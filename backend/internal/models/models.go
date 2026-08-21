package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
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
	ID              uint       `json:"id" gorm:"primaryKey"`
	Username        string     `json:"username" gorm:"uniqueIndex;not null"`
	Email           string     `json:"email" gorm:"uniqueIndex;not null"`
	SupabaseUID     string     `json:"supabase_uid,omitempty" gorm:"size:64;uniqueIndex,where:supabase_uid <> ''"` // Supabase 外部身份标识（UUID），空表示纯本地用户；部分唯一索引：空值允许多个，非空唯一
	Password        string     `json:"-" gorm:"not null"`                                                          // Password hash, never returned in JSON
	FullName        string     `json:"full_name"`
	CompanyID       string     `json:"company_id" gorm:"index"`
	Department      string     `json:"department" gorm:"size:150"` // 所属部门（用于数据隔离）
	DepartmentID    *uint      `json:"department_id" gorm:"index"` // 部门ID（关联Department表，用于部门级数据隔离）
	Role            string     `json:"-" gorm:"-"`                 // 【已废弃】改用 user_roles 关联表。字段保留仅为迁移脚本兼容，GORM 不读写，JSON 不输出
	Active          bool       `json:"active" gorm:"default:true"`
	EmailVerified   bool       `json:"email_verified" gorm:"default:false;index"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	User         User     `json:"user"`
	Permissions  []string `json:"permissions"` // 扁平化权限代码数组，如 ["employee.view","employee.edit"]；JWT 不承载权限（决策 B），前端从响应中读取
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse 刷新 token 响应
type RefreshTokenResponse struct {
	AccessToken  string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	User         User     `json:"user"`
	Permissions  []string `json:"permissions"` // 刷新后一并返回最新权限，确保权限变更即时生效
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    *uint      `json:"user_id,omitempty" gorm:"index"`
	User      *User      `json:"-,omitempty" gorm:"foreignKey:UserID"`
	Token     string     `json:"token" gorm:"uniqueIndex;not null;size:128"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	Used      bool       `json:"used" gorm:"default:false;index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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

// UpdateProfileRequest 是资料更新请求的严格白名单：本期仅允许修改 full_name。
// 处理端使用 DisallowUnknownFields 解码，请求体出现任何其他字段（username/email/id 等）一律拒绝。
type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    *uint      `json:"user_id,omitempty" gorm:"index"`
	User      *User      `json:"-,omitempty" gorm:"foreignKey:UserID"`
	Token     string     `json:"token" gorm:"uniqueIndex;not null;size:128"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	Used      bool       `json:"used" gorm:"default:false;index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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

// AccountAvailabilityRequest represents the payload to check username/email availability
type AccountAvailabilityRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

// AccountAvailabilityResponse represents availability results
type AccountAvailabilityResponse struct {
	EmailAvailable    bool `json:"email_available"`
	UsernameAvailable bool `json:"username_available"`
}

type Period struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	UserID           *uint     `json:"user_id,omitempty" gorm:"index"`
	User             *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	YearMonth        string    `json:"year_month" gorm:"index"`
	Status           string    `json:"status"`
	AllowAdjustments bool      `json:"allow_adjustments" gorm:"default:false"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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

type Employee struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	UserID                uint      `json:"user_id" gorm:"index:idx_employee_user_id_number,unique;uniqueIndex:idx_employee_user_employee_id,where:employee_id <> ''"`
	EmployeeID            string    `json:"employee_id" gorm:"size:100;uniqueIndex:idx_employee_user_employee_id,where:employee_id <> ''"`
	Name                  string    `json:"name" gorm:"size:100;not null;index"`
	Department            string    `json:"department" gorm:"size:150"`
	Position              string    `json:"position" gorm:"size:150"`
	JobLevel              string    `json:"job_level" gorm:"size:100"`
	Gender                string    `json:"gender" gorm:"size:20"`
	HireDate              string    `json:"hire_date" gorm:"size:20"`
	Age                   string    `json:"age" gorm:"size:20"`
	WorkYears             string    `json:"work_years" gorm:"size:20"`
	BirthMonth            string    `json:"birth_month" gorm:"size:20"`
	Education             string    `json:"education" gorm:"size:100"`
	PoliticalStatus       string    `json:"political_status" gorm:"size:100"`
	WorkClothingSize      string    `json:"work_clothing_size" gorm:"size:50"`
	SafetyShoeSize        string    `json:"safety_shoe_size" gorm:"size:50"`
	HouseholdType         string    `json:"household_type" gorm:"size:50"`
	Ethnicity             string    `json:"ethnicity" gorm:"size:50"`
	NativePlace           string    `json:"native_place" gorm:"size:100"`
	IDAddress             string    `json:"id_address" gorm:"size:255"`
	IDNumber              string    `json:"id_number" gorm:"size:40;index:idx_employee_user_id_number,unique"`
	MaritalStatus         string    `json:"marital_status" gorm:"size:50"`
	SocialInsurance       string    `json:"social_insurance" gorm:"size:50"`
	HasBirth              string    `json:"has_birth" gorm:"size:50"`
	Phone                 string    `json:"phone" gorm:"size:50"`
	SocialInsuranceNumber string    `json:"social_insurance_number" gorm:"size:100"`
	ProvidentFundNumber   string    `json:"provident_fund_number" gorm:"size:100"`
	EmergencyContact      string    `json:"emergency_contact" gorm:"size:100"`
	EmergencyPhone        string    `json:"emergency_phone" gorm:"size:100"`
	CurrentAddress        string    `json:"current_address" gorm:"size:255"`
	GraduateSchool        string    `json:"graduate_school" gorm:"size:150"`
	Major                 string    `json:"major" gorm:"size:150"`
	GraduationTime        string    `json:"graduation_time" gorm:"size:20"`
	Email                 string    `json:"email" gorm:"size:120"`
	Remarks               string    `json:"remarks" gorm:"size:255"`
	Status                string    `json:"status" gorm:"size:20;default:'active'"`
	EmploymentStatus      string    `json:"employment_status" gorm:"size:20"`   // 就业状态（trial/formal，正式默认见 employee.go BeforeCreate）
	ProbationEndDate      string    `json:"probation_end_date" gorm:"size:20"`  // 试用期结束日期（YYYY-MM-DD）
	ActualRegularDate     string    `json:"actual_regular_date" gorm:"size:20"` // 实际转正日期（YYYY-MM-DD）
	ResignDate            string    `json:"resign_date" gorm:"size:20"`
	ResignProofPath       string    `json:"-" gorm:"size:255"`
	ResignProofName       string    `json:"resign_proof_name" gorm:"size:255"`
	ResignProofURL        string    `json:"resign_proof_url,omitempty" gorm:"-"`
	ResignReasons         string    `json:"resign_reasons,omitempty" gorm:"type:text"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type SocialInsuranceBatch struct {
	ID               uint                    `json:"id" gorm:"primaryKey"`
	UserID           uint                    `json:"user_id" gorm:"index"`
	ChangeType       string                  `json:"change_type" gorm:"size:20;index"`
	OriginalFileName string                  `json:"original_file_name" gorm:"size:255"`
	StoredFileName   string                  `json:"stored_file_name" gorm:"size:255"`
	StoredFilePath   string                  `json:"stored_file_path" gorm:"size:255"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
	Records          []SocialInsuranceRecord `json:"records,omitempty" gorm:"foreignKey:BatchID;constraint:OnDelete:CASCADE"`
}

type SocialInsuranceRecord struct {
	ID             uint                  `json:"id" gorm:"primaryKey"`
	BatchID        *uint                 `json:"batch_id" gorm:"index"`
	UserID         uint                  `json:"user_id" gorm:"index"`
	ChangeType     string                `json:"change_type" gorm:"size:20;index"`
	EmployeeName   string                `json:"employee_name" gorm:"size:120"`
	Department     string                `json:"department" gorm:"size:150"`
	IdentityNumber string                `json:"identity_number" gorm:"size:40"`
	PersonalNumber string                `json:"personal_number" gorm:"size:40"`
	EffectiveDate  string                `json:"effective_date" gorm:"size:20"`
	Reason         string                `json:"reason" gorm:"size:255"`
	TemplateValues datatypes.JSONMap     `json:"template_values" gorm:"type:json"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	Batch          *SocialInsuranceBatch `json:"batch,omitempty" gorm:"foreignKey:BatchID"`
}

type CallbackUpload struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	UserID       *uint            `json:"user_id,omitempty" gorm:"index"`
	User         *User            `json:"-,omitempty" gorm:"foreignKey:UserID"`
	FileName     string           `json:"file_name" gorm:"size:255"`
	FileSize     int64            `json:"file_size"`
	TotalRecords int              `json:"total_records"`
	UploadedAt   time.Time        `json:"uploaded_at" gorm:"index"`
	RawFile      []byte           `json:"-"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Records      []CallbackRecord `json:"records,omitempty" gorm:"foreignKey:UploadID;constraint:OnDelete:CASCADE"`
}

type CallbackRecord struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	UploadID       uint            `json:"upload_id" gorm:"index"`
	Upload         *CallbackUpload `json:"-,omitempty" gorm:"foreignKey:UploadID"`
	UserID         *uint           `json:"user_id,omitempty" gorm:"index;uniqueIndex:idx_callback_user_identity"`
	User           *User           `json:"-" gorm:"foreignKey:UserID"`
	PersonalNumber string          `json:"personal_number" gorm:"size:120"`
	IdentityNumber string          `json:"identity_number" gorm:"size:60;index;uniqueIndex:idx_callback_user_identity"`
	Name           string          `json:"name" gorm:"size:120"`
	ChangeType     string          `json:"change_type" gorm:"size:120"`
	Phone          string          `json:"phone" gorm:"size:80"`
	Remark         string          `json:"remark" gorm:"size:500"`
	Sequence       int             `json:"sequence"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
