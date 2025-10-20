package service

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"siapp/internal/models"
)

// EmailVerificationService handles email verification operations
type EmailVerificationService struct {
	db *gorm.DB
}

// NewEmailVerificationService creates a new email verification service instance
func NewEmailVerificationService(db *gorm.DB) *EmailVerificationService {
	return &EmailVerificationService{db: db}
}

var (
	ErrEmailAlreadyVerified   = errors.New("邮箱已经验证过了")
	ErrVerificationTokenNotFound = errors.New("验证链接无效")
	ErrVerificationTokenExpired  = errors.New("验证链接已过期")
	ErrVerificationTokenUsed     = errors.New("验证链接已使用")
	ErrUserNotActive            = errors.New("用户账户未激活")
)

// CreateVerificationToken creates a new email verification token for the user
func (s *EmailVerificationService) CreateVerificationToken(userID uint) (*models.EmailVerificationToken, error) {
	// Check if user exists and is active
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if !user.Active {
		return nil, ErrUserNotActive
	}

	// Check if email is already verified
	if user.EmailVerified {
		return nil, ErrEmailAlreadyVerified
	}

	// Invalidate any existing verification tokens for this user
	s.db.Model(&models.EmailVerificationToken{}).
		Where("user_id = ? AND used = ? AND expires_at > ?", userID, false, time.Now()).
		Update("used", true)

	// Create new verification token
	verificationToken := &models.EmailVerificationToken{
		UserID:    &userID,
		ExpiresAt: time.Now().Add(48 * time.Hour), // 48 hours validity
		Used:      false,
	}

	if err := verificationToken.GenerateToken(); err != nil {
		return nil, err
	}

	if err := s.db.Create(verificationToken).Error; err != nil {
		return nil, err
	}

	// Preload user for potential email sending
	s.db.Preload("User").First(verificationToken, verificationToken.ID)

	return verificationToken, nil
}

// CreateVerificationTokenByEmail creates a verification token by email address
func (s *EmailVerificationService) CreateVerificationTokenByEmail(email string) (*models.EmailVerificationToken, error) {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ? AND active = ?", email, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return s.CreateVerificationToken(user.ID)
}

// ValidateVerificationToken validates an email verification token
func (s *EmailVerificationService) ValidateVerificationToken(token string) (*models.EmailVerificationToken, error) {
	var verificationToken models.EmailVerificationToken
	if err := s.db.Preload("User").Where("token = ?", token).First(&verificationToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVerificationTokenNotFound
		}
		return nil, err
	}

	if verificationToken.Used {
		return nil, ErrVerificationTokenUsed
	}

	if verificationToken.IsExpired() {
		return nil, ErrVerificationTokenExpired
	}

	return &verificationToken, nil
}

// VerifyEmail verifies the user's email using a valid token
func (s *EmailVerificationService) VerifyEmail(token string) error {
	// Validate token
	verificationToken, err := s.ValidateVerificationToken(token)
	if err != nil {
		return err
	}

	// Check if user is already verified
	var user models.User
	if err := s.db.First(&user, verificationToken.UserID).Error; err != nil {
		return err
	}

	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Start transaction
	tx := s.db.Begin()

	// Mark user email as verified
	now := time.Now()
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"email_verified":    true,
		"email_verified_at": &now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Mark token as used
	if err := tx.Model(verificationToken).Updates(map[string]interface{}{
		"used":    true,
		"used_at": &now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// IsEmailVerified checks if a user's email is verified
func (s *EmailVerificationService) IsEmailVerified(userID uint) (bool, error) {
	var user models.User
	if err := s.db.Select("email_verified").First(&user, userID).Error; err != nil {
		return false, err
	}
	return user.EmailVerified, nil
}

// GetUserVerificationTokens gets all verification tokens for a user (for admin purposes)
func (s *EmailVerificationService) GetUserVerificationTokens(userID uint) ([]models.EmailVerificationToken, error) {
	var tokens []models.EmailVerificationToken
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

// CleanupExpiredTokens removes expired and used verification tokens older than specified days
func (s *EmailVerificationService) CleanupExpiredTokens(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	result := s.db.Where("(used = ? OR expires_at < ?) AND created_at < ?",
		true, time.Now(), cutoffDate).Delete(&models.EmailVerificationToken{})

	return result.Error
}

// GetUnverifiedUsers returns users who haven't verified their email within specified days
func (s *EmailVerificationService) GetUnverifiedUsers(daysSinceRegistration int) ([]models.User, error) {
	cutoffDate := time.Now().AddDate(0, 0, -daysSinceRegistration)

	var users []models.User
	err := s.db.Where("email_verified = ? AND created_at < ?", false, cutoffDate).
		Find(&users).Error

	return users, err
}