package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"siapp/internal/models"
)

// AuditService handles audit logging operations
type AuditService struct {
	db *gorm.DB
}

// NewAuditService creates a new audit service instance
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// LogAction logs an audit entry for a user action
func (s *AuditService) LogAction(ctx context.Context, params models.CreateAuditLogParams) error {
	auditLog := models.CreateAuditLog(params)

	// Save to database
	if err := s.db.Create(auditLog).Error; err != nil {
		log.Printf("Failed to save audit log: %v", err)
		return err
	}

	// Also log to console for debugging
	s.logToConsole(auditLog)

	return nil
}

// LogHTTPRequest logs an HTTP request with its response details
func (s *AuditService) LogHTTPRequest(r *http.Request, statusCode int, duration time.Duration, userID *uint, action models.ActionType, resource string, resourceID *string, errorMsg string, details *models.LogDetails) {
	status := models.StatusSuccess
	if statusCode >= 400 {
		status = models.StatusError
	}

	params := models.CreateAuditLogParams{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Method:     r.Method,
		Path:       r.URL.Path,
		IPAddress:  getClientIP(r),
		UserAgent:  r.UserAgent(),
		Status:     status,
		StatusCode: statusCode,
		ErrorMsg:   errorMsg,
		Duration:   duration.Milliseconds(),
		Details:    details,
	}

	if err := s.LogAction(r.Context(), params); err != nil {
		log.Printf("Failed to log HTTP request: %v", err)
	}
}

// LogAuthEvent logs authentication-related events
func (s *AuditService) LogAuthEvent(r *http.Request, userID *uint, action models.ActionType, success bool, errorMsg string) {
	status := models.StatusSuccess
	if !success {
		status = models.StatusFailed
	}

	params := models.CreateAuditLogParams{
		UserID:     userID,
		Action:     action,
		Resource:   "auth",
		Method:     r.Method,
		Path:       r.URL.Path,
		IPAddress:  getClientIP(r),
		UserAgent:  r.UserAgent(),
		Status:     status,
		StatusCode: getStatusCodeFromSuccess(success),
		ErrorMsg:   errorMsg,
		Duration:   0,
	}

	if err := s.LogAction(r.Context(), params); err != nil {
		log.Printf("Failed to log auth event: %v", err)
	}
}

// LogFileUpload logs file upload operations
func (s *AuditService) LogFileUpload(ctx context.Context, userID uint, fileName string, fileSize int64, scheme, part string, recordCount int, periodID uint, success bool, errorMsg string, duration time.Duration) {
	status := models.StatusSuccess
	if !success {
		status = models.StatusFailed
	}

	details := &models.LogDetails{
		FileName:    fileName,
		FileSize:    fileSize,
		Scheme:      scheme,
		Part:        part,
		RecordCount: recordCount,
		PeriodID:    periodID,
	}

	params := models.CreateAuditLogParams{
		UserID:     &userID,
		Action:     models.ActionUploadFile,
		Resource:   "files",
		ResourceID: &fileName,
		Status:     status,
		ErrorMsg:   errorMsg,
		Duration:   duration.Milliseconds(),
		Details:    details,
	}

	if err := s.LogAction(ctx, params); err != nil {
		log.Printf("Failed to log file upload: %v", err)
	}
}

// LogDataExport logs data export operations
func (s *AuditService) LogDataExport(ctx context.Context, userID uint, exportType, resourceID string, success bool, errorMsg string, duration time.Duration) {
	status := models.StatusSuccess
	if !success {
		status = models.StatusFailed
	}

	var action models.ActionType
	switch exportType {
	case "charges":
		action = models.ActionExportCharges
	case "scheme":
		action = models.ActionExportScheme
	case "template":
		action = models.ActionDownloadTemplate
	default:
		action = models.ActionExportCharges
	}

	params := models.CreateAuditLogParams{
		UserID:     &userID,
		Action:     action,
		Resource:   "exports",
		ResourceID: &resourceID,
		Status:     status,
		ErrorMsg:   errorMsg,
		Duration:   duration.Milliseconds(),
	}

	if err := s.LogAction(ctx, params); err != nil {
		log.Printf("Failed to log data export: %v", err)
	}
}

// LogSystemEvent logs system-level events
func (s *AuditService) LogSystemEvent(action models.ActionType, message string, details *models.LogDetails) {
	params := models.CreateAuditLogParams{
		Action:   action,
		Resource: "system",
		Status:   models.StatusSuccess,
		ErrorMsg: message,
		Details:  details,
	}

	if err := s.LogAction(context.Background(), params); err != nil {
		log.Printf("Failed to log system event: %v", err)
	}
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (s *AuditService) GetAuditLogs(userID *uint, limit, offset int, action, status string, startDate, endDate *time.Time) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{})

	// Apply filters
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with user preloading
	if err := query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// CleanupOldLogs removes audit logs older than the specified number of days
func (s *AuditService) CleanupOldLogs(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.AuditLog{})
	if result.Error != nil {
		return result.Error
	}

	log.Printf("Cleaned up %d old audit log entries (older than %d days)", result.RowsAffected, daysToKeep)
	return nil
}

// GetUserActivityStats returns activity statistics for a user
func (s *AuditService) GetUserActivityStats(userID uint, days int) (map[string]int, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var results []struct {
		Action string
		Count  int
	}

	err := s.db.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, startDate).
		Group("action").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, result := range results {
		stats[result.Action] = result.Count
	}

	return stats, nil
}

// GetSystemActivityStats returns system-wide activity statistics
func (s *AuditService) GetSystemActivityStats(days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	stats := make(map[string]interface{})

	// Total events
	var totalEvents int64
	s.db.Model(&models.AuditLog{}).Where("created_at >= ?", startDate).Count(&totalEvents)
	stats["total_events"] = totalEvents

	// Events by status
	var statusStats []struct {
		Status string
		Count  int
	}
	s.db.Model(&models.AuditLog{}).
		Select("status, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("status").
		Find(&statusStats)

	statusMap := make(map[string]int)
	for _, stat := range statusStats {
		statusMap[stat.Status] = stat.Count
	}
	stats["by_status"] = statusMap

	// Active users
	var activeUsers int64
	s.db.Model(&models.AuditLog{}).
		Where("created_at >= ? AND user_id IS NOT NULL", startDate).
		Distinct("user_id").
		Count(&activeUsers)
	stats["active_users"] = activeUsers

	// Most common actions
	var actionStats []struct {
		Action string
		Count  int
	}
	s.db.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("action").
		Order("count DESC").
		Limit(10).
		Find(&actionStats)
	stats["top_actions"] = actionStats

	return stats, nil
}

// Helper functions

func (s *AuditService) logToConsole(auditLog *models.AuditLog) {
	userInfo := "system"
	if auditLog.UserID != nil {
		userInfo = fmt.Sprintf("user:%d", *auditLog.UserID)
	}

	log.Printf("[AUDIT] %s %s %s %s (status: %s, duration: %dms) - %s",
		userInfo,
		auditLog.Action,
		auditLog.Method,
		auditLog.Path,
		auditLog.Status,
		auditLog.Duration,
		auditLog.IPAddress,
	)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func getStatusCodeFromSuccess(success bool) int {
	if success {
		return http.StatusOK
	}
	return http.StatusUnauthorized
}