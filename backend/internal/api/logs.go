package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type LogHandler struct {
	db *gorm.DB
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	return &LogHandler{db: db}
}

func (h *LogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.QueryLogs)
	r.Get("/export", h.ExportLogs)
	r.Post("/backup", h.CreateBackup)
	r.Get("/backups", h.ListBackups)
	r.Get("/alert-rules", h.ListAlertRules)
	r.Post("/alert-rules", h.CreateAlertRule)
	r.Put("/alert-rules/{id}", h.UpdateAlertRule)
	r.Delete("/alert-rules/{id}", h.DeleteAlertRule)
	return r
}

func (h *LogHandler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"total": 0,
		"page":  page,
		"size":  size,
	})
}

func (h *LogHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
}

func (h *LogHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}})
}

func (h *LogHandler) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}})
}

func (h *LogHandler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
}

func (h *LogHandler) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
}

func (h *LogHandler) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
}

func (h *LogHandler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=logs-export.xlsx")
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "export started"})
}
