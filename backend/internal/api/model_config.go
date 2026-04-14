package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// ProviderConfig 内置提供商配置
type ProviderConfig struct {
	Name           string `json:"name"`
	Endpoint       string `json:"endpoint"`
	ModelsEndpoint string `json:"models_endpoint"`
	AuthType       string `json:"auth_type"`
}

// BuiltInProviders 内置提供商定义
var BuiltInProviders = map[string]ProviderConfig{
	"openai":      {Name: "OpenAI", Endpoint: "https://api.openai.com/v1", ModelsEndpoint: "https://api.openai.com/v1/models", AuthType: "bearer"},
	"anthropic":   {Name: "Anthropic", Endpoint: "https://api.anthropic.com", ModelsEndpoint: "https://api.anthropic.com/v1/models", AuthType: "x-api-key"},
	"azure":       {Name: "Azure OpenAI", Endpoint: "", ModelsEndpoint: "", AuthType: "azure"},
	"qwen":        {Name: "阿里Qwen", Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1", ModelsEndpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1/models", AuthType: "bearer"},
	"zhipuai":     {Name: "智谱AI", Endpoint: "https://open.bigmodel.cn/api/paas/v4", ModelsEndpoint: "https://open.bigmodel.cn/api/paas/v4/models", AuthType: "bearer"},
	"cohere":      {Name: "Cohere", Endpoint: "https://api.cohere.ai/v1", ModelsEndpoint: "https://api.cohere.ai/v1/models", AuthType: "bearer"},
	"siliconflow": {Name: "硅基流动", Endpoint: "https://api.siliconflow.cn/v1", ModelsEndpoint: "https://api.siliconflow.cn/v1/models", AuthType: "bearer"},
}

func validateModelCapabilities(configType, modelName, apiEndpoint, apiKey string) string {
	if apiEndpoint == "" || apiKey == "" {
		return ""
	}

	base := strings.TrimRight(apiEndpoint, "/")
	if strings.Contains(base, "/ocr/jobs") || strings.Contains(base, "paddleocr") {
		return ""
	}

	var apiURL string
	if strings.Contains(base, "/models") {
		apiURL = base
	} else if strings.Contains(base, "/v1") {
		apiURL = base + "/models"
	} else {
		apiURL = base + "/v1/models"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "验证失败：无法连接到模型服务，请检查端点和Key是否正确"
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "验证失败：无法连接到模型服务，请检查端点和Key是否正确"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "验证失败：模型服务返回错误，请检查配置"
	}

	return ""
}

func detectModelCapabilities(apiEndpoint, apiKey, modelName string) (detectedType string, errMsg string) {
	if apiEndpoint == "" || apiKey == "" {
		return "", ""
	}

	base := strings.TrimRight(apiEndpoint, "/")

	if strings.Contains(base, "/ocr/jobs") || strings.Contains(base, "paddleocr") {
		return "ocr", ""
	}

	var testURL string
	if strings.Contains(base, "/models") {
		testURL = base
	} else if strings.Contains(base, "/v1") {
		testURL = base + "/models"
	} else {
		testURL = base + "/v1/models"
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return "", "无法连接到模型服务"
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", "无法连接到模型服务"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "无法连接到模型服务"
	}

	nameLower := strings.ToLower(modelName)

	switch {
	case strings.Contains(nameLower, "embedding") ||
		strings.Contains(nameLower, "bge-") ||
		strings.Contains(nameLower, "text-vec"):
		return "embedding", ""

	case strings.Contains(nameLower, "rerank") ||
		strings.Contains(nameLower, "bge-rerank"):
		return "rerank", ""

	case strings.Contains(nameLower, "ocr") ||
		strings.Contains(nameLower, "paddle") ||
		strings.Contains(nameLower, "vision") ||
		strings.Contains(nameLower, "layout"):
		return "ocr", ""

	case strings.Contains(nameLower, "gpt") ||
		strings.Contains(nameLower, "claude") ||
		strings.Contains(nameLower, "qwen") ||
		strings.Contains(nameLower, "chat") ||
		strings.Contains(nameLower, "instruct") ||
		strings.Contains(nameLower, "llama") ||
		strings.Contains(nameLower, "gemini") ||
		strings.Contains(nameLower, "deepseek"):
		return "llm", ""

	default:
		return "", ""
	}
}

// maskAPIKey masks API key for security (show first 4 chars + ****)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 4 {
		return "****"
	}
	return apiKey[:4] + "****"
}

// listModelConfigs 获取所有模型配置
// GET /api/settings/models?config_type=ocr|llm|embedding|rerank
func (h *Handler) ListModelConfigs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configType := r.URL.Query().Get("config_type")

	var configs []models.ModelConfig
	query := h.db.Where("user_id = ?", userID)

	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}

	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load model configs", err)
		return
	}

	// Mask API keys in response
	for i := range configs {
		configs[i].APIKey = maskAPIKey(configs[i].APIKey)
	}

	respondJSON(w, http.StatusOK, configs)
}

// createModelConfigRequest request body for creating model config
type createModelConfigRequest struct {
	ConfigType    string                 `json:"config_type"`
	Provider      string                 `json:"provider"`
	ModelName     string                 `json:"model_name"`
	APIKey        string                 `json:"api_key"`
	APIEndpoint   string                 `json:"api_endpoint"`
	ExtraParams   map[string]interface{} `json:"extra_params"`
	Enabled       bool                   `json:"enabled"`
	IsDefault     bool                   `json:"is_default"`
	Role          string                 `json:"role"`
	IsBuiltIn     bool                   `json:"is_built_in"`
	Priority      int                    `json:"priority"`
	ContextLength string                 `json:"context_length"`
	Capabilities  string                 `json:"capabilities"`
	RateLimitRPM  int                    `json:"rate_limit_rpm"`
	RateLimitTPM  int                    `json:"rate_limit_tpm"`
}

// createModelConfig 创建模型配置
// POST /api/settings/models
func (h *Handler) CreateModelConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req createModelConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// Validate required fields
	if req.ConfigType == "" || req.Provider == "" || req.ModelName == "" || req.APIKey == "" || req.APIEndpoint == "" {
		respondError(w, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	// Convert extra params to JSON
	var extraParamsJSON datatypes.JSON
	if req.ExtraParams != nil {
		data, err := json.Marshal(req.ExtraParams)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid extra_params", err)
			return
		}
		extraParamsJSON = data
	}

	if validationError := validateModelCapabilities(req.ConfigType, req.ModelName, req.APIEndpoint, req.APIKey); validationError != "" {
		respondError(w, http.StatusBadRequest, validationError, nil)
		return
	}

	detectedType, detectErr := detectModelCapabilities(req.APIEndpoint, req.APIKey, req.ModelName)
	if detectErr != "" {
		respondError(w, http.StatusBadRequest, detectErr, nil)
		return
	}
	if detectedType != "" && detectedType != req.ConfigType {
		typeNames := map[string]string{
			"llm":       "大语言模型",
			"embedding": "向量模型",
			"rerank":    "重排模型",
			"ocr":       "OCR模型",
		}
		respondError(w, http.StatusBadRequest,
			fmt.Sprintf("检测到该模型为%s，请选择正确的配置类型", typeNames[detectedType]), nil)
		return
	}

	// 如果没有设置priority，设置为最低
	if req.Priority == 0 {
		var maxPriority int64
		h.db.Model(&models.ModelConfig{}).
			Where("user_id = ? AND config_type = ?", userID, req.ConfigType).
			Select("COALESCE(MAX(priority), 0)").
			Scan(&maxPriority)
		req.Priority = int(maxPriority) + 1
	}

	// If setting as default, unset other defaults for this config type
	if req.IsDefault {
		if err := h.db.Model(&models.ModelConfig{}).
			Where("user_id = ? AND config_type = ?", userID, req.ConfigType).
			Update("is_default", false).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update default config", err)
			return
		}
	}

	config := models.ModelConfig{
		UserID:        userID,
		ConfigType:    req.ConfigType,
		Provider:      req.Provider,
		ModelName:     req.ModelName,
		APIKey:        req.APIKey,
		APIEndpoint:   req.APIEndpoint,
		ExtraParams:   extraParamsJSON,
		Enabled:       req.Enabled,
		IsDefault:     req.IsDefault,
		Role:          req.Role,
		IsBuiltIn:     req.IsBuiltIn,
		Priority:      req.Priority,
		ContextLength: req.ContextLength,
		Capabilities:  req.Capabilities,
		RateLimitRPM:  req.RateLimitRPM,
		RateLimitTPM:  req.RateLimitTPM,
	}

	if err := h.db.Create(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create model config", err)
		return
	}

	// Mask API key in response
	config.APIKey = maskAPIKey(config.APIKey)

	respondJSON(w, http.StatusCreated, config)
}

// updateModelConfigRequest request body for updating model config
type updateModelConfigRequest struct {
	ConfigType  string                 `json:"config_type"`
	Provider    string                 `json:"provider"`
	ModelName   string                 `json:"model_name"`
	APIKey      string                 `json:"api_key"`
	APIEndpoint string                 `json:"api_endpoint"`
	ExtraParams map[string]interface{} `json:"extra_params"`
	Enabled     bool                   `json:"enabled"`
	IsDefault   bool                   `json:"is_default"`
}

// updateModelConfig 更新模型配置
// PUT /api/settings/models/{configId}
func (h *Handler) UpdateModelConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configIDStr := chi.URLParam(r, "configId")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid config id", err)
		return
	}

	var req updateModelConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// Check if config exists and belongs to user
	var config models.ModelConfig
	if err := h.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "model config not found", err)
		return
	}

	// Update fields
	updates := map[string]interface{}{
		"config_type":  req.ConfigType,
		"provider":     req.Provider,
		"model_name":   req.ModelName,
		"api_endpoint": req.APIEndpoint,
		"enabled":      req.Enabled,
		"is_default":   req.IsDefault,
	}

	// Only update API key if provided and not masked
	if req.APIKey != "" && !strings.HasSuffix(req.APIKey, "****") {
		updates["api_key"] = req.APIKey
	}

	// Update extra params if provided
	if req.ExtraParams != nil {
		data, err := json.Marshal(req.ExtraParams)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid extra_params", err)
			return
		}
		updates["extra_params"] = datatypes.JSON(data)
	}

	// If setting as default, unset other defaults for this config type
	if req.IsDefault && config.ConfigType != req.ConfigType {
		if err := h.db.Model(&models.ModelConfig{}).
			Where("user_id = ? AND config_type = ? AND id != ?", userID, req.ConfigType, configID).
			Update("is_default", false).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update default config", err)
			return
		}
	} else if req.IsDefault && config.ConfigType == req.ConfigType {
		if err := h.db.Model(&models.ModelConfig{}).
			Where("user_id = ? AND config_type = ? AND id != ?", userID, req.ConfigType, configID).
			Update("is_default", false).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update default config", err)
			return
		}
	}

	if err := h.db.Model(&config).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update model config", err)
		return
	}

	// Reload to get updated data
	if err := h.db.First(&config, configID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload model config", err)
		return
	}

	// Mask API key in response
	config.APIKey = maskAPIKey(config.APIKey)

	respondJSON(w, http.StatusOK, config)
}

// deleteModelConfig 删除模型配置
// DELETE /api/settings/models/{configId}
func (h *Handler) DeleteModelConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configIDStr := chi.URLParam(r, "configId")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid config id", err)
		return
	}

	// Check if config exists and belongs to user
	var config models.ModelConfig
	if err := h.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "model config not found", err)
		return
	}

	if err := h.db.Delete(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete model config", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "model config deleted successfully"})
}

// testModelConfigRequest request body for testing model config
type testModelConfigRequest struct {
	ConfigID uint `json:"config_id"`
}

// testModelConfig 测试模型配置连通性
// POST /api/settings/models/{configId}/test
func (h *Handler) TestModelConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configIDStr := chi.URLParam(r, "configId")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid config id", err)
		return
	}

	// Check if config exists and belongs to user
	var config models.ModelConfig
	if err := h.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "model config not found", err)
		return
	}

	// Simple HTTP connectivity test: GET endpoint and check for non-5xx status
	client := &http.Client{}
	req, err := http.NewRequest("GET", config.APIEndpoint, nil)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("invalid endpoint: %v", err),
		})
		return
	}

	// Add API key as Authorization header if present
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("connection failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Check if status code is 5xx
	if resp.StatusCode >= 500 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("server error: HTTP %d", resp.StatusCode),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("connection successful: HTTP %d", resp.StatusCode),
	})
}

// ListBuiltInProviders 获取内置提供商列表
// GET /api/settings/models/providers
func (h *Handler) ListBuiltInProviders(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, BuiltInProviders)
}

// AvailableModel 可用模型信息
type AvailableModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

// ListAvailableModels 获取提供商的可用模型列表
// GET /api/settings/models/{configId}/available-models
func (h *Handler) ListAvailableModels(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configIDStr := chi.URLParam(r, "configId")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid config id", err)
		return
	}

	// 获取ModelConfig
	var config models.ModelConfig
	if err := h.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "model config not found", err)
		return
	}

	// 根据provider查找BuiltInProviders获取modelsEndpoint
	provider, ok := BuiltInProviders[config.Provider]
	if !ok || provider.ModelsEndpoint == "" {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}

	// 使用APIKey调用modelsEndpoint
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", provider.ModelsEndpoint, nil)
	if err != nil {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}

	// 根据AuthType添加认证头
	switch provider.AuthType {
	case "bearer":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	case "x-api-key":
		req.Header.Set("X-API-Key", config.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}

	// 解析返回的模型列表
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}

	// 根据不同提供商解析响应格式
	var models []AvailableModel
	switch config.Provider {
	case "openai":
		var openaiResp struct {
			Data []struct {
				ID    string `json:"id"`
				Owned bool   `json:"owned_by"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &openaiResp); err == nil {
			for _, m := range openaiResp.Data {
				models = append(models, AvailableModel{
					ID:       m.ID,
					Name:     m.ID,
					Disabled: false,
				})
			}
		}
	case "siliconflow":
		var sfResp struct {
			Data []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Owned bool   `json:"owned_by"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &sfResp); err == nil {
			for _, m := range sfResp.Data {
				models = append(models, AvailableModel{
					ID:       m.ID,
					Name:     m.Name,
					Disabled: false,
				})
			}
		}
	default:
		// 通用格式处理
		var genericResp struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &genericResp); err == nil {
			for _, m := range genericResp.Data {
				models = append(models, AvailableModel{
					ID:       m.ID,
					Name:     m.Name,
					Disabled: false,
				})
			}
		}
	}

	respondJSON(w, http.StatusOK, models)
}

// FetchModelsByEndpoint 通过自定义 endpoint + api_key 预览可用模型列表（无需已保存的 configId）
// GET /api/settings/models/fetch-models?endpoint=xxx&api_key=xxx
func (h *Handler) FetchModelsByEndpoint(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Query().Get("endpoint")
	apiKey := r.URL.Query().Get("api_key")

	if endpoint == "" || apiKey == "" {
		respondError(w, http.StatusBadRequest, "endpoint and api_key are required", nil)
		return
	}

	base := strings.TrimRight(endpoint, "/")
	var modelsURL string

	if strings.Contains(base, "/ocr/jobs") || strings.Contains(base, "paddleocr") {
		respondJSON(w, http.StatusOK, []AvailableModel{})
		return
	}

	if strings.Contains(base, "/models") {
		modelsURL = base
	} else if strings.Contains(base, "/v1") {
		modelsURL = base + "/models"
	} else {
		modelsURL = base + "/v1/models"
	}

	maskLen := 10
	if len(apiKey) < maskLen {
		maskLen = len(apiKey)
	}
	log.Printf("FetchModelsByEndpoint: url=%s key=%s...", modelsURL, apiKey[:maskLen])

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		log.Printf("FetchModelsByEndpoint: build request error: %v", err)
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid endpoint URL: %v", err), nil)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("FetchModelsByEndpoint: HTTP error: %v", err)
		respondError(w, http.StatusBadGateway, fmt.Sprintf("request failed: %v", err), nil)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	preview := string(body)
	if len(preview) > 300 {
		preview = preview[:300]
	}
	log.Printf("FetchModelsByEndpoint: status=%d body_preview=%s", resp.StatusCode, preview)

	if resp.StatusCode != http.StatusOK {
		respondError(w, http.StatusBadGateway,
			fmt.Sprintf("upstream returned HTTP %d: %s", resp.StatusCode, preview), nil)
		return
	}

	var openaiResp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		log.Printf("FetchModelsByEndpoint: unmarshal error: %v", err)
		respondError(w, http.StatusBadGateway,
			fmt.Sprintf("failed to parse response: %v", err), nil)
		return
	}

	availableModels := make([]AvailableModel, 0, len(openaiResp.Data))
	for _, m := range openaiResp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		availableModels = append(availableModels, AvailableModel{ID: m.ID, Name: name})
	}

	log.Printf("FetchModelsByEndpoint: found %d models", len(availableModels))
	respondJSON(w, http.StatusOK, availableModels)
}
