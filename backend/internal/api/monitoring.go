package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/service"
)

// MonitoringHandler handles monitoring and health check requests
type MonitoringHandler struct {
	db                *gorm.DB
	monitoringService *service.MonitoringService
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(db *gorm.DB, monitoringService *service.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		db:                db,
		monitoringService: monitoringService,
	}
}

// HealthCheck returns the health status of the system
func (h *MonitoringHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health, err := h.monitoringService.GetHealthStatus()
	if err != nil {
		http.Error(w, `{"status":"error","message":"Failed to get health status"}`, http.StatusInternalServerError)
		return
	}

	// Set appropriate status code based on health
	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// GetSystemMetrics returns comprehensive system metrics
func (h *MonitoringHandler) GetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated (optional for metrics)
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// In a production system, you might want to restrict this to admin users
	_ = userID

	metrics, err := h.monitoringService.GetSystemMetrics()
	if err != nil {
		http.Error(w, `{"error":"Failed to get system metrics"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetDatabaseStatus returns detailed database status
func (h *MonitoringHandler) GetDatabaseStatus(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	_ = userID

	status, err := h.monitoringService.GetDatabaseStatus()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetSystemInfo returns general system information
func (h *MonitoringHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	_ = userID

	info := h.monitoringService.GetSystemInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// RunMaintenance performs system maintenance tasks
func (h *MonitoringHandler) RunMaintenance(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	_ = userID

	// Run cleanup tasks
	err = h.monitoringService.CleanupExpiredTokens()
	if err != nil {
		http.Error(w, `{"error":"Maintenance task failed"}`, http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"message":    "Maintenance tasks completed successfully",
		"timestamp":  time.Now(),
		"tasks_run": []string{
			"cleanup_expired_password_reset_tokens",
			"cleanup_expired_email_verification_tokens",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetReadiness returns readiness status for Kubernetes-style health checks
func (h *MonitoringHandler) GetReadiness(w http.ResponseWriter, r *http.Request) {
	// Check if the application is ready to serve requests
	ready := true
	message := "Service is ready"

	// Check database connectivity
	sqlDB, err := h.db.DB()
	if err != nil {
		ready = false
		message = "Database connection error"
	} else {
		if err := sqlDB.Ping(); err != nil {
			ready = false
			message = "Database ping failed"
		}
	}

	status := map[string]interface{}{
		"ready":     ready,
		"message":   message,
		"timestamp": time.Now(),
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// GetLiveness returns liveness status for Kubernetes-style health checks
func (h *MonitoringHandler) GetLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - if we can respond, we're alive
	status := map[string]interface{}{
		"alive":     true,
		"message":   "Service is alive",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetVersion returns version information
func (h *MonitoringHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"name":        "社保整合系统",
		"version":     "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339), // Should be set at build time
		"git_commit":  "latest",                         // Should be set at build time
		"go_version":  "go1.21",                         // Should be runtime.Version()
		"environment": "development",                     // Should be from config
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// RegisterMonitoringRoutes registers monitoring-related routes
func (h *MonitoringHandler) RegisterMonitoringRoutes(r chi.Router) {
	// Public health check endpoints (no auth required)
	r.Get("/health", h.HealthCheck)
	r.Get("/health/readiness", h.GetReadiness)
	r.Get("/health/liveness", h.GetLiveness)
	r.Get("/version", h.GetVersion)
}

// RegisterProtectedMonitoringRoutes registers protected monitoring routes
func (h *MonitoringHandler) RegisterProtectedMonitoringRoutes(r chi.Router) {
	r.Route("/monitoring", func(r chi.Router) {
		r.Get("/metrics", h.GetSystemMetrics)
		r.Get("/database", h.GetDatabaseStatus)
		r.Get("/info", h.GetSystemInfo)
		r.Post("/maintenance", h.RunMaintenance)
	})
}