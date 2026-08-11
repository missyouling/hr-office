package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// KnowledgeBase 知识库（独立模块，与现有 document_category 体系分离）
type KnowledgeBase struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           *uint          `json:"user_id" gorm:"index"`
	User             *User          `json:"-" gorm:"foreignKey:UserID"`
	Name             string         `json:"name" gorm:"size:100;not null;uniqueIndex"`
	Description      string         `json:"description" gorm:"size:500"`
	SourceModule     string         `json:"source_module" gorm:"size:50;index;default:'custom'"`
	EmbeddingModelID *uint          `json:"embedding_model_id"`
	ChunkingConfig   datatypes.JSON `json:"chunking_config" gorm:"type:json"`
	Visibility       string         `json:"visibility" gorm:"size:20;default:'restricted'"`
	IsSystem         bool           `json:"is_system" gorm:"default:false"`
	OwnerID          *uint          `json:"owner_id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

// KBAccessRule 知识库访问规则（角色+部门+用户三维；多条 Rule OR 组合）
type KBAccessRule struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	KnowledgeBaseID uint           `json:"knowledge_base_id" gorm:"index;not null"`
	KnowledgeBase   *KnowledgeBase `json:"-" gorm:"foreignKey:KnowledgeBaseID"`
	RoleLevel       *string        `json:"role_level" gorm:"size:20"`
	DepartmentID    *uint          `json:"department_id" gorm:"index"`
	UserID          *uint          `json:"user_id" gorm:"index"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (KBAccessRule) TableName() string { return "kb_access_rules" }

// KBFieldMask 知识库字段脱敏规则
type KBFieldMask struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	KnowledgeBaseID uint           `json:"knowledge_base_id" gorm:"index;not null"`
	KnowledgeBase   *KnowledgeBase `json:"-" gorm:"foreignKey:KnowledgeBaseID"`
	FieldName       string         `json:"field_name" gorm:"size:50;not null"`
	MaskPattern     string         `json:"mask_pattern" gorm:"size:50;default:'front3back4'"`
	ExemptRole      *string        `json:"exempt_role" gorm:"size:20"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (KBFieldMask) TableName() string { return "kb_field_masks" }

// DefaultChunkingConfig 返回默认分块配置（用于新知识库创建时填充 ChunkingConfig 字段）
// 策略：
//   - strategy: "auto" — 自动检测文档结构
//   - enable_parent_child: true — 开启父子分块（子块用于精确向量匹配，父块提供完整上下文）
//   - chunk_size/chunk_overlap — 通用分块参数
//   - parent_chunk_size/child_chunk_size — 父子分块专用尺寸
func DefaultChunkingConfig() datatypes.JSON {
	cfg := map[string]interface{}{
		"strategy":            "auto",
		"chunk_size":          512,
		"chunk_overlap":       80,
		"separators":          []string{"\n\n", "\n", "。", "！", "？", ";", "；"},
		"enable_parent_child": true,
		"parent_chunk_size":   2048,
		"child_chunk_size":    384,
		"token_limit":         0,
	}
	b, _ := json.Marshal(cfg)
	return datatypes.JSON(b)
}
