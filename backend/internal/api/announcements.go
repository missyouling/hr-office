package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

type announcementPayload struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	IsTop   bool   `json:"is_top"`
	Status  string `json:"status"`
}

func (h *Handler) registerAnnouncementRoutes(r chi.Router) {
	// 注意：外层已经是 /announcements，所以这里直接注册子路由
	// 读操作：announcements.view（viewer 及以上可读公告）
	r.With(middleware.RequirePermission(h.db, "announcements", "view")).Get("/", h.listAnnouncements)
	// 写操作：create / edit / delete 分别校验
	r.With(middleware.RequirePermission(h.db, "announcements", "create")).Post("/", h.createAnnouncement)
	r.With(middleware.RequirePermission(h.db, "announcements", "edit")).Put("/{id}", h.updateAnnouncement)
	r.With(middleware.RequirePermission(h.db, "announcements", "delete")).Delete("/{id}", h.deleteAnnouncement)
}

func (h *Handler) listAnnouncements(w http.ResponseWriter, r *http.Request) {
	var announcements []models.Announcement
	query := h.db.Preload("CreatedByUser").Order("is_top DESC, created_at DESC")

	status := r.URL.Query().Get("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&announcements).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if announcements == nil {
		announcements = []models.Announcement{}
	}

	writeJSON(w, announcements)
}

func (h *Handler) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var payload announcementPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	announcement := models.Announcement{
		Title:     payload.Title,
		Content:   payload.Content,
		IsTop:     payload.IsTop,
		Status:    payload.Status,
		CreatedBy: userID,
	}

	if payload.Status == "published" {
		announcement.PublishedAt = &time.Time{}
		*announcement.PublishedAt = time.Now()
	}

	if err := h.db.Create(&announcement).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create announcement", err)
		return
	}

	writeJSON(w, announcement)
}

func (h *Handler) updateAnnouncement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var payload announcementPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	updates := map[string]interface{}{
		"title":   payload.Title,
		"content": payload.Content,
		"is_top":  payload.IsTop,
		"status":  payload.Status,
	}

	if payload.Status == "published" {
		updates["published_at"] = time.Now()
	} else if payload.Status == "draft" {
		updates["published_at"] = nil
	}

	if err := h.db.Model(&models.Announcement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update announcement", err)
		return
	}

	var announcement models.Announcement
	if err := h.db.First(&announcement, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "announcement not found", err)
		return
	}

	writeJSON(w, announcement)
}

func (h *Handler) deleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	if err := h.db.Delete(&models.Announcement{}, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete announcement", err)
		return
	}

	writeJSON(w, map[string]string{"message": "deleted"})
}
