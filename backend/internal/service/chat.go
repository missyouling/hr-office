package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ChatService 问答服务
type ChatService struct {
	db               *gorm.DB
	retrievalService *RetrievalService
	httpClient       *http.Client
}

// ChatResponse 问答响应
type ChatResponse struct {
	Answer    string         `json:"answer"`
	Sources   []SearchResult `json:"sources"`
	SessionID string         `json:"session_id"`
}

// sseEvent SSE 事件结构
type sseEvent struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	MessageID *uint  `json:"message_id,omitempty"`
}

// NewChatService 构造函数
func NewChatService(db *gorm.DB, retrievalService *RetrievalService) *ChatService {
	return &ChatService{
		db:               db,
		retrievalService: retrievalService,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Chat 问答（检索增强生成 RAG）
// 1. 用 retrievalService.HybridSearch 检索相关文档
// 2. 对检索结果应用字段脱敏（不得把原文送给模型）
// 3. 构建 prompt（系统提示 + 脱敏后的检索结果 + 用户问题）
// 4. 调用 LLM API（从 model_configs 读取 llm 配置）
// 5. 对最终答案应用敏感模式脱敏（防御层）
// 6. 保存 ChatMessage 记录（脱敏后的答案与 Sources）
// 返回 ChatResponse
// kbID=0 表示搜索全部可见知识库（按每条结果所属 KB 脱敏）
func (s *ChatService) Chat(userID uint, sessionID string, question string, kbID uint) (*ChatResponse, error) {
	if strings.TrimSpace(question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	if strings.TrimSpace(sessionID) == "" {
		sessionID = fmt.Sprintf("session-%d-%d", userID, time.Now().UnixNano())
	}

	// 检索相关文档（按 kbID 限定范围，kbID=0 即全部可见）
	sources, err := s.retrievalService.HybridSearch(context.Background(), userID, question, 5, kbID)
	if err != nil {
		log.Printf("[chat] hybrid search failed: %v", err)
		sources = []SearchResult{}
	}

	// 对检索结果应用字段脱敏（不得把原文送给模型；脱敏失败安全失败）
	user, uErr := s.loadUser(userID)
	if uErr != nil {
		return nil, fmt.Errorf("加载用户失败: %w", uErr)
	}
	maskedSources, sensitive, exempt, mErr := s.retrievalService.ApplyMaskToResults(s.db, user, kbID, sources)
	if mErr != nil {
		return nil, fmt.Errorf("检索结果脱敏失败: %w", mErr)
	}
	sources = maskedSources

	// 构建 prompt（使用脱敏后的 sources）
	systemPrompt := `你是一个知识库助手。根据提供的文档内容回答用户的问题。
如果文档中没有相关信息，请说明你无法从文档中找到答案。
请保持回答简洁、准确。`

	contextText := ""
	if len(sources) > 0 {
		contextText = "相关文档内容：\n"
		for i, source := range sources {
			contextText += fmt.Sprintf("\n[文档 %d] %s\n%s\n", i+1, source.Title, source.Snippet)
		}
	}

	userPrompt := fmt.Sprintf("%s\n\n用户问题：%s", contextText, question)

	// 调用 LLM API
	answer, err := s.callLLM(userID, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[chat] LLM call failed: %v", err)
		answer = fmt.Sprintf("抱歉，我无法处理您的问题。错误：%v", err)
	}

	// 对最终答案应用敏感模式脱敏（防御层，防止模型复述原文敏感信息）
	// 再应用 KB 字段规则提取的敏感值映射（address/自定义字段等非正则模式的规则）
	// exempt 中的豁免值在防御层被跳过（ExemptRole 用户应看到原文，避免先替换后无法恢复）
	answer = maskSensitiveTextExempt(answer, toExemptSet(exempt))
	answer = applyValueMap(answer, sensitive)

	// 保存用户消息
	userMsg := &models.ChatMessage{
		UserID:    userID,
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
	}
	if err := s.db.Create(userMsg).Error; err != nil {
		log.Printf("[chat] failed to save user message: %v", err)
	}

	// 保存助手消息（脱敏后的答案与 Sources）
	sourcesJSON, _ := json.Marshal(sources)
	assistantMsg := &models.ChatMessage{
		UserID:    userID,
		SessionID: sessionID,
		Role:      "assistant",
		Content:   answer,
		Sources:   sourcesJSON,
	}
	if err := s.db.Create(assistantMsg).Error; err != nil {
		log.Printf("[chat] failed to save assistant message: %v", err)
	}

	return &ChatResponse{
		Answer:    answer,
		Sources:   sources,
		SessionID: sessionID,
	}, nil
}

// loadUser 按 ID 加载用户（脱敏豁免检查需要）
func (s *ChatService) loadUser(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// BuildContext 构建问答上下文
// 1. 使用 HybridSearch 检索相关文档
// 2. 对检索结果应用字段脱敏（不得把原文送给模型）
// 3. 组装 systemPrompt 与 userPrompt
// 返回脱敏后的 sources 供上层溯源使用；sensitive 为 KB 字段规则提取的
// 原始值→脱敏值 映射，供最终答案/SSE 增量脱敏复用；
// exempt 为 ExemptRole 豁免的原始值列表，供答案/SSE 防御层跳过（保留原文）
// kbID=0 表示搜索全部可见知识库（按每条结果所属 KB 脱敏）
func (s *ChatService) BuildContext(userID uint, question string, maxRetrieval int, kbID uint) (systemPrompt string, userPrompt string, sources []SearchResult, sensitive map[string]string, exempt []string, err error) {
	if maxRetrieval <= 0 {
		maxRetrieval = 5
	}

	// 检索相关文档（按 kbID 限定范围）
	sources, err = s.retrievalService.HybridSearch(context.Background(), userID, question, maxRetrieval, kbID)
	if err != nil {
		log.Printf("[chat] BuildContext hybrid search failed: %v", err)
		sources = []SearchResult{}
	}

	// 对检索结果应用字段脱敏（不得把原文送给模型；脱敏失败安全失败）
	user, uErr := s.loadUser(userID)
	if uErr != nil {
		return "", "", nil, nil, nil, fmt.Errorf("加载用户失败: %w", uErr)
	}
	maskedSources, sensitive, exempt, mErr := s.retrievalService.ApplyMaskToResults(s.db, user, kbID, sources)
	if mErr != nil {
		return "", "", nil, nil, nil, fmt.Errorf("检索结果脱敏失败: %w", mErr)
	}
	sources = maskedSources

	// 系统提示
	systemPrompt = `你是一个知识库助手。根据提供的文档内容回答用户的问题。
如果文档中没有相关信息，请说明你无法从文档中找到答案。
请保持回答简洁、准确。`

	// 根据检索结果构建上下文文本（使用脱敏后的 sources）
	contextText := ""
	if len(sources) > 0 {
		contextText = "相关文档内容：\n"
		for i, source := range sources {
			contextText += fmt.Sprintf("\n[文档 %d] %s\n%s\n", i+1, source.Title, source.Snippet)
		}
	}

	userPrompt = fmt.Sprintf("%s\n\n用户问题：%s", contextText, question)
	return systemPrompt, userPrompt, sources, sensitive, exempt, nil
}

// GetOrCreateSession 获取或创建会话
// 若 sessionID 为空或不存在，则新建会话；标题使用问题首行截断
func (s *ChatService) GetOrCreateSession(userID uint, sessionID string, question string) (*models.ChatSession, error) {
	// 先尝试按 sessionID 查询已有会话
	if strings.TrimSpace(sessionID) != "" {
		var existing models.ChatSession
		if err := s.db.Where("user_id = ? AND session_id = ?", userID, sessionID).First(&existing).Error; err == nil {
			return &existing, nil
		} else if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("查询会话失败: %v", err)
		}
	}

	// 生成新的 sessionID
	if strings.TrimSpace(sessionID) == "" {
		sessionID = fmt.Sprintf("session-%d-%d", userID, time.Now().UnixNano())
	}

	// 标题：首行截断，最多 100 字符
	title := strings.TrimSpace(question)
	if idx := strings.IndexAny(title, "\n\r"); idx > 0 {
		title = title[:idx]
	}
	if len(title) > 100 {
		title = title[:100]
	}
	if title == "" {
		title = "新会话"
	}

	session := models.ChatSession{
		UserID:    userID,
		SessionID: sessionID,
		Title:     title,
	}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}
	return &session, nil
}

// SummarizeHistory 对历史消息生成摘要
// 使用非流式 LLM 调用，返回简洁摘要
func (s *ChatService) SummarizeHistory(userID uint, history []models.ChatMessage) (string, error) {
	if len(history) == 0 {
		return "", nil
	}

	// 构建摘要 prompt
	var historyText strings.Builder
	for _, msg := range history {
		historyText.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	systemPrompt := "你是一个对话摘要助手。请将以下对话历史压缩为一段简洁的摘要（不超过200字），保留关键信息和上下文。"
	userPrompt := fmt.Sprintf("对话历史：\n%s\n请生成摘要：", historyText.String())

	// 使用 callLLM 生成摘要
	summary, err := s.callLLM(userID, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[chat] 生成摘要失败: %v", err)
		return "", err
	}
	return summary, nil
}

// CompressContext 对历史消息执行智能压缩
// 策略: 保留最近 recentCount 条消息，更早的压缩为摘要
// 返回: (摘要文本, 最近消息列表, error)
func (s *ChatService) CompressContext(session *models.ChatSession, userID uint, recentCount int) (string, []models.ChatMessage, error) {
	// 加载所有历史消息
	var allHistory []models.ChatMessage
	if err := s.db.Where("user_id = ? AND session_id = ?", userID, session.SessionID).
		Order("created_at ASC").
		Find(&allHistory).Error; err != nil {
		return "", nil, err
	}

	if len(allHistory) <= recentCount {
		// 消息不足，无需压缩
		return session.Summary, allHistory, nil
	}

	// 分割：旧消息 → 摘要，新消息保留
	splitIdx := len(allHistory) - recentCount
	oldMessages := allHistory[:splitIdx]
	recentMessages := allHistory[splitIdx:]

	// 生成摘要（包含已有摘要）
	summary, err := s.SummarizeHistory(userID, oldMessages)
	if err != nil {
		// 降级：直接截断
		log.Printf("[chat] 摘要失败，降级为截断: %v", err)
		return session.Summary, recentMessages, nil
	}

	// 保存摘要到会话
	session.Summary = summary
	s.db.Model(session).Update("summary", summary)

	return summary, recentMessages, nil
}

// RewriteQuery 检测追问意图并改写查询
// 检测代词（他/她/这个/那个/这些）和省略句
// 如果检测到追问，使用 LLM 融合历史上下文改写 query
// 返回: (改写后的 query, 是否为追问, error)
func (s *ChatService) RewriteQuery(userID uint, sessionID string, question string) (string, bool, error) {
	// 快速规则检测
	trimmed := strings.TrimSpace(question)
	hasPronoun := false
	pronouns := []string{"他", "她", "它", "这个", "那个", "这些", "那些", "这里", "那里", "这样", "那样", "这么", "那么", "这方面", "那方面"}
	for _, p := range pronouns {
		if strings.Contains(trimmed, p) {
			hasPronoun = true
			break
		}
	}

	// 短句也可能是追问（如"为什么"、"具体呢"）
	isShort := len([]rune(trimmed)) <= 6

	if !hasPronoun && !isShort {
		return question, false, nil
	}

	// 获取最后一条用户消息作为上下文
	var lastMsg models.ChatMessage
	if err := s.db.Where("user_id = ? AND session_id = ? AND role = ?", userID, sessionID, "user").
		Order("created_at DESC").First(&lastMsg).Error; err != nil {
		// 没有历史，直接返回原问题
		return question, false, nil
	}

	// 使用 LLM 改写 query
	systemPrompt := "你是一个查询改写助手。根据对话上下文，将用户的追问改写为完整、独立的问题。只返回改写后的问题，不要加任何解释或额外文字。"
	userPrompt := fmt.Sprintf("上一条问题: %s\n当前追问: %s\n请改写为完整问题:", lastMsg.Content, trimmed)

	rewritten, err := s.callLLM(userID, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[chat] query rewrite failed: %v", err)
		return question, false, nil
	}

	rewritten = strings.TrimSpace(rewritten)
	if rewritten == "" || rewritten == question {
		return question, false, nil
	}

	log.Printf("[chat] query rewritten: %q -> %q", question, rewritten)
	return rewritten, true, nil
}

// StreamChat 流式问答（SSE）
// 1. 获取或创建会话
// 2. 加载上下文配置，读取历史消息
// 3. 构建当前问题的 prompt（透传 kbID 限定检索范围，检索结果先脱敏再送模型）
// 4. 流式调用 LLM，对增量 token 应用敏感模式脱敏后实时向客户端输出
// 5. 保存聊天记录（脱敏后的答案与 Sources）
// kbID=0 表示搜索全部可见知识库（按每条结果所属 KB 脱敏）
func (s *ChatService) StreamChat(w http.ResponseWriter, userID uint, sessionID string, question string, kbID uint) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "服务器不支持流式响应", http.StatusInternalServerError)
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if strings.TrimSpace(question) == "" {
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: "问题不能为空"})
		return
	}

	// 获取或创建会话
	session, err := s.GetOrCreateSession(userID, sessionID, question)
	if err != nil {
		log.Printf("[chat] StreamChat 获取会话失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: "会话创建失败"})
		return
	}

	// 加载上下文配置，默认智能摘要保留最近 10 条
	ctxConfig := models.ContextConfig{
		MaxTokens:           4000,
		CompressionStrategy: "smart_summarize",
		RecentMessageCount:  10,
	}
	if len(session.ContextConfigJSON) > 0 {
		if err := json.Unmarshal(session.ContextConfigJSON, &ctxConfig); err != nil {
			log.Printf("[chat] 解析上下文配置失败: %v", err)
		}
	}

	// 追问意图识别：检测代词/省略句，融合历史上下文改写 query
	rewritten, isFollowUp, err := s.RewriteQuery(userID, session.SessionID, question)
	if err == nil && isFollowUp {
		question = rewritten
	}

	// 加载历史消息：智能摘要策略优先，否则回退到滑动窗口
	var history []models.ChatMessage
	var summary string
	if ctxConfig.CompressionStrategy == "smart_summarize" && ctxConfig.RecentMessageCount > 0 {
		summary, history, err = s.CompressContext(session, userID, ctxConfig.RecentMessageCount)
		if err != nil {
			log.Printf("[chat] 智能压缩失败: %v", err)
			summary, history = session.Summary, []models.ChatMessage{}
		}
	} else if ctxConfig.CompressionStrategy == "sliding_window" && ctxConfig.RecentMessageCount > 0 {
		if err := s.db.Where("user_id = ? AND session_id = ?", userID, session.SessionID).
			Order("created_at DESC").
			Limit(ctxConfig.RecentMessageCount).
			Find(&history).Error; err != nil {
			log.Printf("[chat] 加载历史消息失败: %v", err)
		} else {
			// 恢复为时间正序
			for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	// 构建当前问题的上下文 prompt（透传 kbID 限定检索范围）
	systemPrompt, userPrompt, sources, sensitive, exempt, err := s.BuildContext(userID, question, 5, kbID)
	if err != nil {
		log.Printf("[chat] StreamChat 构建上下文失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: "构建上下文失败"})
		return
	}

	// 在 systemPrompt 中加入知识库范围说明
	systemPrompt = s.enrichSystemPromptWithScope(systemPrompt, session)

	// 如果有摘要，在 systemPrompt 中追加
	if summary != "" {
		systemPrompt = fmt.Sprintf("%s\n\n对话历史摘要：%s", systemPrompt, summary)
	}

	// 组装 LLM messages：system + 历史消息 + 当前问题
	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}
	for _, msg := range history {
		if msg.Role == "user" || msg.Role == "assistant" {
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userPrompt,
	})

	// 流式调用 LLM 并实时推送 token（增量敏感模式脱敏，防止流式输出泄露原文；
	// exempt 豁免值在防御层被跳过，ExemptRole 用户流式输出保留原文）
	var answerBuilder strings.Builder
	masker := &streamMasker{sensitive: sensitive, exempt: toExemptSet(exempt)}
	err = s.callLLMStream(userID, messages, func(token string) error {
		safe := masker.Push(token)
		if safe == "" {
			return nil
		}
		answerBuilder.WriteString(safe)
		return s.sendSSE(w, flusher, sseEvent{Type: "token", Content: safe})
	})
	if err != nil {
		log.Printf("[chat] StreamChat LLM 流式调用失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: fmt.Sprintf("调用 LLM 失败: %v", err)})
		return
	}

	// 输出剩余缓冲（整体脱敏）
	if tail := masker.Flush(); tail != "" {
		answerBuilder.WriteString(tail)
		if err := s.sendSSE(w, flusher, sseEvent{Type: "token", Content: tail}); err != nil {
			log.Printf("[chat] StreamChat 发送尾部 token 失败: %v", err)
		}
	}

	// 流式结束后保存聊天记录（脱敏后的答案与 Sources）
	assistantMsg, err := s.saveChatMessages(userID, session.SessionID, question, answerBuilder.String(), sources)
	if err != nil {
		log.Printf("[chat] StreamChat 保存助手消息失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: "保存助手消息失败"})
		return
	}

	// 仅在助手消息成功保存后回传真实消息 ID。
	if err := s.sendSSE(w, flusher, sseEvent{Type: "done", MessageID: &assistantMsg.ID}); err != nil {
		log.Printf("[chat] StreamChat 发送 done 事件失败: %v", err)
	}
}

// sendSSE 向客户端发送一个 SSE 事件并立即刷新
func (s *ChatService) sendSSE(w http.ResponseWriter, flusher http.Flusher, ev sseEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// enrichSystemPromptWithScope 根据会话范围配置在 systemPrompt 中追加知识库范围说明
func (s *ChatService) enrichSystemPromptWithScope(systemPrompt string, session *models.ChatSession) string {
	if len(session.ScopeConfigJSON) == 0 {
		return systemPrompt
	}

	var scope models.KBAccessScope
	if err := json.Unmarshal(session.ScopeConfigJSON, &scope); err != nil {
		log.Printf("[chat] 解析范围配置失败: %v", err)
		return systemPrompt
	}

	var parts []string
	if len(scope.CategoryCodes) > 0 {
		parts = append(parts, fmt.Sprintf("一级分类：%s", strings.Join(scope.CategoryCodes, "、")))
	}
	if len(scope.SubCategoryCodes) > 0 {
		parts = append(parts, fmt.Sprintf("二级分类：%s", strings.Join(scope.SubCategoryCodes, "、")))
	}
	if len(scope.FolderPaths) > 0 {
		parts = append(parts, fmt.Sprintf("文件夹：%s", strings.Join(scope.FolderPaths, "、")))
	}
	if len(scope.TagNames) > 0 {
		parts = append(parts, fmt.Sprintf("标签：%s", strings.Join(scope.TagNames, "、")))
	}
	if len(scope.DocumentIDs) > 0 {
		parts = append(parts, fmt.Sprintf("指定文档 ID：%v", scope.DocumentIDs))
	}
	if len(parts) == 0 {
		return systemPrompt
	}

	return fmt.Sprintf("%s\n\n本次问答限定在以下知识库范围内，请仅依据范围内的文档作答：\n%s",
		systemPrompt, strings.Join(parts, "\n"))
}

// saveChatMessages 保存用户与助手消息，并返回成功保存的助手消息。
func (s *ChatService) saveChatMessages(userID uint, sessionID string, question, answer string, sources []SearchResult) (*models.ChatMessage, error) {
	userMsg := &models.ChatMessage{
		UserID:    userID,
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
	}
	if err := s.db.Create(userMsg).Error; err != nil {
		log.Printf("[chat] 保存用户消息失败: %v", err)
	}

	sourcesJSON, _ := json.Marshal(sources)
	assistantMsg := &models.ChatMessage{
		UserID:    userID,
		SessionID: sessionID,
		Role:      "assistant",
		Content:   answer,
		Sources:   sourcesJSON,
	}
	if err := s.db.Create(assistantMsg).Error; err != nil {
		log.Printf("[chat] 保存助手消息失败: %v", err)
		return nil, err
	}
	return assistantMsg, nil
}

// GetChatHistory 获取会话历史
func (s *ChatService) GetChatHistory(userID uint, sessionID string) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	if err := s.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return []models.ChatMessage{}, fmt.Errorf("failed to fetch chat history: %v", err)
	}
	return messages, nil
}

// GetLLMConfig 获取 LLM 配置
// 查询策略：优先查询当前用户的配置，如果没有则查询全局配置（user_id为NULL）
func (s *ChatService) GetLLMConfig(userID uint) (*models.ModelConfig, error) {
	var config models.ModelConfig

	// 优先查询当前用户的LLM配置
	if err := s.db.Where("user_id = ? AND config_type = ? AND enabled = ?", userID, "llm", true).
		Order("is_default DESC, created_at DESC").
		First(&config).Error; err == nil {
		return &config, nil
	}

	// 如果用户没有配置，查询全局配置（user_id为NULL）
	if err := s.db.Where("user_id IS NULL AND config_type = ? AND enabled = ?", "llm", true).
		Order("is_default DESC, created_at DESC").
		First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// parseExtraParams 安全解析 ModelConfig.ExtraParams（datatypes.JSON）为 map。
// 空值返回空 map；无效 JSON 返回带上下文的错误，不静默忽略。
func parseExtraParams(raw datatypes.JSON) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var params map[string]interface{}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("解析模型配置 ExtraParams 失败: %w", err)
	}
	return params, nil
}

// callLLM 调用 LLM API
func (s *ChatService) callLLM(userID uint, systemPrompt, userPrompt string) (string, error) {
	startTime := time.Now()

	config, err := s.GetLLMConfig(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM config: %v", err)
	}

	if config == nil {
		// 返回占位响应
		log.Printf("[chat] no LLM config found for user %d, returning placeholder", userID)
		return s.generatePlaceholderResponse(userPrompt), nil
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
	}

	// 合并 ExtraParams 到请求体顶层（先合并，再显式写回核心键保证权威）
	extraParams, err := parseExtraParams(config.ExtraParams)
	if err != nil {
		return "", err
	}
	for k, v := range extraParams {
		reqBody[k] = v
	}
	reqBody["model"] = config.ModelName
	reqBody["messages"] = []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userPrompt},
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// 调用 API（根据 provider 拼接完整 URL）
	apiURL := config.APIEndpoint

	if config.Provider == "siliconflow" {
		// Siliconflow 需要完整的 chat/completions 路径
		if !strings.HasSuffix(apiURL, "/chat/completions") {
			apiURL = strings.TrimSuffix(apiURL, "/") + "/chat/completions"
		}
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		// Log failed call
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), err.Error())
		log.Printf("[chat] API call failed: %v", err)
		return s.generatePlaceholderResponse(userPrompt), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), fmt.Sprintf("HTTP %d", resp.StatusCode))
		log.Printf("[chat] API returned %d: %s", resp.StatusCode, string(body))
		return s.generatePlaceholderResponse(userPrompt), nil
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), err.Error())
		log.Printf("[chat] failed to decode response: %v", err)
		return s.generatePlaceholderResponse(userPrompt), nil
	}

	// Extract token usage from response
	inputTokens, outputTokens := 0, 0
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			outputTokens = int(ct)
		}
	}

	// 提取答案（假设响应格式为 { "choices": [{ "message": { "content": "..." } }] }）
	answer := ""
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					answer = content
				}
			}
		}
	}

	// Record successful usage
	go s.recordUsage(userID, config, "success", inputTokens, outputTokens, int(elapsed.Milliseconds()), "")

	if answer == "" {
		return s.generatePlaceholderResponse(userPrompt), nil
	}
	return answer, nil
}

// callLLMStream 流式调用 LLM API
// 请求体中加入 stream: true，逐行读取 SSE 响应，对每个 content token 调用 callback
func (s *ChatService) callLLMStream(userID uint, messages []map[string]string, onToken func(string) error) error {
	startTime := time.Now()

	config, err := s.GetLLMConfig(userID)
	if err != nil {
		return fmt.Errorf("获取 LLM 配置失败: %v", err)
	}
	if config == nil {
		return fmt.Errorf("未找到可用的 LLM 配置")
	}

	// 构建流式请求体
	reqBody := map[string]interface{}{
		"model":    config.ModelName,
		"messages": messages,
		"stream":   true,
	}

	// 合并 ExtraParams 到请求体顶层（先合并，再显式写回核心键保证权威）
	extraParams, err := parseExtraParams(config.ExtraParams)
	if err != nil {
		return err
	}
	for k, v := range extraParams {
		reqBody[k] = v
	}
	reqBody["model"] = config.ModelName
	reqBody["messages"] = messages
	reqBody["stream"] = true

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	// 调用 API（根据 provider 拼接完整 URL）
	apiURL := config.APIEndpoint
	if config.Provider == "siliconflow" {
		// Siliconflow 需要完整的 chat/completions 路径
		if !strings.HasSuffix(apiURL, "/chat/completions") {
			apiURL = strings.TrimSuffix(apiURL, "/") + "/chat/completions"
		}
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), err.Error())
		return fmt.Errorf("LLM API 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), fmt.Sprintf("HTTP %d", resp.StatusCode))
		return fmt.Errorf("LLM API 返回错误: HTTP %d, %s", resp.StatusCode, string(body))
	}

	// 逐行读取流式响应
	reader := bufio.NewReader(resp.Body)
	outputTokens := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			go s.recordUsage(userID, config, "failed", 0, outputTokens, int(elapsed.Milliseconds()), err.Error())
			return fmt.Errorf("读取流式响应失败: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "data: [DONE]" {
			break
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("[chat] 解析流式 chunk 失败: %v", err)
			continue
		}

		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok && content != "" {
						outputTokens++
						if err := onToken(content); err != nil {
							go s.recordUsage(userID, config, "failed", 0, outputTokens, int(elapsed.Milliseconds()), err.Error())
							return fmt.Errorf("推送 token 失败: %v", err)
						}
					}
				}
			}
		}
	}

	go s.recordUsage(userID, config, "success", 0, outputTokens, int(elapsed.Milliseconds()), "")
	return nil
}

// recordUsage records model usage asynchronously
func (s *ChatService) recordUsage(userID uint, config *models.ModelConfig, status string, inputTokens, outputTokens, durationMs int, errMsg string) {
	usageLog := &models.ModelUsageLog{
		UserID:       userID,
		ConfigID:     config.ID,
		ModelName:    config.ModelName,
		Provider:     config.Provider,
		ConfigType:   "llm",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Status:       status,
		ErrorMsg:     errMsg,
		DurationMs:   durationMs,
	}
	usageLog.CostUSD = usageLog.CalculateCost()
	if err := s.db.Create(usageLog).Error; err != nil {
		log.Printf("[chat] failed to record usage: %v", err)
	}
}

// generatePlaceholderResponse 生成占位响应
func (s *ChatService) generatePlaceholderResponse(userPrompt string) string {
	return fmt.Sprintf("我已收到您的问题。根据知识库内容，我无法提供完整的答案。请稍后重试或联系管理员配置 LLM 服务。\n\n您的问题：%s", userPrompt)
}
