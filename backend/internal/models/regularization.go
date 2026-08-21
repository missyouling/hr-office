package models

import (
	"errors"
	"time"
)

// 转正导入任务运行状态常量（P12.3.3-4）。
const (
	RegularizationRunStatusSuccess = "success" // 运行成功（含部分失败）
	RegularizationRunStatusFailed  = "failed"  // 运行失败（整体失败）
)

// IsValidRegularizationRunStatus 校验转正 worker 运行状态。
func IsValidRegularizationRunStatus(status string) bool {
	switch status {
	case RegularizationRunStatusSuccess, RegularizationRunStatusFailed:
		return true
	default:
		return false
	}
}

// RegularizationEffectRun 记录单日自动转正任务执行结果。
// RunDate 为上海业务日期，同一天仅允许一条记录，用于避免多实例重复扫描。
type RegularizationEffectRun struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RunDate   string    `json:"run_date" gorm:"size:20;not null;uniqueIndex"`
	Status    string    `json:"status" gorm:"size:20;not null;index"`
	Processed int       `json:"processed" gorm:"default:0"`
	Failed    int       `json:"failed" gorm:"default:0"`
	ErrorMsg  string    `json:"error_msg" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验自动转正任务运行记录。
func (r *RegularizationEffectRun) Validate() error {
	if !IsValidRegularizationRunStatus(r.Status) {
		return errors.New("无效的转正任务运行状态")
	}
	if _, err := time.Parse("2006-01-02", r.RunDate); err != nil {
		return errors.New("任务运行日期格式必须为 YYYY-MM-DD")
	}
	if r.Processed < 0 || r.Failed < 0 {
		return errors.New("任务处理数量不能为负数")
	}
	return nil
}

// 转正生命周期状态常量（P12.3.3-2 数据底座）。
// 本批次仅提供常量与字段校验，不实现业务状态流转（审批/延期/生效均后续批次实现）。
const (
	RegularizationStatusPendingSupervisor      = "pending_supervisor"       // 待直属主管审批
	RegularizationStatusPendingHRReview        = "pending_hr_review"        // 主管通过，待 HR 复核
	RegularizationStatusScheduled              = "scheduled"                // 已排期（等待计划转正日生效）
	RegularizationStatusEffective              = "effective"                // 已生效（转正完成）
	RegularizationStatusRejected               = "rejected"                 // 不通过（驳回）
	RegularizationStatusPostponedScheduled     = "postponed_scheduled"      // 已延期后排期（延期最多一次，由 HR 复核直接处理）
	RegularizationStatusEffectFailed           = "effect_failed"            // 生效失败（worker 重试后仍失败）
	RegularizationStatusCancelledByResignation = "cancelled_by_resignation" // 因离职取消
	RegularizationStatusVoided                 = "voided"                   // 已作废（终态；错误记录作废后新建）
)

// 转正来源常量：区分人工发起与 Excel 批量导入。
const (
	RegularizationSourceManual      = "manual"       // 人工发起
	RegularizationSourceExcelDirect = "excel_direct" // Excel 批量导入（未来日期排期规则后续批次实现）
)

// IsValidRegularizationStatus 校验转正状态是否合法。
func IsValidRegularizationStatus(status string) bool {
	switch status {
	case RegularizationStatusPendingSupervisor,
		RegularizationStatusPendingHRReview,
		RegularizationStatusScheduled,
		RegularizationStatusEffective,
		RegularizationStatusRejected,
		RegularizationStatusPostponedScheduled,
		RegularizationStatusEffectFailed,
		RegularizationStatusCancelledByResignation,
		RegularizationStatusVoided:
		return true
	default:
		return false
	}
}

// IsValidRegularizationSource 校验转正来源是否合法。
func IsValidRegularizationSource(source string) bool {
	switch source {
	case RegularizationSourceManual, RegularizationSourceExcelDirect:
		return true
	default:
		return false
	}
}

// RegularizationRecord 转正记录（P12.3.3-2 数据底座）。
//
// 已确认规则：
//   - 单条审批固定三人且必须不同：发起人（HR）、直属主管审批人、HR 复核人；
//   - 延期最多一次由 HR 复核直接处理（本批次仅预留 ExtensionCount / OriginalPlannedRegularDate 字段）；
//   - 错误记录作废后新建（voided 为终态，本批次仅预留状态常量）；
//   - Excel 未来日期排期规则后续批次实现；
//   - 快照字段（姓名/部门/岗位/用工状态/试用期结束日期）创建后即冻结，
//     本模型不提供通用更新/删除方法，后续审批/生效流程仅允许更新状态与审批字段。
//
// UserID 归属字段使用 json:"-" 不对外输出；所有查询/写入均以登录上下文中的 user_id 为准。
type RegularizationRecord struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;index;uniqueIndex:idx_reg_approval_no_user"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`                                   // 归属用户关联

	EmployeeID *uint `json:"employee_id" gorm:"index"` // 关联员工ID（可空；不建外键约束，避免 GORM 生成反向外键）

	// 快照字段（创建时从员工主表拷贝，之后冻结不随员工变动）
	SnapshotName             string `json:"snapshot_name" gorm:"size:100"`              // 快照姓名
	SnapshotDepartment       string `json:"snapshot_department" gorm:"size:150;index"`  // 快照部门（部门隔离过滤依据）
	SnapshotPosition         string `json:"snapshot_position" gorm:"size:150"`          // 快照岗位
	SnapshotEmploymentStatus string `json:"snapshot_employment_status" gorm:"size:20"`  // 快照用工状态（trial/formal）
	SnapshotProbationEndDate string `json:"snapshot_probation_end_date" gorm:"size:20"` // 快照试用期结束日期（YYYY-MM-DD）

	ContractTermMonths int    `json:"contract_term_months"`                  // 合同期限月数
	EmployeeSelfReview string `json:"employee_self_review" gorm:"type:text"` // 员工自评

	// 三人审批（固定三人且必须不同，校验见 Validate）
	InitiatorHRUserID        *uint `json:"initiator_hr_user_id"`        // 发起人（HR）
	SupervisorApproverUserID *uint `json:"supervisor_approver_user_id"` // 直属主管审批人
	HRReviewerUserID         *uint `json:"hr_reviewer_user_id"`         // HR 复核人

	ApprovalNo string `json:"approval_no" gorm:"size:64;uniqueIndex:idx_reg_approval_no_user,where:approval_no <> ''"` // 审批编号（租户内唯一，空值允许多条）

	PlannedRegularDate string `json:"planned_regular_date" gorm:"size:20"` // 计划转正日期（YYYY-MM-DD）
	ActualRegularDate  string `json:"actual_regular_date" gorm:"size:20"`  // 实际转正日期（YYYY-MM-DD，生效后回填）

	Status string `json:"status" gorm:"size:30;not null;index;default:pending_supervisor"` // 生命周期状态
	Source string `json:"source" gorm:"size:20;not null;index;default:manual"`             // 来源（manual/excel_direct）

	// 延期预留字段（本批次仅持久化，不实现延期动作）
	ExtensionCount             int    `json:"extension_count" gorm:"default:0"`             // 延期次数（规则上限 1 次，后续批次约束）
	OriginalPlannedRegularDate string `json:"original_planned_regular_date" gorm:"size:20"` // 原始计划转正日期（延期前）

	RejectionReason string `json:"rejection_reason" gorm:"type:text"` // 不通过/驳回原因

	// 生效 worker 预留字段（本批次不实现 worker）
	EffectAttemptedAt  *time.Time `json:"effect_attempted_at"`                   // 生效尝试时间
	EffectFailedReason string     `json:"effect_failed_reason" gorm:"type:text"` // 生效失败原因

	// 三人审批时间/意见（发起人提交无意见字段，主管与 HR 复核各自保留时间+意见）
	InitiatorSubmittedAt      *time.Time `json:"initiator_submitted_at"`                       // 发起人提交时间
	SupervisorApprovedAt      *time.Time `json:"supervisor_approved_at"`                       // 主管通过时间
	SupervisorRejectedAt      *time.Time `json:"supervisor_rejected_at"`                       // 主管拒绝时间
	SupervisorApprovalComment string     `json:"supervisor_approval_comment" gorm:"type:text"` // 主管审批意见
	HRReviewedAt              *time.Time `json:"hr_reviewed_at"`                               // HR 复核时间
	HRReviewComment           string     `json:"hr_review_comment" gorm:"type:text"`           // HR 复核意见

	// P12.3.3-3 写接口补充字段：延期原因 / 作废原因与时间
	PostponedReason string     `json:"postponed_reason" gorm:"type:text"` // 延期原因（HR 复核延期时必填）
	VoidReason      string     `json:"void_reason" gorm:"type:text"`      // 作废原因（作废时必填）
	VoidedAt        *time.Time `json:"voided_at"`                         // 作废时间

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验转正记录字段约束（数据底座层，不涉及状态流转）：
//   - 状态/来源必须合法；
//   - 三人审批（发起人/主管/HR复核）已设置时两两不得相同。
func (r *RegularizationRecord) Validate() error {
	if !IsValidRegularizationStatus(r.Status) {
		return errors.New("无效的转正状态")
	}
	if !IsValidRegularizationSource(r.Source) {
		return errors.New("无效的转正来源")
	}
	if err := r.validateApproversDistinct(); err != nil {
		return err
	}
	return nil
}

// validateApproversDistinct 校验三人审批两两不同（单条审批固定三人且必须不同）。
func (r *RegularizationRecord) validateApproversDistinct() error {
	ids := make([]uint, 0, 3)
	for _, p := range []*uint{r.InitiatorHRUserID, r.SupervisorApproverUserID, r.HRReviewerUserID} {
		if p != nil {
			ids = append(ids, *p)
		}
	}
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return errors.New("转正审批的发起人、直属主管、HR复核人必须为三名不同用户")
		}
		seen[id] = struct{}{}
	}
	return nil
}
