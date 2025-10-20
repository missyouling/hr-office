package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"unicode"

	"gorm.io/gorm"
	"siapp/internal/auth"
	"siapp/internal/models"
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		db:         db,
		jwtManager: jwtManager,
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

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate new password
	if err := h.validatePassword(req.NewPassword); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Get current user
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	// Verify current password
	if !user.CheckPassword(req.CurrentPassword) {
		http.Error(w, `{"error":"Current password is incorrect"}`, http.StatusBadRequest)
		return
	}

	// Set new password
	if err := user.SetPassword(req.NewPassword); err != nil {
		http.Error(w, `{"error":"Failed to process password"}`, http.StatusInternalServerError)
		return
	}

	// Save user
	if err := h.db.Save(&user).Error; err != nil {
		http.Error(w, `{"error":"Failed to update password"}`, http.StatusInternalServerError)
		return
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

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}