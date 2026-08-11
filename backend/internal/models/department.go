package models

import (
	"time"
)

// Department 部门模型（用于公司内部数据隔离）
type Department struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`    // 多租户隔离
	Name      string    `json:"name" gorm:"uniqueIndex"` // 部门名称
	ParentID  *uint     `json:"parent_id" gorm:"index"`  // 上级部门ID，支持层级结构
	Code      string    `json:"code"`                    // 部门编码
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DepartmentMember 部门成员关联表（多对多中间表）
type DepartmentMember struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	DepartmentID uint      `json:"department_id" gorm:"index"`
	UserID       uint      `json:"user_id" gorm:"index"`
	Role         string    `json:"role"` // 部门内角色：leader / member
	JoinedAt     time.Time `json:"joined_at"`
}
