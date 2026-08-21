package models

import (
	"errors"
	"time"
)

// 劳动合同生命周期状态常量
const (
	ContractStatusDraft     = "draft"     // 草稿
	ContractStatusActive    = "active"    // 生效中
	ContractStatusExpired   = "expired"   // 已到期（终态）
	ContractStatusCancelled = "cancelled" // 已作废（终态）
)

// 劳动合同类型常量：本期仅支持固定期限
const (
	ContractTypeFixedTerm = "fixed_term" // 固定期限
)

// IsValidContractStatus 校验劳动合同状态是否合法。
func IsValidContractStatus(status string) bool {
	switch status {
	case ContractStatusDraft, ContractStatusActive, ContractStatusExpired, ContractStatusCancelled:
		return true
	default:
		return false
	}
}

// IsValidContractType 校验劳动合同类型是否合法（本期仅固定期限）。
func IsValidContractType(contractType string) bool {
	return contractType == ContractTypeFixedTerm
}

// LaborContract 劳动合同（批次）记录。
//
// 已确认规则：
//   - 仅固定期限合同（contract_type 恒为 fixed_term）；
//   - 生命周期：draft → active → expired / cancelled；
//   - 合同起始日不自动生效，需 contract.edit 权限用户手动触发生效；
//   - 到期仅标记 expired 并提供提醒/查询，不修改员工状态；
//   - active 不可编辑/删除，只能作废（cancelled）后新建；
//   - 复用档案文档关联字段（document_id 关联 Document，用于存放合同扫描件等）；
//   - 快照字段（姓名/部门/岗位/身份证号）创建时从员工拷贝，之后冻结；
//   - UserID 归属字段使用 json:"-" 不对外输出，所有查询/写入以登录上下文 user_id 为准。
type LaborContract struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;index"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	EmployeeID *uint `json:"employee_id" gorm:"index"` // 关联员工ID（可空；不建外键约束，避免 GORM 生成反向外键）

	// 快照字段（创建时从员工主表拷贝，之后冻结不随员工变动）
	SnapshotName       string `json:"snapshot_name" gorm:"size:100"`             // 快照姓名
	SnapshotDepartment string `json:"snapshot_department" gorm:"size:150;index"` // 快照部门（部门隔离过滤依据）
	SnapshotPosition   string `json:"snapshot_position" gorm:"size:150"`         // 快照岗位
	SnapshotIDNumber   string `json:"snapshot_id_number" gorm:"size:40"`         // 快照身份证号

	// 合同信息
	ContractNo   string `json:"contract_no" gorm:"size:64;index"`                         // 合同编号
	ContractType string `json:"contract_type" gorm:"size:20;not null;default:fixed_term"` // 合同类型（仅固定期限）
	StartDate    string `json:"start_date" gorm:"size:20"`                                // 合同起始日（YYYY-MM-DD）
	EndDate      string `json:"end_date" gorm:"size:20"`                                  // 合同到期日（YYYY-MM-DD）
	TermMonths   int    `json:"term_months"`                                              // 合同期限（月）

	// 档案文档关联（复用档案文档关联字段：关联 Document 存放合同扫描件等）
	DocumentID *uint     `json:"document_id" gorm:"index"` // 关联档案文档ID（可空）
	Document   *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`

	Status       string     `json:"status" gorm:"size:20;not null;index;default:draft"` // 生命周期状态
	ActivatedAt  *time.Time `json:"activated_at"`                                       // 手动生效时间
	ExpiredAt    *time.Time `json:"expired_at"`                                         // 到期标记时间
	CancelledAt  *time.Time `json:"cancelled_at"`                                       // 作废时间
	CancelReason string     `json:"cancel_reason" gorm:"type:text"`                     // 作废原因（作废时必填）

	Remarks   string    `json:"remarks" gorm:"type:text"` // 备注
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验劳动合同字段约束（数据底座层，不涉及状态流转）。
func (c *LaborContract) Validate() error {
	if !IsValidContractStatus(c.Status) {
		return errors.New("无效的劳动合同状态")
	}
	if !IsValidContractType(c.ContractType) {
		return errors.New("仅支持固定期限劳动合同")
	}
	if c.StartDate == "" || c.EndDate == "" {
		return errors.New("合同起始日与到期日必填")
	}
	if _, err := time.Parse("2006-01-02", c.StartDate); err != nil {
		return errors.New("合同起始日格式必须为 YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", c.EndDate); err != nil {
		return errors.New("合同到期日格式必须为 YYYY-MM-DD")
	}
	if c.TermMonths <= 0 {
		return errors.New("合同期限月数必须为正整数")
	}
	return nil
}
