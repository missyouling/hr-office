package models

import (
	"errors"
	"strings"
	"time"
)

const (
	OccupationalHealthStatusDraft     = "draft"
	OccupationalHealthStatusCompleted = "completed"
	OccupationalHealthStatusVoided    = "voided"
)

// OccupationalHealthCheck 职业健康检查独立台账。
// 规则：
//   - 必填：员工、检查日期、检查机构、检查类别；
//   - 可选：检查结论、下次检查日期、备注；
//   - 创建时拷贝员工姓名/部门/岗位快照并冻结；
//   - 状态：draft → completed → voided；
//   - draft 可编辑，draft/completed 均可作废；
//   - 不修改员工主表资料，不做附件/审批/提醒/批量导入等扩展。
type OccupationalHealthCheck struct {
	ID                 uint       `json:"id" gorm:"primaryKey"`
	UserID             uint       `json:"-" gorm:"not null;index"`
	EmployeeID         uint       `json:"employee_id" gorm:"not null;index"`
	SnapshotName       string     `json:"employee_name" gorm:"size:100"`
	SnapshotDepartment string     `json:"employee_department" gorm:"size:150;index"`
	SnapshotPosition   string     `json:"employee_position" gorm:"size:150"`
	CheckDate          string     `json:"check_date" gorm:"size:20;not null"`
	CheckInstitution   string     `json:"medical_institution" gorm:"size:255;not null"`
	CheckCategory      string     `json:"check_category" gorm:"size:255;not null"`
	Conclusion         string     `json:"check_conclusion" gorm:"type:text"`
	NextCheckDate      string     `json:"next_check_date" gorm:"size:20"`
	Remarks            string     `json:"remarks" gorm:"type:text"`
	Status             string     `json:"status" gorm:"size:20;not null;index;default:draft"`
	CompletedAt        *time.Time `json:"completed_at"`
	VoidedAt           *time.Time `json:"voided_at"`
	VoidReason         string     `json:"void_reason" gorm:"type:text"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// IsValidOccupationalHealthStatus 校验职业健康检查状态是否合法。
func IsValidOccupationalHealthStatus(status string) bool {
	switch status {
	case OccupationalHealthStatusDraft, OccupationalHealthStatusCompleted, OccupationalHealthStatusVoided:
		return true
	default:
		return false
	}
}

// Validate 校验职业健康检查底座字段。
func (r *OccupationalHealthCheck) Validate() error {
	if !IsValidOccupationalHealthStatus(r.Status) {
		return errors.New("无效的职业健康检查状态")
	}
	if r.EmployeeID == 0 {
		return errors.New("关联员工必填")
	}
	if strings.TrimSpace(r.CheckDate) == "" {
		return errors.New("检查日期必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(r.CheckDate)); err != nil {
		return errors.New("检查日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(r.CheckInstitution) == "" {
		return errors.New("检查机构必填")
	}
	if strings.TrimSpace(r.CheckCategory) == "" {
		return errors.New("检查类别必填")
	}
	if next := strings.TrimSpace(r.NextCheckDate); next != "" {
		if _, err := time.Parse("2006-01-02", next); err != nil {
			return errors.New("下次检查日期格式必须为 YYYY-MM-DD")
		}
	}
	return nil
}
