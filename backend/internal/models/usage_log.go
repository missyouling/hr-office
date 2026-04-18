package models

import "time"

// ModelUsageLog 模型使用日志
type ModelUsageLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index"`
	ConfigID     uint      `json:"config_id" gorm:"index"`
	ModelName    string    `json:"model_name" gorm:"size:200"`
	Provider     string    `json:"provider" gorm:"size:100"`
	ConfigType   string    `json:"config_type" gorm:"size:20;index"` // ocr/llm/embedding/rerank
	InputTokens  int       `json:"input_tokens" gorm:"default:0"`
	OutputTokens int       `json:"output_tokens" gorm:"default:0"`
	TotalTokens  int       `json:"total_tokens" gorm:"default:0"`
	Status       string    `json:"status" gorm:"size:20;index"` // success/failed
	ErrorMsg     string    `json:"error_msg" gorm:"size:500"`
	DurationMs   int       `json:"duration_ms" gorm:"default:0"`
	CostUSD      float64   `json:"cost_usd" gorm:"type:decimal(10,6);default:0"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// GetStandardCost returns standard cost per 1M tokens (input, output)
func GetStandardCost(configType string) (float64, float64) {
	rates := map[string][2]float64{
		"llm":       {0.5, 1.5},
		"embedding": {0.1, 0},
		"rerank":    {0.5, 0},
		"ocr":       {1.5, 0},
	}
	if r, ok := rates[configType]; ok {
		return r[0], r[1]
	}
	return 0.5, 1.0
}

// CalculateCost computes estimated cost
func (m *ModelUsageLog) CalculateCost() float64 {
	inCost, outCost := GetStandardCost(m.ConfigType)
	return float64(m.InputTokens)/1000000.0*inCost + float64(m.OutputTokens)/1000000.0*outCost
}
