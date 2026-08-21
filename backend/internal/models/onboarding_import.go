package models

import "time"

// 入职导入任务运行状态常量（P12.3.2.3）
const (
	OnboardingRunStatusSuccess = "success" // 运行成功（含部分失败）
	OnboardingRunStatusFailed  = "failed"  // 运行失败（整体失败）
)

// IsValidOnboardingRunStatus 校验运行状态。
func IsValidOnboardingRunStatus(status string) bool {
	switch status {
	case OnboardingRunStatusSuccess, OnboardingRunStatusFailed:
		return true
	default:
		return false
	}
}

// OnboardingImportRun 入职导入定时任务运行记录（P12.3.2.3）。
// RunDate 唯一索引保证同日幂等：同一自然日（Asia/Shanghai）仅允许一次运行，
// 并发启动时唯一约束兜底，插入失败即视为当日已运行，不自动重试。
type OnboardingImportRun struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RunDate   string    `json:"run_date" gorm:"size:20;not null;uniqueIndex"` // 运行日期（Asia/Shanghai，YYYY-MM-DD）
	Status    string    `json:"status" gorm:"size:20;not null"`               // success / failed
	Processed int       `json:"processed"`                                    // 成功处理记录数
	Failed    int       `json:"failed"`                                       // 失败记录数
	ErrorMsg  string    `json:"error_msg" gorm:"type:text"`                   // 失败原因（整体失败或部分失败摘要）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
