package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

func (h *Handler) updateTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadTrainingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.TrainingStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可编辑", nil)
		return
	}
	var payload trainingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateTrainingPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.buildTrainingRecord(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updates := map[string]any{"employee_id": updated.EmployeeID, "snapshot_name": updated.SnapshotName, "snapshot_department": updated.SnapshotDepartment, "snapshot_position": updated.SnapshotPosition, "topic": updated.Topic, "training_type": updated.TrainingType, "training_date": updated.TrainingDate, "trainer_or_institution": updated.TrainerOrInstitution, "result": updated.Result, "remarks": updated.Remarks}
	if err := h.db.Model(record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新培训记录失败", err)
		return
	}
	h.respondTrainingRecord(w, record.ID)
}

func (h *Handler) deleteTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadTrainingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.TrainingStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", record.ID, userID).Delete(&models.TrainingRecord{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除培训记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": record.ID})
}

func (h *Handler) completeTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadTrainingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.TrainingStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可完成", nil)
		return
	}
	if err := h.db.Model(record).Updates(map[string]any{"status": models.TrainingStatusCompleted, "completed_at": time.Now()}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "完成培训记录失败", err)
		return
	}
	h.respondTrainingRecord(w, record.ID)
}

func (h *Handler) voidTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadTrainingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.TrainingStatusDraft && record.Status != models.TrainingStatusCompleted {
		respondError(w, http.StatusConflict, "仅草稿或已完成记录可作废", nil)
		return
	}
	var payload trainingVoidPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	if err := h.db.Model(record).Updates(map[string]any{"status": models.TrainingStatusVoided, "void_reason": reason, "voided_at": time.Now()}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "作废培训记录失败", err)
		return
	}
	h.respondTrainingRecord(w, record.ID)
}
