package api

import (
	"time"

	"siapp/internal/models"
)

// AuthUserResponse 是认证接口暴露的安全用户字段，不直接复用数据库模型作为 API 契约。
type AuthUserResponse struct {
	ID              uint       `json:"id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	FullName        string     `json:"full_name"`
	CompanyID       string     `json:"company_id"`
	Department      string     `json:"department"`
	DepartmentID    *uint      `json:"department_id"`
	Active          bool       `json:"active"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AuthUserPayload 是登录、刷新和个人资料接口共用的认证用户结构。
type AuthUserPayload struct {
	User        AuthUserResponse `json:"user"`
	Permissions []string         `json:"permissions"`
}

// AuthTokenResponse 在认证用户结构上附加现有令牌字段。
type AuthTokenResponse struct {
	AuthUserPayload
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func newAuthUserPayload(user models.User, permissions []string) AuthUserPayload {
	return AuthUserPayload{
		User: AuthUserResponse{
			ID: user.ID, Username: user.Username, Email: user.Email, FullName: user.FullName,
			CompanyID: user.CompanyID, Department: user.Department, DepartmentID: user.DepartmentID,
			Active: user.Active, EmailVerified: user.EmailVerified, EmailVerifiedAt: user.EmailVerifiedAt,
			CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
		},
		Permissions: permissions,
	}
}
