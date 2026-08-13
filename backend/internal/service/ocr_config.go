package service

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// GetOCRConfig 从 model_configs 表获取用户可用的 OCR 配置。
func (s *OCRService) GetOCRConfig(userID uint) (*models.ModelConfig, error) {
	var config models.ModelConfig
	if err := s.db.Where("user_id = ? AND config_type = ? AND is_default = ? AND enabled = ?", userID, "ocr", true, true).First(&config).Error; err == nil {
		return &config, nil
	}
	if err := s.db.Where("user_id = ? AND config_type = ? AND enabled = ?", userID, "ocr", true).First(&config).Error; err == nil {
		return &config, nil
	}
	if err := s.db.Where("user_id IS NULL AND config_type = ? AND is_default = ? AND enabled = ?", "ocr", true, true).First(&config).Error; err == nil {
		return &config, nil
	}
	return nil, nil
}

// GetGlobalDefaultOCRConfig 查询仅属于系统的启用默认 OCR 配置，不回退到用户配置。
func (s *OCRService) GetGlobalDefaultOCRConfig() (*models.ModelConfig, error) {
	var config models.ModelConfig
	err := s.db.Where("user_id IS NULL AND config_type = ? AND enabled = ? AND is_default = ?", "ocr", true, true).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("未配置启用的全局默认 OCR")
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *OCRService) fallbackExtract(userID uint, filePath string, fileType int) (*OCRResult, error) {
	log.Printf("[OCR] Fallback: attempting visual model for user %d, file: %s, type: %d", userID, filePath, fileType)
	return &OCRResult{Success: false, Error: "OCR API failed and fallback not yet implemented"}, nil
}
