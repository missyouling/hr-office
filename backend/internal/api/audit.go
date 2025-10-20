package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

// AuditHandler handles audit log related HTTP requests
type AuditHandler struct {
	db           *gorm.DB
	auditService *service.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(db *gorm.DB, auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{
		db:           db,
		auditService: auditService,
	}
}

// GetAuditLogsRequest represents the request for getting audit logs
type GetAuditLogsRequest struct {
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	UserOnly  bool   `json:"user_only"`
}

// GetAuditLogsResponse represents the response for audit logs
type GetAuditLogsResponse struct {
	Logs  []models.AuditLog `json:"logs"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// UserActivityStatsResponse represents user activity statistics
type UserActivityStatsResponse struct {
	UserID uint           `json:"user_id"`
	Days   int            `json:"days"`
	Stats  map[string]int `json:"stats"`
}

// SystemActivityStatsResponse represents system activity statistics
type SystemActivityStatsResponse struct {
	Days  int                    `json:"days"`
	Stats map[string]interface{} `json:"stats"`
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	action := r.URL.Query().Get("action")
	status := r.URL.Query().Get("status")
	userOnlyStr := r.URL.Query().Get("user_only")
	userOnly := userOnlyStr == "true"

	// Parse date filters
	var startDate, endDate *time.Time
	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &parsed
		}
	}
	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			// Set to end of day
			end := parsed.Add(24*time.Hour - time.Second)
			endDate = &end
		}
	}

	// Get user ID from context
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Determine which user's logs to fetch
	var targetUserID *uint
	if userOnly {
		targetUserID = &userID
	}
	// If not userOnly, admin users could see all logs (future feature)

	// Get audit logs
	logs, total, err := h.auditService.GetAuditLogs(targetUserID, limit, offset, action, status, startDate, endDate)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch audit logs"}`, http.StatusInternalServerError)
		return
	}

	// Calculate page number
	page := (offset / limit) + 1

	response := GetAuditLogsResponse{
		Logs:  logs,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserActivityStats returns activity statistics for the current user
func (h *AuditHandler) GetUserActivityStats(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse days parameter
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30 // Default to 30 days
	}

	// Get user activity stats
	stats, err := h.auditService.GetUserActivityStats(userID, days)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch user activity stats"}`, http.StatusInternalServerError)
		return
	}

	response := UserActivityStatsResponse{
		UserID: userID,
		Days:   days,
		Stats:  stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSystemActivityStats returns system-wide activity statistics
func (h *AuditHandler) GetSystemActivityStats(w http.ResponseWriter, r *http.Request) {
	// This could be restricted to admin users in the future
	// For now, any authenticated user can access

	// Parse days parameter
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 7 // Default to 7 days
	}

	// Get system activity stats
	stats, err := h.auditService.GetSystemActivityStats(days)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch system activity stats"}`, http.StatusInternalServerError)
		return
	}

	response := SystemActivityStatsResponse{
		Days:  days,
		Stats: stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAuditLogDetails returns detailed information about a specific audit log
func (h *AuditHandler) GetAuditLogDetails(w http.ResponseWriter, r *http.Request) {
	// Get log ID from URL
	logIDStr := chi.URLParam(r, "id")
	logID, err := strconv.ParseUint(logIDStr, 10, 32)
	if err != nil {
		http.Error(w, `{"error":"Invalid log ID"}`, http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Find the audit log
	var auditLog models.AuditLog
	query := h.db.Preload("User").Where("id = ?", uint(logID))

	// Users can only see their own logs (unless admin in future)
	query = query.Where("user_id = ?", userID)

	if err := query.First(&auditLog).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, `{"error":"Audit log not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to fetch audit log"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLog)
}

// CleanupOldLogs manually triggers cleanup of old audit logs
func (h *AuditHandler) CleanupOldLogs(w http.ResponseWriter, r *http.Request) {
	// This should be restricted to admin users in production
	// For now, any authenticated user can trigger cleanup

	// Parse days to keep parameter
	daysToKeep, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if daysToKeep <= 0 {
		daysToKeep = 90 // Default to keep 90 days
	}

	// Minimum days to keep for safety
	if daysToKeep < 7 {
		http.Error(w, `{"error":"Cannot keep less than 7 days of logs"}`, http.StatusBadRequest)
		return
	}

	// Perform cleanup
	err := h.auditService.CleanupOldLogs(daysToKeep)
	if err != nil {
		http.Error(w, `{"error":"Failed to cleanup old logs"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":      "Old audit logs cleaned up successfully",
		"days_kept":    daysToKeep,
		"cleaned_at":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterAuditRoutes registers audit-related routes
func (h *AuditHandler) RegisterAuditRoutes(r chi.Router) {
	r.Route("/audit", func(r chi.Router) {
		r.Get("/logs", h.GetAuditLogs)
		r.Get("/logs/{id}", h.GetAuditLogDetails)
		r.Get("/stats/user", h.GetUserActivityStats)
		r.Get("/stats/system", h.GetSystemActivityStats)
		r.Post("/cleanup", h.CleanupOldLogs)
	})
}