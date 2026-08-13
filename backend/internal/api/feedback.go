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
	"siapp/internal/service"
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
	Username          string          `json:"username"`
	FullName          string          `json:"full_name"`
	Question          string          `json:"question"`
	Answer            string          `json:"answer"`
	Sources           json.RawMessage `json:"sources"`
	AnswerUnavailable bool            `json:"answer_unavailable"`
}
type feedbackStatsResponse struct {
	Total          int64          `json:"total"`
	Positive       int64          `json:"positive"`
	Negative       int64          `json:"negative"`
	PositiveRate   float64        `json:"positive_rate"`
	Pending        int64          `json:"pending"`
	Replied        int64          `json:"replied"`
	Closed         int64          `json:"closed"`
	RecentNegative []feedbackItem `json:"recent_negative"`
}

func isValidRating(rating string) bool {
	rating = strings.ToLower(strings.TrimSpace(rating))
	return rating == "positive" || rating == "negative"
}

func (h *Handler) submitFeedback(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var req submitFeedbackRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}
	req.MessageID, req.SessionID, req.Rating, req.Comment = strings.TrimSpace(req.MessageID), strings.TrimSpace(req.SessionID), strings.ToLower(strings.TrimSpace(req.Rating)), strings.TrimSpace(req.Comment)
	if req.MessageID == "" {
		respondError(w, http.StatusBadRequest, "message_id is required", nil)
		return
	}
	if !isValidRating(req.Rating) {
		respondError(w, http.StatusBadRequest, "rating must be positive or negative", nil)
		return
	}
	var fb models.ChatFeedback
	result := h.db.Where("message_id = ?", req.MessageID).First(&fb)
	now := time.Now()
	if result.Error == nil {
		fb.UserID, fb.SessionID, fb.Rating, fb.Comment, fb.UpdatedAt = userID, req.SessionID, req.Rating, req.Comment, now
		if err = h.db.Save(&fb).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update feedback", err)
			return
		}
		respondJSON(w, http.StatusOK, fb)
		return
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		respondError(w, http.StatusInternalServerError, "failed to check existing feedback", result.Error)
		return
	}
	fb = models.ChatFeedback{UserID: userID, MessageID: req.MessageID, SessionID: req.SessionID, Rating: req.Rating, Comment: req.Comment, Status: models.FeedbackStatusPending, CreatedAt: now, UpdatedAt: now}
	if err = h.db.Create(&fb).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create feedback", err)
		return
	}
	respondJSON(w, http.StatusCreated, fb)
}

func parseFeedbackTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return &parsed, err
}

func feedbackQuery(db *gorm.DB, q map[string]string) (*gorm.DB, error) {
	query := db.Model(&models.ChatFeedback{})
	if q["rating"] != "" {
		if !isValidRating(q["rating"]) {
			return nil, errors.New("invalid rating filter")
		}
		query = query.Where("rating = ?", q["rating"])
	}
	if q["status"] != "" {
		if !models.IsValidFeedbackStatus(q["status"]) {
			return nil, errors.New("invalid status filter")
		}
		query = query.Where("status = ?", q["status"])
	}
	start, err := parseFeedbackTime(q["start_at"])
	if err != nil {
		return nil, errors.New("invalid start_at")
	}
	end, err := parseFeedbackTime(q["end_at"])
	if err != nil {
		return nil, errors.New("invalid end_at")
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at <= ?", *end)
	}
	if q["user_id"] != "" {
		id, parseErr := strconv.ParseUint(q["user_id"], 10, 64)
		if parseErr != nil || id == 0 {
			return nil, errors.New("invalid user_id")
		}
		query = query.Where("user_id = ?", id)
	}
	return query, nil
}

func (h *Handler) listFeedback(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	q := map[string]string{"rating": strings.ToLower(strings.TrimSpace(r.URL.Query().Get("rating"))), "status": strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status"))), "start_at": r.URL.Query().Get("start_at"), "end_at": r.URL.Query().Get("end_at"), "user_id": r.URL.Query().Get("user_id")}
	query, err := feedbackQuery(h.db, q)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count feedback", err)
		return
	}
	var rows []models.ChatFeedback
	if err = query.Preload("User").Order("created_at DESC").Limit(feedbackPageSize).Offset((page - 1) * feedbackPageSize).Find(&rows).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list feedback", err)
		return
	}
	items := h.buildFeedbackItems(rows)
	respondJSON(w, http.StatusOK, feedbackListResponse{Items: items, Total: total, Page: page, PageSize: feedbackPageSize})
}

func (h *Handler) buildFeedbackItems(rows []models.ChatFeedback) []feedbackItem {
	items := make([]feedbackItem, 0, len(rows))
	for _, fb := range rows {
		items = append(items, h.buildFeedbackItem(fb))
	}
	return items
}

func (h *Handler) buildFeedbackItem(fb models.ChatFeedback) feedbackItem {
	item := feedbackItem{ChatFeedback: fb, Username: getUserName(fb.User), FullName: getUserFullName(fb.User), Sources: json.RawMessage("[]"), AnswerUnavailable: true}
	var question models.ChatMessage
	h.db.Where("user_id = ? AND session_id = ? AND role = ?", fb.UserID, fb.SessionID, "user").Order("created_at DESC").First(&question)
	item.Question = question.Content
	var answer models.ChatMessage
	if id, err := strconv.ParseUint(fb.MessageID, 10, 64); err == nil {
		h.db.Where("id = ? AND role = ?", id, "assistant").First(&answer)
	}
	if answer.ID != 0 {
		item.Answer, item.Sources, item.AnswerUnavailable = answer.Content, json.RawMessage(answer.Sources), false
	}
	if item.Status == "" {
		item.Status = models.FeedbackStatusPending
	}
	return item
}

func (h *Handler) replyFeedback(w http.ResponseWriter, r *http.Request) {
	fb, err := h.loadFeedback(r)
	if err != nil {
		respondError(w, feedbackErrorStatus(err), "failed to load feedback", err)
		return
	}
	var req feedbackReplyRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}
	now := time.Now()
	fb.Reply, fb.RepliedAt, fb.Status, fb.UpdatedAt = strings.TrimSpace(req.Reply), &now, models.FeedbackStatusReplied, now
	if err = h.db.Save(&fb).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save reply", err)
		return
	}
	h.logFeedbackAudit(r, models.ActionReplyFeedback, fb.ID)
	respondJSON(w, http.StatusOK, h.buildFeedbackItem(fb))
}

func (h *Handler) closeFeedback(w http.ResponseWriter, r *http.Request) {
	fb, err := h.loadFeedback(r)
	if err != nil {
		respondError(w, feedbackErrorStatus(err), "failed to load feedback", err)
		return
	}
	now := time.Now()
	fb.Status, fb.ClosedAt, fb.UpdatedAt = models.FeedbackStatusClosed, &now, now
	if err = h.db.Save(&fb).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to close feedback", err)
		return
	}
	h.logFeedbackAudit(r, models.ActionCloseFeedback, fb.ID)
	respondJSON(w, http.StatusOK, h.buildFeedbackItem(fb))
}

func (h *Handler) loadFeedback(r *http.Request) (models.ChatFeedback, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id == 0 {
		return models.ChatFeedback{}, errors.New("invalid feedback id")
	}
	var fb models.ChatFeedback
	err = h.db.Preload("User").First(&fb, id).Error
	return fb, err
}
func feedbackErrorStatus(err error) int {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}
func (h *Handler) logFeedbackAudit(r *http.Request, action models.ActionType, id uint) {
	uid, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		return
	}
	rid := strconv.FormatUint(uint64(id), 10)
	_ = service.NewAuditService(h.db).LogAction(r.Context(), models.CreateAuditLogParams{UserID: &uid, Action: action, Resource: "chat_feedback", ResourceID: &rid, Method: r.Method, Path: r.URL.Path, Status: models.StatusSuccess, StatusCode: http.StatusOK})
}

func (h *Handler) feedbackStats(w http.ResponseWriter, r *http.Request) {
	q := map[string]string{"start_at": r.URL.Query().Get("start_at"), "end_at": r.URL.Query().Get("end_at")}
	query, err := feedbackQuery(h.db, q)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var s feedbackStatsResponse
	if err = query.Count(&s.Total).Error; err != nil {
		respondError(w, 500, "failed to count feedback", err)
		return
	}
	query.Session(&gorm.Session{}).Where("rating = ?", "positive").Count(&s.Positive)
	query.Session(&gorm.Session{}).Where("rating = ?", "negative").Count(&s.Negative)
	if s.Total > 0 {
		s.PositiveRate = float64(s.Positive) / float64(s.Total)
	}
	query.Session(&gorm.Session{}).Where("status = ? OR status = ''", models.FeedbackStatusPending).Count(&s.Pending)
	query.Session(&gorm.Session{}).Where("status = ?", models.FeedbackStatusReplied).Count(&s.Replied)
	query.Session(&gorm.Session{}).Where("status = ?", models.FeedbackStatusClosed).Count(&s.Closed)
	var recent []models.ChatFeedback
	query.Session(&gorm.Session{}).Where("rating = ?", "negative").Preload("User").Order("created_at DESC").Limit(5).Find(&recent)
	s.RecentNegative = h.buildFeedbackItems(recent)
	respondJSON(w, http.StatusOK, s)
}

func (h *Handler) myFeedback(w http.ResponseWriter, r *http.Request) {
	uid, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	page := 1
	if p, e := strconv.Atoi(r.URL.Query().Get("page")); e == nil && p > 0 {
		page = p
	}
	query := h.db.Model(&models.ChatFeedback{}).Where("user_id = ?", uid)
	var total int64
	query.Count(&total)
	var rows []models.ChatFeedback
	if err = query.Preload("User").Order("created_at DESC").Limit(feedbackPageSize).Offset((page - 1) * feedbackPageSize).Find(&rows).Error; err != nil {
		respondError(w, 500, "failed to list feedback", err)
		return
	}
	respondJSON(w, 200, feedbackListResponse{Items: h.buildFeedbackItems(rows), Total: total, Page: page, PageSize: feedbackPageSize})
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
