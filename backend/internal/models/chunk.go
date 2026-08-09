package models

import (
	"time"

	"gorm.io/datatypes"
)

// ============================================================
// DocumentChunk — 文档分块与向量索引（替代 DocumentEmbedding）
// 借鉴 WeKnora Chunk 模型，保留核心字段，裁剪多租户/图谱/多模态
// ============================================================

// ChunkType 分块类型常量
const (
	ChunkTypeText    = "text"        // 普通文本块（参与向量索引）
	ChunkTypeParent  = "parent_text" // 父子分块父块（仅上下文，不参与向量索引）
	ChunkTypeSummary = "summary"     // 文档摘要块
)

// IndexStatus 索引状态常量
const (
	IndexStatusReady      = "ready"      // 已同步到向量库
	IndexStatusProcessing = "processing" // 正在重建索引
	IndexStatusFailed     = "failed"     // 索引失败（允许用户编辑修正后重试）
)

// DocumentChunk 文档分块（含 pgvector 向量）
// embedding 列通过数据库迁移手动创建（GORM 不原生支持 vector 类型），
// 应用层通过 raw SQL 操作向量，embedding_json 作为降级读取的 JSON 副本。
type DocumentChunk struct {
	ID    uint      `json:"id" gorm:"primaryKey"`
	DocID uint      `json:"doc_id" gorm:"index;not null"`
	Doc   *Document `json:"-" gorm:"foreignKey:DocID"`

	ChunkIndex int    `json:"chunk_index" gorm:"not null"`                  // 在原文档中的序号
	ChunkType  string `json:"chunk_type" gorm:"size:20;default:text;index"` // text / parent_text / summary

	// 内容
	Content       string `json:"content" gorm:"type:text;not null"` // 当前可编辑内容
	SourceContent string `json:"source_content" gorm:"type:text"`   // 不可变解析器输出（首次编辑时惰性回填）
	ContentHash   string `json:"content_hash" gorm:"size:64"`       // SHA-256（去重/变更检测）

	// 上下文头部（借鉴 WeKnora ContextHeader）
	// 解析时从 Markdown/HTML 标题层级提取的面包屑，向量化时前置到 content 之前
	ContextHeader string `json:"context_header" gorm:"type:text"`

	// 坐标（原文中的字符位置）
	StartAt int `json:"start_at"`
	EndAt   int `json:"end_at"`

	// 块链（指向前后块，用于上下文扩展）
	PreChunkID  *uint `json:"pre_chunk_id"`
	NextChunkID *uint `json:"next_chunk_id"`

	// 父子分块（父块提供更完整的上下文，子块用于精确向量匹配）
	ParentChunkID *uint `json:"parent_chunk_id" gorm:"index"`

	// 版本控制（借鉴 WeKnora）
	ContentRevision int    `json:"content_revision" gorm:"default:1"`               // 每次编辑 +1，乐观锁
	IndexStatus     string `json:"index_status" gorm:"size:20;default:ready;index"` // 索引同步状态

	// 向量（两层存储）
	EmbeddingJSON datatypes.JSON `json:"embedding_json" gorm:"type:jsonb"` // JSON 副本（降级读取/迁移）
	// EmbeddingVec 为 pgvector 原生类型，由数据库迁移脚本创建，应用层通过 raw SQL 操作
	// CREATE TABLE 中包含: embedding vector(768) 或 vector(1536)
	// 查询: SELECT ... ORDER BY embedding <=> $1

	ModelName string    `json:"model_name" gorm:"size:100"` // 向量模型名称
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DocumentChunk) TableName() string {
	return "document_chunks"
}

// EmbeddingContent 将 ContextHeader 前置到 Content，供 embedding 模型使用
// 借鉴 WeKnora Chunk.EmbeddingContent() 设计
func (c *DocumentChunk) EmbeddingContent() string {
	if c == nil {
		return ""
	}
	if c.ContextHeader == "" {
		return c.Content
	}
	return c.ContextHeader + "\n" + c.Content
}

// NeedsReindex 判断是否需要重新向量索引
func (c *DocumentChunk) NeedsReindex() bool {
	return c != nil && c.IndexStatus != IndexStatusReady
}

// ============================================================
// ChunkRevision — 分块版本快照（借鉴 WeKnora ChunkRevision）
// ============================================================

// ChunkRevision 分块编辑历史快照
// 当前版本的内容存于 DocumentChunk.Content，历史版本存于此表
type ChunkRevision struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	ChunkID    uint           `json:"chunk_id" gorm:"index;not null"`
	Chunk      *DocumentChunk `json:"-" gorm:"foreignKey:ChunkID"`
	Revision   int            `json:"revision" gorm:"not null"`                  // 版本号
	Content    string         `json:"content" gorm:"type:text;not null"`         // 该版本的快照内容
	EditorID   *uint          `json:"editor_id"`                                 // 编辑者用户 ID
	EditSource string         `json:"edit_source" gorm:"size:20;default:manual"` // manual / ocr_correction / ai_fix
	CreatedAt  time.Time      `json:"created_at"`

	// 联合唯一：同一 chunk 同一版本号只存一条
	_ struct{} `gorm:"uniqueIndex:idx_chunk_revision"`
}

// TableName 指定表名
func (ChunkRevision) TableName() string {
	return "chunk_revisions"
}

// ============================================================
// ChatSession — 知识库问答会话管理
// ============================================================

// ChatSession 知识库问答会话
// 扩展现有 ChatMessage，添加会话级别管理（列表/重命名/置顶/范围控制）
type ChatSession struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"index;not null"`
	User      *User      `json:"-" gorm:"foreignKey:UserID"`
	Title     string     `json:"title" gorm:"size:200"`            // 会话标题（自动从首条问题截取）
	SessionID string     `json:"session_id" gorm:"size:100;index"` // 会话外部标识（与 ChatMessage 关联）
	IsPinned  bool       `json:"is_pinned" gorm:"default:false"`   // 是否置顶
	PinnedAt  *time.Time `json:"pinned_at"`                        // 置顶时间

	// 问答范围（JSON），存 KBAccessScope
	ScopeConfigJSON datatypes.JSON `json:"scope_config" gorm:"type:jsonb"`

	// 上下文配置（JSON），借鉴 WeKnora ContextConfig
	// {"max_tokens": 4000, "compression": "sliding_window", "recent_message_count": 10}
	ContextConfigJSON datatypes.JSON `json:"context_config" gorm:"type:jsonb"`

	// 对话历史摘要，用于上下文压缩
	Summary string `json:"summary" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// KBAccessScope 知识库访问范围
type KBAccessScope struct {
	CategoryCodes    []string `json:"category_codes,omitempty"`     // 一级分类代码
	SubCategoryCodes []string `json:"sub_category_codes,omitempty"` // 二级分类代码
	FolderPaths      []string `json:"folder_paths,omitempty"`       // 文件夹路径
	TagNames         []string `json:"tag_names,omitempty"`          // 标签
	DocumentIDs      []uint   `json:"document_ids,omitempty"`       // 指定文档 ID
}

// ContextConfig 上下文压缩配置
type ContextConfig struct {
	MaxTokens           int    `json:"max_tokens"`                    // 上下文上限
	CompressionStrategy string `json:"compression_strategy"`          // sliding_window / smart / none
	RecentMessageCount  int    `json:"recent_message_count"`          // 滑动窗口保留消息数
	SummarizeThreshold  int    `json:"summarize_threshold,omitempty"` // 智能压缩：超过此 token 数触发摘要
}
