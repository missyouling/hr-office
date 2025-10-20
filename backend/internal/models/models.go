package models

import (
	"time"
)

type Part string

const (
	PartPersonal Part = "personal"
	PartUnit     Part = "unit"
)

type FileType string

const (
	FileTypeNormal     FileType = "normal"
	FileTypeAdjustment FileType = "adjustment"
)

type Scheme string

const (
	SchemePension        Scheme = "pension"
	SchemeMedical        Scheme = "medical"
	SchemeSeriousIllness Scheme = "serious_illness"
	SchemeUnemployment   Scheme = "unemployment"
	SchemeInjury         Scheme = "injury"
)

type Period struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	YearMonth string    `json:"year_month" gorm:"uniqueIndex"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SourceFile struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	PeriodID     uint      `json:"period_id" gorm:"index"`
	Period       Period    `json:"-"`
	FileName     string    `json:"file_name"`
	StoredPath   string    `json:"stored_path"`
	Scheme       Scheme    `json:"scheme" gorm:"index"`
	Part         Part      `json:"part" gorm:"index"`
	FileType     FileType  `json:"file_type" gorm:"index;default:normal"`
	Rows         int       `json:"rows"`
	Status       string    `json:"status"`
	UploadedAt   time.Time `json:"uploaded_at"`
	OriginalName string    `json:"original_name"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RawRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	PeriodID     uint      `json:"period_id" gorm:"index"`
	SourceFileID uint      `json:"source_file_id" gorm:"index"`
	Sequence     int       `json:"sequence"`
	Name         string    `json:"name"`
	IDType       string    `json:"id_type"`
	IDNumber     string    `json:"id_number" gorm:"index"`
	Department   string    `json:"department"`
	PaySalary    float64   `json:"pay_salary"`
	PayBase      float64   `json:"pay_base"`
	RateText     string    `json:"rate_text"`
	AmountDue    float64   `json:"amount_due"`
	AmountAdjust float64   `json:"amount_adjust"`
	PersonCode   string    `json:"person_code"`
	Scheme       Scheme    `json:"scheme" gorm:"index"`
	Part         Part      `json:"part" gorm:"index"`
	FileType     FileType  `json:"file_type" gorm:"index;default:normal"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PeriodSummary struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	PeriodID     uint      `json:"period_id" gorm:"index"`
	Scheme       Scheme    `json:"scheme"`
	Part         Part      `json:"part"`
	Headcount    int       `json:"headcount"`
	BaseTotal    float64   `json:"base_total"`
	AmountTotal  float64   `json:"amount_total"`
	IsAdjustment bool      `json:"is_adjustment" gorm:"index;default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PersonalCharge struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	PeriodID         uint      `json:"period_id" gorm:"index"`
	Name             string    `json:"name"`
	IDNumber         string    `json:"id_number" gorm:"index"`
	Department       string    `json:"department"`
	Base             float64   `json:"base"`
	Pension          float64   `json:"pension"`
	MedicalMaternity float64   `json:"medical_maternity"`
	SeriousIllness   float64   `json:"serious_illness"`
	Unemployment     float64   `json:"unemployment"`
	Subtotal         float64   `json:"subtotal"`
	IsAdjustment     bool      `json:"is_adjustment" gorm:"index;default:false"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UnitCharge struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	PeriodID         uint      `json:"period_id" gorm:"index"`
	Name             string    `json:"name"`
	IDNumber         string    `json:"id_number" gorm:"index"`
	Department       string    `json:"department"`
	Base             float64   `json:"base"`
	Pension          float64   `json:"pension"`
	MedicalMaternity float64   `json:"medical_maternity"`
	SeriousIllness   float64   `json:"serious_illness"`
	Injury           float64   `json:"injury"`
	Unemployment     float64   `json:"unemployment"`
	Subtotal         float64   `json:"subtotal"`
	IsAdjustment     bool      `json:"is_adjustment" gorm:"index;default:false"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RosterEntry struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	PeriodID   uint      `json:"period_id" gorm:"index"`
	Name       string    `json:"name"`
	IDNumber   string    `json:"id_number" gorm:"index"`
	Department string    `json:"department"`
	Title      string    `json:"title"`
	Remarks    string    `json:"remarks"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
