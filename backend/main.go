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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/api"
	"siapp/internal/auth"
	auditmw "siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/supabase"
)

// connectDatabase connects to database based on environment configuration
func connectDatabase() (*gorm.DB, error) {
	dbType := os.Getenv("SIAPP_DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite" // Default to SQLite
	}

	var db *gorm.DB
	var err error

	switch dbType {
	case "postgres", "postgresql":
		// PostgreSQL connection
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

		// 创建自定义拨号器，优先尝试 IPv4，失败时回退到默认拨号逻辑
		connConfig, err := pgx.ParseConfig(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			dbHost, dbUser, dbPassword, dbName, dbPort, sslMode))
		if err != nil {
			return nil, fmt.Errorf("parse postgres config: %v", err)
		}

		// 优先尝试解析 IPv4 地址，若不可用则回退到原始地址（允许 IPv6）
		connConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
			if err == nil && len(ips) > 0 {
				ipv4Addr := net.JoinHostPort(ips[0].String(), port)
				log.Printf("Resolved %s to IPv4: %s", host, ipv4Addr)

				if conn, dialErr := dialer.DialContext(ctx, "tcp", ipv4Addr); dialErr == nil {
					return conn, nil
				} else {
					log.Printf("Failed to dial IPv4 %s: %v, falling back to %s", ipv4Addr, dialErr, addr)
				}
			} else if err != nil {
				log.Printf("IPv4 lookup failed for %s: %v, falling back to %s", host, err, addr)
			} else {
				log.Printf("No IPv4 address found for %s, falling back to %s", host, addr)
			}

			return dialer.DialContext(ctx, network, addr)
		}

		connStr := stdlib.RegisterConnConfig(connConfig)
		log.Printf("Connecting to PostgreSQL database: host=%s dbname=%s", dbHost, dbName)
		db, err = gorm.Open(postgres.New(postgres.Config{
			DriverName: "pgx",
			DSN:        connStr,
		}), &gorm.Config{})

	case "sqlite":
		// SQLite connection (default)
		dbPath := os.Getenv("SIAPP_DATABASE_PATH")
		if dbPath == "" {
			dbPath = "./data/siapp.db"
		}

		if err := os.MkdirAll("./data", 0o755); err != nil {
			return nil, fmt.Errorf("create data directory: %v", err)
		}

		log.Printf("Connecting to SQLite database: %s", dbPath)
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})

	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: sqlite, postgres)", dbType)
	}

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
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Initialize default admin user
	if err := initializeDefaultAdmin(db); err != nil {
		log.Fatalf("initialize default admin: %v", err)
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager()

	// Create services
	auditService := service.NewAuditService(db)
	passwordResetService := service.NewPasswordResetService(db)
	emailVerificationService := service.NewEmailVerificationService(db)
	emailService := service.NewEmailService()
	monitoringService := service.NewMonitoringService(db)

	// Create handlers
	handler := api.NewHandler(db)
	authHandler := api.NewAuthHandler(db, jwtManager, passwordResetService, emailVerificationService, emailService)
	auditHandler := api.NewAuditHandler(db, auditService)
	monitoringHandler := api.NewMonitoringHandler(db, monitoringService)

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
		corsOptions.AllowedOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
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

		// Protected routes with Supabase JWT auth first, then audit logging
		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(supabase.SupabaseJWTMiddleware())
			protectedRouter.Use(auditmw.AuditMiddleware(auditService))

			// Auth profile routes
			protectedRouter.Get("/auth/profile", authHandler.GetProfile)
			protectedRouter.Post("/auth/logout", authHandler.Logout)
			protectedRouter.Post("/auth/change-password", authHandler.ChangePassword)
			protectedRouter.Get("/auth/check-email-verification", authHandler.CheckEmailVerificationStatus)

			// Audit log routes
			auditHandler.RegisterAuditRoutes(protectedRouter)

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
