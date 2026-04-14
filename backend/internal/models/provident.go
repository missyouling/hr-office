package models

import "time"

// ProvidentFundRecord represents a single employee's provident fund configuration.
type ProvidentFundRecord struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	UserID            *uint      `json:"user_id" gorm:"index:idx_fund_user_identity,unique"`
	PersonalAccount   string     `json:"personal_account" gorm:"not null"`
	Name              string     `json:"name"`
	IdentityNumber    string     `json:"identity_number" gorm:"index:idx_fund_user_identity,unique"`
	PersonalBase      float64    `json:"personal_base"`
	ContributionRatio float64    `json:"contribution_ratio" gorm:"default:6"`
	PersonalAmount    float64    `json:"personal_amount"`
	CompanyAmount     float64    `json:"company_amount"`
	TotalAmount       float64    `json:"total_amount"`
	Status            string     `json:"status" gorm:"index;default:active"`
	Notes             string     `json:"notes"`
	SealedAt          *time.Time `json:"sealed_at"`
	UnsealedAt        *time.Time `json:"unsealed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ProvidentFundSettings stores unit level provident fund configuration.
type ProvidentFundSettings struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      *uint     `json:"user_id" gorm:"uniqueIndex"`
	UnitName    string    `json:"unit_name"`
	UnitAccount string    `json:"unit_account"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProvidentFundBill represents a generated monthly summary.
type ProvidentFundBill struct {
	ID                  uint                    `json:"id" gorm:"primaryKey"`
	UserID              *uint                   `json:"user_id" gorm:"index"`
	MonthLabel          string                  `json:"month_label" gorm:"index"`
	RecordCount         int                     `json:"record_count"`
	PersonalAmountTotal float64                 `json:"personal_amount_total"`
	CompanyAmountTotal  float64                 `json:"company_amount_total"`
	CombinedAmountTotal float64                 `json:"combined_amount_total"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	Items               []ProvidentFundBillItem `json:"items" gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE;"`
}

// ProvidentFundBillItem stores each member contribution snapshot for a bill.
type ProvidentFundBillItem struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	BillID          uint      `json:"bill_id" gorm:"index"`
	RecordID        uint      `json:"record_id" gorm:"index"`
	PersonalAccount string    `json:"personal_account"`
	Name            string    `json:"name"`
	IdentityNumber  string    `json:"identity_number"`
	PersonalAmount  float64   `json:"personal_amount"`
	CompanyAmount   float64   `json:"company_amount"`
	TotalAmount     float64   `json:"total_amount"`
	CreatedAt       time.Time `json:"created_at"`
}
