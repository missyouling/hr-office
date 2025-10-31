package service

import (
	"strings"
	"testing"
	"time"

	"siapp/internal/models"
)

func TestEmailService_BaseURLFallback(t *testing.T) {
	t.Setenv("BASE_URL", "")
	service := NewEmailService()
	if service.baseURL != "http://localhost:3000" {
		t.Errorf("基础URL回退不正确，期望 http://localhost:3000，实际为 %s", service.baseURL)
	}
}

func TestEmailService_TestEmailConfiguration(t *testing.T) {
	t.Setenv("BASE_URL", "https://hr-office.example.com")
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USERNAME", "tester")

	service := NewEmailService()
	if err := service.TestEmailConfiguration(); err != nil {
		t.Fatalf("邮件配置检测应通过，实际失败: %v", err)
	}
}

func TestEmailService_SendPasswordResetEmail_MissingConfig(t *testing.T) {
	t.Setenv("BASE_URL", "https://hr-office.example.com")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")

	service := NewEmailService()
	user := &models.User{
		Username:  "tester",
		Email:     "tester@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	token := &models.PasswordResetToken{
		Token:     "reset-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := service.SendPasswordResetEmail(user, token)
	if err == nil {
		t.Fatal("缺少SMTP配置时应返回错误")
	}
	if !strings.Contains(err.Error(), "SMTP配置不完整") {
		t.Errorf("错误信息不符合预期: %v", err)
	}
}
