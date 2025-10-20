package models

import (
	"encoding/json"
	"time"
)

// AuditLog represents an audit log entry for tracking user operations
type AuditLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      *uint     `json:"user_id,omitempty" gorm:"index"` // nullable for system events
	User        *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Action      string    `json:"action" gorm:"size:100;not null;index"` // e.g., "LOGIN", "UPLOAD_FILE", "CREATE_PERIOD"
	Resource    string    `json:"resource" gorm:"size:100;index"`         // e.g., "users", "periods", "files"
	ResourceID  *string   `json:"resource_id,omitempty" gorm:"size:50"`   // ID of affected resource
	Method      string    `json:"method" gorm:"size:10"`                  // HTTP method
	Path        string    `json:"path" gorm:"size:255"`                   // Request path
	IPAddress   string    `json:"ip_address" gorm:"size:45;index"`        // Client IP address
	UserAgent   string    `json:"user_agent" gorm:"size:500"`             // Client user agent
	Status      string    `json:"status" gorm:"size:20;index"`            // SUCCESS, FAILED, ERROR
	StatusCode  int       `json:"status_code" gorm:"index"`               // HTTP status code
	ErrorMsg    string    `json:"error_msg,omitempty" gorm:"size:1000"`   // Error message if failed
	Duration    int64     `json:"duration"` // Request duration in milliseconds
	Details     string    `json:"details,omitempty" gorm:"type:text"`     // Additional JSON details
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}

// ActionType defines the types of actions that can be logged
type ActionType string

const (
	// Authentication actions
	ActionLogin               ActionType = "LOGIN"
	ActionLogout              ActionType = "LOGOUT"
	ActionRegister            ActionType = "REGISTER"
	ActionChangePassword      ActionType = "CHANGE_PASSWORD"
	ActionTokenRefresh        ActionType = "TOKEN_REFRESH"
	ActionRequestPasswordReset ActionType = "REQUEST_PASSWORD_RESET"
	ActionResetPassword       ActionType = "RESET_PASSWORD"
	ActionValidateResetToken  ActionType = "VALIDATE_RESET_TOKEN"
	ActionSendVerificationEmail ActionType = "SEND_VERIFICATION_EMAIL"
	ActionVerifyEmail         ActionType = "VERIFY_EMAIL"
	ActionResendVerification  ActionType = "RESEND_VERIFICATION"

	// Period management actions
	ActionCreatePeriod ActionType = "CREATE_PERIOD"
	ActionDeletePeriod ActionType = "DELETE_PERIOD"
	ActionResetPeriod  ActionType = "RESET_PERIOD"
	ActionProcessPeriod ActionType = "PROCESS_PERIOD"

	// File operations
	ActionUploadFile       ActionType = "UPLOAD_FILE"
	ActionUploadBatch      ActionType = "UPLOAD_BATCH"
	ActionUploadRoster     ActionType = "UPLOAD_ROSTER"
	ActionUploadAdjustment ActionType = "UPLOAD_ADJUSTMENT"
	ActionClearFiles       ActionType = "CLEAR_FILES"
	ActionClearAdjustments ActionType = "CLEAR_ADJUSTMENTS"

	// Data export actions
	ActionExportCharges ActionType = "EXPORT_CHARGES"
	ActionExportScheme  ActionType = "EXPORT_SCHEME"
	ActionDownloadTemplate ActionType = "DOWNLOAD_TEMPLATE"

	// System actions
	ActionSystemStart  ActionType = "SYSTEM_START"
	ActionSystemError  ActionType = "SYSTEM_ERROR"
	ActionHealthCheck  ActionType = "HEALTH_CHECK"

	// Security events
	ActionAuthFailure     ActionType = "AUTH_FAILURE"
	ActionPermissionDenied ActionType = "PERMISSION_DENIED"
	ActionInvalidToken    ActionType = "INVALID_TOKEN"
	ActionRateLimitHit    ActionType = "RATE_LIMIT_HIT"
)

// LogStatus defines the status of logged actions
type LogStatus string

const (
	StatusSuccess LogStatus = "SUCCESS"
	StatusFailed  LogStatus = "FAILED"
	StatusError   LogStatus = "ERROR"
)

// LogDetails provides structured details for audit logs
type LogDetails struct {
	FileName   string                 `json:"file_name,omitempty"`
	FileSize   int64                  `json:"file_size,omitempty"`
	FileType   string                 `json:"file_type,omitempty"`
	Scheme     string                 `json:"scheme,omitempty"`
	Part       string                 `json:"part,omitempty"`
	RecordCount int                   `json:"record_count,omitempty"`
	PeriodID   uint                   `json:"period_id,omitempty"`
	YearMonth  string                 `json:"year_month,omitempty"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// ToJSON converts LogDetails to JSON string
func (d *LogDetails) ToJSON() string {
	if d == nil {
		return ""
	}
	data, _ := json.Marshal(d)
	return string(data)
}

// ParseLogDetails parses JSON string to LogDetails
func ParseLogDetails(jsonStr string) *LogDetails {
	if jsonStr == "" {
		return nil
	}
	var details LogDetails
	json.Unmarshal([]byte(jsonStr), &details)
	return &details
}

// CreateAuditLogParams holds parameters for creating an audit log entry
type CreateAuditLogParams struct {
	UserID     *uint
	Action     ActionType
	Resource   string
	ResourceID *string
	Method     string
	Path       string
	IPAddress  string
	UserAgent  string
	Status     LogStatus
	StatusCode int
	ErrorMsg   string
	Duration   int64
	Details    *LogDetails
}

// CreateAuditLog creates a new audit log entry
func CreateAuditLog(params CreateAuditLogParams) *AuditLog {
	log := &AuditLog{
		UserID:     params.UserID,
		Action:     string(params.Action),
		Resource:   params.Resource,
		ResourceID: params.ResourceID,
		Method:     params.Method,
		Path:       params.Path,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		Status:     string(params.Status),
		StatusCode: params.StatusCode,
		ErrorMsg:   params.ErrorMsg,
		Duration:   params.Duration,
		Details:    params.Details.ToJSON(),
		CreatedAt:  time.Now(),
	}
	return log
}

// GetParsedDetails returns parsed details as LogDetails struct
func (al *AuditLog) GetParsedDetails() *LogDetails {
	return ParseLogDetails(al.Details)
}

// TableName specifies the table name for GORM
func (AuditLog) TableName() string {
	return "audit_logs"
}