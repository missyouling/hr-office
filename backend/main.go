package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/api"
	"siapp/internal/auth"
	"siapp/internal/models"
)

func main() {
	dbPath := os.Getenv("SIAPP_DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/siapp.db"
	}

	if err := os.MkdirAll("./data", 0o755); err != nil {
		log.Fatalf("create data directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Period{},
		&models.SourceFile{},
		&models.RawRecord{},
		&models.PeriodSummary{},
		&models.PersonalCharge{},
		&models.UnitCharge{},
		&models.RosterEntry{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager()

	// Create handlers
	handler := api.NewHandler(db)
	authHandler := api.NewAuthHandler(db, jwtManager)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Improved CORS settings - more secure
	corsOptions := cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}

	// Check for allowed origins from environment variable
	if allowedOrigins := os.Getenv("ALLOWED_ORIGINS"); allowedOrigins != "" {
		corsOptions.AllowedOrigins = []string{allowedOrigins}
	} else {
		// Default for development - should be configured for production
		corsOptions.AllowedOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	r.Use(cors.Handler(corsOptions))

	r.Route("/api", func(apiRouter chi.Router) {
		// Public authentication routes
		apiRouter.Post("/auth/register", authHandler.Register)
		apiRouter.Post("/auth/login", authHandler.Login)

		// Protected routes
		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(auth.JWTMiddleware(jwtManager))

			// Auth profile routes
			protectedRouter.Get("/auth/profile", authHandler.GetProfile)
			protectedRouter.Post("/auth/logout", authHandler.Logout)
			protectedRouter.Post("/auth/change-password", authHandler.ChangePassword)

			// All existing routes are now protected
			handler.RegisterRoutes(protectedRouter)
		})
	})

	addr := os.Getenv("SIAPP_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}

	log.Printf("social insurance server listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
