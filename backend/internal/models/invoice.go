package models

import (
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Invoice 发票状态常量
const (
	InvoiceStatusDraft      = "draft"      // 草稿
	InvoiceStatusSubmitted  = "submitted"  // 已提交
	InvoiceStatusApproved   = "approved"   // 已审批
	InvoiceStatusRejected   = "rejected"   // 已驳回
	InvoiceStatusReimbursed = "reimbursed" // 已报销
)

// InvoiceArchiveStatus 发票归档状态，与审批/报销状态独立。
type InvoiceArchiveStatus string

const (
	InvoiceArchiveStatusPending   InvoiceArchiveStatus = "pending"
	InvoiceArchiveStatusConfirmed InvoiceArchiveStatus = "confirmed"
	InvoiceArchiveStatusVoided    InvoiceArchiveStatus = "voided"
)

// IsValid 判断归档状态是否为受支持的值。
func (status InvoiceArchiveStatus) IsValid() bool {
	return status == InvoiceArchiveStatusPending ||
		status == InvoiceArchiveStatusConfirmed ||
		status == InvoiceArchiveStatusVoided
}

// InvoiceVoucherType 发票或凭证的归档类型。
type InvoiceVoucherType string

const (
	InvoiceVoucherTypeVATInput     InvoiceVoucherType = "vat_input"
	InvoiceVoucherTypeReceipt      InvoiceVoucherType = "receipt"
	InvoiceVoucherTypePaymentProof InvoiceVoucherType = "payment_proof"
	InvoiceVoucherTypeItinerary    InvoiceVoucherType = "e_itinerary"
	InvoiceVoucherTypeOther        InvoiceVoucherType = "other"
)

// IsValid 判断凭证类型是否为受支持的值。
func (voucherType InvoiceVoucherType) IsValid() bool {
	return voucherType == InvoiceVoucherTypeVATInput ||
		voucherType == InvoiceVoucherTypeReceipt ||
		voucherType == InvoiceVoucherTypePaymentProof ||
		voucherType == InvoiceVoucherTypeItinerary ||
		voucherType == InvoiceVoucherTypeOther
}

// BeforeSave 在持久化前填充归档默认值并校验枚举。
func (invoice *Invoice) BeforeSave(*gorm.DB) error {
	if invoice.ArchiveStatus == "" {
		invoice.ArchiveStatus = InvoiceArchiveStatusPending
	}
	if invoice.VoucherType == "" {
		invoice.VoucherType = InvoiceVoucherTypeOther
	}
	if !invoice.ArchiveStatus.IsValid() {
		return errors.New("无效的发票归档状态")
	}
	if !invoice.VoucherType.IsValid() {
		return errors.New("无效的发票凭证类型")
	}
	return nil
}

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
	User   *User `json:"-,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`

	// 发票基本信息
	InvoiceNo           string    `json:"invoice_no" gorm:"size:50;index"`
	InvoiceCode         string    `json:"invoice_code" gorm:"size:50;index"`
	ElectronicInvoiceNo string    `json:"electronic_invoice_no" gorm:"size:100;index"`
	IdentityKey         *string   `json:"identity_key" gorm:"size:200;index"`
	InvoiceDate         time.Time `json:"invoice_date" gorm:"not null"`
	InvoiceType         string    `json:"invoice_type" gorm:"size:20"` // 增值税普通发票/专用发票/电子发票等
	Amount              float64   `json:"amount" gorm:"not null"`
	TaxAmount           float64   `json:"tax_amount"`   // 税额
	TotalAmount         float64   `json:"total_amount"` // 含税总额
	Seller              string    `json:"seller" gorm:"size:200;not null"`
	SellerTaxNo         string    `json:"seller_tax_no" gorm:"size:50"`
	Buyer               string    `json:"buyer" gorm:"size:200"`    // 购方（默认本公司）
	Purpose             string    `json:"purpose" gorm:"type:text"` // 用途说明
	Remark              string    `json:"remark" gorm:"type:text"`  // 备注
	// AttachmentURL 仅兼容历史数据；新附件必须通过 AttachmentFileID 关联 SysFile。
	AttachmentURL    string   `json:"attachment_url" gorm:"size:500"`
	AttachmentFileID *uint    `json:"attachment_file_id" gorm:"index"`
	AttachmentFile   *SysFile `json:"attachment_file,omitempty" gorm:"foreignKey:AttachmentFileID;constraint:OnDelete:RESTRICT"`
	FileSHA256       string   `json:"file_sha256" gorm:"size:64;index"`

	// 归档信息，与现有审批/报销状态机独立。
	ArchiveStatus     InvoiceArchiveStatus     `json:"archive_status" gorm:"size:20;not null;default:'pending';index"`
	VoucherType       InvoiceVoucherType       `json:"voucher_type" gorm:"size:30;not null;default:'other';index"`
	BuyerTaxNo        string                   `json:"buyer_tax_no" gorm:"size:50"`
	BuyerMatched      bool                     `json:"buyer_matched" gorm:"default:false;index"`
	BuyerMatchNote    string                   `json:"buyer_match_note" gorm:"size:500"`
	RecognitionSource string                   `json:"recognition_source" gorm:"size:30;index"`
	OriginalText      string                   `json:"original_text" gorm:"type:text"`
	FieldConfidence   datatypes.JSON           `json:"field_confidence" gorm:"type:json"`
	DeletedAt         gorm.DeletedAt           `json:"-" gorm:"index"`
	PurgeAfter        *time.Time               `json:"purge_after,omitempty" gorm:"index"`
	DeletedBy         *uint                    `json:"deleted_by" gorm:"index"`
	DeletedByUser     *User                    `json:"deleted_by_user,omitempty" gorm:"foreignKey:DeletedBy;constraint:OnDelete:SET NULL"`
	Items             []InvoiceItem            `json:"items,omitempty" gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	ParsingTasks      []InvoiceParsingTask     `json:"parsing_tasks,omitempty" gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	CorrectionAudits  []InvoiceCorrectionAudit `json:"correction_audits,omitempty" gorm:"foreignKey:InvoiceID;constraint:OnDelete:RESTRICT"`

	// 关联业务（软引用，不强制外键约束，避免循环依赖）
	SourceType string `json:"source_type" gorm:"size:30;index"` // 关联类型
	SourceID   *uint  `json:"source_id" gorm:"index"`           // 关联业务 ID

	// 报销信息
	ApplicantID     *uint   `json:"applicant_id" gorm:"index"`
	Applicant       *User   `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID;constraint:OnDelete:SET NULL"`
	ReimburseAmount float64 `json:"reimburse_amount"` // 实报销金额

	// 状态与审批
	Status         string     `json:"status" gorm:"size:20;index;default:'draft'"`
	ApproverID     *uint      `json:"approver_id" gorm:"index"`
	Approver       *User      `json:"approver,omitempty" gorm:"foreignKey:ApproverID;constraint:OnDelete:SET NULL"`
	ApprovedAt     *time.Time `json:"approved_at"`
	ApprovalRemark string     `json:"approval_remark" gorm:"type:text"`

	ConfirmedBy     *uint      `json:"confirmed_by" gorm:"index"`
	ConfirmedByUser *User      `json:"confirmed_by_user,omitempty" gorm:"foreignKey:ConfirmedBy;constraint:OnDelete:SET NULL"`
	ConfirmedAt     *time.Time `json:"confirmed_at" gorm:"index"`
	VoidedBy        *uint      `json:"voided_by" gorm:"index"`
	VoidedByUser    *User      `json:"voided_by_user,omitempty" gorm:"foreignKey:VoidedBy;constraint:OnDelete:SET NULL"`
	VoidedAt        *time.Time `json:"voided_at" gorm:"index"`
	VoidedReason    string     `json:"voided_reason" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Invoice) TableName() string { return "invoices" }
