package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// OCRService 封装 OCR 调用逻辑
type OCRService struct {
	db         *gorm.DB
	httpClient *http.Client
}

// OCRResult OCR 提取结果
type OCRResult struct {
	Text      string `json:"text"`       // 提取的文本
	Markdown  string `json:"markdown"`   // Markdown 格式
	Provider  string `json:"provider"`   // 使用的服务商
	Model     string `json:"model"`      // 使用的模型
	Success   bool   `json:"success"`    // 是否成功
	Error     string `json:"error"`      // 错误信息
	RawResult string `json:"raw_result"` // 原始结果（JSON 字符串）
}

// NewOCRService 构造函数
func NewOCRService(db *gorm.DB) *OCRService {
	return &OCRService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExtractSync 同步 OCR 提取（小文件，直接返回结果）
// 调用 PaddleOCR layout-parsing API（同步）
// URL: 从 model_configs 表读取 OCR 配置，默认 https://i0lau9j0n9iavbk3.aistudio-app.com/layout-parsing
// Token: 从 model_configs 表读取
// 请求体: { "file": base64, "fileType": 0|1 }
// 响应: 解析 result.layoutParsingResults[].markdown.text
// Fallback: 如果 API 返回非 200 或超时(30s)，尝试视觉模型 fallback（记录日志，返回空结果+错误标记）
func (s *OCRService) ExtractSync(userID uint, filePath string, fileType int) (*OCRResult, error) {
	return s.ExtractSyncWithContext(context.Background(), userID, filePath, fileType)
}

// ExtractSyncWithContext 允许后台受控任务取消外部 OCR 请求。
func (s *OCRService) ExtractSyncWithContext(ctx context.Context, userID uint, filePath string, fileType int) (*OCRResult, error) {
	config, err := s.GetOCRConfig(userID)
	if err != nil {
		return nil, err
	}
	return s.extractSyncWithConfig(ctx, &userID, filePath, fileType, config, true)
}

// ExtractGlobalDefaultWithContext 仅供无用户归属的后台任务使用严格全局默认 OCR 配置。
func (s *OCRService) ExtractGlobalDefaultWithContext(ctx context.Context, filePath string, fileType int) (*OCRResult, error) {
	config, err := s.GetGlobalDefaultOCRConfig()
	if err != nil {
		return nil, err
	}
	return s.extractSyncWithConfig(ctx, nil, filePath, fileType, config, false)
}

func (s *OCRService) extractSyncWithConfig(ctx context.Context, usageUserID *uint, filePath string, fileType int, config *models.ModelConfig, allowFallback bool) (*OCRResult, error) {
	startTime := time.Now()

	// 读取文件
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return &OCRResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	// 默认 API 端点
	apiURL := "https://i0lau9j0n9iavbk3.aistudio-app.com/layout-parsing"
	token := ""

	if config != nil {
		if config.APIEndpoint != "" {
			apiURL = config.APIEndpoint
		}
		if config.APIKey != "" {
			token = config.APIKey
		}
	}

	// 编码文件为 base64
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	// 构建请求
	reqBody := map[string]interface{}{
		"file":     base64Data,
		"fileType": fileType,
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return &OCRResult{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal request: %v", err),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return &OCRResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	// 发送请求
	resp, err := s.httpClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		s.recordOCRFailure(usageUserID, config, elapsed, err.Error())
		if allowFallback {
			return s.fallbackExtract(*usageUserID, filePath, fileType)
		}
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.recordOCRFailure(usageUserID, config, elapsed, err.Error())
		if allowFallback {
			return s.fallbackExtract(*usageUserID, filePath, fileType)
		}
		return nil, err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("OCR API returned HTTP %d", resp.StatusCode)
		s.recordOCRFailure(usageUserID, config, elapsed, err.Error())
		if allowFallback {
			return s.fallbackExtract(*usageUserID, filePath, fileType)
		}
		return nil, err
	}

	// 解析响应
	var apiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		s.recordOCRFailure(usageUserID, config, elapsed, err.Error())
		if allowFallback {
			return s.fallbackExtract(*usageUserID, filePath, fileType)
		}
		return nil, err
	}

	// 提取文本
	text := ""
	markdown := ""

	if result, ok := apiResp["result"].(map[string]interface{}); ok {
		if layoutResults, ok := result["layoutParsingResults"].([]interface{}); ok && len(layoutResults) > 0 {
			if firstResult, ok := layoutResults[0].(map[string]interface{}); ok {
				if markdownObj, ok := firstResult["markdown"].(map[string]interface{}); ok {
					if markdownText, ok := markdownObj["text"].(string); ok {
						markdown = markdownText
						text = markdownText
					}
				}
			}
		}
	}

	provider := "PaddleOCR"
	model := "layout-parsing"
	if config != nil {
		provider = config.Provider
		model = config.ModelName
		if usageUserID != nil {
			go s.recordUsage(*usageUserID, config, "success", 0, 0, int(elapsed.Milliseconds()), "")
		}
	}

	return &OCRResult{
		Text:      text,
		Markdown:  markdown,
		Provider:  provider,
		Model:     model,
		Success:   true,
		RawResult: string(respBody),
	}, nil
}

func (s *OCRService) recordOCRFailure(userID *uint, config *models.ModelConfig, elapsed time.Duration, message string) {
	if userID != nil && config != nil {
		go s.recordUsage(*userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), message)
	}
}

// recordUsage records model usage asynchronously
func (s *OCRService) recordUsage(userID uint, config *models.ModelConfig, status string, inputTokens, outputTokens, durationMs int, errMsg string) {
	usageLog := &models.ModelUsageLog{
		UserID:       userID,
		ConfigID:     config.ID,
		ModelName:    config.ModelName,
		Provider:     config.Provider,
		ConfigType:   "ocr",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Status:       status,
		ErrorMsg:     errMsg,
		DurationMs:   durationMs,
	}
	usageLog.CostUSD = usageLog.CalculateCost()
	if err := s.db.Create(usageLog).Error; err != nil {
		log.Printf("[ocr] failed to record usage: %v", err)
	}
}

// ExtractAsync 异步 OCR 提取（大文件，创建 Job）
// 调用 PaddleOCR async Job API
// URL: https://paddleocr.aistudio-app.com/api/v2/ocr/jobs
// Token: bearer token
// 创建 OCRJob 记录，返回 job_id
func (s *OCRService) ExtractAsync(userID uint, filePath string, documentID *uint) (uint, error) {
	// 获取 OCR 配置
	config, err := s.GetOCRConfig(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get OCR config: %w", err)
	}

	// 默认 API 端点
	apiURL := "https://paddleocr.aistudio-app.com/api/v2/ocr/jobs"

	if config != nil {
		if config.APIEndpoint != "" {
			apiURL = config.APIEndpoint
		}
	}

	// 创建 OCRJob 记录
	job := &models.OCRJob{
		UserID:     userID,
		DocumentID: documentID,
		FilePath:   filePath,
		Status:     "pending",
		Provider:   "PaddleOCR",
	}

	if config != nil {
		job.Provider = config.Provider
	}

	if err := s.db.Create(job).Error; err != nil {
		return 0, fmt.Errorf("failed to create OCR job: %w", err)
	}

	// 异步调用 API（这里仅记录日志，实际应该在后台任务中处理）
	log.Printf("[OCR] Created async job %d for user %d, file: %s, API: %s", job.ID, userID, filePath, apiURL)

	return job.ID, nil
}

// CheckJobStatus 查询异步任务状态
func (s *OCRService) CheckJobStatus(jobID uint) (*models.OCRJob, error) {
	var job models.OCRJob
	if err := s.db.First(&job, jobID).Error; err != nil {
		return nil, fmt.Errorf("failed to find job: %w", err)
	}
	return &job, nil
}
