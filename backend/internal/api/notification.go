package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"

	"siapp/internal/models"
	"siapp/internal/service"
)

func (h *Handler) ListNotificationConfigs(w http.ResponseWriter, r *http.Request) {
	var configs []models.NotificationConfig
	if err := h.db.Order("channel, created_at DESC").Find(&configs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load configs", err)
		return
	}

	// 脱敏：Config JSON 中的 token/secret 不返回原文
	for i := range configs {
		maskNotificationConfig(&configs[i])
	}

	respondJSON(w, http.StatusOK, configs)
}

type createNotificationConfigRequest struct {
	Channel string                 `json:"channel"`
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
	Status  string                 `json:"status"`
	Remark  string                 `json:"remark"`
}

func (h *Handler) CreateNotificationConfig(w http.ResponseWriter, r *http.Request) {
	var req createNotificationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Channel == "" {
		respondError(w, http.StatusBadRequest, "channel is required", nil)
		return
	}

	var configJSON datatypes.JSON
	if req.Config != nil {
		data, err := json.Marshal(req.Config)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid config", err)
			return
		}
		configJSON = data
	}

	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	config := models.NotificationConfig{
		Channel: req.Channel,
		Name:    req.Name,
		Enabled: req.Enabled,
		Config:  configJSON,
		Status:  status,
		Remark:  req.Remark,
	}

	if err := h.db.Create(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create config", err)
		return
	}

	maskNotificationConfig(&config)
	respondJSON(w, http.StatusOK, config)
}

func (h *Handler) GetNotificationConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var config models.NotificationConfig
	if err := h.db.First(&config, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "config not found", err)
		return
	}

	maskNotificationConfig(&config)
	respondJSON(w, http.StatusOK, config)
}

type updateNotificationConfigRequest struct {
	Channel string                 `json:"channel,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Status  string                 `json:"status,omitempty"`
	Remark  string                 `json:"remark,omitempty"`
}

func (h *Handler) UpdateNotificationConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var config models.NotificationConfig
	if err := h.db.First(&config, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "config not found", err)
		return
	}

	var req updateNotificationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Channel != "" {
		config.Channel = req.Channel
	}
	if req.Name != "" {
		config.Name = req.Name
	}
	config.Enabled = req.Enabled
	if req.Status != "" {
		config.Status = req.Status
	}
	if req.Remark != "" {
		config.Remark = req.Remark
	}

	if req.Config != nil {
		data, err := json.Marshal(req.Config)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid config", err)
			return
		}
		config.Config = data
	}

	if err := h.db.Save(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update config", err)
		return
	}

	maskNotificationConfig(&config)
	respondJSON(w, http.StatusOK, config)
}

func (h *Handler) DeleteNotificationConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.db.Delete(&models.NotificationConfig{}, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete config", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type smtpSendRequest struct {
	Config  json.RawMessage `json:"config,omitempty"`
	To      string          `json:"to"`
	Subject string          `json:"subject"`
	Content string          `json:"content"`
}

func (h *Handler) SendSMTPNotification(w http.ResponseWriter, r *http.Request) {
	var req smtpSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.To == "" || req.Content == "" {
		respondError(w, http.StatusBadRequest, "to and content are required", nil)
		return
	}

	subject := req.Subject
	if subject == "" {
		subject = "通知"
	}

	ns := service.NewNotificationService()
	if err := ns.SendSMTPEmail(&service.SendRequest{
		Config:  req.Config,
		To:      req.To,
		Subject: subject,
		Content: req.Content,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "send failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type smsSendRequest struct {
	Config   json.RawMessage   `json:"config,omitempty"`
	Phone    string            `json:"phone"`
	Template map[string]string `json:"template"`
}

func (h *Handler) SendSMSNotification(w http.ResponseWriter, r *http.Request) {
	var req smsSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Phone == "" || req.Template == nil {
		respondError(w, http.StatusBadRequest, "phone and template are required", nil)
		return
	}

	ns := service.NewNotificationService()
	if err := ns.SendAliyunSMS(&service.SendSMSTemplate{
		Config:   req.Config,
		Phone:    req.Phone,
		Template: req.Template,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "send failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type telegramSendRequest struct {
	Config json.RawMessage `json:"config,omitempty"`
	ChatID string          `json:"chat_id,omitempty"`
	Text   string          `json:"text"`
}

func (h *Handler) SendTelegramNotification(w http.ResponseWriter, r *http.Request) {
	var req telegramSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Text == "" {
		respondError(w, http.StatusBadRequest, "text is required", nil)
		return
	}

	ns := service.NewNotificationService()
	if err := ns.SendTelegram(&service.SendTelegramRequest{
		Config: req.Config,
		ChatID: req.ChatID,
		Text:   req.Text,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "send failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type webhookSendRequest struct {
	Config  json.RawMessage   `json:"config,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

func (h *Handler) SendWebhookNotification(w http.ResponseWriter, r *http.Request) {
	var req webhookSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	ns := service.NewNotificationService()
	if err := ns.SendWebhook(&service.SendWebhookRequest{
		Config:  req.Config,
		Method:  req.Method,
		Headers: req.Headers,
		Body:    req.Body,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "send failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type testNotificationRequest struct {
	Channel string          `json:"channel"`
	Config  json.RawMessage `json:"config,omitempty"`
	To      string          `json:"to"`
	Content string          `json:"content"`
}

func (h *Handler) TestNotification(w http.ResponseWriter, r *http.Request) {
	var req testNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Channel == "" {
		respondError(w, http.StatusBadRequest, "channel is required", nil)
		return
	}

	ns := service.NewNotificationService()
	var err error

	switch req.Channel {
	case "smtp":
		subject := "测试邮件"
		if req.Content != "" {
			subject = req.Content
		}
		err = ns.SendSMTPEmail(&service.SendRequest{
			Config:  req.Config,
			To:      req.To,
			Subject: subject,
			Content: "这是一封测试邮件。",
		})
	case "sms":
		err = ns.SendAliyunSMS(&service.SendSMSTemplate{
			Config:   req.Config,
			Phone:    req.To,
			Template: map[string]string{"code": "123456"},
		})
	case "telegram":
		text := "测试消息"
		if req.Content != "" {
			text = req.Content
		}
		err = ns.SendTelegram(&service.SendTelegramRequest{
			Config: req.Config,
			ChatID: req.To,
			Text:   text,
		})
	case "webhook":
		err = ns.SendWebhook(&service.SendWebhookRequest{
			Config: req.Config,
			Method: "GET",
		})
	default:
		respondError(w, http.StatusBadRequest, "unsupported channel", nil)
		return
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "test failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "test passed"})
}
