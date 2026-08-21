package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// safetyInspectionPayload 创建/更新安全检查请求体。
// 必填：inspection_type/inspection_date/location/responsible_person/issue_description/rectification_requirement。
type safetyInspectionPayload struct {
	InspectionType           string `json:"inspection_type"`
	InspectionDate           string `json:"inspection_date"`
	Location                 string `json:"location"`
	ResponsiblePerson        string `json:"responsible_person"`
	IssueDescription         string `json:"issue_description"`
	RectificationRequirement string `json:"rectification_requirement"`
}

// safetyInspectionVoidPayload 作废安全检查请求体（原因必填）。
type safetyInspectionVoidPayload struct {
	Reason string `json:"reason"`
}

// registerSafetyInspectionRoutes 注册安全检查路由（P12.3.9）。
// 权限：列表/详情 safety.view；创建 safety.create；
// 编辑/完成 safety.edit；删除/作废 safety.delete。
func (h *Handler) registerSafetyInspectionRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "safety", "view")).Get("/safety-inspections", h.listSafetyInspections)
	r.With(middleware.RequirePermission(h.db, "safety", "create")).Post("/safety-inspections", h.createSafetyInspection)
	r.With(middleware.RequirePermission(h.db, "safety", "view")).Get("/safety-inspections/{id}", h.getSafetyInspection)
	r.With(middleware.RequirePermission(h.db, "safety", "edit")).Put("/safety-inspections/{id}", h.updateSafetyInspection)
	r.With(middleware.RequirePermission(h.db, "safety", "delete")).Delete("/safety-inspections/{id}", h.deleteSafetyInspection)
	r.With(middleware.RequirePermission(h.db, "safety", "edit")).Post("/safety-inspections/{id}/complete", h.completeSafetyInspection)
	r.With(middleware.RequirePermission(h.db, "safety", "delete")).Post("/safety-inspections/{id}/void", h.voidSafetyInspection)
}

// safetyInspectionQuery 构造安全检查查询条件：仅按登录态租户隔离（无部门快照）。
func safetyInspectionQuery(ctx context.Context, db *gorm.DB, userID uint) *gorm.DB {
	return db.Where("user_id = ?", userID)
}

// loadSafetyInspection 按租户隔离加载单条安全检查记录。
func (h *Handler) loadSafetyInspection(w http.ResponseWriter, r *http.Request, userID uint) (*models.SafetyInspection, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的安全检查 ID", err)
		return nil, false
	}
	var record models.SafetyInspection
	err = safetyInspectionQuery(r.Context(), h.db.Where("id = ?", id), userID).First(&record).Error
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "未找到安全检查记录", err)
		return nil, false
	}
	return &record, true
}

// validateSafetyInspectionPayload 校验创建/更新安全检查必填字段。
func validateSafetyInspectionPayload(p *safetyInspectionPayload) error {
	record := models.SafetyInspection{
		InspectionType:           p.InspectionType,
		InspectionDate:           p.InspectionDate,
		Location:                 p.Location,
		ResponsiblePerson:        p.ResponsiblePerson,
		IssueDescription:         p.IssueDescription,
		RectificationRequirement: p.RectificationRequirement,
		Status:                   models.SafetyInspectionStatusDraft,
	}
	return record.Validate()
}

// buildSafetyInspectionFromPayload 组装安全检查记录（初始草稿态）。
func buildSafetyInspectionFromPayload(userID uint, p *safetyInspectionPayload) *models.SafetyInspection {
	return &models.SafetyInspection{
		UserID:                   userID,
		InspectionType:           p.InspectionType,
		InspectionDate:           strings.TrimSpace(p.InspectionDate),
		Location:                 strings.TrimSpace(p.Location),
		ResponsiblePerson:        strings.TrimSpace(p.ResponsiblePerson),
		IssueDescription:         strings.TrimSpace(p.IssueDescription),
		RectificationRequirement: strings.TrimSpace(p.RectificationRequirement),
		Status:                   models.SafetyInspectionStatusDraft,
	}
}

// listSafetyInspections 列表查询安全检查记录（safety.view）：支持 status / inspection_type 过滤，按检查日期倒序。
func (h *Handler) listSafetyInspections(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := safetyInspectionQuery(r.Context(), h.db, userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidSafetyInspectionStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的检查状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	if kind := strings.TrimSpace(r.URL.Query().Get("inspection_type")); kind != "" {
		if !models.IsValidSafetyInspectionType(kind) {
			respondError(w, http.StatusBadRequest, "无效的检查类型", nil)
			return
		}
		query = query.Where("inspection_type = ?", kind)
	}
	var records []models.SafetyInspection
	if err := query.Order("inspection_date DESC, created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询安全检查记录失败", err)
		return
	}
	if records == nil {
		records = []models.SafetyInspection{}
	}
	respondJSON(w, http.StatusOK, records)
}

// getSafetyInspection 查询单条安全检查记录详情（safety.view）。
func (h *Handler) getSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadSafetyInspection(w, r, userID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// createSafetyInspection 创建安全检查记录草稿（safety.create）。
func (h *Handler) createSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload safetyInspectionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateSafetyInspectionPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	record := buildSafetyInspectionFromPayload(userID, &payload)
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建安全检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

// respondSafetyInspection 重新加载并返回单条安全检查记录。
func (h *Handler) respondSafetyInspection(w http.ResponseWriter, id uint) {
	var record models.SafetyInspection
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重新加载安全检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}
