package service

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"sort"
	"strings"
	"time"
)

// NotificationService handles notification operations
type NotificationService struct{}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

type SMTPConfigFields struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	UseTLS   bool   `json:"use_tls"`
}

type SMSConfigFields struct {
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SignName        string `json:"sign_name"`
	TemplateCode    string `json:"template_code"`
}

type TelegramConfigFields struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type WebhookConfigFields struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Auth    string            `json:"auth"`
}

type SendRequest struct {
	ConfigID *uint           `json:"config_id,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
	To       string          `json:"to"`
	Subject  string          `json:"subject,omitempty"`
	Content  string          `json:"content"`
}

type SendSMSTemplate struct {
	ConfigID *uint             `json:"config_id,omitempty"`
	Config   json.RawMessage   `json:"config,omitempty"`
	Phone    string            `json:"phone"`
	Template map[string]string `json:"template"`
}

type SendTelegramRequest struct {
	ConfigID *uint           `json:"config_id,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
	ChatID   string          `json:"chat_id,omitempty"`
	Text     string          `json:"text"`
}

type SendWebhookRequest struct {
	ConfigID *uint             `json:"config_id,omitempty"`
	Config   json.RawMessage   `json:"config,omitempty"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
}

func (s *NotificationService) SendSMTPEmail(req *SendRequest) error {
	var cfg SMTPConfigFields

	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("配置解析失败: %v", err)
		}
	} else {
		cfg.Host = os.Getenv("SMTP_HOST")
		cfg.Port = os.Getenv("SMTP_PORT")
		cfg.Username = os.Getenv("SMTP_USERNAME")
		cfg.Password = os.Getenv("SMTP_PASSWORD")
		cfg.From = os.Getenv("SMTP_FROM")
		cfg.FromName = os.Getenv("FROM_NAME")
		cfg.UseTLS = os.Getenv("SMTP_USE_TLS") == "true"
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.Username == "" || cfg.Password == "" || cfg.From == "" {
		return fmt.Errorf("SMTP配置不完整")
	}

	fromName := cfg.FromName
	if fromName == "" {
		fromName = cfg.From
	}
	from := fmt.Sprintf("%s <%s>", fromName, cfg.From)

	message := fmt.Sprintf("From: %s\r\n", from) +
		fmt.Sprintf("To: %s\r\n", req.To) +
		fmt.Sprintf("Subject: %s\r\n", req.Subject) +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		req.Content

	serverName := net.JoinHostPort(cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	tlsConfig := &tls.Config{InsecureSkipVerify: false, ServerName: cfg.Host}

	conn, err := net.Dial("tcp", serverName)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %v", err)
	}
	defer client.Quit()

	if cfg.UseTLS {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("启动TLS失败: %v", err)
		}
	}

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP身份验证失败: %v", err)
	}

	if err = client.Mail(cfg.From); err != nil {
		return fmt.Errorf("设置发送者失败: %v", err)
	}

	if err = client.Rcpt(req.To); err != nil {
		return fmt.Errorf("设置接收者失败: %v", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备邮件数据失败: %v", err)
	}

	if _, err = writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("写入邮件数据失败: %v", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("关闭邮件数据失败: %v", err)
	}

	log.Printf("SMTP邮件发送成功: %s", req.To)
	return nil
}

func (s *NotificationService) SendAliyunSMS(req *SendSMSTemplate) error {
	var cfg SMSConfigFields

	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("配置解析失败: %v", err)
		}
	}

	if cfg.AccessKeyId == "" || cfg.AccessKeySecret == "" || cfg.SignName == "" || cfg.TemplateCode == "" {
		return fmt.Errorf("阿里短信配置不完整")
	}

	signName := cfg.SignName
	templateCode := cfg.TemplateCode

	params := make(map[string]string)
	params["PhoneNumbers"] = req.Phone
	params["SignName"] = signName
	params["TemplateCode"] = templateCode

	templateParam, _ := json.Marshal(req.Template)
	params["TemplateParam"] = string(templateParam)

	accessKeyId := cfg.AccessKeyId
	accessKeySecret := cfg.AccessKeySecret

	signature := s.aliyunSign(accessKeySecret, params)
	params["Signature"] = signature
	params["AccessKeyId"] = accessKeyId
	params["Format"] = "JSON"
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Action"] = "SendSms"
	params["Version"] = "2017-05-25"

	url := "https://dysmscon.aliyuncs.com/"
	httpReq, _ := http.NewRequest("POST", url, strings.NewReader(s.aliyunParam(params)))
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("短信请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode != 200 {
		return fmt.Errorf("短信发送失败: %s", string(body))
	}

	if code, ok := result["Code"].(string); ok && code != "OK" {
		return fmt.Errorf("短信发送失败: %s", result["Message"])
	}

	log.Printf("阿里短信发送成功: %s", req.Phone)
	return nil
}

func (s *NotificationService) aliyunSign(secret string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sorted string
	for _, k := range keys {
		if sorted != "" {
			sorted += "&"
		}
		sorted += k + "%3D" + params[k]
	}

	stringToSign := "POST&%2F&" + sorted
	h := md5sum([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h)
}

func md5sum(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

func (s *NotificationService) aliyunParam(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result string
	for _, k := range keys {
		if result != "" {
			result += "&"
		}
		result += k + "=" + params[k]
	}
	return result
}

func (s *NotificationService) SendTelegram(req *SendTelegramRequest) error {
	var cfg TelegramConfigFields

	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("配置解析失败: %v", err)
		}
	}

	if cfg.BotToken == "" {
		return fmt.Errorf("Telegram BotToken未配置")
	}

	chatID := req.ChatID
	if chatID == "" {
		chatID = cfg.ChatID
	}
	if chatID == "" {
		return fmt.Errorf("ChatID未指定")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	body := fmt.Sprintf(`{"chat_id": %s, "text": "%s"}`, chatID, req.Text)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBufferString(body))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Telegram请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if resp.StatusCode != 200 {
		return fmt.Errorf("Telegram发送失败: %s", string(respBody))
	}

	log.Printf("Telegram消息发送成功: %s", chatID)
	return nil
}

func (s *NotificationService) SendWebhook(req *SendWebhookRequest) error {
	var cfg WebhookConfigFields

	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("配置解析失败: %v", err)
		}
	}

	if cfg.URL == "" {
		return fmt.Errorf("Webhook URL未配置")
	}

	method := req.Method
	if method == "" {
		method = "POST"
	}
	if method == "" {
		method = cfg.Method
	}
	if method == "" {
		method = "POST"
	}

	httpReq, _ := http.NewRequest(method, cfg.URL, bytes.NewBufferString(req.Body))
	httpReq.Header.Set("Content-Type", "application/json")

	for k, v := range cfg.Headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Webhook请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook请求失败: %d %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Webhook请求成功: %s %s", method, cfg.URL)
	return nil
}
