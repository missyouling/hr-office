package models

import "gorm.io/gorm"

// 员工生命周期状态常量（P12.3.3-1 数据底座；保持 active/resigned 语义不变）
const (
	EmployeeStatusActive   = "active"   // 在职
	EmployeeStatusResigned = "resigned" // 离职
)

// 就业（用工）状态常量：定义在员工模型侧（员工可用的稳定位置），
// 入职模块（onboarding）复用本处定义，避免 onboarding 依赖倒置。
const (
	EmploymentStatusTrial  = "trial"  // 试用期
	EmploymentStatusFormal = "formal" // 正式
)

// IsValidEmploymentStatus 校验就业状态是否合法。
func IsValidEmploymentStatus(status string) bool {
	switch status {
	case EmploymentStatusTrial, EmploymentStatusFormal:
		return true
	default:
		return false
	}
}

// BeforeCreate 为新建员工提供安全的就业状态默认值：
// 未显式指定（空值）且非离职时默认 formal（正式），
// 避免旧创建路径误把员工置为 trial（试用）；离职员工保持空值。
func (e *Employee) BeforeCreate(tx *gorm.DB) error {
	if e.EmploymentStatus == "" && e.Status != EmployeeStatusResigned {
		e.EmploymentStatus = EmploymentStatusFormal
	}
	return nil
}

// MigrateEmployeeEmploymentStatus 回填历史员工的就业状态（幂等，AutoMigrate 后调用）：
//   - 仅将 employment_status 为空的 active（在职）员工置为 formal（正式）；
//   - 不覆盖已有 trial/formal 值；
//   - 不改动 resigned（离职）员工的 employment_status（保持空值）。
//
// 使用标准 SQL 条件，PostgreSQL 与 SQLite 均适用。
func MigrateEmployeeEmploymentStatus(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Model(&Employee{}).
		Where("status = ? AND (employment_status IS NULL OR employment_status = '')", EmployeeStatusActive).
		Update("employment_status", EmploymentStatusFormal).Error
}
