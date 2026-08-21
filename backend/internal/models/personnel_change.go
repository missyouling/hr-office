package models

import (
	"errors"
	"strings"
	"time"
)

const (
	PersonnelChangeTypeTransfer  = "transfer"
	PersonnelChangeTypePromotion = "promotion"
	PersonnelChangeTypeDemotion  = "demotion"

	PersonnelChangeStatusDraft     = "draft"
	PersonnelChangeStatusEffective = "effective"
	PersonnelChangeStatusVoided    = "voided"
)

// PersonnelChange 人事异动台账，前快照在创建时冻结，后快照在生效时确认。
type PersonnelChange struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	UserID        uint   `json:"-" gorm:"not null;index"`
	EmployeeID    uint   `json:"employee_id" gorm:"not null;index"`
	ChangeType    string `json:"change_type" gorm:"size:20;not null;index"`
	EffectiveDate string `json:"effective_date" gorm:"size:20;not null"`
	Reason        string `json:"reason" gorm:"type:text;not null"`

	BeforeDepartmentID *uint  `json:"before_department_id" gorm:"index"`
	BeforeDepartment   string `json:"before_department" gorm:"size:150;index"`
	BeforePosition     string `json:"before_position" gorm:"size:150"`
	BeforeJobLevel     string `json:"before_job_level" gorm:"size:100"`
	AfterDepartmentID  *uint  `json:"after_department_id" gorm:"index"`
	AfterDepartment    string `json:"after_department" gorm:"size:150;index"`
	AfterPosition      string `json:"after_position" gorm:"size:150"`
	AfterJobLevel      string `json:"after_job_level" gorm:"size:100"`

	Status      string     `json:"status" gorm:"size:20;not null;index;default:draft"`
	VoidReason  string     `json:"void_reason" gorm:"type:text"`
	CreatedBy   uint       `json:"created_by" gorm:"not null"`
	EffectiveBy *uint      `json:"effective_by"`
	EffectiveAt *time.Time `json:"effective_at"`
	VoidedAt    *time.Time `json:"voided_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func IsValidPersonnelChangeType(value string) bool {
	return value == PersonnelChangeTypeTransfer || value == PersonnelChangeTypePromotion || value == PersonnelChangeTypeDemotion
}

func IsValidPersonnelChangeStatus(value string) bool {
	return value == PersonnelChangeStatusDraft || value == PersonnelChangeStatusEffective || value == PersonnelChangeStatusVoided
}

func (p *PersonnelChange) Validate() error {
	if !IsValidPersonnelChangeType(p.ChangeType) {
		return errors.New("无效的异动类型")
	}
	if !IsValidPersonnelChangeStatus(p.Status) {
		return errors.New("无效的异动状态")
	}
	if p.EmployeeID == 0 {
		return errors.New("关联员工必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.EffectiveDate)); err != nil {
		return errors.New("生效日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(p.Reason) == "" {
		return errors.New("异动原因必填")
	}
	return nil
}
