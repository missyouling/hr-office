package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// updateReward 编辑奖惩记录草稿（仅 draft 可编辑；effective/voided 不可编辑）。
func (h *Handler) updateReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadReward(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.RewardStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可编辑，生效/作废记录不可修改", nil)
		return
	}
	var payload rewardPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateRewardPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.checkRewardDocument(userID, payload.DocumentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// 编辑时允许调整关联员工并刷新快照（仅草稿阶段）
	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", payload.EmployeeID, userID).First(&employee).Error; err != nil {
		respondError(w, http.StatusBadRequest, "关联的员工不存在或不属于当前租户", nil)
		return
	}
	updates := map[string]any{
		"employee_id":         payload.EmployeeID,
		"snapshot_name":       employee.Name,
		"snapshot_department": employee.Department,
		"snapshot_position":   employee.Position,
		"record_type":         payload.RecordType,
		"occurred_date":       strings.TrimSpace(payload.OccurredDate),
		"reason":              strings.TrimSpace(payload.Reason),
		"level":               strings.TrimSpace(payload.Level),
		"score":               payload.Score,
		"amount":              payload.Amount,
		"owner":               strings.TrimSpace(payload.Owner),
		"document_id":         payload.DocumentID,
		"remarks":             strings.TrimSpace(payload.Remarks),
	}
	if err := h.db.Model(&models.RewardRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update reward record", err)
		return
	}
	var updated models.RewardRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload reward record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// deleteReward 删除奖惩记录草稿（仅 draft 可删除；effective/voided 不可删除）。
func (h *Handler) deleteReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadReward(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.RewardStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可删除，生效/作废记录不可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", record.ID, userID).Delete(&models.RewardRecord{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete reward record", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": record.ID})
}

// activateReward 手动生效奖惩记录（reward.edit 权限用户触发）。
// 仅 draft → effective；不改变员工状态或薪资。
func (h *Handler) activateReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadReward(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.RewardStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可生效", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(&models.RewardRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
		"status":       models.RewardStatusEffective,
		"effective_at": now,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to activate reward record", err)
		return
	}
	var updated models.RewardRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload reward record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// voidReward 作废奖惩记录（draft/effective → voided；voided 为终态不可再作废）。
// 作废原因必填；不改变员工状态或薪资。
func (h *Handler) voidReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadReward(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.RewardStatusDraft && record.Status != models.RewardStatusEffective {
		respondError(w, http.StatusConflict, "仅草稿或生效中记录可作废", nil)
		return
	}
	var payload rewardVoidPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(&models.RewardRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
		"status":      models.RewardStatusVoided,
		"voided_at":   now,
		"void_reason": reason,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to void reward record", err)
		return
	}
	var updated models.RewardRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload reward record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}
