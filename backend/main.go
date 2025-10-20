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

	handler := api.NewHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))

	r.Route("/api", func(apiRouter chi.Router) {
		handler.RegisterRoutes(apiRouter)
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
