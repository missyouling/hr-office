package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	db *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

func (h *NotificationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListNotifications)
	r.Get("/unread-count", h.GetUnreadCount)
	r.Put("/{id}/read", h.MarkAsRead)
	r.Put("/read-all", h.MarkAllAsRead)
	return r
}

type Notification struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	Read      bool   `json:"read"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}

	// TODO: Query from database
	notifications := []Notification{}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  notifications,
		"total": 0,
		"page":  page,
		"size":  size,
		"unread": 0,
	})
}

func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	// TODO: Get actual unread count from database
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"unread": 0,
	})
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// TODO: Update database
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"id":     id,
	})
}

func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	// TODO: Mark all as read in database
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
	})
}