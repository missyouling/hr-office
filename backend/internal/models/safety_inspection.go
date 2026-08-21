package models

import (
	"errors"
	"strings"
	"time"
)

// 安全检查类型常量（按需扩展，当前仅记录台账，不区分具体类型枚举）。
const (
	SafetyInspectionTypeRoutine = "routine" // 例行检查
	SafetyInspectionTypeSpecial = "special" // 专项检查
)

// 安全检查生命周期状态常量。
const (
	SafetyInspectionStatusDraft     = "draft"     // 草稿
	SafetyInspectionStatusCompleted = "completed" // 已完成
	SafetyInspectionStatusVoided    = "voided"    // 已作废（终态）
)

// IsValidSafetyInspectionType 校验安全检查类型是否合法。
func IsValidSafetyInspectionType(value string) bool {
	return value == SafetyInspectionTypeRoutine || value == SafetyInspectionTypeSpecial
}

// IsValidSafetyInspectionStatus 校验安全检查状态是否合法。
func IsValidSafetyInspectionStatus(value string) bool {
	switch value {
	case SafetyInspectionStatusDraft, SafetyInspectionStatusCompleted, SafetyInspectionStatusVoided:
		return true
	default:
		return false
	}
}

// SafetyInspection 安全检查台账（P12.3.9）。
//
// 已确认规则：
//   - 独立台账，无隐患分派、审批、附件、定时任务、批量导入、员工联动；
//   - 生命周期：draft → completed → voided；创建即草稿，safety.edit 手动完成；
//   - draft/completed 均可由 safety.delete 填写原因作废（voided 为终态）；
//   - 必填：inspection_type、inspection_date、location、responsible_person、
//     issue_description、rectification_requirement；
//   - 登录态注入租户（UserID），按租户隔离，无部门快照。
type SafetyInspection struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;index"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	InspectionType           string `json:"inspection_type" gorm:"size:20;not null;index"`       // 检查类型（routine/special）
	InspectionDate           string `json:"inspection_date" gorm:"size:20;not null"`             // 检查日期（YYYY-MM-DD）
	Location                 string `json:"location" gorm:"size:255;not null"`                   // 检查地点
	ResponsiblePerson        string `json:"responsible_person" gorm:"size:100;not null"`         // 责任人
	IssueDescription         string `json:"issue_description" gorm:"type:text;not null"`         // 问题描述
	RectificationRequirement string `json:"rectification_requirement" gorm:"type:text;not null"` // 整改要求

	Status      string     `json:"status" gorm:"size:20;not null;index;default:draft"` // 生命周期状态
	CompletedAt *time.Time `json:"completed_at"`                                       // 手动完成时间
	VoidedAt    *time.Time `json:"voided_at"`                                          // 作废时间
	VoidReason  string     `json:"void_reason" gorm:"type:text"`                       // 作废原因（作废时必填）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验安全检查字段约束（数据底座层，不涉及状态流转）。
func (s *SafetyInspection) Validate() error {
	if !IsValidSafetyInspectionType(s.InspectionType) {
		return errors.New("无效的检查类型")
	}
	if !IsValidSafetyInspectionStatus(s.Status) {
		return errors.New("无效的检查状态")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(s.InspectionDate)); err != nil {
		return errors.New("检查日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(s.Location) == "" {
		return errors.New("检查地点必填")
	}
	if strings.TrimSpace(s.ResponsiblePerson) == "" {
		return errors.New("责任人必填")
	}
	if strings.TrimSpace(s.IssueDescription) == "" {
		return errors.New("问题描述必填")
	}
	if strings.TrimSpace(s.RectificationRequirement) == "" {
		return errors.New("整改要求必填")
	}
	return nil
}
