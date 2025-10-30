package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// User represents a system user
type User struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Username      string    `json:"username" gorm:"uniqueIndex;not null"`
	Email         string    `json:"email" gorm:"uniqueIndex;not null"`
	Password      string    `json:"-" gorm:"not null"` // Password hash, never returned in JSON
	FullName      string    `json:"full_name"`
	CompanyID     string    `json:"company_id" gorm:"index"`
	Active        bool      `json:"active" gorm:"default:true"`
	EmailVerified bool      `json:"email_verified" gorm:"default:false;index"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SetPassword hashes and sets the user password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FullName  string `json:"full_name" binding:"max=100"`
	CompanyID string `json:"companyId" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id,omitempty" gorm:"index"`
	User      *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:128"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	Used      bool      `json:"used" gorm:"default:false;index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GenerateToken creates a new secure random token
func (prt *PasswordResetToken) GenerateToken() error {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	prt.Token = hex.EncodeToString(bytes)
	return nil
}

// IsExpired checks if the token has expired
func (prt *PasswordResetToken) IsExpired() bool {
	return time.Now().After(prt.ExpiresAt)
}

// IsValid checks if the token is valid (not used and not expired)
func (prt *PasswordResetToken) IsValid() bool {
	return !prt.Used && !prt.IsExpired()
}

// PasswordResetRequest represents the password reset request payload
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirmRequest represents the password reset confirmation payload
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id,omitempty" gorm:"index"`
	User      *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:128"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	Used      bool      `json:"used" gorm:"default:false;index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GenerateToken creates a new secure random token for email verification
func (evt *EmailVerificationToken) GenerateToken() error {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	evt.Token = hex.EncodeToString(bytes)
	return nil
}

// IsExpired checks if the email verification token has expired
func (evt *EmailVerificationToken) IsExpired() bool {
	return time.Now().After(evt.ExpiresAt)
}

// IsValid checks if the token is valid (not used and not expired)
func (evt *EmailVerificationToken) IsValid() bool {
	return !evt.Used && !evt.IsExpired()
}

// EmailVerificationRequest represents the email verification request payload
type EmailVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EmailVerificationConfirmRequest represents the email verification confirmation payload
type EmailVerificationConfirmRequest struct {
	Token string `json:"token" binding:"required"`
}

type Period struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id,omitempty" gorm:"index"`
	User      *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	YearMonth string    `json:"year_month" gorm:"index"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SourceFile struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       *uint     `json:"user_id,omitempty" gorm:"index"`
	User         *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
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
	UserID       *uint     `json:"user_id,omitempty" gorm:"index"`
	User         *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
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
	UserID       *uint     `json:"user_id,omitempty" gorm:"index"`
	User         *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
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
	UserID           *uint     `json:"user_id,omitempty" gorm:"index"`
	User             *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
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
	UserID           *uint     `json:"user_id,omitempty" gorm:"index"`
	User             *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
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
	UserID     *uint     `json:"user_id,omitempty" gorm:"index"`
	User       *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	PeriodID   uint      `json:"period_id" gorm:"index"`
	Name       string    `json:"name"`
	IDNumber   string    `json:"id_number" gorm:"index"`
	Department string    `json:"department"`
	Title      string    `json:"title"`
	Remarks    string    `json:"remarks"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
