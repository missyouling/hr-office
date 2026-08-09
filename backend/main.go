package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"siapp/internal/api"
	"siapp/internal/auth"
	auditmw "siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/service/storage"
	"siapp/internal/supabase"
)

// connectDatabase connects to database based on environment configuration
func connectDatabase() (*gorm.DB, error) {
	dbType := os.Getenv("SIAPP_DATABASE_TYPE")
	if dbType == "" {
		// SQLite is disabled. PostgreSQL is required.
		return nil, fmt.Errorf("SIAPP_DATABASE_TYPE environment variable is required and must be set to 'postgres' or 'postgresql'. SQLite is no longer supported. Example: export SIAPP_DATABASE_TYPE=postgres")
	}

	// Only allow PostgreSQL
	if dbType != "postgres" && dbType != "postgresql" {
		return nil, fmt.Errorf("unsupported database type: %s. Only 'postgres' and 'postgresql' are supported. SQLite has been disabled. (SIAPP_DATABASE_TYPE=%s)", dbType, dbType)
	}

	var db *gorm.DB
	var err error

	// PostgreSQL connection (only supported option)
	dbHost := os.Getenv("SIAPP_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("SIAPP_DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("SIAPP_DB_USER")
	if dbUser == "" {
		dbUser = "siapp"
	}

	dbPassword := os.Getenv("SIAPP_DB_PASSWORD")
	if dbPassword == "" {
		return nil, fmt.Errorf("SIAPP_DB_PASSWORD environment variable is required for PostgreSQL")
	}

	dbName := os.Getenv("SIAPP_DB_NAME")
	if dbName == "" {
		dbName = "siapp"
	}

	sslMode := os.Getenv("SIAPP_DB_SSLMODE")
	if sslMode == "" {
		sslMode = "require"
	}

	// Parse PostgreSQL connection config
	connConfig, err := pgx.ParseConfig(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, sslMode))
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %v", err)
	}

	// Supabase pooler（PgBouncer）在事务模式下不支持预编译语句缓存，
	// 使用 exec 模式避免 "prepared statement already exists (42P05)" 错误
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	// PostgreSQL connection strategy: prefer IPv4, allow IPv6 if IPv4 unavailable
	connConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		// Try IPv4 first
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
		if err == nil && len(ips) > 0 {
			ipv4Addr := net.JoinHostPort(ips[0].String(), port)
			log.Printf("Resolved %s to IPv4: %s", host, ipv4Addr)
			if conn, dialErr := dialer.DialContext(ctx, "tcp", ipv4Addr); dialErr == nil {
				return conn, nil
			}
			log.Printf("IPv4 connection failed for %s, attempting IPv6 fallback", host)
		}

		// Fallback to IPv6 if IPv4 fails or unavailable
		ips, err = net.DefaultResolver.LookupIP(ctx, "ip6", host)
		if err == nil && len(ips) > 0 {
			ipv6Addr := net.JoinHostPort(ips[0].String(), port)
			log.Printf("Resolved %s to IPv6: %s", host, ipv6Addr)
			if conn, dialErr := dialer.DialContext(ctx, "tcp", ipv6Addr); dialErr == nil {
				return conn, nil
			}
		}

		// Last resort: use host directly (PostgreSQL driver will handle DNS resolution)
		log.Printf("IPv4/IPv6 lookup failed, using default connection to %s", addr)
		return dialer.DialContext(ctx, network, addr)
	}

	connStr := stdlib.RegisterConnConfig(connConfig)
	log.Printf("Connecting to PostgreSQL database: host=%s port=%s dbname=%s user=%s sslmode=%s", dbHost, dbPort, dbName, dbUser, sslMode)
	db, err = gorm.Open(postgres.New(postgres.Config{
		DriverName: "pgx",
		DSN:        connStr,
	}), &gorm.Config{})

	return db, err
}

// initializeDefaultAdmin creates a default admin user if it doesn't exist
func initializeDefaultAdmin(db *gorm.DB) error {
	// Check if admin user already exists
	var existingAdmin models.User
	result := db.Where("username = ?", "admin").First(&existingAdmin)

	if result.Error != nil && result.Error.Error() == "record not found" {
		// Admin user doesn't exist, create it
		admin := models.User{
			Username:      "admin",
			Email:         "admin@system.local",
			FullName:      "系统管理员",
			Role:          "super_admin",
			Active:        true,
			EmailVerified: true, // Admin account is pre-verified
		}

		// Set password to "admin123"
		if err := admin.SetPassword("admin123"); err != nil {
			return fmt.Errorf("set admin password: %v", err)
		}

		// Set email verification timestamp
		now := time.Now()
		admin.EmailVerifiedAt = &now

		// Create the admin user
		if err := db.Create(&admin).Error; err != nil {
			return fmt.Errorf("create admin user: %v", err)
		}

		log.Printf("Default admin user created: username=admin, password=admin123")
		return nil
	} else if result.Error != nil {
		return fmt.Errorf("check for existing admin user: %v", result.Error)
	}

	// Admin user already exists
	log.Printf("Default admin user already exists")
	return nil
}

func ensureDefaultAdminSuperAdminRole(db *gorm.DB) error {
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find default admin user: %v", err)
	}

	var superAdminRole models.Role
	if err := db.Where("name = ?", "super_admin").First(&superAdminRole).Error; err != nil {
		return fmt.Errorf("find super_admin role: %v", err)
	}

	if adminUser.Role != "super_admin" {
		if err := db.Model(&models.User{}).Where("id = ?", adminUser.ID).Update("role", "super_admin").Error; err != nil {
			return fmt.Errorf("update admin role to super_admin: %v", err)
		}
	}

	var userRole models.UserRole
	if err := db.Where("user_id = ? AND role_id = ?", adminUser.ID, superAdminRole.ID).First(&userRole).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("check admin user-role mapping: %v", err)
		}
		if err := db.Create(&models.UserRole{UserID: adminUser.ID, RoleID: superAdminRole.ID}).Error; err != nil {
			return fmt.Errorf("create admin user-role mapping: %v", err)
		}
	}

	return nil
}

func relaxSocialInsuranceConstraints(db *gorm.DB) {
	if db == nil {
		return
	}
	if !strings.EqualFold(db.Dialector.Name(), "postgres") {
		return
	}
	if err := db.Exec("ALTER TABLE social_insurance_records ALTER COLUMN batch_id DROP NOT NULL").Error; err != nil {
		log.Printf("failed to relax social insurance constraint: %v", err)
	}
}

// ensureKnowledgeBaseInfrastructure 确保知识库基础设施可用
// 在 GORM AutoMigrate 之后执行：
//  1. 启用 pgvector 扩展
//  2. 为 document_chunks 添加 embedding 向量列（GORM 不原生支持 vector 类型）
//  3. 创建 HNSW 向量索引
//  4. 添加 tsvector 全文搜索列及 GIN 索引
func ensureKnowledgeBaseInfrastructure(db *gorm.DB) {
	if db == nil {
		return
	}
	if !strings.EqualFold(db.Dialector.Name(), "postgres") {
		return
	}

	// 1. 启用 pgvector 扩展
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Printf("[kb-infra] failed to enable vector extension: %v", err)
		return
	}

	// 2. 为 document_chunks 添加 embedding 列
	if err := db.Exec("ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS embedding vector(768)").Error; err != nil {
		log.Printf("[kb-infra] failed to add embedding column: %v", err)
	}

	// 3. 创建 HNSW 向量索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunk_embedding_hnsw
		ON document_chunks USING hnsw (embedding vector_cosine_ops)
		WITH (m = 16, ef_construction = 64)
	`).Error; err != nil {
		log.Printf("[kb-infra] failed to create HNSW index: %v", err)
	}

	// 4. 为 documents 添加 tsvector 列 + 触发器
	if err := db.Exec("ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_tsv tsvector").Error; err != nil {
		log.Printf("[kb-infra] failed to add documents content_tsv: %v", err)
	}
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION documents_tsvector_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsv := to_tsvector('simple', COALESCE(NEW.content_text, '') || ' ' || COALESCE(NEW.file_name, ''));
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`).Error; err != nil {
		log.Printf("[kb-infra] failed to create documents tsvector trigger function: %v", err)
	}
	if err := db.Exec("DROP TRIGGER IF EXISTS trg_documents_tsvector ON documents").Error; err != nil {
		log.Printf("[kb-infra] failed to drop old documents trigger: %v", err)
	}
	if err := db.Exec("CREATE TRIGGER trg_documents_tsvector BEFORE INSERT OR UPDATE OF content_text, file_name ON documents FOR EACH ROW EXECUTE FUNCTION documents_tsvector_trigger()").Error; err != nil {
		log.Printf("[kb-infra] failed to create documents tsvector trigger: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_documents_content_tsv ON documents USING GIN (content_tsv)").Error; err != nil {
		log.Printf("[kb-infra] failed to create documents tsvector GIN index: %v", err)
	}

	// 5. 为 document_chunks 添加 tsvector 列 + 触发器
	if err := db.Exec("ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector").Error; err != nil {
		log.Printf("[kb-infra] failed to add chunks content_tsv: %v", err)
	}
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION chunks_tsvector_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsv := to_tsvector('simple', COALESCE(NEW.content, ''));
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`).Error; err != nil {
		log.Printf("[kb-infra] failed to create chunks tsvector trigger function: %v", err)
	}
	if err := db.Exec("DROP TRIGGER IF EXISTS trg_chunks_tsvector ON document_chunks").Error; err != nil {
		log.Printf("[kb-infra] failed to drop old chunks trigger: %v", err)
	}
	if err := db.Exec("CREATE TRIGGER trg_chunks_tsvector BEFORE INSERT OR UPDATE OF content ON document_chunks FOR EACH ROW EXECUTE FUNCTION chunks_tsvector_trigger()").Error; err != nil {
		log.Printf("[kb-infra] failed to create chunks tsvector trigger: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_chunks_content_tsv ON document_chunks USING GIN (content_tsv)").Error; err != nil {
		log.Printf("[kb-infra] failed to create chunks tsvector GIN index: %v", err)
	}

	log.Println("[kb-infra] knowledge base infrastructure initialized successfully")
}

func dedupeUserPreferences(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	type duplicate struct {
		UserID  *uint
		PrefKey string
		Count   int64
	}
	var duplicates []duplicate
	if err := db.Table("user_preferences").Select("user_id, pref_key, COUNT(*) as count").
		Group("user_id, pref_key").Having("COUNT(*) > 1").Scan(&duplicates).Error; err != nil {
		return err
	}
	for _, dup := range duplicates {
		var prefs []models.UserPreference
		query := db.Where("pref_key = ?", dup.PrefKey)
		if dup.UserID == nil {
			query = query.Where("user_id IS NULL")
		} else {
			query = query.Where("user_id = ?", *dup.UserID)
		}
		if err := query.Order("updated_at DESC, id DESC").Find(&prefs).Error; err != nil {
			return err
		}
		if len(prefs) <= 1 {
			continue
		}
		var removeIDs []uint
		for i := 1; i < len(prefs); i++ {
			removeIDs = append(removeIDs, prefs[i].ID)
		}
		if len(removeIDs) == 0 {
			continue
		}
		if err := db.Where("id IN ?", removeIDs).Delete(&models.UserPreference{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureUserPreferenceIndex(db *gorm.DB) {
	if db == nil {
		return
	}
	if err := dedupeUserPreferences(db); err != nil {
		log.Printf("failed to dedupe user preferences: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_pref_user_key ON user_preferences(user_id, pref_key)").Error; err != nil {
		log.Printf("failed to ensure user preference unique index: %v", err)
	}
}

func ensureModelUsageLogsTable(db *gorm.DB) {
	if db == nil {
		return
	}
	// Check if table exists
	if !db.Migrator().HasTable(&models.ModelUsageLog{}) {
		if err := db.Migrator().CreateTable(&models.ModelUsageLog{}); err != nil {
			log.Printf("failed to create model_usage_logs table: %v", err)
			return
		}
		log.Printf("created model_usage_logs table")
	}
	// Create indexes
	if err := db.Migrator().CreateIndex(&models.ModelUsageLog{}, "user_id"); err != nil {
		log.Printf("failed to create user_id index: %v", err)
	}
	if err := db.Migrator().CreateIndex(&models.ModelUsageLog{}, "config_id"); err != nil {
		log.Printf("failed to create config_id index: %v", err)
	}
	if err := db.Migrator().CreateIndex(&models.ModelUsageLog{}, "config_type"); err != nil {
		log.Printf("failed to create config_type index: %v", err)
	}
	if err := db.Migrator().CreateIndex(&models.ModelUsageLog{}, "status"); err != nil {
		log.Printf("failed to create status index: %v", err)
	}
	if err := db.Migrator().CreateIndex(&models.ModelUsageLog{}, "created_at"); err != nil {
		log.Printf("failed to create created_at index: %v", err)
	}
}

func main() {
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.Period{},
		&models.SourceFile{},
		&models.RawRecord{},
		&models.PeriodSummary{},
		&models.PersonalCharge{},
		&models.UnitCharge{},
		&models.RosterEntry{},
		&models.Employee{},
		&models.AuditLog{}, // Add audit log table
		&models.SocialInsuranceBatch{},
		&models.SocialInsuranceRecord{},
		&models.CallbackUpload{},
		&models.CallbackRecord{},
		&models.DormSite{},
		&models.DormBuilding{},
		&models.DormRoom{},
		&models.DormBed{},
		&models.DormRoomAsset{},
		&models.DormContract{},
		&models.DormCheckout{},
		&models.DormMeterItem{},
		&models.DormMeterReading{},
		&models.DormBillingRule{},
		&models.DormBill{},
		&models.DormBillItem{},
		&models.UserPreference{},
		&models.ProvidentFundRecord{},
		&models.ProvidentFundSettings{},
		&models.ProvidentFundBill{},
		&models.ProvidentFundBillItem{},
		&models.DocumentCategory{},
		&models.DocumentSubCategory{},
		&models.Document{},
		&models.RetentionPeriod{},
		&models.StorageLocation{},
		&models.ExpirationReminder{},
		&models.Announcement{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.ShareLink{},
		&models.ArchiveSharedField{},
		&models.ArchiveFieldGroup{},
		&models.ArchiveFieldDefinition{},
		&models.CodeRule{},
		&models.CodeRulePlaceholder{},
		&models.ArchiveConfig{},
		&models.ArchiveTag{},
		&models.DocumentTagLink{},
		&models.StorageModuleConfig{},
		&models.StorageConfig{},
		&models.StorageRule{},
		// SysFile 文件元数据表
		&models.SysFile{},
		&models.NotificationConfig{},
		&models.SMTPConfig{},
		&models.DocumentContent{},
		&models.DocumentEmbedding{},
		&models.DocumentChunk{},
		&models.ChunkRevision{},
		&models.ModelConfig{},
		&models.DocumentTypeField{},
		&models.TypeDefaultColumn{},
		&models.OCRJob{},
		&models.ChatMessage{},
		&models.ChatSession{},
		&models.ChatFeedback{},
		&models.ModelUsageLog{},
		&api.SystemLog{},
		&api.LogBackup{},
		&api.AlertRule{},
		&api.BackupSettings{},
	); err != nil {
		log.Printf("auto migrate warning: %v", err)
	}
	ensureUserPreferenceIndex(db)
	ensureModelUsageLogsTable(db)
	relaxSocialInsuranceConstraints(db)
	ensureKnowledgeBaseInfrastructure(db)

	// Seed document categories
	if err := seedDocumentCategories(db); err != nil {
		log.Printf("seed document categories: %v", err)
	}

	// Seed shared fields and field groups
	if err := seedSharedFieldsAndGroups(db); err != nil {
		log.Printf("seed shared fields and groups: %v", err)
	}

	// Seed model configs
	if err := seedModelConfigs(db); err != nil {
		log.Printf("seed model configs: %v", err)
	}

	// Seed RBAC
	if err := seedRBAC(db); err != nil {
		log.Printf("seed RBAC: %v", err)
	}

	// Initialize default admin user
	if err := initializeDefaultAdmin(db); err != nil {
		log.Fatalf("initialize default admin: %v", err)
	}
	if err := ensureDefaultAdminSuperAdminRole(db); err != nil {
		log.Fatalf("ensure default admin super role: %v", err)
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager()

	// Create services
	auditService := service.NewAuditService(db)
	passwordResetService := service.NewPasswordResetService(db)
	emailVerificationService := service.NewEmailVerificationService(db)
	emailService := service.NewEmailService()
	monitoringService := service.NewMonitoringService(db)

	storageManager := storage.NewStorageManager(db, storage.DefaultRegistry)
	if err := storageManager.Init(); err != nil {
		log.Printf("storage manager init: %v", err)
	}
	storage.GlobalManager = storageManager

	healthMonitor := storage.NewHealthMonitor(db, storage.DefaultRegistry)
	healthMonitor.Start()

	// Create handlers
	handler := api.NewHandler(db)
	authHandler := api.NewAuthHandler(db, jwtManager, passwordResetService, emailVerificationService, emailService)
	auditHandler := api.NewAuditHandler(db, auditService)
	monitoringHandler := api.NewMonitoringHandler(db, monitoringService)

	// Auto-cleanup scheduler for logs older than 30 days
	runLogCleanup := func(db *gorm.DB) {
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		auditResult := db.Where("created_at < ?", thirtyDaysAgo).Delete(&models.AuditLog{})
		systemResult := db.Where("created_at < ?", thirtyDaysAgo).Delete(&api.SystemLog{})
		totalDeleted := auditResult.RowsAffected + systemResult.RowsAffected
		log.Printf("auto-cleanup completed: deleted %d logs older than 30 days", totalDeleted)
	}

	// Run cleanup on startup, then daily
	runCleanupTask := func() {
		time.Sleep(10 * time.Second)
		runLogCleanup(db)
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			if now.Hour() >= 2 {
				next = next.AddDate(0, 0, 1)
			}
			duration := next.Sub(now)
			log.Printf("log auto-cleanup scheduled in %v hours", duration.Hours())
			time.Sleep(duration)
			runLogCleanup(db)
		}
	}
	go runCleanupTask()

	// Start usage log cleanup goroutine
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().AddDate(0, 0, -30)
			result := db.Where("created_at < ?", cutoff).Delete(&models.ModelUsageLog{})
			if result.Error != nil {
				log.Printf("[cleanup] failed to clean usage logs: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("[cleanup] deleted %d old usage logs", result.RowsAffected)
			}
		}
	}()

	// Log system startup
	dbType := os.Getenv("SIAPP_DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	customDetails := map[string]interface{}{
		"database_type": dbType,
		"listen_addr":   os.Getenv("SIAPP_ADDR"),
	}

	if dbType == "sqlite" {
		customDetails["database_path"] = os.Getenv("SIAPP_DATABASE_PATH")
	} else if dbType == "postgres" || dbType == "postgresql" {
		customDetails["database_host"] = os.Getenv("SIAPP_DB_HOST")
		customDetails["database_name"] = os.Getenv("SIAPP_DB_NAME")
	}

	auditService.LogSystemEvent(
		models.ActionSystemStart,
		"Social insurance server starting up",
		&models.LogDetails{
			Custom: customDetails,
		},
	)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Add audit context first
	r.Use(auditmw.AuditContext(auditService))

	// Improved CORS settings - more secure
	corsOptions := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}

	// Check for allowed origins from environment variable
	if allowedOrigins := os.Getenv("ALLOWED_ORIGINS"); allowedOrigins != "" {
		origins := strings.Split(allowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		corsOptions.AllowedOrigins = origins
	} else {
		// Default for development - should be configured for production
		corsOptions.AllowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:10010",
			"http://127.0.0.1:10010",
			"http://localhost:10086",
			"http://127.0.0.1:10086",
		}
	}

	r.Use(cors.Handler(corsOptions))

	// Public monitoring endpoints (no /api prefix for standard health checks)
	monitoringHandler.RegisterMonitoringRoutes(r)

	r.Route("/api", func(apiRouter chi.Router) {
		// Public authentication routes with audit logging
		apiRouter.Group(func(publicRouter chi.Router) {
			publicRouter.Use(auditmw.AuditMiddleware(auditService))
			publicRouter.Post("/auth/register", authHandler.Register)
			publicRouter.Post("/auth/check-availability", authHandler.CheckAccountAvailability)
			publicRouter.Post("/auth/login", authHandler.Login)
			publicRouter.Post("/auth/request-password-reset", authHandler.RequestPasswordReset)
			publicRouter.Post("/auth/reset-password", authHandler.ResetPassword)
			publicRouter.Get("/auth/validate-reset-token", authHandler.ValidatePasswordResetToken)
			publicRouter.Get("/auth/verify-email", authHandler.VerifyEmail)
			publicRouter.Post("/auth/resend-verification", authHandler.ResendVerificationEmail)
		})

		// Public archive share link access (no auth required)
		apiRouter.Get("/archives/shared/{token}", handler.AccessSharedDocument)

		// Protected routes with Supabase JWT auth first, then audit logging
		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(supabase.SupabaseJWTMiddleware())
			protectedRouter.Use(auditmw.DepartmentContext(db))
			protectedRouter.Use(auditmw.AuditMiddleware(auditService))

			// Auth profile routes
			protectedRouter.Get("/auth/profile", authHandler.GetProfile)
			protectedRouter.Post("/auth/logout", authHandler.Logout)
			protectedRouter.Post("/auth/change-password", authHandler.ChangePassword)
			protectedRouter.Get("/auth/check-email-verification", authHandler.CheckEmailVerificationStatus)

			// Audit log routes
			auditHandler.RegisterAuditRoutes(protectedRouter)

			// Log management routes
			logsHandler := api.NewLogHandler(db)
			protectedRouter.Mount("/logs", logsHandler.Routes())

			// Notification routes - 推送历史、已读标记、配置管理
			protectedRouter.Route("/notifications", func(notifRouter chi.Router) {
				// 推送历史与已读标记
				notificationHandler := api.NewNotificationHandler(db)
				notifRouter.Get("/", notificationHandler.ListNotifications)
				notifRouter.Get("/unread-count", notificationHandler.GetUnreadCount)
				notifRouter.Put("/{id}/read", notificationHandler.MarkAsRead)
				notifRouter.Put("/read-all", notificationHandler.MarkAllAsRead)

				// 通知配置 CRUD 与测试
				notifRouter.Get("/configs", handler.ListNotificationConfigs)
				notifRouter.Post("/configs", handler.CreateNotificationConfig)
				notifRouter.Get("/configs/{id}", handler.GetNotificationConfig)
				notifRouter.Put("/configs/{id}", handler.UpdateNotificationConfig)
				notifRouter.Delete("/configs/{id}", handler.DeleteNotificationConfig)

				notifRouter.Post("/smtp/send", handler.SendSMTPNotification)
				notifRouter.Post("/sms/send", handler.SendSMSNotification)
				notifRouter.Post("/telegram/send", handler.SendTelegramNotification)
				notifRouter.Post("/webhook/send", handler.SendWebhookNotification)
				notifRouter.Post("/test", handler.TestNotification)
			})

			// Protected monitoring routes
			monitoringHandler.RegisterProtectedMonitoringRoutes(protectedRouter)

			// All existing routes are now protected
			handler.RegisterRoutes(protectedRouter)
		})
	})

	addr := os.Getenv("SIAPP_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}

	dbTypeForLog := os.Getenv("SIAPP_DATABASE_TYPE")
	if dbTypeForLog == "" {
		dbTypeForLog = "sqlite"
	}
	log.Printf("social insurance server listening on %s (db: %s)", addr, dbTypeForLog)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
