package models

import (
	"errors"
	"strings"
	"time"
)

// 行政合同生命周期状态常量（独立于劳动合同与宿舍合同）。
const (
	AdminContractStatusDraft     = "draft"     // 草稿
	AdminContractStatusActive    = "active"    // 生效中
	AdminContractStatusExpired   = "expired"   // 已到期（终态）
	AdminContractStatusCancelled = "cancelled" // 已作废（终态）
)

// IsValidAdminContractStatus 校验行政合同状态是否合法。
func IsValidAdminContractStatus(status string) bool {
	switch status {
	case AdminContractStatusDraft, AdminContractStatusActive, AdminContractStatusExpired, AdminContractStatusCancelled:
		return true
	default:
		return false
	}
}

// AdminContract 行政合同（P12.3.5）台账记录。
//
// 已确认规则：
//   - 独立于劳动合同与宿舍合同，覆盖全部外部主体（相对方为外部单位/个人自由文本）；
//   - 必填：合同编号、名称、相对方名称、合同类型、起止日期；选填：含税金额、币种、负责人、备注；
//   - 生命周期：draft → active → expired / cancelled；contract.edit 手动生效，不自动生效；
//   - 到期仅标记 expired 并提供页面提醒/工作台统一提醒（30 天），不联动其他模块；
//   - draft/active 均可由 admin_contract.delete 填写原因作废，作废后可新建替代；
//   - 复用档案 Document 关联（document_id 关联 Document 存放扫描件等）；
//   - UserID 归属字段使用 json:"-" 不对外输出，所有查询/写入以登录上下文 user_id 为准（租户隔离）。
type AdminContract struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;index"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	// 必填字段
	ContractNo   string `json:"contract_no" gorm:"size:64;index"`      // 合同编号
	Name         string `json:"name" gorm:"size:200"`                  // 合同名称
	Counterparty string `json:"counterparty" gorm:"size:200;index"`    // 相对方名称（外部主体）
	ContractType string `json:"contract_type" gorm:"size:50;not null"` // 合同类型（如 服务/采购/租赁）
	StartDate    string `json:"start_date" gorm:"size:20"`             // 合同起始日（YYYY-MM-DD）
	EndDate      string `json:"end_date" gorm:"size:20;index"`         // 合同到期日（YYYY-MM-DD）

	// 选填字段
	AmountInclTax *float64 `json:"amount_incl_tax"`          // 含税金额（可空）
	Currency      string   `json:"currency" gorm:"size:20"`  // 币种
	Owner         string   `json:"owner" gorm:"size:100"`    // 负责人
	Remarks       string   `json:"remarks" gorm:"type:text"` // 备注

	// 档案文档关联（复用档案 Document 关联字段）
	DocumentID *uint     `json:"document_id" gorm:"index"` // 关联档案文档ID（可空）
	Document   *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`

	Status       string     `json:"status" gorm:"size:20;not null;index;default:draft"` // 生命周期状态
	ActivatedAt  *time.Time `json:"activated_at"`                                       // 手动生效时间
	ExpiredAt    *time.Time `json:"expired_at"`                                         // 到期标记时间
	CancelledAt  *time.Time `json:"cancelled_at"`                                       // 作废时间
	CancelReason string     `json:"cancel_reason" gorm:"type:text"`                     // 作废原因（作废时必填）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验行政合同必填字段与格式（数据底座层，不涉及状态流转）。
func (c *AdminContract) Validate() error {
	if !IsValidAdminContractStatus(c.Status) {
		return errors.New("无效的行政合同状态")
	}
	if strings.TrimSpace(c.ContractNo) == "" {
		return errors.New("合同编号必填")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("合同名称必填")
	}
	if strings.TrimSpace(c.Counterparty) == "" {
		return errors.New("相对方名称必填")
	}
	if strings.TrimSpace(c.ContractType) == "" {
		return errors.New("合同类型必填")
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
	return nil
}
