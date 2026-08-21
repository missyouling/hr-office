package models

import (
	"errors"
	"time"
)

// 业务待办状态常量（P12.3.2 数据底座）
const (
	WorkTodoStatusPending   = "pending"   // 待处理
	WorkTodoStatusCompleted = "completed" // 已完成
)

// IsValidWorkTodoStatus 校验业务待办状态。
func IsValidWorkTodoStatus(status string) bool {
	switch status {
	case WorkTodoStatusPending, WorkTodoStatusCompleted:
		return true
	default:
		return false
	}
}

// WorkTodo 业务待办（P12.3.2 数据底座）。
// 按 (user_id, business_type, business_id) 复合唯一去重：同一租户下
// 同一业务类型 + 业务ID 仅允许一条待办；不同租户互不影响。
// 本阶段仅提供字段与校验，不实现业务状态流转。
// UserID 归属字段使用 json:"-" 不对外输出，防止响应泄露归属关系；
// 所有查询/写入均以登录上下文中的 user_id 为准，不接受请求体指定。
type WorkTodo struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"-" gorm:"not null;index;uniqueIndex:idx_work_todos_business"`               // 租户隔离归属（仅由服务端从登录态注入）
	User         *User      `json:"-" gorm:"foreignKey:UserID"`                                                // 归属用户关联
	BusinessType string     `json:"business_type" gorm:"size:50;not null;uniqueIndex:idx_work_todos_business"` // 业务类型（如 onboarding）
	BusinessID   uint       `json:"business_id" gorm:"not null;uniqueIndex:idx_work_todos_business"`           // 业务记录ID
	Title        string     `json:"title" gorm:"size:200;not null"`                                            // 待办标题（必填）
	Description  string     `json:"description" gorm:"type:text"`                                              // 待办描述（可选）
	Status       string     `json:"status" gorm:"size:20;not null;index;default:pending"`                      // 状态
	AssigneeID   *uint      `json:"assignee_id" gorm:"index"`                                                  // 归属人（默认租户管理员，可空）
	Assignee     *User      `json:"-" gorm:"foreignKey:AssigneeID"`                                            // 归属人关联
	DueDate      *time.Time `json:"due_date"`                                                                  // 截止时间（可选）
	CompletedAt  *time.Time `json:"completed_at"`                                                              // 完成时间（可选）
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Validate 校验业务待办字段约束（数据底座层，不涉及状态流转）。
func (t *WorkTodo) Validate() error {
	if t.BusinessType == "" {
		return errors.New("业务类型必填")
	}
	if t.BusinessID == 0 {
		return errors.New("业务ID必填")
	}
	if t.Title == "" {
		return errors.New("待办标题必填")
	}
	if !IsValidWorkTodoStatus(t.Status) {
		return errors.New("无效的待办状态")
	}
	return nil
}
