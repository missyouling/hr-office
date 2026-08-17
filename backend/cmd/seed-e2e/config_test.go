package main

import (
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.ModelConfig{}); err != nil {
		t.Fatalf("迁移 ModelConfig 表失败: %v", err)
	}
	return db
}

// TestEnsureGlobalLLMConfig_WritesEnableThinkingFalse 全局 LLM 配置应写入 enable_thinking=false
// 且保持 user_id 为 nil（全局配置），避免反馈闭环 E2E 被 Qwen3 推理流拖至 60 秒超时
func TestEnsureGlobalLLMConfig_WritesEnableThinkingFalse(t *testing.T) {
	db := setupConfigDB(t)
	if err := ensureGlobalLLMConfig(db); err != nil {
		t.Fatalf("初始化全局 LLM 配置失败: %v", err)
	}

	var cfg models.ModelConfig
	if err := db.Where("user_id IS NULL AND config_type = ? AND enabled = ?", "llm", true).First(&cfg).Error; err != nil {
		t.Fatalf("查询全局 LLM 配置失败: %v", err)
	}
	if cfg.UserID != nil {
		t.Errorf("全局配置 user_id 应为 nil，实际: %v", cfg.UserID)
	}
	var params map[string]interface{}
	if err := json.Unmarshal(cfg.ExtraParams, &params); err != nil {
		t.Fatalf("解析 ExtraParams 失败: %v", err)
	}
	enableThinking, ok := params["enable_thinking"].(bool)
	if !ok || enableThinking {
		t.Errorf("ExtraParams 应包含 enable_thinking=false，实际: %v", params)
	}
}
