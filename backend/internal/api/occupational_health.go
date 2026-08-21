package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

type occupationalHealthPayload struct {
	EmployeeID       uint   `json:"employee_id"`
	CheckDate        string `json:"check_date"`
	CheckInstitution string `json:"medical_institution"`
	CheckCategory    string `json:"check_category"`
	Conclusion       string `json:"check_conclusion"`
	NextCheckDate    string `json:"next_check_date"`
	Remarks          string `json:"remarks"`
}

type occupationalHealthVoidPayload struct {
	Reason string `json:"reason"`
}

func (h *Handler) registerOccupationalHealthRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "occupational_health", "view")).Get("/occupational-health-checks", h.listOccupationalHealthChecks)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "create")).Post("/occupational-health-checks", h.createOccupationalHealthCheck)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "view")).Get("/occupational-health-checks/{id}", h.getOccupationalHealthCheck)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "edit")).Put("/occupational-health-checks/{id}", h.updateOccupationalHealthCheck)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "delete")).Delete("/occupational-health-checks/{id}", h.deleteOccupationalHealthCheck)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "edit")).Post("/occupational-health-checks/{id}/complete", h.completeOccupationalHealthCheck)
	r.With(middleware.RequirePermission(h.db, "occupational_health", "delete")).Post("/occupational-health-checks/{id}/void", h.voidOccupationalHealthCheck)
}

func occupationalHealthQuery(ctx context.Context, db *gorm.DB, userID uint) *gorm.DB {
	query := db.Where("user_id = ?", userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(ctx); ok && dept != "" {
		query = query.Where("snapshot_department = ?", dept)
	}
	return query
}

func (h *Handler) loadOccupationalHealthCheck(w http.ResponseWriter, r *http.Request, userID uint) (*models.OccupationalHealthCheck, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的职业健康检查记录 ID", err)
		return nil, false
	}
	var record models.OccupationalHealthCheck
	err = occupationalHealthQuery(r.Context(), h.db.Where("id = ?", id), userID).First(&record).Error
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "未找到职业健康检查记录", err)
		return nil, false
	}
	return &record, true
}

func validateOccupationalHealthPayload(p *occupationalHealthPayload) error {
	record := models.OccupationalHealthCheck{
		EmployeeID:       p.EmployeeID,
		CheckDate:        p.CheckDate,
		CheckInstitution: p.CheckInstitution,
		CheckCategory:    p.CheckCategory,
		NextCheckDate:    p.NextCheckDate,
		Status:           models.OccupationalHealthStatusDraft,
	}
	return record.Validate()
}

func (h *Handler) buildOccupationalHealthCheck(userID uint, p *occupationalHealthPayload) (*models.OccupationalHealthCheck, error) {
	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", p.EmployeeID, userID).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("关联员工不存在或不属于当前租户")
		}
		return nil, err
	}
	return &models.OccupationalHealthCheck{
		UserID:             userID,
		EmployeeID:         p.EmployeeID,
		SnapshotName:       employee.Name,
		SnapshotDepartment: employee.Department,
		SnapshotPosition:   employee.Position,
		CheckDate:          strings.TrimSpace(p.CheckDate),
		CheckInstitution:   strings.TrimSpace(p.CheckInstitution),
		CheckCategory:      strings.TrimSpace(p.CheckCategory),
		Conclusion:         strings.TrimSpace(p.Conclusion),
		NextCheckDate:      strings.TrimSpace(p.NextCheckDate),
		Remarks:            strings.TrimSpace(p.Remarks),
		Status:             models.OccupationalHealthStatusDraft,
	}, nil
}

func (h *Handler) listOccupationalHealthChecks(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := occupationalHealthQuery(r.Context(), h.db, userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidOccupationalHealthStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的职业健康检查状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var records []models.OccupationalHealthCheck
	if err := query.Order("check_date DESC, created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询职业健康检查记录失败", err)
		return
	}
	if records == nil {
		records = []models.OccupationalHealthCheck{}
	}
	respondJSON(w, http.StatusOK, records)
}

func (h *Handler) getOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOccupationalHealthCheck(w, r, userID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, record)
}

func (h *Handler) createOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload occupationalHealthPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateOccupationalHealthPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	record, err := h.buildOccupationalHealthCheck(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建职业健康检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

func (h *Handler) updateOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOccupationalHealthCheck(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OccupationalHealthStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可编辑", nil)
		return
	}
	var payload occupationalHealthPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateOccupationalHealthPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.buildOccupationalHealthCheck(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updates := map[string]any{
		"employee_id":         updated.EmployeeID,
		"snapshot_name":       updated.SnapshotName,
		"snapshot_department": updated.SnapshotDepartment,
		"snapshot_position":   updated.SnapshotPosition,
		"check_date":          updated.CheckDate,
		"check_institution":   updated.CheckInstitution,
		"check_category":      updated.CheckCategory,
		"conclusion":          updated.Conclusion,
		"next_check_date":     updated.NextCheckDate,
		"remarks":             updated.Remarks,
	}
	if err := h.db.Model(record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新职业健康检查记录失败", err)
		return
	}
	h.respondOccupationalHealthCheck(w, record.ID)
}

func (h *Handler) deleteOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOccupationalHealthCheck(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OccupationalHealthStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", record.ID, userID).Delete(&models.OccupationalHealthCheck{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除职业健康检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": record.ID})
}

func (h *Handler) completeOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOccupationalHealthCheck(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OccupationalHealthStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可完成", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(record).Updates(map[string]any{"status": models.OccupationalHealthStatusCompleted, "completed_at": now}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "完成职业健康检查记录失败", err)
		return
	}
	h.respondOccupationalHealthCheck(w, record.ID)
}

func (h *Handler) voidOccupationalHealthCheck(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOccupationalHealthCheck(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OccupationalHealthStatusDraft && record.Status != models.OccupationalHealthStatusCompleted {
		respondError(w, http.StatusConflict, "仅草稿或已完成记录可作废", nil)
		return
	}
	var payload occupationalHealthVoidPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(record).Updates(map[string]any{"status": models.OccupationalHealthStatusVoided, "void_reason": reason, "voided_at": now}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "作废职业健康检查记录失败", err)
		return
	}
	h.respondOccupationalHealthCheck(w, record.ID)
}

func (h *Handler) respondOccupationalHealthCheck(w http.ResponseWriter, id uint) {
	var record models.OccupationalHealthCheck
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重新加载职业健康检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}
