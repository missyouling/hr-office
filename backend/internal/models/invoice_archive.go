package models

import (
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// InvoiceItem 发票货物或服务明细。
type InvoiceItem struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	InvoiceID     uint      `json:"invoice_id" gorm:"not null;uniqueIndex:idx_invoice_items_invoice_line"`
	LineNo        int       `json:"line_no" gorm:"not null;uniqueIndex:idx_invoice_items_invoice_line"`
	Name          string    `json:"name" gorm:"size:300;not null"`
	Specification string    `json:"specification" gorm:"size:200"`
	Unit          string    `json:"unit" gorm:"size:30"`
	Quantity      float64   `json:"quantity"`
	UnitPrice     float64   `json:"unit_price"`
	Amount        float64   `json:"amount"`
	TaxAmount     float64   `json:"tax_amount"`
	TaxRate       float64   `json:"tax_rate"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// InvoiceParsingTaskStatus 标识持久化解析任务的处理状态。
type InvoiceParsingTaskStatus string

const (
	InvoiceParsingTaskPending   InvoiceParsingTaskStatus = "pending"
	InvoiceParsingTaskRunning   InvoiceParsingTaskStatus = "running"
	InvoiceParsingTaskSucceeded InvoiceParsingTaskStatus = "succeeded"
	InvoiceParsingTaskFailed    InvoiceParsingTaskStatus = "failed"
)

// IsValid 判断解析任务状态是否为受支持的值。
func (status InvoiceParsingTaskStatus) IsValid() bool {
	return status == InvoiceParsingTaskPending ||
		status == InvoiceParsingTaskRunning ||
		status == InvoiceParsingTaskSucceeded ||
		status == InvoiceParsingTaskFailed
}

// BeforeSave 在持久化前填充解析任务默认状态并校验枚举。
func (task *InvoiceParsingTask) BeforeSave(*gorm.DB) error {
	if task.Status == "" {
		task.Status = InvoiceParsingTaskPending
	}
	if !task.Status.IsValid() {
		return errors.New("无效的发票解析任务状态")
	}
	return nil
}

// InvoiceParsingTask 保存可恢复的发票解析任务，不包含工作器逻辑。
type InvoiceParsingTask struct {
	ID           uint                     `json:"id" gorm:"primaryKey"`
	InvoiceID    uint                     `json:"invoice_id" gorm:"not null;uniqueIndex"`
	Status       InvoiceParsingTaskStatus `json:"status" gorm:"size:20;not null;default:'pending';index"`
	AttemptCount int                      `json:"attempt_count" gorm:"default:0"`
	MaxAttempts  int                      `json:"max_attempts" gorm:"not null;default:3"`
	AvailableAt  *time.Time               `json:"available_at" gorm:"index"`
	LockedBy     string                   `json:"locked_by" gorm:"size:100"`
	LockedUntil  *time.Time               `json:"locked_until" gorm:"index"`
	ErrorCode    string                   `json:"error_code" gorm:"size:100"`
	LastError    string                   `json:"last_error" gorm:"type:text"`
	StartedAt    *time.Time               `json:"started_at"`
	CompletedAt  *time.Time               `json:"completed_at"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// InvoiceFileCleanupTask 记录上传后失败时待删除的存储对象，不向客户端暴露。
type InvoiceFileCleanupTask struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	StorageConfigID uint       `json:"-" gorm:"not null;index"`
	ObjectPath      string     `json:"-" gorm:"size:500;not null"`
	SysFileID       *uint      `json:"-" gorm:"index"`
	Status          string     `json:"-" gorm:"size:20;not null;default:'pending';index"`
	Attempts        int        `json:"-" gorm:"not null;default:0"`
	OwnerToken      string     `json:"-" gorm:"size:36;index"`
	LockedUntil     *time.Time `json:"-" gorm:"index"`
	LastError       string     `json:"-" gorm:"type:text"`
	CreatedAt       time.Time  `json:"-"`
	UpdatedAt       time.Time  `json:"-"`
}

// InvoiceCorrectionAudit 记录已确认字段的更正差异。
type InvoiceCorrectionAudit struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	InvoiceID   uint           `json:"invoice_id" gorm:"not null;index"`
	Invoice     *Invoice       `json:"-" gorm:"foreignKey:InvoiceID;constraint:OnDelete:RESTRICT"`
	Changes     datatypes.JSON `json:"changes" gorm:"type:json;not null"`
	Reason      string         `json:"reason" gorm:"type:text;not null"`
	CorrectedBy uint           `json:"corrected_by" gorm:"not null;index"`
	Corrector   *User          `json:"corrector,omitempty" gorm:"foreignKey:CorrectedBy;constraint:OnDelete:RESTRICT"`
	CreatedAt   time.Time      `json:"created_at"`
}
