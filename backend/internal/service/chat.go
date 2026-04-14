package service

import (
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
	Answer    string        `json:"answer"`
	Sources   []SearchResult `json:"sources"`
	SessionID string        `json:"session_id"`
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
func (s *ChatService) GetLLMConfig(userID uint) (*models.ModelConfig, error) {
	var config models.ModelConfig
	if err := s.db.Where("user_id = ? AND config_type = ? AND enabled = ?", userID, "llm", true).
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

	// 调用 API
	req, err := http.NewRequest("POST", config.APIEndpoint, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[chat] API call failed: %v", err)
		return s.generatePlaceholderResponse(userPrompt), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[chat] API returned %d: %s", resp.StatusCode, string(body))
		return s.generatePlaceholderResponse(userPrompt), nil
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[chat] failed to decode response: %v", err)
		return s.generatePlaceholderResponse(userPrompt), nil
	}

	// 提取答案（假设响应格式为 { "choices": [{ "message": { "content": "..." } }] }）
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	// 占位响应
	return s.generatePlaceholderResponse(userPrompt), nil
}

// generatePlaceholderResponse 生成占位响应
func (s *ChatService) generatePlaceholderResponse(userPrompt string) string {
	return fmt.Sprintf("我已收到您的问题。根据知识库内容，我无法提供完整的答案。请稍后重试或联系管理员配置 LLM 服务。\n\n您的问题：%s", userPrompt)
}
