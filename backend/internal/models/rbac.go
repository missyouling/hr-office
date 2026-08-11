package models

import (
	"time"
)

// 系统内置角色名称常量（4 角色体系）
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleEditor  = "editor"
	RoleViewer  = "viewer"
)

// 兼容映射：super_admin → admin（无需数据迁移）
var legacyRoleMapping = map[string]string{
	"super_admin": RoleAdmin,
}

// NormalizeRole 将旧的角色名称映射到新 4 角色体系
func NormalizeRole(name string) string {
	if mapped, ok := legacyRoleMapping[name]; ok {
		return mapped
	}
	return name
}

type Role struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:50;uniqueIndex;not null"`
	Label       string    `json:"label" gorm:"size:100;not null"`
	Description string    `json:"description" gorm:"size:500"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"` // 系统内置角色不可删除
	UserCount   int       `json:"user_count" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Permission struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Module    string    `json:"module" gorm:"size:50;not null;index"`
	Action    string    `json:"action" gorm:"size:50;not null"` // view, create, edit, delete
	Label     string    `json:"label" gorm:"size:100;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RolePermission struct {
	ID           uint `json:"id" gorm:"primaryKey"`
	RoleID       uint `json:"role_id" gorm:"index;not null"`
	PermissionID uint `json:"permission_id" gorm:"index;not null"`
}

type UserRole struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	UserID uint `json:"user_id" gorm:"index;not null"`
	RoleID uint `json:"role_id" gorm:"index;not null"`
}
