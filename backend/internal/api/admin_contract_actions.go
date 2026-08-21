package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// adminContractExpiringDefaultDays 到期提醒查询 days 参数缺省值（天）。
const adminContractExpiringDefaultDays = 30

// adminContractExpiringMaxDays 到期提醒查询 days 参数合法范围上限（天）。
const adminContractExpiringMaxDays = 365

// updateAdminContract 编辑行政合同草稿（仅 draft 可编辑；active/expired/cancelled 不可编辑）。
func (h *Handler) updateAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadAdminContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.AdminContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可编辑，生效/到期/作废合同不可修改", nil)
		return
	}
	var payload adminContractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateAdminContractPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.checkAdminContractDocument(userID, payload.DocumentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updates := map[string]any{
		"contract_no":     strings.TrimSpace(payload.ContractNo),
		"name":            strings.TrimSpace(payload.Name),
		"counterparty":    strings.TrimSpace(payload.Counterparty),
		"contract_type":   strings.TrimSpace(payload.ContractType),
		"start_date":      strings.TrimSpace(payload.StartDate),
		"end_date":        strings.TrimSpace(payload.EndDate),
		"amount_incl_tax": payload.AmountInclTax,
		"currency":        strings.TrimSpace(payload.Currency),
		"owner":           strings.TrimSpace(payload.Owner),
		"remarks":         strings.TrimSpace(payload.Remarks),
		"document_id":     payload.DocumentID,
	}
	if err := h.db.Model(&models.AdminContract{}).Where("id = ?", contract.ID).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update admin contract", err)
		return
	}
	var updated models.AdminContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload admin contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// deleteAdminContract 删除行政合同草稿（仅 draft 可删除；active/expired/cancelled 不可删除）。
func (h *Handler) deleteAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadAdminContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.AdminContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可删除，生效/到期/作废合同不可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", contract.ID, userID).Delete(&models.AdminContract{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete admin contract", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": contract.ID})
}

// activateAdminContract 手动生效行政合同（admin_contract.edit 权限用户触发）。
// 仅 draft → active；合同起始日不自动生效，不校验日期是否已到。
func (h *Handler) activateAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadAdminContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.AdminContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可生效", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(&models.AdminContract{}).Where("id = ?", contract.ID).Updates(map[string]any{
		"status":       models.AdminContractStatusActive,
		"activated_at": now,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to activate admin contract", err)
		return
	}
	var updated models.AdminContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload admin contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// cancelAdminContract 作废行政合同（draft/active → cancelled；expired 为终态不可作废）。
// 作废原因必填；作废后需新建替代合同。
func (h *Handler) cancelAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadAdminContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.AdminContractStatusDraft && contract.Status != models.AdminContractStatusActive {
		respondError(w, http.StatusConflict, "仅草稿或生效中合同可作废", nil)
		return
	}
	var payload adminContractCancelPayload
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
	if err := h.db.Model(&models.AdminContract{}).Where("id = ?", contract.ID).Updates(map[string]any{
		"status":        models.AdminContractStatusCancelled,
		"cancelled_at":  now,
		"cancel_reason": reason,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to cancel admin contract", err)
		return
	}
	var updated models.AdminContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload admin contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// markExpiredAdminContracts 惰性标记当前租户已到期行政合同（幂等）：
// active 且 end_date < 今日 → expired，记录到期标记时间；不联动其他模块。
func (h *Handler) markExpiredAdminContracts(userID uint) error {
	today := time.Now().Format("2006-01-02")
	return h.db.Model(&models.AdminContract{}).
		Where("user_id = ? AND status = ? AND end_date < ?", userID, models.AdminContractStatusActive, today).
		Updates(map[string]any{
			"status":     models.AdminContractStatusExpired,
			"expired_at": time.Now(),
		}).Error
}

// expireAdminContracts 手动触发到期标记（admin_contract.edit 权限）。
// 返回本次标记为 expired 的合同数量。
func (h *Handler) expireAdminContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if err := h.markExpiredAdminContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired admin contracts", err)
		return
	}
	var count int64
	if err := h.db.Model(&models.AdminContract{}).
		Where("user_id = ? AND status = ?", userID, models.AdminContractStatusExpired).Count(&count).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count expired admin contracts", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"expired": count})
}

// listExpiringAdminContracts 到期提醒查询（admin_contract.view）：
// 返回当前租户 active 且未来 days 天内到期的行政合同，按到期日升序。
func (h *Handler) listExpiringAdminContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	days, err := parseAdminContractExpiringDays(r.URL.Query().Get("days"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// 先惰性标记已到期合同，避免过期合同混入提醒
	if err := h.markExpiredAdminContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired admin contracts", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	query := h.db.Where("user_id = ?", userID).
		Where("status = ? AND end_date >= ? AND end_date <= ?", models.AdminContractStatusActive, today, end)
	var contracts []models.AdminContract
	if err := query.Order("end_date ASC").Find(&contracts).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list expiring admin contracts", err)
		return
	}
	if contracts == nil {
		contracts = []models.AdminContract{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"days": days, "contracts": contracts})
}

// parseAdminContractExpiringDays 解析 days 查询参数：缺省 30，非法（非数字/越界）报错。
func parseAdminContractExpiringDays(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return adminContractExpiringDefaultDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > adminContractExpiringMaxDays {
		return 0, errors.New("invalid days: must be an integer between 1 and 365")
	}
	return days, nil
}
