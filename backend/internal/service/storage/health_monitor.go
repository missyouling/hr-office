package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"sync"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// HealthMonitor periodically checks storage health
type HealthMonitor struct {
	db       *gorm.DB
	registry *Registry
	mu       sync.Mutex
	cancel   context.CancelFunc
	running  bool
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(db *gorm.DB, registry *Registry) *HealthMonitor {
	return &HealthMonitor{
		db:       db,
		registry: registry,
	}
}

// Start begins the health monitoring loop
func (m *HealthMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true

	go m.run(ctx)
	log.Println("[HealthMonitor] started")
}

// Stop stops the health monitoring loop
func (m *HealthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.cancel()
	m.running = false
	log.Println("[HealthMonitor] stopped")
}

func (m *HealthMonitor) run(ctx context.Context) {
	// Initial check after 30 seconds
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			m.checkAll(ctx)
			// Reset timer to minimum interval among enabled configs
			interval := m.getMinInterval()
			timer.Reset(time.Duration(interval) * time.Second)
		}
	}
}

func (m *HealthMonitor) getMinInterval() int {
	var configs []models.StorageConfig
	m.db.Where("health_check_enabled = ? AND enabled = ?", true, true).Find(&configs)

	minInterval := 300 // default 5 minutes
	for _, c := range configs {
		if c.HealthCheckInterval > 0 && c.HealthCheckInterval < minInterval {
			minInterval = c.HealthCheckInterval
		}
	}
	return minInterval
}

func (m *HealthMonitor) checkAll(ctx context.Context) {
	var configs []models.StorageConfig
	if err := m.db.Where("health_check_enabled = ? AND enabled = ?", true, true).Find(&configs).Error; err != nil {
		log.Printf("[HealthMonitor] failed to load configs: %v", err)
		return
	}

	for _, config := range configs {
		select {
		case <-ctx.Done():
			return
		default:
			m.checkOne(ctx, &config)
		}
	}
}

func (m *HealthMonitor) checkOne(ctx context.Context, config *models.StorageConfig) {
	driver, err := m.registry.Create(config.Type, ([]byte)(config.Config))
	if err != nil {
		log.Printf("[HealthMonitor] failed to create driver for %s (id=%d): %v", config.Name, config.ID, err)
		m.markError(config, "failed to create driver: "+err.Error())
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	status, err := driver.Test(checkCtx)
	now := time.Now()

	if err != nil || (status != nil && !status.Healthy) {
		config.FailCount++
		config.LastHealthCheck = &now

		message := "unknown error"
		if err != nil {
			message = err.Error()
		} else if status != nil {
			message = status.Message
		}

		if config.FailCount >= config.MaxFailCount {
			config.Status = "error"
			log.Printf("[HealthMonitor] %s (id=%d) marked as ERROR after %d failures: %s",
				config.Name, config.ID, config.FailCount, message)
			m.sendDisconnectNotification(config, message)
			// Auto-failover: if this was the default, try to promote backup
			if config.IsDefault {
				m.promoteBackup(config)
			}
		} else {
			config.Status = "checking"
			log.Printf("[HealthMonitor] %s (id=%d) check failed (%d/%d): %s",
				config.Name, config.ID, config.FailCount, config.MaxFailCount, message)
		}
	} else {
		// Healthy
		if config.Status == "error" || config.Status == "checking" {
			log.Printf("[HealthMonitor] %s (id=%d) recovered", config.Name, config.ID)
			// Trigger background migration for files in fallback storage
			go m.migrateFilesFromFallback(config.ID)
		}
		config.FailCount = 0
		config.Status = "active"
		config.LastHealthCheck = &now
	}

	m.db.Model(config).Updates(map[string]interface{}{
		"status":            config.Status,
		"fail_count":        config.FailCount,
		"last_health_check": config.LastHealthCheck,
	})
}

func (m *HealthMonitor) markError(config *models.StorageConfig, message string) {
	config.FailCount++
	now := time.Now()
	config.LastHealthCheck = &now

	if config.FailCount >= config.MaxFailCount {
		config.Status = "error"
		m.sendDisconnectNotification(config, message)
		if config.IsDefault {
			m.promoteBackup(config)
		}
	}

	m.db.Model(config).Updates(map[string]interface{}{
		"status":            config.Status,
		"fail_count":        config.FailCount,
		"last_health_check": config.LastHealthCheck,
	})
}

func (m *HealthMonitor) promoteBackup(failedConfig *models.StorageConfig) {
	var backup models.StorageConfig
	err := m.db.Where("user_id = ? AND is_backup = ? AND enabled = ? AND status = ? AND id != ?",
		failedConfig.UserID, true, true, "active", failedConfig.ID).
		Order("priority DESC").
		First(&backup).Error

	if err != nil {
		log.Printf("[HealthMonitor] no backup available for user %v", failedConfig.UserID)
		return
	}

	// Demote failed config
	m.db.Model(failedConfig).Update("is_default", false)
	// Promote backup
	m.db.Model(&backup).Update("is_default", true)
	log.Printf("[HealthMonitor] promoted backup %s (id=%d) as default for user %v",
		backup.Name, backup.ID, failedConfig.UserID)
}

// sendDisconnectNotification sends an email notification when storage disconnects
func (m *HealthMonitor) sendDisconnectNotification(config *models.StorageConfig, message string) {
	// Only notify when fail_count FIRST reaches max_fail_count
	// (config.FailCount == config.MaxFailCount, not >)
	if config.FailCount != config.MaxFailCount {
		return
	}

	// Load SMTP config from database
	var smtpConfig models.SMTPConfig
	if err := m.db.First(&smtpConfig).Error; err != nil {
		log.Printf("[HealthMonitor] no SMTP config found, skipping notification for %s", config.Name)
		return
	}

	// Skip if SMTP not configured
	if smtpConfig.Host == "" {
		log.Printf("[HealthMonitor] SMTP not configured, skipping notification for %s", config.Name)
		return
	}

	// Get admin users to notify
	var admins []models.User
	m.db.Where("role = ?", "admin").Find(&admins)
	if len(admins) == 0 {
		log.Printf("[HealthMonitor] no admin users found for notification")
		return
	}

	subject := fmt.Sprintf("存储断连告警: %s", config.Name)
	body := fmt.Sprintf(
		"存储节点 \"%s\" (类型: %s) 已连续 %d 次健康检查失败，已标记为断开状态。\n\n"+
			"错误信息: %s\n\n"+
			"请及时检查存储服务状态。",
		config.Name, config.Type, config.FailCount, message,
	)

	// Send to each admin
	for _, admin := range admins {
		if admin.Email == "" {
			continue
		}
		go func(email string) {
			if err := sendSMTPEmail(smtpConfig, email, subject, body); err != nil {
				log.Printf("[HealthMonitor] failed to send notification to %s: %v", email, err)
			} else {
				log.Printf("[HealthMonitor] disconnect notification sent to %s for storage %s", email, config.Name)
			}
		}(admin.Email)
	}
}

// CheckSingle performs an immediate health check on a single storage config
func (m *HealthMonitor) CheckSingle(ctx context.Context, configID uint) (*HealthStatus, error) {
	var config models.StorageConfig
	if err := m.db.First(&config, configID).Error; err != nil {
		return nil, err
	}

	driver, err := m.registry.Create(config.Type, ([]byte)(config.Config))
	if err != nil {
		return &HealthStatus{
			Healthy:   false,
			Message:   "failed to create driver: " + err.Error(),
			CheckedAt: time.Now(),
		}, nil
	}

	return driver.Test(ctx)
}

// migrateFilesFromFallback migrates files from fallback storage back to primary storage
func (m *HealthMonitor) migrateFilesFromFallback(primaryConfigID uint) {
	log.Printf("[HealthMonitor] starting migration for files in fallback storage (primary config %d)", primaryConfigID)

	var fallbackFiles []models.SysFile
	if err := m.db.Where("primary_config_id = ? AND is_fallback = ? AND migration_status = ?",
		primaryConfigID, true, "pending").Find(&fallbackFiles).Error; err != nil {
		log.Printf("[HealthMonitor] failed to query fallback files: %v", err)
		return
	}

	if len(fallbackFiles) == 0 {
		log.Printf("[HealthMonitor] no pending migrations for config %d", primaryConfigID)
		return
	}

	log.Printf("[HealthMonitor] found %d files to migrate for config %d", len(fallbackFiles), primaryConfigID)

	var primaryConfig models.StorageConfig
	if err := m.db.First(&primaryConfig, primaryConfigID).Error; err != nil {
		log.Printf("[HealthMonitor] failed to load primary config %d: %v", primaryConfigID, err)
		return
	}

	primaryDriver, err := m.registry.Create(primaryConfig.Type, []byte(primaryConfig.Config))
	if err != nil {
		log.Printf("[HealthMonitor] failed to create primary driver for config %d: %v", primaryConfigID, err)
		return
	}

	var fallbackConfig models.StorageConfig
	if len(fallbackFiles) > 0 && fallbackFiles[0].StorageConfigID != nil {
		if err := m.db.First(&fallbackConfig, *fallbackFiles[0].StorageConfigID).Error; err != nil {
			log.Printf("[HealthMonitor] failed to load fallback config: %v", err)
			return
		}
	}

	fallbackDriver, err := m.registry.Create(fallbackConfig.Type, []byte(fallbackConfig.Config))
	if err != nil {
		log.Printf("[HealthMonitor] failed to create fallback driver: %v", err)
		return
	}

	for _, file := range fallbackFiles {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := m.migrateFile(ctx, &file, fallbackDriver, primaryDriver, &primaryConfig); err != nil {
			log.Printf("[HealthMonitor] migration failed for file %d: %v", file.ID, err)
			m.db.Model(&file).Updates(map[string]interface{}{
				"migration_status": "failed",
			})
		}
	}

	log.Printf("[HealthMonitor] migration completed for config %d", primaryConfigID)
}

// migrateFile migrates a single file from fallback to primary storage
func (m *HealthMonitor) migrateFile(ctx context.Context, file *models.SysFile, fallbackDriver, primaryDriver Driver, primaryConfig *models.StorageConfig) error {
	reader, err := fallbackDriver.Download(ctx, file.Path)
	if err != nil {
		return fmt.Errorf("failed to download from fallback: %w", err)
	}
	defer reader.Close()

	if err := primaryDriver.Upload(ctx, file.Path, reader, file.Size); err != nil {
		return fmt.Errorf("failed to upload to primary: %w", err)
	}

	if err := fallbackDriver.Delete(ctx, file.Path); err != nil {
		log.Printf("[HealthMonitor] warning: failed to delete from fallback storage: %v", err)
	}

	if err := m.db.Model(file).Updates(map[string]interface{}{
		"is_fallback":       false,
		"migration_status":  "completed",
		"storage_type":      primaryDriver.Type(),
		"storage_config_id": primaryConfig.ID,
		"primary_config_id": nil,
	}).Error; err != nil {
		return fmt.Errorf("failed to update file metadata: %w", err)
	}

	log.Printf("[HealthMonitor] successfully migrated file %d to primary storage", file.ID)
	return nil
}

// sendSMTPEmail sends an email via SMTP with TLS support
func sendSMTPEmail(cfg models.SMTPConfig, to, subject, body string) error {
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.FromName, from, to, subject, body)

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	if cfg.UseTLS {
		tlsConfig := &tls.Config{ServerName: cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("SMTP client failed: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		if err = client.Mail(from); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
