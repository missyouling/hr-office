package models

import (
	"time"

	"gorm.io/datatypes"
)

// DocumentContent OCR 提取的文本内容
type DocumentContent struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	DocID       uint           `json:"doc_id" gorm:"index"`
	Document    *Document      `json:"-" gorm:"foreignKey:DocID"`
	Content     string         `json:"content" gorm:"type:text"`      // OCR 提取的文本
	OCRProvider string         `json:"ocr_provider" gorm:"size:50"`   // OCR 服务商
	OCRModel    string         `json:"ocr_model" gorm:"size:100"`     // OCR 模型名称
	OCRVersion  string         `json:"ocr_version" gorm:"size:50"`    // OCR 版本
	RawResponse datatypes.JSON `json:"raw_response" gorm:"type:json"` // 原始 OCR 响应
	CreatedAt   time.Time      `json:"created_at"`
}

// DocumentEmbedding 文档向量嵌入
type DocumentEmbedding struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DocID        uint           `json:"doc_id" gorm:"index"`
	Document     *Document      `json:"-" gorm:"foreignKey:DocID"`
	ChunkIndex   int            `json:"chunk_index" gorm:"index"`       // 分块索引
	ChunkContent string         `json:"chunk_content" gorm:"type:text"` // 分块内容
	Embedding    datatypes.JSON `json:"embedding" gorm:"type:json"`     // 向量（JSON 格式）
	ModelName    string         `json:"model_name" gorm:"size:100"`     // 向量模型名称
	ModelVersion string         `json:"model_version" gorm:"size:50"`   // 向量模型版本
	CreatedAt    time.Time      `json:"created_at"`
}

// ModelConfig 模型配置（OCR/LLM/向量/重排）
type ModelConfig struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        *uint          `json:"user_id" gorm:"index"`                  // 可空：NULL 表示全局配置，所有用户可用
	User          *User          `json:"-" gorm:"foreignKey:UserID"`
	ConfigType    string         `json:"config_type" gorm:"size:20;index"`      // ocr/llm/embedding/rerank
	Provider      string         `json:"provider" gorm:"size:100"`              // 服务商名称
	ModelName     string         `json:"model_name" gorm:"size:200"`            // 模型名称
	APIKey        string         `json:"api_key,omitempty" gorm:"size:500"`     // API 密钥（敏感，加密存储标记）
	APIEndpoint   string         `json:"api_endpoint" gorm:"size:500"`          // API 端点
	ExtraParams   datatypes.JSON `json:"extra_params" gorm:"type:json"`         // 额外参数
	Enabled       bool           `json:"enabled" gorm:"default:true"`           // 是否启用
	IsDefault     bool           `json:"is_default" gorm:"default:false"`       // 是否为默认配置
	Role          string         `json:"role" gorm:"size:20;default:'primary'"` // primary/backup
	Priority      int            `json:"priority" gorm:"default:0"`             // 优先级（数字越小越优先）
	IsBuiltIn     bool           `json:"is_built_in" gorm:"default:false"`      // 是否为内置模型
	ContextLength string         `json:"context_length" gorm:"size:20"`         // 如 "256K", "128K", "8K"
	Capabilities  string         `json:"capabilities" gorm:"size:200"`          // 如 "vision,tool_call"
	RateLimitRPM  int            `json:"rate_limit_rpm" gorm:"default:0"`       // 每分钟请求限制
	RateLimitTPM  int            `json:"rate_limit_tpm" gorm:"default:0"`       // 每分钟令牌限制
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// DocumentTypeField 文档类型字段映射
type DocumentTypeField struct {
	ID            uint                 `json:"id" gorm:"primaryKey"`
	SubCategoryID uint                 `json:"sub_category_id" gorm:"index"`
	SubCategory   *DocumentSubCategory `json:"-" gorm:"foreignKey:SubCategoryID"`
	FieldName     string               `json:"field_name" gorm:"size:100;index"`     // 字段英文名
	FieldLabel    string               `json:"field_label" gorm:"size:200"`          // 字段显示名
	FieldType     string               `json:"field_type" gorm:"size:50"`            // 字段类型
	IsShared      bool                 `json:"is_shared" gorm:"default:false"`       // 是否跨类型共享
	IsOCRFillable bool                 `json:"is_ocr_fillable" gorm:"default:false"` // 是否可由 OCR 填充
	SortOrder     int                  `json:"sort_order" gorm:"default:0"`          // 排序
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// TypeDefaultColumn 文档类型默认显示列配置
type TypeDefaultColumn struct {
	ID            uint                 `json:"id" gorm:"primaryKey"`
	SubCategoryID uint                 `json:"sub_category_id" gorm:"index"`
	SubCategory   *DocumentSubCategory `json:"-" gorm:"foreignKey:SubCategoryID"`
	UserID        uint                 `json:"user_id" gorm:"index"`
	User          *User                `json:"-" gorm:"foreignKey:UserID"`
	ColumnKeys    datatypes.JSON       `json:"column_keys" gorm:"type:json"` // 列字段名数组
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// OCRJob OCR 异步任务
type OCRJob struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"index"`
	User         *User          `json:"-" gorm:"foreignKey:UserID"`
	DocumentID   *uint          `json:"document_id" gorm:"index"` // 关联文档（可为空）
	Document     *Document      `json:"-" gorm:"foreignKey:DocumentID"`
	FilePath     string         `json:"file_path" gorm:"size:500"`      // 文件路径
	Status       string         `json:"status" gorm:"size:20;index"`    // pending/processing/completed/failed
	Provider     string         `json:"provider" gorm:"size:100"`       // OCR 服务商
	Result       datatypes.JSON `json:"result" gorm:"type:json"`        // OCR 结果
	ErrorMessage string         `json:"error_message" gorm:"type:text"` // 错误信息
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ChatMessage 问答消息记录
type ChatMessage struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"index"`
	User      *User          `json:"-" gorm:"foreignKey:UserID"`
	SessionID string         `json:"session_id" gorm:"size:100;index"` // 会话 ID
	Role      string         `json:"role" gorm:"size:20"`              // user/assistant
	Content   string         `json:"content" gorm:"type:text"`         // 消息内容
	Sources   datatypes.JSON `json:"sources" gorm:"type:json"`         // 引用溯源（文档 ID 数组）
	CreatedAt time.Time      `json:"created_at"`
}
