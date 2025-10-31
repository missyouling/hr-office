package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
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

	log.Printf("=== 发送密码重置邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)

	if err := s.sendEmail(user.Email, template.Subject, template.Body); err != nil {
		log.Printf("密码重置邮件发送失败: %v", err)
		return fmt.Errorf("邮件发送失败: %v", err)
	}

	log.Printf("密码重置邮件发送成功")
	return nil
}

// SendEmailVerificationEmail sends an email verification email
func (s *EmailService) SendEmailVerificationEmail(user *models.User, verificationToken *models.EmailVerificationToken) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, verificationToken.Token)

	template := s.getEmailVerificationTemplate(user.Username, verificationURL)

	log.Printf("=== 发送邮箱验证邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)

	if err := s.sendEmail(user.Email, template.Subject, template.Body); err != nil {
		log.Printf("邮箱验证邮件发送失败: %v", err)
		return fmt.Errorf("邮件发送失败: %v", err)
	}

	log.Printf("邮箱验证邮件发送成功")
	return nil
}

// SendWelcomeEmail sends a welcome email to new users (now sent after email verification)
func (s *EmailService) SendWelcomeEmail(user *models.User) error {
	template := s.getWelcomeTemplate(user.Username)

	log.Printf("=== 发送欢迎邮件 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)

	if err := s.sendEmail(user.Email, template.Subject, template.Body); err != nil {
		log.Printf("欢迎邮件发送失败: %v", err)
		return fmt.Errorf("邮件发送失败: %v", err)
	}

	log.Printf("欢迎邮件发送成功")
	return nil
}

// SendPasswordChangedEmail sends a notification when password is changed
func (s *EmailService) SendPasswordChangedEmail(user *models.User) error {
	template := s.getPasswordChangedTemplate(user.Username)

	log.Printf("=== 发送密码更改通知 ===")
	log.Printf("收件人: %s <%s>", user.Username, user.Email)
	log.Printf("主题: %s", template.Subject)

	if err := s.sendEmail(user.Email, template.Subject, template.Body); err != nil {
		log.Printf("密码更改通知邮件发送失败: %v", err)
		return fmt.Errorf("邮件发送失败: %v", err)
	}

	log.Printf("密码更改通知邮件发送成功")
	return nil
}

func (s *EmailService) getPasswordResetTemplate(username, resetURL string) EmailTemplate {
	return EmailTemplate{
		Subject: "人事行政管理系统 (hr-office) - 密码重置",
		Body: fmt.Sprintf(`尊敬的 %s，

您好！我们收到了您的密码重置请求。

请点击以下链接重置您的密码：
%s

此链接将在24小时后过期。

如果您没有请求重置密码，请忽略此邮件。

---
人事行政管理系统 (hr-office)`, username, resetURL),
	}
}

func (s *EmailService) getEmailVerificationTemplate(username, verificationURL string) EmailTemplate {
	return EmailTemplate{
		Subject: "人事行政管理系统 (hr-office) - 邮箱验证",
		Body: fmt.Sprintf(`尊敬的 %s，

感谢您注册人事行政管理系统 (hr-office)！

为了确保您的账户安全，请点击以下链接验证您的邮箱地址：
%s

此链接将在48小时后过期。

验证邮箱后，您就可以开始使用系统的全部功能：
- 社保账期管理
- 多险种文件上传
- 数据处理和报表生成

如果您没有注册此账户，请忽略此邮件。

---
人事行政管理系统 (hr-office)`, username, verificationURL),
	}
}

func (s *EmailService) getWelcomeTemplate(username string) EmailTemplate {
	return EmailTemplate{
		Subject: "邮箱验证成功 - 欢迎使用人事行政管理系统 (hr-office)",
		Body: fmt.Sprintf(`尊敬的 %s，

恭喜！您的邮箱验证成功！

您现在可以完全使用人事行政管理系统 (hr-office) 的全部功能：
- 社保账期管理
- 多险种文件上传
- 数据处理和报表生成

如有任何问题，请联系系统管理员。

---
人事行政管理系统 (hr-office)`, username),
	}
}

func (s *EmailService) getPasswordChangedTemplate(username string) EmailTemplate {
	return EmailTemplate{
		Subject: "人事行政管理系统 (hr-office) - 密码已更改",
		Body: fmt.Sprintf(`尊敬的 %s，

您的账户密码已成功更改。

如果这不是您本人的操作，请立即联系系统管理员。

---
人事行政管理系统 (hr-office)`, username),
	}
}

// TestEmailConfiguration tests if email service is properly configured
func (s *EmailService) TestEmailConfiguration() error {
	log.Printf("邮件服务配置测试")
	log.Printf("基础URL: %s", s.baseURL)

	// Test SMTP connection
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")

	if host == "" || port == "" || username == "" {
		return fmt.Errorf("SMTP配置不完整")
	}

	log.Printf("邮件服务配置正常: %s:%s", host, port)
	return nil
}

// sendEmail implements actual SMTP email sending with TLS support
func (s *EmailService) sendEmail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" || port == "" || username == "" || password == "" || from == "" {
		return fmt.Errorf("SMTP配置不完整")
	}

	// 构建邮件内容
	message := fmt.Sprintf("From: %s\r\n", from) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body

	// SMTP服务器地址（支持IPv4和IPv6）
	serverName := net.JoinHostPort(host, port)

	// SMTP认证
	auth := smtp.PlainAuth("", username, password, host)

	// TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	// 建立连接
	conn, err := net.Dial("tcp", serverName)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %v", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %v", err)
	}
	defer client.Quit()

	// 启动TLS
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("启动TLS失败: %v", err)
	}

	// 身份验证
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP身份验证失败: %v", err)
	}

	// 设置发送者
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发送者失败: %v", err)
	}

	// 设置接收者
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("设置接收者失败: %v", err)
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备邮件数据失败: %v", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("写入邮件数据失败: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("关闭邮件数据失败: %v", err)
	}

	return nil
}
