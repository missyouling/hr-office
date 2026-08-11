package models

import (
	"time"
)

// Invoice 发票状态常量
const (
	InvoiceStatusDraft      = "draft"      // 草稿
	InvoiceStatusSubmitted  = "submitted"  // 已提交
	InvoiceStatusApproved   = "approved"   // 已审批
	InvoiceStatusRejected   = "rejected"   // 已驳回
	InvoiceStatusReimbursed = "reimbursed" // 已报销
)

// Invoice 关联业务来源类型常量
const (
	InvoiceSourceOffice         = "office"          // 关联办公用品采购
	InvoiceSourceCanteen        = "canteen"         // 关联食堂支出
	InvoiceSourcePaymentRequest = "payment_request" // 关联请款单
	InvoiceSourceIndependent    = "independent"     // 独立发票（无关联业务）
)

// Invoice 发票管理模型（P7.3）
type Invoice struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID *uint `json:"user_id" gorm:"index"`
	User   *User `json:"-,omitempty" gorm:"foreignKey:UserID"`

	// 发票基本信息
	InvoiceNo     string    `json:"invoice_no" gorm:"size:50;uniqueIndex;not null"`
	InvoiceDate   time.Time `json:"invoice_date" gorm:"not null"`
	InvoiceType   string    `json:"invoice_type" gorm:"size:20"` // 增值税普通发票/专用发票/电子发票等
	Amount        float64   `json:"amount" gorm:"not null"`
	TaxAmount     float64   `json:"tax_amount"`   // 税额
	TotalAmount   float64   `json:"total_amount"` // 含税总额
	Seller        string    `json:"seller" gorm:"size:200;not null"`
	SellerTaxNo   string    `json:"seller_tax_no" gorm:"size:50"`
	Buyer         string    `json:"buyer" gorm:"size:200"`          // 购方（默认本公司）
	Purpose       string    `json:"purpose" gorm:"type:text"`       // 用途说明
	Remark        string    `json:"remark" gorm:"type:text"`        // 备注
	AttachmentURL string    `json:"attachment_url" gorm:"size:500"` // 发票扫描件 URL

	// 关联业务（软引用，不强制外键约束，避免循环依赖）
	SourceType string `json:"source_type" gorm:"size:30;index"` // 关联类型
	SourceID   *uint  `json:"source_id" gorm:"index"`           // 关联业务 ID

	// 报销信息
	ApplicantID     *uint   `json:"applicant_id" gorm:"index"`
	Applicant       *User   `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID"`
	ReimburseAmount float64 `json:"reimburse_amount"` // 实报销金额

	// 状态与审批
	Status         string     `json:"status" gorm:"size:20;index;default:'draft'"`
	ApproverID     *uint      `json:"approver_id"`
	Approver       *User      `json:"approver,omitempty" gorm:"foreignKey:ApproverID"`
	ApprovedAt     *time.Time `json:"approved_at"`
	ApprovalRemark string     `json:"approval_remark" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Invoice) TableName() string { return "invoices" }
