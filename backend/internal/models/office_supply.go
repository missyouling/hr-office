package models

import (
	"time"
)

// OfficeCategory 办公用品分类表
type OfficeCategory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OfficeCategory) TableName() string { return "office_categories" }

// OfficeSupplier 供应商表（办公用品与食堂共用）
type OfficeSupplier struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      *uint     `json:"user_id" gorm:"index"`
	User        *User     `json:"-" gorm:"foreignKey:UserID"`
	Name        string    `json:"name" gorm:"not null"`
	Contact     string    `json:"contact"`
	Phone       string    `json:"phone"`
	BankName    string    `json:"bank_name"`
	BankAccount string    `json:"bank_account"`
	IsDefault   int       `json:"is_default" gorm:"default:0"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (OfficeSupplier) TableName() string { return "office_suppliers" }

// OfficeSupply 办公用品字典表
type OfficeSupply struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	UserID         *uint           `json:"user_id" gorm:"index"`
	User           *User           `json:"-" gorm:"foreignKey:UserID"`
	Name           string          `json:"name" gorm:"not null;index"`
	Spec           string          `json:"spec"`
	Unit           string          `json:"unit" gorm:"default:个"`
	ReferencePrice float64         `json:"reference_price" gorm:"default:0"`
	SafetyStock    int             `json:"safety_stock" gorm:"default:0"`
	CategoryID     *uint           `json:"category_id"`
	Category       *OfficeCategory `json:"-" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	SupplierID     *uint           `json:"supplier_id"`
	Supplier       *OfficeSupplier `json:"-" gorm:"foreignKey:SupplierID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Status         string          `json:"status" gorm:"default:active;index"`
	Remark         string          `json:"remark"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (OfficeSupply) TableName() string { return "office_supplies" }

// OfficePurchase 办公用品采购单主表
type OfficePurchase struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	UserID        *uint           `json:"user_id" gorm:"index"`
	User          *User           `json:"-" gorm:"foreignKey:UserID"`
	OrderNo       string          `json:"order_no" gorm:"uniqueIndex;not null"`
	PurchaseDate  time.Time       `json:"purchase_date" gorm:"not null"`
	TotalAmount   float64         `json:"total_amount" gorm:"not null"`
	Status        string          `json:"status" gorm:"default:draft;index"`
	Remark        string          `json:"remark"`
	SupplierID    *uint           `json:"supplier_id"`
	Supplier      *OfficeSupplier `json:"-" gorm:"foreignKey:SupplierID"`
	SupplierName  string          `json:"supplier_name"`
	PaymentStatus string          `json:"payment_status" gorm:"default:未付款"`
	PaymentDate   *time.Time      `json:"payment_date"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (OfficePurchase) TableName() string { return "office_purchases" }

// OfficePurchaseItem 办公用品采购明细表
type OfficePurchaseItem struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	UserID     *uint           `json:"user_id" gorm:"index"`
	User       *User           `json:"-" gorm:"foreignKey:UserID"`
	PurchaseID uint            `json:"purchase_id" gorm:"not null;index"`
	Purchase   *OfficePurchase `json:"-" gorm:"foreignKey:PurchaseID;constraint:OnDelete:CASCADE"`
	SupplyID   uint            `json:"supply_id" gorm:"not null;index"`
	Supply     *OfficeSupply   `json:"-" gorm:"foreignKey:SupplyID;constraint:OnDelete:RESTRICT"`
	Quantity   int             `json:"quantity" gorm:"not null"`
	UnitPrice  float64         `json:"unit_price" gorm:"not null"`
	Subtotal   float64         `json:"subtotal" gorm:"not null"`
	Date       *time.Time      `json:"date"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (OfficePurchaseItem) TableName() string { return "office_purchase_items" }

// OfficeBackupLog 备份记录表
type OfficeBackupLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      *uint     `json:"user_id" gorm:"index"`
	User        *User     `json:"-" gorm:"foreignKey:UserID"`
	Filename    string    `json:"filename" gorm:"not null"`
	Description string    `json:"description"`
	FileSize    int64     `json:"file_size" gorm:"default:0"`
	Data        string    `json:"data" gorm:"not null;type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OfficeBackupLog) TableName() string { return "office_backup_logs" }

// OfficePaymentRequest 请款单表
type OfficePaymentRequest struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          *uint     `json:"user_id" gorm:"index"`
	User            *User     `json:"-" gorm:"foreignKey:UserID"`
	RequestNo       string    `json:"request_no" gorm:"uniqueIndex;not null"`
	PaymentUnit     string    `json:"payment_unit" gorm:"not null"`
	Department      string    `json:"department"`
	Applicant       string    `json:"applicant"`
	RequestDate     time.Time `json:"request_date" gorm:"not null"`
	Content         string    `json:"content"`
	Payee           string    `json:"payee"`
	PayeeSupplierID *uint     `json:"payee_supplier_id"`
	BankName        string    `json:"bank_name"`
	BankAccount     string    `json:"bank_account"`
	Amount          float64   `json:"amount" gorm:"default:0"`
	AmountCN        string    `json:"amount_cn"`
	PaymentMethod   string    `json:"payment_method" gorm:"default:转支"`
	Remark          string    `json:"remark"`
	CompanyHead     string    `json:"company_head"`
	FinanceHead     string    `json:"finance_head"`
	DeptHead        string    `json:"dept_head"`
	Handler         string    `json:"handler"`
	Status          string    `json:"status" gorm:"default:draft"`
	PurchaseIDs     string    `json:"purchase_ids"` // 逗号分隔的采购单ID字符串
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (OfficePaymentRequest) TableName() string { return "office_payment_requests" }
