package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// updateSafetyInspection 编辑安全检查记录草稿（仅 draft 可编辑；completed/voided 不可编辑）。
func (h *Handler) updateSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadSafetyInspection(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.SafetyInspectionStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可编辑，已完成/作废记录不可修改", nil)
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
	updated := buildSafetyInspectionFromPayload(userID, &payload)
	updates := map[string]any{
		"inspection_type":           updated.InspectionType,
		"inspection_date":           updated.InspectionDate,
		"location":                  updated.Location,
		"responsible_person":        updated.ResponsiblePerson,
		"issue_description":         updated.IssueDescription,
		"rectification_requirement": updated.RectificationRequirement,
	}
	if err := h.db.Model(record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新安全检查记录失败", err)
		return
	}
	h.respondSafetyInspection(w, record.ID)
}

// deleteSafetyInspection 删除安全检查记录草稿（仅 draft 可删除；completed/voided 不可删除）。
func (h *Handler) deleteSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadSafetyInspection(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.SafetyInspectionStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可删除，已完成/作废记录不可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", record.ID, userID).Delete(&models.SafetyInspection{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除安全检查记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": record.ID})
}

// completeSafetyInspection 手动完成安全检查记录（safety.edit 权限用户触发）。
// 仅 draft → completed；不联动任何其他业务。
func (h *Handler) completeSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadSafetyInspection(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.SafetyInspectionStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可完成", nil)
		return
	}
	if err := h.db.Model(record).Updates(map[string]any{
		"status":       models.SafetyInspectionStatusCompleted,
		"completed_at": time.Now(),
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "完成安全检查记录失败", err)
		return
	}
	h.respondSafetyInspection(w, record.ID)
}

// voidSafetyInspection 作废安全检查记录（draft/completed → voided；voided 为终态不可再作废）。
// 作废原因必填；不联动任何其他业务。
func (h *Handler) voidSafetyInspection(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadSafetyInspection(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.SafetyInspectionStatusDraft && record.Status != models.SafetyInspectionStatusCompleted {
		respondError(w, http.StatusConflict, "仅草稿或已完成记录可作废", nil)
		return
	}
	var payload safetyInspectionVoidPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	if err := h.db.Model(record).Updates(map[string]any{
		"status":      models.SafetyInspectionStatusVoided,
		"void_reason": reason,
		"voided_at":   time.Now(),
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "作废安全检查记录失败", err)
		return
	}
	h.respondSafetyInspection(w, record.ID)
}
