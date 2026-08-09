package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

const feedbackPageSize = 20

type submitFeedbackRequest struct {
	MessageID string `json:"message_id"`
	SessionID string `json:"session_id"`
	Rating    string `json:"rating"`
	Comment   string `json:"comment"`
}

type feedbackReplyRequest struct {
	Reply string `json:"reply"`
}

type feedbackListResponse struct {
	Items    []feedbackItem `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type feedbackItem struct {
	models.ChatFeedback
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

type feedbackStatsResponse struct {
	Total          int64          `json:"total"`
	Positive       int64          `json:"positive"`
	Negative       int64          `json:"negative"`
	PositiveRate   float64        `json:"positive_rate"`
	RecentNegative []feedbackItem `json:"recent_negative"`
}

func isValidRating(rating string) bool {
	switch strings.ToLower(strings.TrimSpace(rating)) {
	case "positive", "negative":
		return true
	default:
		return false
	}
}

func (h *Handler) currentUserIsAdmin(r *http.Request) (bool, uint, error) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		return false, 0, err
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return false, userID, err
	}
	return user.Role == "admin" || user.Role == "super_admin", userID, nil
}

// submitFeedback POST /api/feedback
func (h *Handler) submitFeedback(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req submitFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	req.MessageID = strings.TrimSpace(req.MessageID)
	req.Rating = strings.ToLower(strings.TrimSpace(req.Rating))
	if req.MessageID == "" {
		respondError(w, http.StatusBadRequest, "message_id is required", nil)
		return
	}
	if !isValidRating(req.Rating) {
		respondError(w, http.StatusBadRequest, "rating must be positive or negative", nil)
		return
	}

	var existing models.ChatFeedback
	result := h.db.Where("message_id = ?", req.MessageID).First(&existing)
	now := time.Now()
	if result.Error == nil {
		// 已存在则更新
		existing.UserID = userID
		existing.SessionID = strings.TrimSpace(req.SessionID)
		existing.Rating = req.Rating
		existing.Comment = strings.TrimSpace(req.Comment)
		existing.UpdatedAt = now
		if err := h.db.Save(&existing).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update feedback", err)
			return
		}
		respondJSON(w, http.StatusOK, existing)
		return
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		respondError(w, http.StatusInternalServerError, "failed to check existing feedback", result.Error)
		return
	}

	feedback := models.ChatFeedback{
		UserID:    userID,
		MessageID: req.MessageID,
		SessionID: strings.TrimSpace(req.SessionID),
		Rating:    req.Rating,
		Comment:   strings.TrimSpace(req.Comment),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.db.Create(&feedback).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create feedback", err)
		return
	}
	respondJSON(w, http.StatusCreated, feedback)
}

// listFeedback GET /api/feedback?page=1&rating=negative
func (h *Handler) listFeedback(w http.ResponseWriter, r *http.Request) {
	isAdmin, _, err := h.currentUserIsAdmin(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if !isAdmin {
		respondError(w, http.StatusForbidden, "admin access required", nil)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			page = parsed
		}
	}
	rating := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("rating")))
	if rating != "" && !isValidRating(rating) {
		respondError(w, http.StatusBadRequest, "invalid rating filter", nil)
		return
	}

	query := h.db.Model(&models.ChatFeedback{})
	if rating != "" {
		query = query.Where("rating = ?", rating)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count feedback", err)
		return
	}

	var rows []models.ChatFeedback
	offset := (page - 1) * feedbackPageSize
	if err := query.Preload("User").Order("created_at DESC").Limit(feedbackPageSize).Offset(offset).Find(&rows).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list feedback", err)
		return
	}

	items := make([]feedbackItem, 0, len(rows))
	for _, fb := range rows {
		items = append(items, feedbackItem{
			ChatFeedback: fb,
			Username:     getUserName(fb.User),
			FullName:     getUserFullName(fb.User),
		})
	}

	respondJSON(w, http.StatusOK, feedbackListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: feedbackPageSize,
	})
}

// replyFeedback PUT /api/feedback/{id}/reply
func (h *Handler) replyFeedback(w http.ResponseWriter, r *http.Request) {
	isAdmin, _, err := h.currentUserIsAdmin(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if !isAdmin {
		respondError(w, http.StatusForbidden, "admin access required", nil)
		return
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "feedback id is required", nil)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid feedback id", err)
		return
	}

	var req feedbackReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	var fb models.ChatFeedback
	if err := h.db.Preload("User").First(&fb, id).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load feedback", err)
		return
	}

	now := time.Now()
	fb.Reply = strings.TrimSpace(req.Reply)
	fb.RepliedAt = &now
	fb.UpdatedAt = now
	if err := h.db.Save(&fb).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save reply", err)
		return
	}

	respondJSON(w, http.StatusOK, feedbackItem{
		ChatFeedback: fb,
		Username:     getUserName(fb.User),
		FullName:     getUserFullName(fb.User),
	})
}

// feedbackStats GET /api/feedback/stats
func (h *Handler) feedbackStats(w http.ResponseWriter, r *http.Request) {
	isAdmin, _, err := h.currentUserIsAdmin(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if !isAdmin {
		respondError(w, http.StatusForbidden, "admin access required", nil)
		return
	}

	var total, positive, negative int64
	if err := h.db.Model(&models.ChatFeedback{}).Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count feedback", err)
		return
	}
	if err := h.db.Model(&models.ChatFeedback{}).Where("rating = ?", "positive").Count(&positive).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count positive feedback", err)
		return
	}
	if err := h.db.Model(&models.ChatFeedback{}).Where("rating = ?", "negative").Count(&negative).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count negative feedback", err)
		return
	}

	positiveRate := float64(0)
	if total > 0 {
		positiveRate = float64(positive) / float64(total)
	}

	var recent []models.ChatFeedback
	if err := h.db.Where("rating = ?", "negative").Preload("User").Order("created_at DESC").Limit(5).Find(&recent).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load recent negative feedback", err)
		return
	}
	recentItems := make([]feedbackItem, 0, len(recent))
	for _, fb := range recent {
		recentItems = append(recentItems, feedbackItem{
			ChatFeedback: fb,
			Username:     getUserName(fb.User),
			FullName:     getUserFullName(fb.User),
		})
	}

	respondJSON(w, http.StatusOK, feedbackStatsResponse{
		Total:          total,
		Positive:       positive,
		Negative:       negative,
		PositiveRate:   positiveRate,
		RecentNegative: recentItems,
	})
}

func getUserName(u *models.User) string {
	if u == nil {
		return ""
	}
	return u.Username
}

func getUserFullName(u *models.User) string {
	if u == nil {
		return ""
	}
	return u.FullName
}
