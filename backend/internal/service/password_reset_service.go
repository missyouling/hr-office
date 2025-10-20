package service

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"siapp/internal/models"
)

// PasswordResetService handles password reset operations
type PasswordResetService struct {
	db *gorm.DB
}

// NewPasswordResetService creates a new password reset service instance
func NewPasswordResetService(db *gorm.DB) *PasswordResetService {
	return &PasswordResetService{db: db}
}

var (
	ErrUserNotFound         = errors.New("用户不存在")
	ErrTokenNotFound        = errors.New("密码重置链接无效")
	ErrTokenExpired         = errors.New("密码重置链接已过期")
	ErrTokenAlreadyUsed     = errors.New("密码重置链接已使用")
	ErrInvalidCurrentPassword = errors.New("当前密码错误")
)

// CreateResetToken creates a new password reset token for the user
func (s *PasswordResetService) CreateResetToken(email string) (*models.PasswordResetToken, error) {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ? AND active = ?", email, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Invalidate any existing tokens for this user
	s.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used = ? AND expires_at > ?", user.ID, false, time.Now()).
		Update("used", true)

	// Create new token
	resetToken := &models.PasswordResetToken{
		UserID:    &user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours validity
		Used:      false,
	}

	if err := resetToken.GenerateToken(); err != nil {
		return nil, err
	}

	if err := s.db.Create(resetToken).Error; err != nil {
		return nil, err
	}

	// Preload user for potential email sending
	s.db.Preload("User").First(resetToken, resetToken.ID)

	return resetToken, nil
}

// ValidateResetToken validates a password reset token
func (s *PasswordResetService) ValidateResetToken(token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken
	if err := s.db.Preload("User").Where("token = ?", token).First(&resetToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	if resetToken.Used {
		return nil, ErrTokenAlreadyUsed
	}

	if resetToken.IsExpired() {
		return nil, ErrTokenExpired
	}

	return &resetToken, nil
}

// ResetPassword resets the user's password using a valid token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	// Validate token
	resetToken, err := s.ValidateResetToken(token)
	if err != nil {
		return err
	}

	// Update user password
	var user models.User
	if err := s.db.First(&user, resetToken.UserID).Error; err != nil {
		return err
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// Start transaction
	tx := s.db.Begin()

	// Update user password
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Mark token as used
	now := time.Now()
	if err := tx.Model(resetToken).Updates(map[string]interface{}{
		"used":    true,
		"used_at": &now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ChangePassword changes the user's password with current password verification
func (s *PasswordResetService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	// Verify current password
	if !user.CheckPassword(currentPassword) {
		return ErrInvalidCurrentPassword
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.db.Save(&user).Error
}

// CleanupExpiredTokens removes expired and used tokens older than specified days
func (s *PasswordResetService) CleanupExpiredTokens(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	result := s.db.Where("(used = ? OR expires_at < ?) AND created_at < ?",
		true, time.Now(), cutoffDate).Delete(&models.PasswordResetToken{})

	return result.Error
}

// GetUserResetTokens gets all reset tokens for a user (for admin purposes)
func (s *PasswordResetService) GetUserResetTokens(userID uint) ([]models.PasswordResetToken, error) {
	var tokens []models.PasswordResetToken
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}