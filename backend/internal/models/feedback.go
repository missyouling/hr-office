package models

import "time"

const (
	FeedbackStatusPending = "pending"
	FeedbackStatusReplied = "replied"
	FeedbackStatusClosed  = "closed"
)

// IsValidFeedbackStatus 校验反馈状态。
func IsValidFeedbackStatus(status string) bool {
	switch status {
	case FeedbackStatusPending, FeedbackStatusReplied, FeedbackStatusClosed:
		return true
	default:
		return false
	}
}

// ChatFeedback 用户对 AI 回答的反馈（点赞/点踩 + 文字反馈 + 管理员回复）
type ChatFeedback struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"index;not null"`
	User      *User      `json:"-" gorm:"foreignKey:UserID"`
	MessageID string     `json:"message_id" gorm:"size:100;index;not null;uniqueIndex:idx_chat_feedback_message_id"` // ChatMessage.ID
	SessionID string     `json:"session_id" gorm:"size:100;index"`
	Rating    string     `json:"rating" gorm:"size:10;not null"` // "positive" / "negative"
	Comment   string     `json:"comment" gorm:"type:text"`       // 用户文字反馈
	Reply     string     `json:"reply" gorm:"type:text"`         // 管理员回复
	RepliedAt *time.Time `json:"replied_at"`                     // 回复时间
	Status    string     `json:"status" gorm:"size:20;index;not null;default:pending"`
	ClosedAt  *time.Time `json:"closed_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
