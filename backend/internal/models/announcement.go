package models

import (
	"time"
)

// Announcement 公告
type Announcement struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	Title         string     `json:"title" gorm:"size:200;not null"`
	Content       string     `json:"content" gorm:"type:text"`
	IsTop         bool       `json:"is_top" gorm:"default:false"`
	Status        string     `json:"status" gorm:"size:20;default:draft"` // draft, published
	PublishedAt   *time.Time `json:"published_at"`
	CreatedBy     uint       `json:"created_by" gorm:"index"`
	CreatedByUser *User      `json:"-" gorm:"foreignKey:CreatedBy"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
