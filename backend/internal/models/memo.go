package models

import "time"

// Memo 当前登录用户私有备忘录。
// UserID 归属字段使用 json:"-" 不对外输出，防止响应泄露归属关系；
// 所有查询/写入均以登录上下文中的 user_id 为准，不接受请求体指定。
type Memo struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"not null;index"`        // 归属用户（仅由服务端从登录态注入）
	Title     string    `json:"title" gorm:"size:200;not null"` // 标题（必填，≤200字符）
	Content   string    `json:"content" gorm:"size:5000"`       // 正文（可选，≤5000字符）
	Pinned    bool      `json:"pinned" gorm:"default:false"`    // 是否置顶
	Completed bool      `json:"completed" gorm:"default:false"` // 是否已完成
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
