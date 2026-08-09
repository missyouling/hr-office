package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

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
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
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
// 2. 构建 prompt（系统提示 + 检索结果 + 用户问题）
// 3. 调用 LLM API（从 model_configs 读取 llm 配置）
// 4. 保存 ChatMessage 记录
// 返回 ChatResponse
func (s *ChatService) Chat(userID uint, sessionID string, question string) (*ChatResponse, error) {
	if strings.TrimSpace(question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	if strings.TrimSpace(sessionID) == "" {
		sessionID = fmt.Sprintf("session-%d-%d", userID, time.Now().UnixNano())
	}

	// 检索相关文档
	sources, err := s.retrievalService.HybridSearch(userID, question, 5)
	if err != nil {
		log.Printf("[chat] hybrid search failed: %v", err)
		sources = []SearchResult{}
	}

	// 构建 prompt
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

	// 保存助手消息
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

// BuildContext 构建问答上下文
// 1. 使用 HybridSearch 检索相关文档
// 2. 组装 systemPrompt 与 userPrompt
// 返回 sources 供上层溯源使用
func (s *ChatService) BuildContext(userID uint, question string, maxRetrieval int) (systemPrompt string, userPrompt string, sources []SearchResult, err error) {
	if maxRetrieval <= 0 {
		maxRetrieval = 5
	}

	// 检索相关文档
	sources, err = s.retrievalService.HybridSearch(userID, question, maxRetrieval)
	if err != nil {
		log.Printf("[chat] BuildContext hybrid search failed: %v", err)
		sources = []SearchResult{}
	}

	// 系统提示
	systemPrompt = `你是一个知识库助手。根据提供的文档内容回答用户的问题。
如果文档中没有相关信息，请说明你无法从文档中找到答案。
请保持回答简洁、准确。`

	// 根据检索结果构建上下文文本
	contextText := ""
	if len(sources) > 0 {
		contextText = "相关文档内容：\n"
		for i, source := range sources {
			contextText += fmt.Sprintf("\n[文档 %d] %s\n%s\n", i+1, source.Title, source.Snippet)
		}
	}

	userPrompt = fmt.Sprintf("%s\n\n用户问题：%s", contextText, question)
	return systemPrompt, userPrompt, sources, nil
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

// StreamChat 流式问答（SSE）
// 1. 获取或创建会话
// 2. 加载上下文配置，读取历史消息
// 3. 构建当前问题的 prompt
// 4. 流式调用 LLM，实时向客户端输出 token
// 5. 保存聊天记录
func (s *ChatService) StreamChat(w http.ResponseWriter, userID uint, sessionID string, question string) {
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

	// 加载上下文配置，默认滑动窗口保留最近 10 条
	ctxConfig := models.ContextConfig{
		MaxTokens:           4000,
		CompressionStrategy: "sliding_window",
		RecentMessageCount:  10,
	}
	if len(session.ContextConfigJSON) > 0 {
		if err := json.Unmarshal(session.ContextConfigJSON, &ctxConfig); err != nil {
			log.Printf("[chat] 解析上下文配置失败: %v", err)
		}
	}

	// 加载历史消息（滑动窗口策略）
	var history []models.ChatMessage
	if ctxConfig.CompressionStrategy == "sliding_window" && ctxConfig.RecentMessageCount > 0 {
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

	// 构建当前问题的上下文 prompt
	systemPrompt, userPrompt, sources, err := s.BuildContext(userID, question, 5)
	if err != nil {
		log.Printf("[chat] StreamChat 构建上下文失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: "构建上下文失败"})
		return
	}

	// 在 systemPrompt 中加入知识库范围说明
	systemPrompt = s.enrichSystemPromptWithScope(systemPrompt, session)

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

	// 流式调用 LLM 并实时推送 token
	var answerBuilder strings.Builder
	err = s.callLLMStream(userID, messages, func(token string) error {
		answerBuilder.WriteString(token)
		return s.sendSSE(w, flusher, sseEvent{Type: "token", Content: token})
	})
	if err != nil {
		log.Printf("[chat] StreamChat LLM 流式调用失败: %v", err)
		s.sendSSE(w, flusher, sseEvent{Type: "error", Content: fmt.Sprintf("调用 LLM 失败: %v", err)})
		return
	}

	// 发送完成事件
	if err := s.sendSSE(w, flusher, sseEvent{Type: "done"}); err != nil {
		log.Printf("[chat] StreamChat 发送 done 事件失败: %v", err)
	}

	// 流式结束后保存聊天记录
	s.saveChatMessages(userID, session.SessionID, question, answerBuilder.String(), sources)
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

// saveChatMessages 保存用户与助手消息
func (s *ChatService) saveChatMessages(userID uint, sessionID string, question, answer string, sources []SearchResult) {
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
	}
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
