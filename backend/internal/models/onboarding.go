package models

import (
	"errors"
	"time"
)

// 入职生命周期状态常量（P12.3.2 数据底座）
const (
	OnboardingStatusPending   = "pending"   // 待入职
	OnboardingStatusOnboarded = "onboarded" // 已入职
	OnboardingStatusAbandoned = "abandoned" // 已放弃
)

// 就业（用工）状态常量与校验函数定义在员工模型侧（employee.go），
// 此处直接复用，避免 onboarding 依赖倒置（入职模块不反向定义员工字段的常量）。

// IsValidOnboardingStatus 校验入职生命周期状态。
func IsValidOnboardingStatus(status string) bool {
	switch status {
	case OnboardingStatusPending, OnboardingStatusOnboarded, OnboardingStatusAbandoned:
		return true
	default:
		return false
	}
}

// OnboardingRecord 入职记录（P12.3.2 数据底座）。
// 生命周期：pending → onboarded / abandoned；本阶段仅提供状态常量与字段校验，
// 不实现业务状态流转。放弃原因/时间/备注一旦写入永久保留，不随状态变化清空。
// UserID 归属字段使用 json:"-" 不对外输出，防止响应泄露归属关系；
// 所有查询/写入均以登录上下文中的 user_id 为准，不接受请求体指定。
type OnboardingRecord struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	UserID           uint       `json:"-" gorm:"not null;index"`                              // 租户隔离归属（仅由服务端从登录态注入）
	User             *User      `json:"-" gorm:"foreignKey:UserID"`                           // 归属用户关联
	Name             string     `json:"name" gorm:"size:100;not null"`                        // 候选人姓名（必填）
	IDNumber         string     `json:"id_number" gorm:"size:40;index"`                       // 身份证号（创建/确认时对员工全量冲突检查）
	Phone            string     `json:"phone" gorm:"size:50"`                                 // 联系电话（可选）
	Department       string     `json:"department" gorm:"size:150"`                           // 拟入职部门（可选）
	Position         string     `json:"position" gorm:"size:150"`                             // 拟入职岗位（可选）
	PlannedHireDate  string     `json:"planned_hire_date" gorm:"size:20;not null"`            // 计划入职日期（必填，YYYY-MM-DD）
	ActualHireDate   *string    `json:"actual_hire_date" gorm:"size:20"`                      // 实际入职日期（可空，入职后回填）
	Status           string     `json:"status" gorm:"size:20;not null;index;default:pending"` // 生命周期状态
	EmploymentStatus string     `json:"employment_status" gorm:"size:20"`                     // 用工状态（trial/formal，仅已入职记录可设置）
	EmployeeID       *uint      `json:"employee_id" gorm:"index"`                             // 关联员工ID（可空，入职后回填；不建外键约束，避免 GORM 在 employees 表生成反向外键）
	AbandonReason    string     `json:"abandon_reason" gorm:"type:text"`                      // 放弃原因（永久保留）
	AbandonedAt      *time.Time `json:"abandoned_at"`                                         // 放弃时间（永久保留）
	Remarks          string     `json:"remarks" gorm:"type:text"`                             // 备注（永久保留）
	OfferID          string     `json:"offer_id" gorm:"size:100"`                             // 预留：Offer 编号
	OfferSource      string     `json:"offer_source" gorm:"size:50"`                          // 预留：Offer 来源
	OfferConfirmTime *time.Time `json:"offer_confirm_time"`                                   // 预留：Offer 确认时间
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Validate 校验入职记录字段约束（数据底座层，不涉及状态流转）。
func (r *OnboardingRecord) Validate() error {
	if !IsValidOnboardingStatus(r.Status) {
		return errors.New("无效的入职生命周期状态")
	}
	if r.PlannedHireDate == "" {
		return errors.New("计划入职日期必填")
	}
	if r.EmploymentStatus != "" && !IsValidEmploymentStatus(r.EmploymentStatus) {
		return errors.New("无效的用工状态")
	}
	if r.EmploymentStatus != "" && r.Status != OnboardingStatusOnboarded {
		return errors.New("用工状态仅已入职记录可设置")
	}
	return nil
}
