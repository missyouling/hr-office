package main

import (
	"encoding/json"
	"log"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// seedModelConfigs 预填充内置模型配置
func seedModelConfigs(db *gorm.DB) error {
	// 检查是否已有内置模型
	var count int64
	db.Model(&models.ModelConfig{}).Where("is_built_in = ?", true).Count(&count)
	if count > 0 {
		log.Println("内置模型配置已存在，跳过")
		return nil
	}

	// 获取admin用户ID（内置模型关联到admin用户）
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Printf("找不到admin用户，跳过内置模型创建: %v", err)
		return nil
	}
	adminUserID := adminUser.ID

	siliconflowKey := "sk-lqmkmhmzqhynebyaseuaduwvedicorvqnvoqmqldpkpkjfhi"
	siliconflowEndpoint := "https://api.siliconflow.cn/v1"

	ocrV5Params, _ := json.Marshal(map[string]interface{}{
		"model": "PP-OCRv5",
	})

	builtInModels := []models.ModelConfig{
		// LLM 主模型
		{
			ConfigType:    "llm",
			Provider:      "siliconflow",
			ModelName:     "Qwen/Qwen3.5-4B",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     true,
			Role:          "primary",
			IsBuiltIn:     true,
			ContextLength: "256K",
			Capabilities:  "vision,tool_call",
			RateLimitRPM:  1000,
			RateLimitTPM:  80000,
		},
		// LLM 备模型
		{
			ConfigType:    "llm",
			Provider:      "siliconflow",
			ModelName:     "Qwen/Qwen3-8B",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     false,
			Role:          "backup",
			IsBuiltIn:     true,
			ContextLength: "128K",
			Capabilities:  "tool_call",
			RateLimitRPM:  1000,
			RateLimitTPM:  50000,
		},
		// OCR 主模型 - PaddleOCR-VL-1.5 官方/异步解析
		{
			ConfigType:   "ocr",
			Provider:     "baidu",
			ModelName:    "PaddleOCR-VL-1.5 官方/异步解析",
			APIKey:       "aaae759ff9b85860acddea64bf31133b0165fd67",
			APIEndpoint:  "https://paddleocr.aistudio-app.com/api/v2/ocr/jobs",
			Priority:     1,
			Enabled:      true,
			IsDefault:    true,
			Role:         "primary",
			IsBuiltIn:    true,
			Capabilities: "sync,async",
		},
		// OCR 备用模型1 - PP-OCRv5 官方/异步解析
		{
			ConfigType:   "ocr",
			Provider:     "baidu",
			ModelName:    "PP-OCRv5 官方/异步解析",
			APIKey:       "aaae759ff9b85860acddea64bf31133b0165fd67",
			APIEndpoint:  "https://paddleocr.aistudio-app.com/api/v2/ocr/jobs",
			ExtraParams:  datatypes.JSON(ocrV5Params),
			Priority:     2,
			Enabled:      true,
			IsDefault:    false,
			Role:         "backup",
			IsBuiltIn:    true,
			Capabilities: "sync,async",
		},
		// OCR 备用模型2 - 硅基流动 PaddleOCR
		{
			ConfigType:   "ocr",
			Provider:     "siliconflow",
			ModelName:    "PaddlePaddle/PaddleOCR-VL-1.5",
			APIKey:       siliconflowKey,
			APIEndpoint:  siliconflowEndpoint,
			Priority:     3,
			Enabled:      true,
			IsDefault:    false,
			Role:         "backup",
			IsBuiltIn:    true,
			Capabilities: "vision",
			RateLimitRPM: 1000,
			RateLimitTPM: 80000,
		},
		// Embedding 主模型
		{
			ConfigType:    "embedding",
			Provider:      "siliconflow",
			ModelName:     "BAAI/bge-m3",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     true,
			Role:          "primary",
			IsBuiltIn:     true,
			ContextLength: "8K",
			Capabilities:  "",
			RateLimitRPM:  2000,
			RateLimitTPM:  500000,
		},
		// Embedding 备模型
		{
			ConfigType:    "embedding",
			Provider:      "siliconflow",
			ModelName:     "netease-youdao/bce-embedding-base_v1",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     false,
			Role:          "backup",
			IsBuiltIn:     true,
			ContextLength: "8K",
			Capabilities:  "",
			RateLimitRPM:  2000,
			RateLimitTPM:  500000,
		},
		// Reranker 主模型
		{
			ConfigType:    "rerank",
			Provider:      "siliconflow",
			ModelName:     "BAAI/bge-reranker-v2-m3",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     true,
			Role:          "primary",
			IsBuiltIn:     true,
			ContextLength: "8K",
			Capabilities:  "",
			RateLimitRPM:  2000,
			RateLimitTPM:  500000,
		},
		// Reranker 备模型
		{
			ConfigType:    "rerank",
			Provider:      "siliconflow",
			ModelName:     "netease-youdao/bce-reranker-base_v1",
			APIKey:        siliconflowKey,
			APIEndpoint:   siliconflowEndpoint,
			Enabled:       true,
			IsDefault:     false,
			Role:          "backup",
			IsBuiltIn:     true,
			ContextLength: "8K",
			Capabilities:  "",
			RateLimitRPM:  2000,
			RateLimitTPM:  500000,
		},
	}

	for _, m := range builtInModels {
		m.UserID = adminUserID
		if err := db.Create(&m).Error; err != nil {
			log.Printf("创建内置模型失败 %s/%s: %v", m.ConfigType, m.ModelName, err)
		}
	}

	log.Println("内置模型配置预填充完成")
	return nil
}
