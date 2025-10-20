package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"unicode"

	"gorm.io/gorm"
	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	db                        *gorm.DB
	jwtManager                *auth.JWTManager
	passwordResetService      *service.PasswordResetService
	emailVerificationService  *service.EmailVerificationService
	emailService              *service.EmailService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager, passwordResetService *service.PasswordResetService, emailVerificationService *service.EmailVerificationService, emailService *service.EmailService) *AuthHandler {
	return &AuthHandler{
		db:                        db,
		jwtManager:                jwtManager,
		passwordResetService:      passwordResetService,
		emailVerificationService:  emailVerificationService,
		emailService:              emailService,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
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
		http.Error(w, `{"error":"Username already exists"}`, http.StatusConflict)
		return
	}

	// Check if email already exists
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		http.Error(w, `{"error":"Email already exists"}`, http.StatusConflict)
		return
	}

	// Create new user
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		Active:   true,
	}

	if err := user.SetPassword(req.Password); err != nil {
		http.Error(w, `{"error":"Failed to process password"}`, http.StatusInternalServerError)
		return
	}

	if err := h.db.Create(&user).Error; err != nil {
		http.Error(w, `{"error":"Failed to create user"}`, http.StatusInternalServerError)
		return
	}

	// Create email verification token
	verificationToken, err := h.emailVerificationService.CreateVerificationToken(user.ID)
	if err != nil {
		http.Error(w, `{"error":"Failed to create verification token"}`, http.StatusInternalServerError)
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
		"message": "Registration successful! Please check your email to verify your account.",
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
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Find user by username
	var user models.User
	if err := h.db.Where("username = ? AND active = ?", req.Username, true).First(&user).Error; err != nil {
		http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Check if email is verified
	if !user.EmailVerified {
		http.Error(w, `{"error":"Please verify your email address before logging in"}`, http.StatusForbidden)
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(&user)
	if err != nil {
		http.Error(w, `{"error":"Failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	// Return user and token
	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Logout handles user logout (client-side token invalidation)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// In a JWT-based system, logout is typically handled client-side
	// by removing the token from storage. Here we just return success.
	// For more security, you could implement a token blacklist.

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
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
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
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
		return &ValidationError{"Username must be between 3 and 50 characters"}
	}

	// Username should only contain alphanumeric characters and underscores
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(req.Username) {
		return &ValidationError{"Username can only contain letters, numbers, and underscores"}
	}

	// Validate email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return &ValidationError{"Invalid email format"}
	}

	// Validate password
	if err := h.validatePassword(req.Password); err != nil {
		return err
	}

	// Validate full name length
	if len(req.FullName) > 100 {
		return &ValidationError{"Full name must be less than 100 characters"}
	}

	return nil
}

// validatePassword validates password requirements
func (h *AuthHandler) validatePassword(password string) error {
	if len(password) < 6 {
		return &ValidationError{"Password must be at least 6 characters long"}
	}

	if len(password) > 128 {
		return &ValidationError{"Password must be less than 128 characters long"}
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
		return &ValidationError{"Password must contain at least one letter and one number"}
	}

	return nil
}

// RequestPasswordReset handles password reset requests
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		http.Error(w, `{"error":"Invalid email format"}`, http.StatusBadRequest)
		return
	}

	// Create reset token
	resetToken, err := h.passwordResetService.CreateResetToken(req.Email)
	if err != nil {
		if err == service.ErrUserNotFound {
			// Don't reveal if email exists or not for security
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "If the email exists, a password reset link has been sent",
			})
			return
		}
		http.Error(w, `{"error":"Failed to process request"}`, http.StatusInternalServerError)
		return
	}

	// Send password reset email (non-blocking)
	go func() {
		if err := h.emailService.SendPasswordResetEmail(&resetToken.User, resetToken); err != nil {
			// Log error but don't expose to user
			// log.Printf("Failed to send password reset email: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset confirmation
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
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
			http.Error(w, `{"error":"Invalid or expired reset link"}`, http.StatusBadRequest)
		case service.ErrTokenExpired:
			http.Error(w, `{"error":"Reset link has expired"}`, http.StatusBadRequest)
		case service.ErrTokenAlreadyUsed:
			http.Error(w, `{"error":"Reset link has already been used"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"Failed to reset password"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password reset successfully",
	})
}

// ValidatePasswordResetToken validates a password reset token without using it
func (h *AuthHandler) ValidatePasswordResetToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"Token is required"}`, http.StatusBadRequest)
		return
	}

	// Validate token
	resetToken, err := h.passwordResetService.ValidateResetToken(token)
	if err != nil {
		switch err {
		case service.ErrTokenNotFound:
			http.Error(w, `{"error":"Invalid reset link"}`, http.StatusBadRequest)
		case service.ErrTokenExpired:
			http.Error(w, `{"error":"Reset link has expired"}`, http.StatusBadRequest)
		case service.ErrTokenAlreadyUsed:
			http.Error(w, `{"error":"Reset link has already been used"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"Failed to validate token"}`, http.StatusInternalServerError)
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
		h.emailService.SendWelcomeEmail(&verificationToken.User)
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
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
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
		if err := h.emailService.SendEmailVerificationEmail(&verificationToken.User, verificationToken); err != nil {
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

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}