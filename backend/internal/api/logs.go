package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

type LogHandler struct {
	db *gorm.DB
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	return &LogHandler{db: db}
}

// SystemLog represents a system log entry
type SystemLog struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	Level     string         `json:"level" gorm:"size:20;index"`
	TraceID   string         `json:"trace_id" gorm:"size:100;index"`
	Source    string         `json:"source" gorm:"size:100"`
	Message   string         `json:"message" gorm:"type:text"`
	Details   datatypes.JSON `json:"details" gorm:"type:jsonb"`
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
}

// LogBackup represents a backup record
type LogBackup struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Filename    string    `json:"filename" gorm:"size:255;not null"`
	FilePath    string    `json:"file_path" gorm:"size:500;not null"`
	FileSize    int64     `json:"file_size"`
	RecordCount int       `json:"record_count"`
	BackupType  string    `json:"backup_type" gorm:"size:50"`
	Status      string    `json:"status" gorm:"size:50"`
	CreatedBy   string    `json:"created_by" gorm:"size:100"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}

// AlertRule represents an alert rule
type AlertRule struct {
	ID                   int64          `json:"id" gorm:"primaryKey"`
	Name                 string         `json:"name" gorm:"size:255;not null"`
	Keywords             datatypes.JSON `json:"keywords" gorm:"type:text[]"`
	Threshold            int            `json:"threshold" gorm:"default:10"`
	TimeWindow           int            `json:"time_window" gorm:"default:5"`
	Enabled              bool           `json:"enabled" gorm:"default:true;index"`
	NotificationChannel  string         `json:"notification_channel" gorm:"size:100;default:'in-app'"`
	CreatedBy            string         `json:"created_by" gorm:"size:100"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// TableName specifies table names for GORM
func (SystemLog) TableName() string {
	return "system_logs"
}

func (LogBackup) TableName() string {
	return "log_backups"
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

func (h *LogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.QueryLogs)
	r.Get("/export", h.ExportLogs)
	r.Post("/backup", h.CreateBackup)
	r.Get("/backups", h.ListBackups)
	r.Delete("/backups/{id}", h.DeleteBackup)
	r.Post("/cleanup", h.CleanupLogs)
	r.Get("/alert-rules", h.ListAlertRules)
	r.Post("/alert-rules", h.CreateAlertRule)
	r.Put("/alert-rules/{id}", h.UpdateAlertRule)
	r.Delete("/alert-rules/{id}", h.DeleteAlertRule)
	return r
}

// QueryLogs queries audit_logs and system_logs with filtering and pagination
func (h *LogHandler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	logType := r.URL.Query().Get("log_type")
	if logType == "" {
		logType = r.URL.Query().Get("type") // fallback
	}
	action := r.URL.Query().Get("action")
	userID := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		keyword = r.URL.Query().Get("search")
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	offset := (page - 1) * size
	var logs []interface{}
	var total int64

	if logType == "system" {
		// Query system_logs table
		var systemLogs []SystemLog
		query := h.db.Model(&SystemLog{})

		if action != "" {
			query = query.Where("level = ?", action)
		}
		if keyword != "" {
			query = query.Where("message ILIKE ?", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Count(&total).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to count system logs", err)
			return
		}

		if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&systemLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch system logs", err)
			return
		}

		for _, log := range systemLogs {
			logs = append(logs, log)
		}
	} else if logType == "login" {
		// Query audit_logs WHERE action LIKE 'LOGIN%' OR action='REGISTER' OR action='LOGOUT'
		var auditLogs []models.AuditLog
		query := h.db.Model(&models.AuditLog{}).Where(
			"action LIKE ? OR action = ? OR action = ?",
			"LOGIN%", "REGISTER", "LOGOUT",
		)

		if userID != "" {
			query = query.Where("user_id = ?", userID)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if keyword != "" {
			query = query.Where("path ILIKE ? OR ip_address ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Count(&total).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to count login logs", err)
			return
		}

		if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&auditLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch login logs", err)
			return
		}

		for _, log := range auditLogs {
			logs = append(logs, log)
		}
	} else {
		// Default: Query audit_logs WHERE action NOT LIKE 'LOGIN%' (operation logs)
		var auditLogs []models.AuditLog
		query := h.db.Model(&models.AuditLog{}).Where("action NOT LIKE ?", "LOGIN%")

		if action != "" {
			query = query.Where("action = ?", action)
		}
		if userID != "" {
			query = query.Where("user_id = ?", userID)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if keyword != "" {
			query = query.Where("path ILIKE ? OR resource ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Count(&total).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to count operation logs", err)
			return
		}

		if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&auditLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch operation logs", err)
			return
		}

		for _, log := range auditLogs {
			logs = append(logs, log)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// ExportLogs generates an Excel file with log data
func (h *LogHandler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	logType := r.URL.Query().Get("log_type")
	if logType == "" {
		logType = r.URL.Query().Get("type")
	}
	action := r.URL.Query().Get("action")
	userID := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	keyword := r.URL.Query().Get("keyword")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheetName := f.GetSheetName(0)

	if logType == "system" {
		// Export system logs
		headers := []string{"ID", "Level", "Trace ID", "Source", "Message", "Created At"}
		for idx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
			if err := f.SetCellValue(sheetName, cell, header); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write header", err)
				return
			}
		}

		var systemLogs []SystemLog
		query := h.db.Model(&SystemLog{})

		if action != "" {
			query = query.Where("level = ?", action)
		}
		if keyword != "" {
			query = query.Where("message ILIKE ?", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Order("created_at DESC").Find(&systemLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch system logs", err)
			return
		}

		for idx, log := range systemLogs {
			values := []interface{}{
				log.ID,
				log.Level,
				log.TraceID,
				log.Source,
				log.Message,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			for colIdx, value := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
				if err := f.SetCellValue(sheetName, cell, value); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to write data", err)
					return
				}
			}
		}
	} else if logType == "login" {
		// Export login logs
		headers := []string{"ID", "User ID", "Action", "IP Address", "Status", "Status Code", "Created At"}
		for idx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
			if err := f.SetCellValue(sheetName, cell, header); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write header", err)
				return
			}
		}

		var auditLogs []models.AuditLog
		query := h.db.Model(&models.AuditLog{}).Where(
			"action LIKE ? OR action = ? OR action = ?",
			"LOGIN%", "REGISTER", "LOGOUT",
		)

		if userID != "" {
			query = query.Where("user_id = ?", userID)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if keyword != "" {
			query = query.Where("path ILIKE ? OR ip_address ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Order("created_at DESC").Find(&auditLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch login logs", err)
			return
		}

		for idx, log := range auditLogs {
			userIDStr := ""
			if log.UserID != nil {
				userIDStr = strconv.FormatUint(uint64(*log.UserID), 10)
			}
			values := []interface{}{
				log.ID,
				userIDStr,
				log.Action,
				log.IPAddress,
				log.Status,
				log.StatusCode,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			for colIdx, value := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
				if err := f.SetCellValue(sheetName, cell, value); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to write data", err)
					return
				}
			}
		}
	} else {
		// Export operation logs
		headers := []string{"ID", "User ID", "Action", "Resource", "Method", "Path", "Status", "Status Code", "Created At"}
		for idx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
			if err := f.SetCellValue(sheetName, cell, header); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write header", err)
				return
			}
		}

		var auditLogs []models.AuditLog
		query := h.db.Model(&models.AuditLog{}).Where("action NOT LIKE ?", "LOGIN%")

		if action != "" {
			query = query.Where("action = ?", action)
		}
		if userID != "" {
			query = query.Where("user_id = ?", userID)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if keyword != "" {
			query = query.Where("path ILIKE ? OR resource ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		if startDate != "" {
			query = query.Where("created_at >= ?", startDate)
		}
		if endDate != "" {
			query = query.Where("created_at <= ?", endDate)
		}

		if err := query.Order("created_at DESC").Find(&auditLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch operation logs", err)
			return
		}

		for idx, log := range auditLogs {
			userIDStr := ""
			if log.UserID != nil {
				userIDStr = strconv.FormatUint(uint64(*log.UserID), 10)
			}
			values := []interface{}{
				log.ID,
				userIDStr,
				log.Action,
				log.Resource,
				log.Method,
				log.Path,
				log.Status,
				log.StatusCode,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			for colIdx, value := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
				if err := f.SetCellValue(sheetName, cell, value); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to write data", err)
					return
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=logs-export.xlsx")
	if err := f.Write(w); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to write excel file", err)
		return
	}
}

// CreateBackup creates a JSON backup of logs and saves to disk
func (h *LogHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	backupType := r.URL.Query().Get("type") // "audit", "system", "all"
	if backupType == "" {
		backupType = "all"
	}

	createdBy := r.Header.Get("X-User-ID")
	if createdBy == "" {
		createdBy = "system"
	}

	// Create backup directory if not exists
	backupDir := "./data/log-backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create backup directory", err)
		return
	}

	// Collect logs to backup
	backupData := map[string]interface{}{}
	var recordCount int

	if backupType == "audit" || backupType == "all" {
		var auditLogs []models.AuditLog
		if err := h.db.Find(&auditLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch audit logs", err)
			return
		}
		backupData["audit_logs"] = auditLogs
		recordCount += len(auditLogs)
	}

	if backupType == "system" || backupType == "all" {
		var systemLogs []SystemLog
		if err := h.db.Find(&systemLogs).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch system logs", err)
			return
		}
		backupData["system_logs"] = systemLogs
		recordCount += len(systemLogs)
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to marshal backup data", err)
		return
	}

	// Write to file
	filename := fmt.Sprintf("logs-backup-%s-%s.json", backupType, time.Now().Format("20060102-150405"))
	filePath := filepath.Join(backupDir, filename)

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to write backup file", err)
		return
	}

	// Record in database
	backup := &LogBackup{
		Filename:    filename,
		FilePath:    filePath,
		FileSize:    int64(len(jsonData)),
		RecordCount: recordCount,
		BackupType:  backupType,
		Status:      "completed",
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(backup).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to record backup in database", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "success",
		"backup_id":    backup.ID,
		"filename":     backup.Filename,
		"file_size":    backup.FileSize,
		"record_count": backup.RecordCount,
	})
}

// ListBackups queries log_backups table
func (h *LogHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size
	var backups []LogBackup
	var total int64

	if err := h.db.Model(&LogBackup{}).Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count backups", err)
		return
	}

	if err := h.db.Order("created_at DESC").Offset(offset).Limit(size).Find(&backups).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch backups", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  backups,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// DeleteBackup deletes a backup record and file from disk
func (h *LogHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid backup id", err)
		return
	}

	var backup LogBackup
	if err := h.db.First(&backup, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(w, http.StatusNotFound, "backup not found", nil)
		} else {
			respondError(w, http.StatusInternalServerError, "failed to fetch backup", err)
		}
		return
	}

	// Delete file from disk
	if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
		respondError(w, http.StatusInternalServerError, "failed to delete backup file", err)
		return
	}

	// Delete record from database
	if err := h.db.Delete(&backup).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete backup record", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "backup deleted",
	})
}

// CleanupLogs deletes logs older than 30 days
func (h *LogHandler) CleanupLogs(w http.ResponseWriter, r *http.Request) {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// Delete old audit logs
	auditResult := h.db.Where("created_at < ?", thirtyDaysAgo).Delete(&models.AuditLog{})
	if auditResult.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to cleanup audit logs", auditResult.Error)
		return
	}

	// Delete old system logs
	systemResult := h.db.Where("created_at < ?", thirtyDaysAgo).Delete(&SystemLog{})
	if systemResult.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to cleanup system logs", systemResult.Error)
		return
	}

	totalDeleted := auditResult.RowsAffected + systemResult.RowsAffected

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": fmt.Sprintf("deleted %d logs older than 30 days", totalDeleted),
		"audit_logs_deleted": auditResult.RowsAffected,
		"system_logs_deleted": systemResult.RowsAffected,
	})
}

// ListAlertRules queries alert_rules table
func (h *LogHandler) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size
	var rules []AlertRule
	var total int64

	if err := h.db.Model(&AlertRule{}).Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count alert rules", err)
		return
	}

	if err := h.db.Order("created_at DESC").Offset(offset).Limit(size).Find(&rules).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch alert rules", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  rules,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// CreateAlertRule inserts into alert_rules table
func (h *LogHandler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string   `json:"name"`
		Keywords             []string `json:"keywords"`
		Threshold            int      `json:"threshold"`
		TimeWindow           int      `json:"time_window"`
		Enabled              bool     `json:"enabled"`
		NotificationChannel  string   `json:"notification_channel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	if len(req.Keywords) == 0 {
		respondError(w, http.StatusBadRequest, "keywords are required", nil)
		return
	}

	createdBy := r.Header.Get("X-User-ID")
	if createdBy == "" {
		createdBy = "system"
	}

	keywordsJSON, _ := json.Marshal(req.Keywords)

	rule := &AlertRule{
		Name:                req.Name,
		Keywords:            keywordsJSON,
		Threshold:           req.Threshold,
		TimeWindow:          req.TimeWindow,
		Enabled:             req.Enabled,
		NotificationChannel: req.NotificationChannel,
		CreatedBy:           createdBy,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := h.db.Create(rule).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create alert rule", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"rule":   rule,
	})
}

// UpdateAlertRule updates alert_rules by ID
func (h *LogHandler) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid rule id", err)
		return
	}

	var req struct {
		Name                 string   `json:"name"`
		Keywords             []string `json:"keywords"`
		Threshold            int      `json:"threshold"`
		TimeWindow           int      `json:"time_window"`
		Enabled              bool     `json:"enabled"`
		NotificationChannel  string   `json:"notification_channel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	var rule AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(w, http.StatusNotFound, "alert rule not found", nil)
		} else {
			respondError(w, http.StatusInternalServerError, "failed to fetch alert rule", err)
		}
		return
	}

	keywordsJSON, _ := json.Marshal(req.Keywords)

	updates := map[string]interface{}{
		"name":                   req.Name,
		"keywords":               keywordsJSON,
		"threshold":              req.Threshold,
		"time_window":            req.TimeWindow,
		"enabled":                req.Enabled,
		"notification_channel":   req.NotificationChannel,
		"updated_at":             time.Now(),
	}

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update alert rule", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"rule":   rule,
	})
}

// DeleteAlertRule deletes from alert_rules by ID
func (h *LogHandler) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid rule id", err)
		return
	}

	var rule AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(w, http.StatusNotFound, "alert rule not found", nil)
		} else {
			respondError(w, http.StatusInternalServerError, "failed to fetch alert rule", err)
		}
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete alert rule", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "alert rule deleted",
	})
}
