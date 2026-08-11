package models

import "time"

// AuthToken 记录已签发的 access / refresh token（hash 后存库，用于旋转与吊销）
type AuthToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	TokenHash string    `json:"-" gorm:"size:64;uniqueIndex;not null"` // SHA-256(token 原文)
	Type      string    `json:"type" gorm:"size:20;not null"`          // "access" | "refresh"
	IsRevoked bool      `json:"is_revoked" gorm:"default:false;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (AuthToken) TableName() string { return "auth_tokens" }
