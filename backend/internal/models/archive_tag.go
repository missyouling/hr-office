package models

import (
	"time"
)

// ============================================================
// ArchiveTag — 档案标签（独立表，替代 Document.Tags JSON 字符串）
// ============================================================

// ArchiveTag 档案标签
type ArchiveTag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`               // NULL = 全局标签
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Name      string    `json:"name" gorm:"size:50;not null"`
	Color     string    `json:"color" gorm:"size:20;default:'#3b82f6'"` // 标签颜色（hex）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 联合唯一：同用户下标签名不重复
	_ struct{} `gorm:"uniqueIndex:idx_tag_user_name"`
}

// TableName 指定表名
func (ArchiveTag) TableName() string {
	return "archive_tags"
}

// ============================================================
// DocumentTagLink — 文档-标签关联表
// ============================================================

// DocumentTagLink 文档标签关联
type DocumentTagLink struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	DocumentID uint      `json:"document_id" gorm:"index;not null"`
	Document   *Document `json:"-" gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE"`
	TagID      uint      `json:"tag_id" gorm:"index;not null"`
	Tag        *ArchiveTag `json:"tag,omitempty" gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
	CreatedAt  time.Time `json:"created_at"`

	// 联合唯一：同一文档同一标签不重复
	_ struct{} `gorm:"uniqueIndex:idx_doc_tag"`
}

// TableName 指定表名
func (DocumentTagLink) TableName() string {
	return "document_tag_links"
}

// ============================================================
// TagWithCount — 标签（含文档统计，查询时计算）
// ============================================================

// TagWithCount 带文档计数的标签（用于侧边栏筛选）
type TagWithCount struct {
	ArchiveTag
	DocumentCount int64 `json:"document_count"`
}
