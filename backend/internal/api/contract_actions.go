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

// contractExpiringDefaultDays 到期提醒查询 days 参数缺省值（天）。
const contractExpiringDefaultDays = 30

// contractExpiringMaxDays 到期提醒查询 days 参数合法范围上限（天）。
const contractExpiringMaxDays = 365

// updateContract 编辑劳动合同草稿（仅 draft 可编辑；active/expired/cancelled 不可编辑）。
func (h *Handler) updateContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.ContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可编辑，生效/到期/作废合同不可修改", nil)
		return
	}
	var payload contractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateContractPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.checkContractDocument(userID, payload.DocumentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updates := map[string]any{
		"contract_no": strings.TrimSpace(payload.ContractNo),
		"start_date":  strings.TrimSpace(payload.StartDate),
		"end_date":    strings.TrimSpace(payload.EndDate),
		"term_months": payload.TermMonths,
		"document_id": payload.DocumentID,
		"remarks":     strings.TrimSpace(payload.Remarks),
	}
	// 编辑时允许调整快照字段（仅草稿阶段）
	if payload.EmployeeID != nil {
		updates["employee_id"] = payload.EmployeeID
		var employee models.Employee
		if err := h.db.Where("id = ? AND user_id = ?", *payload.EmployeeID, userID).First(&employee).Error; err != nil {
			respondError(w, http.StatusBadRequest, "关联的员工不存在或不属于当前租户", nil)
			return
		}
		updates["snapshot_name"] = employee.Name
		updates["snapshot_department"] = employee.Department
		updates["snapshot_position"] = employee.Position
		updates["snapshot_id_number"] = employee.IDNumber
	} else {
		updates["employee_id"] = nil
		updates["snapshot_name"] = strings.TrimSpace(payload.Name)
		updates["snapshot_department"] = strings.TrimSpace(payload.Department)
		updates["snapshot_position"] = strings.TrimSpace(payload.Position)
		updates["snapshot_id_number"] = strings.TrimSpace(payload.IDNumber)
		if strings.TrimSpace(payload.Department) == "" {
			respondError(w, http.StatusBadRequest, "未关联员工时部门必填", nil)
			return
		}
	}
	if err := h.db.Model(&models.LaborContract{}).Where("id = ?", contract.ID).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update contract", err)
		return
	}
	var updated models.LaborContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// deleteContract 删除劳动合同草稿（仅 draft 可删除；active/expired/cancelled 不可删除）。
func (h *Handler) deleteContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.ContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可删除，生效/到期/作废合同不可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", contract.ID, userID).Delete(&models.LaborContract{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete contract", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": contract.ID})
}

// activateContract 手动生效劳动合同（contract.edit 权限用户触发）。
// 仅 draft → active；合同起始日不自动生效，不校验日期是否已到。
func (h *Handler) activateContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.ContractStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿合同可生效", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(&models.LaborContract{}).Where("id = ?", contract.ID).Updates(map[string]any{
		"status":       models.ContractStatusActive,
		"activated_at": now,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to activate contract", err)
		return
	}
	var updated models.LaborContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// cancelContract 作废劳动合同（draft/active → cancelled；expired 为终态不可作废）。
// 作废原因必填；作废后需新建合同。
func (h *Handler) cancelContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadContract(w, r, userID)
	if !ok {
		return
	}
	if contract.Status != models.ContractStatusDraft && contract.Status != models.ContractStatusActive {
		respondError(w, http.StatusConflict, "仅草稿或生效中合同可作废", nil)
		return
	}
	var payload contractCancelPayload
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
	if err := h.db.Model(&models.LaborContract{}).Where("id = ?", contract.ID).Updates(map[string]any{
		"status":        models.ContractStatusCancelled,
		"cancelled_at":  now,
		"cancel_reason": reason,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to cancel contract", err)
		return
	}
	var updated models.LaborContract
	if err := h.db.First(&updated, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload contract", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// markExpiredContracts 惰性标记当前租户已到期合同（幂等）：
// active 且 end_date < 今日 → expired，记录到期标记时间；不修改员工状态。
func (h *Handler) markExpiredContracts(userID uint) error {
	today := time.Now().Format("2006-01-02")
	return h.db.Model(&models.LaborContract{}).
		Where("user_id = ? AND status = ? AND end_date < ?", userID, models.ContractStatusActive, today).
		Updates(map[string]any{
			"status":     models.ContractStatusExpired,
			"expired_at": time.Now(),
		}).Error
}

// expireContracts 手动触发到期标记（contract.edit 权限）。
// 返回本次标记为 expired 的合同数量。
func (h *Handler) expireContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if err := h.markExpiredContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired contracts", err)
		return
	}
	var count int64
	if err := h.db.Model(&models.LaborContract{}).
		Where("user_id = ? AND status = ?", userID, models.ContractStatusExpired).Count(&count).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to count expired contracts", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"expired": count})
}

// listExpiringContracts 到期提醒查询（contract.view）：
// 返回当前租户 active 且未来 days 天内到期的合同，按到期日升序。
func (h *Handler) listExpiringContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	days, err := parseContractExpiringDays(r.URL.Query().Get("days"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// 先惰性标记已到期合同，避免过期合同混入提醒
	if err := h.markExpiredContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired contracts", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	query := applyContractDepartmentFilter(r.Context(), h.db.Where("user_id = ?", userID)).
		Where("status = ? AND end_date >= ? AND end_date <= ?", models.ContractStatusActive, today, end)
	var contracts []models.LaborContract
	if err := query.Order("end_date ASC").Find(&contracts).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list expiring contracts", err)
		return
	}
	if contracts == nil {
		contracts = []models.LaborContract{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"days": days, "contracts": contracts})
}

// parseContractExpiringDays 解析 days 查询参数：缺省 30，非法（非数字/越界）报错。
func parseContractExpiringDays(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return contractExpiringDefaultDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > contractExpiringMaxDays {
		return 0, errors.New("invalid days: must be an integer between 1 and 365")
	}
	return days, nil
}
