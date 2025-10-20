package service

import (
	"fmt"
	"log"
	"os"

	"siapp/internal/models"
)

// EmailService handles email operations
type EmailService struct {
	baseURL string
}

// NewEmailService creates a new email service instance
func NewEmailService() *EmailService {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000" // Default frontend URL
	}
	return &EmailService{baseURL: baseURL}
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject string
	Body    string
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(user *models.User, resetToken *models.PasswordResetToken) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, resetToken.Token)

	template := s.getPasswordResetTemplate(user.Username, resetURL)

	// In a real implementation, you would use an email service like SendGrid, AWS SES, etc.
	// For now, we'll log the email content
	log.Printf("=== 密码重置邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)
	log.Printf("内容:\n%s", template.Body)
	log.Printf("==================")

	// TODO: Implement actual email sending
	// This is where you would integrate with your email service provider

	return nil
}

// SendEmailVerificationEmail sends an email verification email
func (s *EmailService) SendEmailVerificationEmail(user *models.User, verificationToken *models.EmailVerificationToken) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, verificationToken.Token)

	template := s.getEmailVerificationTemplate(user.Username, verificationURL)

	log.Printf("=== 邮箱验证邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)
	log.Printf("内容:\n%s", template.Body)
	log.Printf("==================")

	return nil
}

// SendWelcomeEmail sends a welcome email to new users (now sent after email verification)
func (s *EmailService) SendWelcomeEmail(user *models.User) error {
	template := s.getWelcomeTemplate(user.Username)

	log.Printf("=== 欢迎邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)
	log.Printf("内容:\n%s", template.Body)
	log.Printf("==============")

	return nil
}

// SendPasswordChangedEmail sends a notification when password is changed
func (s *EmailService) SendPasswordChangedEmail(user *models.User) error {
	template := s.getPasswordChangedTemplate(user.Username)

	log.Printf("=== 密码更改通知 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)
	log.Printf("内容:\n%s", template.Body)
	log.Printf("================")

	return nil
}

func (s *EmailService) getPasswordResetTemplate(username, resetURL string) EmailTemplate {
	return EmailTemplate{
		Subject: "社保整合系统 - 密码重置",
		Body: fmt.Sprintf(`尊敬的 %s，

您好！我们收到了您的密码重置请求。

请点击以下链接重置您的密码：
%s

此链接将在24小时后过期。

如果您没有请求重置密码，请忽略此邮件。

---
社保整合系统`, username, resetURL),
	}
}

func (s *EmailService) getEmailVerificationTemplate(username, verificationURL string) EmailTemplate {
	return EmailTemplate{
		Subject: "社保整合系统 - 邮箱验证",
		Body: fmt.Sprintf(`尊敬的 %s，

感谢您注册社保整合系统！

为了确保您的账户安全，请点击以下链接验证您的邮箱地址：
%s

此链接将在48小时后过期。

验证邮箱后，您就可以开始使用系统的所有功能了：
- 社保账期管理
- 多险种文件上传
- 数据处理和报表生成

如果您没有注册此账户，请忽略此邮件。

---
社保整合系统`, username, verificationURL),
	}
}

func (s *EmailService) getWelcomeTemplate(username string) EmailTemplate {
	return EmailTemplate{
		Subject: "邮箱验证成功 - 欢迎使用社保整合系统",
		Body: fmt.Sprintf(`尊敬的 %s，

恭喜！您的邮箱验证成功！

您现在可以完全使用社保整合系统的所有功能：
- 社保账期管理
- 多险种文件上传
- 数据处理和报表生成

如有任何问题，请联系系统管理员。

---
社保整合系统`, username),
	}
}

func (s *EmailService) getPasswordChangedTemplate(username string) EmailTemplate {
	return EmailTemplate{
		Subject: "社保整合系统 - 密码已更改",
		Body: fmt.Sprintf(`尊敬的 %s，

您的账户���码已成功更改。

如果这不是您本人的操作，请立即联系系统管理员。

---
社保整合系统`, username),
	}
}

// TestEmailConfiguration tests if email service is properly configured
func (s *EmailService) TestEmailConfiguration() error {
	log.Printf("邮件服务配置测试")
	log.Printf("基础URL: %s", s.baseURL)

	// In a real implementation, you would test SMTP connection or API credentials
	log.Printf("邮件服务配置正常（模拟模式）")

	return nil
}