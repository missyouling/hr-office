package models

import "time"

// CalendarEvent 当前登录用户私有个人日历事件。
// UserID 归属字段使用 json:"-" 不对外输出，防止响应泄露归属关系；
// 所有查询/写入均以登录上下文中的 user_id 为准，不接受请求体指定。
type CalendarEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"not null;index"`        // 归属用户（仅由服务端从登录态注入）
	Title     string    `json:"title" gorm:"size:200;not null"` // 标题（必填）
	StartAt   time.Time `json:"start_at" gorm:"not null;index"` // 开始时间（RFC3339）
	EndAt     time.Time `json:"end_at" gorm:"not null"`         // 结束时间（RFC3339，不得早于开始）
	Location  string    `json:"location" gorm:"size:200"`       // 地点（可选）
	Notes     string    `json:"notes" gorm:"size:1000"`         // 备注（可选）
	AllDay    bool      `json:"all_day" gorm:"default:false"`   // 是否全天事件
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
