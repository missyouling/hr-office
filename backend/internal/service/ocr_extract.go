package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// OCRExtractService OCR字段提取服务
type OCRExtractService struct {
	db         *gorm.DB
	ocrService *OCRService
}

// NewOCRExtractService 构造函数
func NewOCRExtractService(db *gorm.DB, ocrService *OCRService) *OCRExtractService {
	return &OCRExtractService{db: db, ocrService: ocrService}
}

// ExtractResult OCR提取结果
type ExtractResult struct {
	OCRStatus         string                 `json:"ocr_status"`         // success/failed/skipped
	OCRText           string                 `json:"ocr_text"`           // OCR提取的原文
	SharedFields      map[string]interface{} `json:"shared_fields"`      // 共用字段预填充值
	ProprietaryFields map[string]interface{} `json:"proprietary_fields"` // 专用字段预填充值
	ErrorMessage      string                 `json:"error_message"`      // 错误信息
}

// ExtractFieldsFromFile 从上传文件中提取字段
func (s *OCRExtractService) ExtractFieldsFromFile(
	file multipart.File,
	header *multipart.FileHeader,
	subCategoryCode string,
	userID uint,
) (*ExtractResult, error) {
	result := &ExtractResult{
		SharedFields:      make(map[string]interface{}),
		ProprietaryFields: make(map[string]interface{}),
	}

	// 1. 提取基础共用字段（不需要OCR）
	result.SharedFields["archive_title"] = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	result.SharedFields["file_format"] = strings.TrimPrefix(filepath.Ext(header.Filename), ".")
	result.SharedFields["file_size"] = header.Size
	result.SharedFields["archive_date"] = time.Now().Format("2006-01-02")

	// 2. 尝试调用OCR
	ocrText, err := s.callOCR(file, header, userID)
	if err != nil {
		// OCR失败 - 标记为手动填写模式
		result.OCRStatus = "failed"
		result.ErrorMessage = err.Error()
		return result, nil // 不返回error，让前端知道OCR失败但基础字段已填充
	}

	result.OCRStatus = "success"
	result.OCRText = ocrText

	// 3. 从OCR文本提取摘要和标签
	if len(ocrText) > 200 {
		result.SharedFields["summary"] = ocrText[:200] + "..."
	} else if ocrText != "" {
		result.SharedFields["summary"] = ocrText
	}

	// 4. 提取专用字段（根据二级分类）
	s.extractProprietaryFromOCR(result, ocrText, subCategoryCode)

	return result, nil
}

// callOCR 调用OCR服务
func (s *OCRExtractService) callOCR(file multipart.File, header *multipart.FileHeader, userID uint) (string, error) {
	// 检查文件类型是否支持OCR
	ext := strings.ToLower(filepath.Ext(header.Filename))
	supportedExts := map[string]bool{
		".pdf":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".bmp":  true,
		".tiff": true,
		".gif":  true,
		".webp": true,
	}
	if !supportedExts[ext] {
		return "", fmt.Errorf("unsupported file type for OCR: %s", ext)
	}

	// 调用现有OCR服务（如果可用）
	if s.ocrService == nil {
		return "", fmt.Errorf("no OCR service configured")
	}

	// 重置文件指针
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read file failed: %v", err)
	}

	// 创建临时文件用于OCR处理
	// 由于OCRService.ExtractSync需要文件路径，我们需要保存临时文件
	tmpDir := "/tmp"
	tmpFile := filepath.Join(tmpDir, "ocr_"+header.Filename)

	// 写入临时文件
	if err := writeFileBytes(tmpFile, data); err != nil {
		return "", fmt.Errorf("write temp file failed: %v", err)
	}

	// 调用OCR
	ocrResult, err := s.ocrService.ExtractSync(userID, tmpFile, 0)
	if err != nil {
		return "", fmt.Errorf("OCR service error: %v", err)
	}

	if ocrResult == nil || !ocrResult.Success {
		errMsg := "OCR extraction failed"
		if ocrResult != nil && ocrResult.Error != "" {
			errMsg = ocrResult.Error
		}
		return "", errors.New(errMsg)
	}

	return ocrResult.Text, nil
}

// writeFileBytes 写入文件字节
func writeFileBytes(filePath string, data []byte) error {
	return os.WriteFile(filePath, data, 0644)
}

// extractProprietaryFromOCR 从OCR文本中提取专用字段
func (s *OCRExtractService) extractProprietaryFromOCR(result *ExtractResult, ocrText string, subCategoryCode string) {
	if ocrText == "" {
		return
	}

	// 基于关键词匹配的简单提取逻辑
	// 后续可以接入LLM进行智能提取
	// 目前仅做基础的关键词匹配示例

	// 示例：根据二级分类代码提取不同的字段
	switch subCategoryCode {
	case "0101": // 示例分类
		// 提取特定字段的逻辑
		if strings.Contains(ocrText, "合同") {
			result.ProprietaryFields["contract_type"] = "employment_contract"
		}
	}
}
