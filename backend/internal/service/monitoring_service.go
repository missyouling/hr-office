package service

import (
	"runtime"
	"time"

	"gorm.io/gorm"
	"siapp/internal/models"
)

// MonitoringService handles system monitoring operations
type MonitoringService struct {
	db        *gorm.DB
	startTime time.Time
}

// NewMonitoringService creates a new monitoring service instance
func NewMonitoringService(db *gorm.DB) *MonitoringService {
	return &MonitoringService{
		db:        db,
		startTime: time.Now(),
	}
}

// HealthStatus represents the health status of the system
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	Timestamp time.Time `json:"timestamp"`
	Runtime   RuntimeMetrics `json:"runtime"`
	Database  DatabaseMetrics `json:"database"`
	HTTP      HTTPMetrics `json:"http"`
}

// RuntimeMetrics contains Go runtime metrics
type RuntimeMetrics struct {
	GoVersion      string  `json:"go_version"`
	Goroutines     int     `json:"goroutines"`
	MemoryUsage    int64   `json:"memory_usage_bytes"`
	MemoryAlloc    int64   `json:"memory_alloc_bytes"`
	MemorySys      int64   `json:"memory_sys_bytes"`
	GCCount        uint32  `json:"gc_count"`
	NextGC         int64   `json:"next_gc_bytes"`
	CPUCount       int     `json:"cpu_count"`
}

// DatabaseMetrics contains database performance metrics
type DatabaseMetrics struct {
	Status           string  `json:"status"`
	ActiveConnections int     `json:"active_connections"`
	IdleConnections   int     `json:"idle_connections"`
	TotalUsers        int64   `json:"total_users"`
	TotalPeriods      int64   `json:"total_periods"`
	TotalAuditLogs    int64   `json:"total_audit_logs"`
	AvgResponseTime   float64 `json:"avg_response_time_ms"`
}

// HTTPMetrics contains HTTP request metrics (would be collected from middleware)
type HTTPMetrics struct {
	TotalRequests    int64   `json:"total_requests"`
	ActiveRequests   int     `json:"active_requests"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	RequestsByStatus map[string]int64 `json:"requests_by_status"`
}

// ResourceUsage represents system resource usage
type ResourceUsage struct {
	CPU    CPUUsage    `json:"cpu"`
	Memory MemoryUsage `json:"memory"`
	Disk   DiskUsage   `json:"disk"`
}

// CPUUsage represents CPU usage information
type CPUUsage struct {
	Percent float64 `json:"percent"`
	Cores   int     `json:"cores"`
}

// MemoryUsage represents memory usage information
type MemoryUsage struct {
	Total     int64   `json:"total_bytes"`
	Used      int64   `json:"used_bytes"`
	Available int64   `json:"available_bytes"`
	Percent   float64 `json:"percent"`
}

// DiskUsage represents disk usage information
type DiskUsage struct {
	Total     int64   `json:"total_bytes"`
	Used      int64   `json:"used_bytes"`
	Available int64   `json:"available_bytes"`
	Percent   float64 `json:"percent"`
}

// GetHealthStatus returns the overall health status of the system
func (s *MonitoringService) GetHealthStatus() (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.startTime).String(),
		Version:   "1.0.0", // Should be read from config or build info
		Services:  make(map[string]string),
	}

	// Check database connectivity
	sqlDB, err := s.db.DB()
	if err != nil {
		status.Services["database"] = "unhealthy"
		status.Status = "unhealthy"
	} else {
		if err := sqlDB.Ping(); err != nil {
			status.Services["database"] = "unhealthy"
			status.Status = "unhealthy"
		} else {
			status.Services["database"] = "healthy"
		}
	}

	// Check other services can be added here
	status.Services["auth"] = "healthy"
	status.Services["email"] = "healthy"
	status.Services["audit"] = "healthy"

	return status, nil
}

// GetSystemMetrics returns comprehensive system metrics
func (s *MonitoringService) GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Get runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics.Runtime = RuntimeMetrics{
		GoVersion:   runtime.Version(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryUsage: int64(m.Alloc),
		MemoryAlloc: int64(m.Alloc),
		MemorySys:   int64(m.Sys),
		GCCount:     m.NumGC,
		NextGC:      int64(m.NextGC),
		CPUCount:    runtime.NumCPU(),
	}

	// Get database metrics
	dbMetrics, err := s.getDatabaseMetrics()
	if err != nil {
		return nil, err
	}
	metrics.Database = *dbMetrics

	// Get HTTP metrics (placeholder - would be collected from middleware)
	metrics.HTTP = HTTPMetrics{
		TotalRequests:   0, // Would be collected from metrics middleware
		ActiveRequests:  0,
		AvgResponseTime: 0,
		RequestsByStatus: map[string]int64{
			"2xx": 0,
			"4xx": 0,
			"5xx": 0,
		},
	}

	return metrics, nil
}

// getDatabaseMetrics collects database-specific metrics
func (s *MonitoringService) getDatabaseMetrics() (*DatabaseMetrics, error) {
	metrics := &DatabaseMetrics{
		Status: "healthy",
	}

	// Get connection pool stats
	sqlDB, err := s.db.DB()
	if err != nil {
		metrics.Status = "unhealthy"
		return metrics, nil
	}

	stats := sqlDB.Stats()
	metrics.ActiveConnections = stats.InUse
	metrics.IdleConnections = stats.Idle

	// Get table counts
	s.db.Model(&models.User{}).Count(&metrics.TotalUsers)
	s.db.Model(&models.Period{}).Count(&metrics.TotalPeriods)
	s.db.Model(&models.AuditLog{}).Count(&metrics.TotalAuditLogs)

	// Calculate average response time (simple ping test)
	start := time.Now()
	err = sqlDB.Ping()
	if err != nil {
		metrics.Status = "unhealthy"
		metrics.AvgResponseTime = -1
	} else {
		metrics.AvgResponseTime = float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds
	}

	return metrics, nil
}

// GetDatabaseStatus returns detailed database status
func (s *MonitoringService) GetDatabaseStatus() (map[string]interface{}, error) {
	status := make(map[string]interface{})

	sqlDB, err := s.db.DB()
	if err != nil {
		status["status"] = "error"
		status["error"] = err.Error()
		return status, err
	}

	// Test connectivity
	start := time.Now()
	err = sqlDB.Ping()
	pingDuration := time.Since(start)

	if err != nil {
		status["status"] = "error"
		status["error"] = err.Error()
		status["ping_time"] = pingDuration.String()
		return status, err
	}

	status["status"] = "healthy"
	status["ping_time"] = pingDuration.String()

	// Get connection stats
	stats := sqlDB.Stats()
	status["max_open_connections"] = stats.MaxOpenConnections
	status["open_connections"] = stats.OpenConnections
	status["in_use"] = stats.InUse
	status["idle"] = stats.Idle
	status["wait_count"] = stats.WaitCount
	status["wait_duration"] = stats.WaitDuration.String()
	status["max_idle_closed"] = stats.MaxIdleClosed
	status["max_idle_time_closed"] = stats.MaxIdleTimeClosed
	status["max_lifetime_closed"] = stats.MaxLifetimeClosed

	// Get table information
	tables := make(map[string]int64)
	var count int64

	s.db.Model(&models.User{}).Count(&count)
	tables["users"] = count

	s.db.Model(&models.Period{}).Count(&count)
	tables["periods"] = count

	s.db.Model(&models.SourceFile{}).Count(&count)
	tables["source_files"] = count

	s.db.Model(&models.RawRecord{}).Count(&count)
	tables["raw_records"] = count

	s.db.Model(&models.AuditLog{}).Count(&count)
	tables["audit_logs"] = count

	s.db.Model(&models.PasswordResetToken{}).Count(&count)
	tables["password_reset_tokens"] = count

	s.db.Model(&models.EmailVerificationToken{}).Count(&count)
	tables["email_verification_tokens"] = count

	status["tables"] = tables

	return status, nil
}

// GetSystemInfo returns general system information
func (s *MonitoringService) GetSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["service_name"] = "人事行政管理系统 (hr-office)"
	info["version"] = "1.0.0"
	info["go_version"] = runtime.Version()
	info["build_time"] = s.startTime.Format(time.RFC3339)
	info["uptime"] = time.Since(s.startTime).String()
	info["timezone"] = time.Now().Format("MST")

	// Runtime info
	info["goroutines"] = runtime.NumGoroutine()
	info["cpu_count"] = runtime.NumCPU()

	// Memory info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info["memory"] = map[string]interface{}{
		"alloc_bytes":      m.Alloc,
		"total_alloc_bytes": m.TotalAlloc,
		"sys_bytes":        m.Sys,
		"gc_count":         m.NumGC,
		"next_gc_bytes":    m.NextGC,
	}

	return info
}

// CleanupExpiredTokens performs maintenance tasks
func (s *MonitoringService) CleanupExpiredTokens() error {
	// Cleanup expired password reset tokens
	cutoffDate := time.Now().AddDate(0, 0, -7) // Keep 7 days

	result := s.db.Where("(used = ? OR expires_at < ?) AND created_at < ?",
		true, time.Now(), cutoffDate).Delete(&models.PasswordResetToken{})

	if result.Error != nil {
		return result.Error
	}

	// Cleanup expired email verification tokens
	result = s.db.Where("(used = ? OR expires_at < ?) AND created_at < ?",
		true, time.Now(), cutoffDate).Delete(&models.EmailVerificationToken{})

	return result.Error
}
