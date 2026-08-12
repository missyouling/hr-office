package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	db                       *gorm.DB
	jwtManager               *auth.JWTManager
	passwordResetService     *service.PasswordResetService
	emailVerificationService *service.EmailVerificationService
	emailService             *service.EmailService
	rateLimiter              *middleware.LoginRateLimiter
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager, passwordResetService *service.PasswordResetService, emailVerificationService *service.EmailVerificationService, emailService *service.EmailService, rateLimiter *middleware.LoginRateLimiter) *AuthHandler {
	return &AuthHandler{
		db:                       db,
		jwtManager:               jwtManager,
		passwordResetService:     passwordResetService,
		emailVerificationService: emailVerificationService,
		emailService:             emailService,
		rateLimiter:              rateLimiter,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// Validate input
	if err := h.validateRegistration(&req); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var existingUser models.User
	if err := h.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		http.Error(w, `{"error":"用户名已存在"}`, http.StatusConflict)
		return
	}

	// Check if email already exists
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		http.Error(w, `{"error":"邮箱地址已存在"}`, http.StatusConflict)
		return
	}

	// Create new user
	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		FullName:  req.FullName,
		CompanyID: req.CompanyID,
		Active:    true,
	}

	if err := user.SetPassword(req.Password); err != nil {
		http.Error(w, `{"error":"密码处理失败"}`, http.StatusInternalServerError)
		return
	}

	if err := h.db.Create(&user).Error; err != nil {
		http.Error(w, `{"error":"创建用户失败"}`, http.StatusInternalServerError)
		return
	}

	// Create email verification token
	verificationToken, err := h.emailVerificationService.CreateVerificationToken(user.ID)
	if err != nil {
		http.Error(w, `{"error":"创建验证令牌失败"}`, http.StatusInternalServerError)
		return
	}

	// Send email verification email (non-blocking)
	go func() {
		if err := h.emailService.SendEmailVerificationEmail(&user, verificationToken); err != nil {
			// Log error but don't fail registration
			// log.Printf("Failed to send verification email to %s: %v", user.Email, err)
		}
	}()

	// Return success message (no token until email is verified)
	response := map[string]interface{}{
		"message": "注册成功！请检查邮箱以验证您的账户。",
		"email":   user.Email,
		"user_id": user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// 查找活跃用户
	var user models.User
	if err := h.db.Where("username = ? AND active = ?", req.Username, true).First(&user).Error; err != nil {
		http.Error(w, `{"error":"用户名或密码错误"}`, http.StatusUnauthorized)
		return
	}

	// 获取客户端 IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	}

	// 限流检查：账号锁定（连续 5 次失败 → 锁定 15 分钟）
	if h.rateLimiter.IsUserLocked(req.Username) {
		http.Error(w, `{"error":"账号已被锁定，请15分钟后再试"}`, http.StatusLocked)
		return
	}

	// 限流检查：IP 频率限制（每分钟最多 10 次失败）
	if h.rateLimiter.IsIPBlocked(clientIP) {
		http.Error(w, `{"error":"请求过于频繁，请稍后再试"}`, http.StatusTooManyRequests)
		return
	}

	// 校验密码
	if !user.CheckPassword(req.Password) {
		h.rateLimiter.RecordFailure(req.Username, clientIP)
		if h.rateLimiter.IsUserLocked(req.Username) {
			http.Error(w, `{"error":"账号已被锁定，请15分钟后再试"}`, http.StatusLocked)
			return
		}
		http.Error(w, `{"error":"用户名或密码错误"}`, http.StatusUnauthorized)
		return
	}

	// 密码正确，清空限流计数
	h.rateLimiter.Reset(req.Username, clientIP)

	// 检查邮箱是否已验证
	if !user.EmailVerified {
		http.Error(w, `{"error":"请先验证邮箱地址后再登录"}`, http.StatusForbidden)
		return
	}

	// 签发 access + refresh token 对
	now := time.Now()
	accessExpiry := now.Add(24 * time.Hour)
	refreshExpiry := now.Add(7 * 24 * time.Hour)

	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		http.Error(w, `{"error":"生成登录令牌失败"}`, http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		http.Error(w, `{"error":"生成刷新令牌失败"}`, http.StatusInternalServerError)
		return
	}

	// 入库 auth_tokens（事务保护）
	tx := h.db.Begin()
	if err := saveAuthToken(tx, user.ID, accessToken, "access", accessExpiry); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"保存令牌失败"}`, http.StatusInternalServerError)
		return
	}
	if err := saveAuthToken(tx, user.ID, refreshToken, "refresh", refreshExpiry); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"保存刷新令牌失败"}`, http.StatusInternalServerError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		http.Error(w, `{"error":"提交令牌失败"}`, http.StatusInternalServerError)
		return
	}

	// 加载用户权限（从 user_roles → role_permissions → permissions 联表查询）
	permissions, err := loadUserPermissions(h.db, user.ID)
	if err != nil {
		// 权限加载失败不应阻断登录，降级返回空数组
		permissions = []string{}
	}

	// 返回用户信息和 token 对
	response := AuthTokenResponse{
		AuthUserPayload: newAuthUserPayload(user, permissions),
		Token:           accessToken,
		RefreshToken:    refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckAccountAvailability verifies whether username/email is already registered in Supabase
func (h *AuthHandler) CheckAccountAvailability(w http.ResponseWriter, r *http.Request) {
	var req models.AccountAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(req.Email)
	username := strings.TrimSpace(req.Username)

	if email == "" && username == "" {
		http.Error(w, `{"error":"缺少邮箱或用户名"}`, http.StatusBadRequest)
		return
	}

	supabaseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if supabaseURL == "" || serviceKey == "" {
		http.Error(w, `{"error":"Supabase 配置缺失"}`, http.StatusInternalServerError)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	emailAvailable := true
	if email != "" {
		exists, err := supabaseEmailExists(client, supabaseURL, serviceKey, email)
		if err != nil {
			http.Error(w, `{"error":"检查邮箱可用性失败"}`, http.StatusInternalServerError)
			return
		}
		emailAvailable = !exists
	}

	usernameAvailable := true
	if username != "" {
		exists, err := supabaseUsernameExists(client, supabaseURL, serviceKey, username)
		if err != nil {
			http.Error(w, `{"error":"检查用户名可用性失败"}`, http.StatusInternalServerError)
			return
		}
		usernameAvailable = !exists
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AccountAvailabilityResponse{
		EmailAvailable:    emailAvailable,
		UsernameAvailable: usernameAvailable,
	})
}

func supabaseEmailExists(client *http.Client, baseURL, serviceKey, email string) (bool, error) {
	checkURL := fmt.Sprintf("%s/auth/v1/admin/users?per_page=200&page=1", baseURL)
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Authorization", "Bearer "+serviceKey)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("supabase admin users returned status %d", resp.StatusCode)
	}

	var out struct {
		Users []struct {
			Email string `json:"email"`
		} `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("decode supabase admin users response: %w", err)
	}

	for _, user := range out.Users {
		if strings.EqualFold(user.Email, email) {
			return true, nil
		}
	}

	return false, nil
}

func supabaseUsernameExists(client *http.Client, baseURL, serviceKey, username string) (bool, error) {
	escaped := url.QueryEscape(username)
	checkURL := fmt.Sprintf("%s/rest/v1/profiles?select=username&username=eq.%s", baseURL, escaped)
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Authorization", "Bearer "+serviceKey)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("supabase profiles returned status %d", resp.StatusCode)
	}

	var profiles []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return false, fmt.Errorf("decode supabase profiles response: %w", err)
	}

	for _, profile := range profiles {
		if strings.EqualFold(profile.Username, username) {
			return true, nil
		}
	}

	return false, nil
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	permissions, err := loadUserPermissions(h.db, user.ID)
	if err != nil {
		permissions = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newAuthUserPayload(user, permissions))
}

// Logout handles user logout — 吊销所有 refresh token，access token 等自然过期
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		// 未登录也返回成功（幂等）
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "已退出登录"})
		return
	}

	// 吊销该用户所有未过期的 refresh token
	now := time.Now()
	h.db.Model(&models.AuthToken{}).
		Where("user_id = ? AND type = ? AND is_revoked = ? AND expires_at > ?", userID, "refresh", false, now).
		Update("is_revoked", true)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "已退出登录"})
}

// ChangePassword handles password change for authenticated users
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// Validate new password
	if err := h.validatePassword(req.NewPassword); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Use password reset service to change password
	if err := h.passwordResetService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		if err == service.ErrInvalidCurrentPassword {
			http.Error(w, `{"error":"Current password is incorrect"}`, http.StatusBadRequest)
		} else {
			http.Error(w, `{"error":"Failed to change password"}`, http.StatusInternalServerError)
		}
		return
	}

	// Get user for email notification
	var user models.User
	if err := h.db.First(&user, userID).Error; err == nil {
		// Send password changed notification (non-blocking)
		go func() {
			h.emailService.SendPasswordChangedEmail(&user)
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}

// validateRegistration validates the registration request
func (h *AuthHandler) validateRegistration(req *models.RegisterRequest) error {
	// Validate username
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return &ValidationError{"用户名长度必须在3-50个字符之间"}
	}

	// Username should only contain alphanumeric characters and underscores
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(req.Username) {
		return &ValidationError{"用户名只能包含字母、数字和下划线"}
	}

	// Validate email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return &ValidationError{"邮箱格式无效"}
	}

	// Validate password
	if err := h.validatePassword(req.Password); err != nil {
		return err
	}

	// Validate full name length
	if len(req.FullName) > 100 {
		return &ValidationError{"姓名长度不能超过100个字符"}
	}

	// Validate company ID
	if req.CompanyID == "" {
		return &ValidationError{"请选择所属公司"}
	}

	return nil
}

// validatePassword validates password requirements
func (h *AuthHandler) validatePassword(password string) error {
	if len(password) < 6 {
		return &ValidationError{"密码长度至少需要6个字符"}
	}

	if len(password) > 128 {
		return &ValidationError{"密码长度不能超过128个字符"}
	}

	// Check for at least one letter and one number
	hasLetter := false
	hasNumber := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsNumber(char) {
			hasNumber = true
		}
	}

	if !hasLetter || !hasNumber {
		return &ValidationError{"密码必须包含至少一个字母和一个数字"}
	}

	return nil
}

// RequestPasswordReset handles password reset requests
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		http.Error(w, `{"error":"邮箱格式不正确"}`, http.StatusBadRequest)
		return
	}

	// Create reset token
	resetToken, err := h.passwordResetService.CreateResetToken(req.Email)
	if err != nil {
		if err == service.ErrResetRequestRateLimited {
			http.Error(w, `{"error":"密码重置请求过于频繁，请在5分钟后再试"}`, http.StatusTooManyRequests)
			return
		}
		if err == service.ErrResetRequestDailyLimit {
			http.Error(w, `{"error":"今日密码重置次数已达上限，请明日再试或联系管理员"}`, http.StatusTooManyRequests)
			return
		}
		if err == service.ErrUserNotFound {
			// Don't reveal if email exists or not for security
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "如果该邮箱存在，我们已发送密码重置链接",
			})
			return
		}
		http.Error(w, `{"error":"请求处理失败"}`, http.StatusInternalServerError)
		return
	}

	// Send password reset email (non-blocking)
	go func() {
		if err := h.emailService.SendPasswordResetEmail(resetToken.User, resetToken); err != nil {
			// Log error but don't expose to user
			// log.Printf("Failed to send password reset email: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "如果该邮箱存在，我们已发送密码重置链接",
	})
}

// ResetPassword handles password reset confirmation
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// Validate new password
	if err := h.validatePassword(req.NewPassword); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Reset password using token
	err := h.passwordResetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		switch err {
		case service.ErrTokenNotFound:
			http.Error(w, `{"error":"重置链接无效"}`, http.StatusBadRequest)
		case service.ErrTokenExpired:
			http.Error(w, `{"error":"重置链接已过期"}`, http.StatusBadRequest)
		case service.ErrTokenAlreadyUsed:
			http.Error(w, `{"error":"重置链接已被使用"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"重置密码失败"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "密码重置成功",
	})
}

// ValidatePasswordResetToken validates a password reset token without using it
func (h *AuthHandler) ValidatePasswordResetToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"缺少重置令牌"}`, http.StatusBadRequest)
		return
	}

	// Validate token
	resetToken, err := h.passwordResetService.ValidateResetToken(token)
	if err != nil {
		switch err {
		case service.ErrTokenNotFound:
			http.Error(w, `{"error":"重置链接无效"}`, http.StatusBadRequest)
		case service.ErrTokenExpired:
			http.Error(w, `{"error":"重置链接已过期"}`, http.StatusBadRequest)
		case service.ErrTokenAlreadyUsed:
			http.Error(w, `{"error":"重置链接已被使用"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"校验重置链接失败"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"email": resetToken.User.Email,
	})
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"Token is required"}`, http.StatusBadRequest)
		return
	}

	// Get verification token before verifying (to get user info)
	verificationToken, err := h.emailVerificationService.ValidateVerificationToken(token)
	if err != nil {
		switch err {
		case service.ErrVerificationTokenNotFound:
			http.Error(w, `{"error":"Invalid verification link"}`, http.StatusBadRequest)
		case service.ErrVerificationTokenExpired:
			http.Error(w, `{"error":"Verification link has expired"}`, http.StatusBadRequest)
		case service.ErrVerificationTokenUsed:
			http.Error(w, `{"error":"Verification link has already been used"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"Failed to validate token"}`, http.StatusInternalServerError)
		}
		return
	}

	// Verify email using token
	err = h.emailVerificationService.VerifyEmail(token)
	if err != nil {
		if err == service.ErrEmailAlreadyVerified {
			http.Error(w, `{"error":"Email is already verified"}`, http.StatusBadRequest)
		} else {
			http.Error(w, `{"error":"Failed to verify email"}`, http.StatusInternalServerError)
		}
		return
	}

	// Send welcome email (non-blocking)
	go func() {
		h.emailService.SendWelcomeEmail(verificationToken.User)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully! You can now log in.",
	})
}

// ResendVerificationEmail resends the email verification email
func (h *AuthHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	var req models.EmailVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		http.Error(w, `{"error":"Invalid email format"}`, http.StatusBadRequest)
		return
	}

	// Create new verification token
	verificationToken, err := h.emailVerificationService.CreateVerificationTokenByEmail(req.Email)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			// Don't reveal if email exists or not for security
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "If the email exists and is not verified, a verification link has been sent",
			})
			return
		case service.ErrEmailAlreadyVerified:
			http.Error(w, `{"error":"Email is already verified"}`, http.StatusBadRequest)
			return
		case service.ErrUserNotActive:
			http.Error(w, `{"error":"User account is not active"}`, http.StatusBadRequest)
			return
		default:
			http.Error(w, `{"error":"Failed to process request"}`, http.StatusInternalServerError)
			return
		}
	}

	// Send verification email (non-blocking)
	go func() {
		if err := h.emailService.SendEmailVerificationEmail(verificationToken.User, verificationToken); err != nil {
			// Log error but don't expose to user
			// log.Printf("Failed to send verification email: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "If the email exists and is not verified, a verification link has been sent",
	})
}

// CheckEmailVerificationStatus checks if a user's email is verified
func (h *AuthHandler) CheckEmailVerificationStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	verified, err := h.emailVerificationService.IsEmailVerified(userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to check verification status"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email_verified": verified,
	})
}

// saveAuthToken 将 access/refresh token 的 SHA-256 哈希写入 auth_tokens 表
func saveAuthToken(tx *gorm.DB, userID uint, rawToken, tokenType string, expiresAt time.Time) error {
	tokenHash := auth.HashToken(rawToken)
	record := models.AuthToken{
		UserID:    userID,
		TokenHash: tokenHash,
		Type:      tokenType,
		ExpiresAt: expiresAt,
	}
	return tx.Create(&record).Error
}

// Refresh 使用 refresh token 续签新的 access + refresh 对（旋转机制）
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"无效的请求内容"}`, http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, `{"error":"refresh_token 不能为空"}`, http.StatusBadRequest)
		return
	}

	// 验证 refresh token 的 JWT 签名与有效期
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"refresh_token 无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	// 确认 token 类型为 refresh（防止用 access token 续签）
	if claims.ID != "refresh" {
		http.Error(w, `{"error":"仅支持 refresh token 刷新"}`, http.StatusUnauthorized)
		return
	}

	// 查库：refresh token 必须存在、未被吊销
	tokenHash := auth.HashToken(req.RefreshToken)
	var storedToken models.AuthToken
	if err := h.db.Where("token_hash = ? AND type = ? AND is_revoked = ?", tokenHash, "refresh", false).
		First(&storedToken).Error; err != nil {
		http.Error(w, `{"error":"refresh_token 已被吊销或不存在"}`, http.StatusUnauthorized)
		return
	}

	// 如果 refresh token 已过期，标记吊销并返回错误
	if time.Now().After(storedToken.ExpiresAt) {
		h.db.Model(&storedToken).Update("is_revoked", true)
		http.Error(w, `{"error":"refresh_token 已过期"}`, http.StatusUnauthorized)
		return
	}

	// 旧 refresh token 标记吊销（旋转）
	h.db.Model(&storedToken).Update("is_revoked", true)

	// 签发新的 access + refresh 对
	now := time.Now()
	accessExpiry := now.Add(24 * time.Hour)
	refreshExpiry := now.Add(7 * 24 * time.Hour)

	// 获取用户名
	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		http.Error(w, `{"error":"用户不存在"}`, http.StatusUnauthorized)
		return
	}

	newAccessToken, err := h.jwtManager.GenerateAccessToken(claims.UserID, user.Username)
	if err != nil {
		http.Error(w, `{"error":"生成新令牌失败"}`, http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := h.jwtManager.GenerateRefreshToken(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"生成新刷新令牌失败"}`, http.StatusInternalServerError)
		return
	}

	// 入库新 token 对
	tx := h.db.Begin()
	if err := saveAuthToken(tx, claims.UserID, newAccessToken, "access", accessExpiry); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"保存新令牌失败"}`, http.StatusInternalServerError)
		return
	}
	if err := saveAuthToken(tx, claims.UserID, newRefreshToken, "refresh", refreshExpiry); err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"保存新刷新令牌失败"}`, http.StatusInternalServerError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		http.Error(w, `{"error":"提交令牌失败"}`, http.StatusInternalServerError)
		return
	}

	// 加载用户权限（刷新时也返回最新权限，确保权限变更即时生效）
	permissions, err := loadUserPermissions(h.db, user.ID)
	if err != nil {
		permissions = []string{}
	}

	resp := AuthTokenResponse{
		AuthUserPayload: newAuthUserPayload(user, permissions),
		Token:           newAccessToken,
		RefreshToken:    newRefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
