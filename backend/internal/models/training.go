package models

import (
	"errors"
	"strings"
	"time"
)

const (
	TrainingTypeInternal = "internal"
	TrainingTypeExternal = "external"
	TrainingTypeOnline   = "online"

	TrainingStatusDraft     = "draft"
	TrainingStatusCompleted = "completed"
	TrainingStatusVoided    = "voided"
)

// TrainingRecord 培训管理独立台账；员工快照在关联员工时创建并冻结。
type TrainingRecord struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	UserID               uint       `json:"-" gorm:"not null;index"`
	EmployeeID           *uint      `json:"employee_id,omitempty" gorm:"index"`
	SnapshotName         string     `json:"snapshot_name" gorm:"size:100"`
	SnapshotDepartment   string     `json:"snapshot_department" gorm:"size:150;index"`
	SnapshotPosition     string     `json:"snapshot_position" gorm:"size:150"`
	Topic                string     `json:"topic" gorm:"size:255;not null"`
	TrainingType         string     `json:"training_type" gorm:"size:20;not null;index"`
	TrainingDate         string     `json:"training_date" gorm:"size:20;not null"`
	TrainerOrInstitution string     `json:"trainer_or_institution" gorm:"size:255"`
	Result               string     `json:"result" gorm:"type:text"`
	Remarks              string     `json:"remarks" gorm:"type:text"`
	Status               string     `json:"status" gorm:"size:20;not null;index;default:draft"`
	CompletedAt          *time.Time `json:"completed_at"`
	VoidedAt             *time.Time `json:"voided_at"`
	VoidReason           string     `json:"void_reason" gorm:"type:text"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func IsValidTrainingType(value string) bool {
	return value == TrainingTypeInternal || value == TrainingTypeExternal || value == TrainingTypeOnline
}

func IsValidTrainingStatus(value string) bool {
	return value == TrainingStatusDraft || value == TrainingStatusCompleted || value == TrainingStatusVoided
}

func (r *TrainingRecord) Validate() error {
	if !IsValidTrainingType(r.TrainingType) {
		return errors.New("无效的培训类型")
	}
	if !IsValidTrainingStatus(r.Status) {
		return errors.New("无效的培训状态")
	}
	if strings.TrimSpace(r.Topic) == "" {
		return errors.New("培训主题必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(r.TrainingDate)); err != nil {
		return errors.New("培训日期格式必须为 YYYY-MM-DD")
	}
	return nil
}
