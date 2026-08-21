package models

import (
	"errors"
	"strings"
	"time"
)

// 奖惩记录类型常量（仅两类）。
const (
	RewardTypeReward     = "reward"     // 奖励
	RewardTypePunishment = "punishment" // 惩罚
)

// 奖惩记录生命周期状态常量。
const (
	RewardStatusDraft     = "draft"     // 草稿
	RewardStatusEffective = "effective" // 已生效
	RewardStatusVoided    = "voided"    // 已作废（终态）
)

// IsValidRewardType 校验奖惩记录类型是否合法。
func IsValidRewardType(recordType string) bool {
	switch recordType {
	case RewardTypeReward, RewardTypePunishment:
		return true
	default:
		return false
	}
}

// IsValidRewardStatus 校验奖惩记录状态是否合法。
func IsValidRewardStatus(status string) bool {
	switch status {
	case RewardStatusDraft, RewardStatusEffective, RewardStatusVoided:
		return true
	default:
		return false
	}
}

// RewardRecord 奖惩记录（P12.3.6）台账。
//
// 已确认规则：
//   - 仅奖励（reward）与惩罚（punishment）两类；
//   - 生命周期：draft → effective → voided；创建即草稿，reward.edit 手动生效；
//   - draft/effective 均可由 reward.delete 填写原因作废（voided 为终态）；
//   - 不改变员工状态或薪资，仅作台账记录；
//   - 必填：employee_id、record_type、occurred_date、reason、level；
//     选填：score/amount/owner/document_id/remarks；
//   - 快照字段（姓名/部门/岗位）创建时从员工主表拷贝，之后冻结；
//   - 复用档案文档关联字段（document_id 关联 Document）；
//   - UserID 归属字段使用 json:"-" 不对外输出，所有查询/写入以登录上下文 user_id 为准（租户隔离）。
type RewardRecord struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;index"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	EmployeeID uint `json:"employee_id" gorm:"not null;index"` // 关联员工ID（必填；不建外键约束，避免 GORM 生成反向外键）

	// 快照字段（创建时从员工主表拷贝，之后冻结不随员工变动）
	SnapshotName       string `json:"snapshot_name" gorm:"size:100"`             // 快照姓名
	SnapshotDepartment string `json:"snapshot_department" gorm:"size:150;index"` // 快照部门（部门隔离过滤依据）
	SnapshotPosition   string `json:"snapshot_position" gorm:"size:150"`         // 快照岗位

	// 记录信息
	RecordType   string   `json:"record_type" gorm:"size:20;not null;index"` // 记录类型（reward/punishment）
	OccurredDate string   `json:"occurred_date" gorm:"size:20;not null"`     // 发生日期（YYYY-MM-DD）
	Reason       string   `json:"reason" gorm:"type:text;not null"`          // 事由（必填）
	Level        string   `json:"level" gorm:"size:50;not null"`             // 等级（如 嘉奖/记功/警告/记过）
	Score        *float64 `json:"score"`                                     // 分值（选填）
	Amount       *float64 `json:"amount"`                                    // 金额（选填）
	Owner        string   `json:"owner" gorm:"size:100"`                     // 经办人（选填）
	Remarks      string   `json:"remarks" gorm:"type:text"`                  // 备注（选填）

	// 档案文档关联（复用档案 Document 关联字段）
	DocumentID *uint     `json:"document_id" gorm:"index"` // 关联档案文档ID（可空）
	Document   *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`

	Status      string     `json:"status" gorm:"size:20;not null;index;default:draft"` // 生命周期状态
	EffectiveAt *time.Time `json:"effective_at"`                                       // 手动生效时间
	VoidedAt    *time.Time `json:"voided_at"`                                          // 作废时间
	VoidReason  string     `json:"void_reason" gorm:"type:text"`                       // 作废原因（作废时必填）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验奖惩记录字段约束（数据底座层，不涉及状态流转）。
func (r *RewardRecord) Validate() error {
	if !IsValidRewardStatus(r.Status) {
		return errors.New("无效的奖惩记录状态")
	}
	if !IsValidRewardType(r.RecordType) {
		return errors.New("记录类型必须为 reward 或 punishment")
	}
	if r.EmployeeID == 0 {
		return errors.New("关联员工必填")
	}
	if strings.TrimSpace(r.OccurredDate) == "" {
		return errors.New("发生日期必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(r.OccurredDate)); err != nil {
		return errors.New("发生日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("事由必填")
	}
	if strings.TrimSpace(r.Level) == "" {
		return errors.New("等级必填")
	}
	return nil
}
