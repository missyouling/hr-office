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

const (
	passwordResetTokenValidity   = 48 * time.Hour
	passwordResetRequestCooldown = 5 * time.Minute
	passwordResetDailyLimit      = 3
)

// NewPasswordResetService creates a new password reset service instance
func NewPasswordResetService(db *gorm.DB) *PasswordResetService {
	return &PasswordResetService{db: db}
}

var (
	ErrUserNotFound              = errors.New("用户不存在")
	ErrTokenNotFound             = errors.New("密码重置链接无效")
	ErrTokenExpired              = errors.New("密码重置链接已过期")
	ErrTokenAlreadyUsed          = errors.New("密码重置链接已使用")
	ErrInvalidCurrentPassword    = errors.New("当前密码错误")
	ErrPasswordManagedExternally = errors.New("账号密码由外部身份提供方管理")
	ErrResetRequestRateLimited   = errors.New("密码重置请求过于频繁")
	ErrResetRequestDailyLimit    = errors.New("密码重置次数超出每日限制")
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

	now := time.Now()

	// Daily limit: restrict total reset requests per day
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var dailyCount int64
	if err := s.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND created_at >= ?", user.ID, startOfDay).
		Count(&dailyCount).Error; err != nil {
		return nil, err
	}
	if dailyCount >= int64(passwordResetDailyLimit) {
		return nil, ErrResetRequestDailyLimit
	}

	// Rate limiting: disallow frequent requests within cooldown
	var recentToken models.PasswordResetToken
	cutoff := now.Add(-passwordResetRequestCooldown)
	if err := s.db.Where("user_id = ? AND used = ? AND created_at >= ?", user.ID, false, cutoff).
		Order("created_at DESC").
		First(&recentToken).Error; err == nil {
		if !recentToken.IsExpired() {
			return nil, ErrResetRequestRateLimited
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Invalidate any existing tokens for this user
	s.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used = ? AND expires_at > ?", user.ID, false, now).
		Update("used", true)

	// Create new token
	resetToken := &models.PasswordResetToken{
		UserID:    &user.ID,
		ExpiresAt: now.Add(passwordResetTokenValidity),
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

// ChangePassword 校验当前密码后更新密码，并在同一事务内吊销该用户所有未过期 refresh token。
// SupabaseUID 非空的用户密码由外部身份提供方（Supabase Auth）管理，本地改密无法同步，明确拒绝。
func (s *PasswordResetService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不泄露用户是否存在，与"当前密码错误"等价处理
			return ErrInvalidCurrentPassword
		}
		return err
	}

	// 外部身份管理：明确拒绝，禁止只改本地密码造成双源不一致
	if user.SupabaseUID != "" {
		return ErrPasswordManagedExternally
	}

	// 校验当前密码
	if !user.CheckPassword(currentPassword) {
		return ErrInvalidCurrentPassword
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// 同库事务：更新密码 + 吊销该用户所有未过期 refresh token，任一步失败整体回滚
	tx := s.db.Begin()
	if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("password", user.Password).Error; err != nil {
		tx.Rollback()
		return err
	}
	now := time.Now()
	if err := tx.Model(&models.AuthToken{}).
		Where("user_id = ? AND type = ? AND is_revoked = ? AND expires_at > ?", userID, "refresh", false, now).
		Update("is_revoked", true).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
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
