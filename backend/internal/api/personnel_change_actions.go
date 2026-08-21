package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

func (h *Handler) updatePersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadPersonnelChange(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.PersonnelChangeStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可编辑", nil)
		return
	}
	var payload personnelChangePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validatePersonnelChangePayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, err := h.buildPersonnelChange(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updates := map[string]any{"employee_id": updated.EmployeeID, "change_type": updated.ChangeType, "effective_date": updated.EffectiveDate, "reason": updated.Reason, "before_department": updated.BeforeDepartment, "before_position": updated.BeforePosition, "before_job_level": updated.BeforeJobLevel, "after_department_id": updated.AfterDepartmentID, "after_department": updated.AfterDepartment, "after_position": updated.AfterPosition, "after_job_level": updated.AfterJobLevel}
	if err := h.db.Model(record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新人事异动记录失败", err)
		return
	}
	h.respondPersonnelChange(w, record.ID, http.StatusOK)
}

func (h *Handler) deletePersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadPersonnelChange(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.PersonnelChangeStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可删除", nil)
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", record.ID, userID).Delete(&models.PersonnelChange{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除人事异动记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": record.ID})
}

func personnelChangeEmployeeMatches(record models.PersonnelChange, employee models.Employee) bool {
	return employee.Department == record.BeforeDepartment && employee.Position == record.BeforePosition && employee.JobLevel == record.BeforeJobLevel
}

func (h *Handler) activatePersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadPersonnelChange(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.PersonnelChangeStatusDraft {
		respondError(w, http.StatusConflict, "仅草稿记录可生效", nil)
		return
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var fresh models.PersonnelChange
		if err := tx.Where("id = ? AND user_id = ?", record.ID, userID).First(&fresh).Error; err != nil {
			return err
		}
		var employee models.Employee
		if err := tx.Where("id = ? AND user_id = ?", fresh.EmployeeID, userID).First(&employee).Error; err != nil {
			return err
		}
		if fresh.Status != models.PersonnelChangeStatusDraft || employee.Status != models.EmployeeStatusActive || !personnelChangeEmployeeMatches(fresh, employee) {
			return errors.New("员工当前信息已变化，无法生效人事异动")
		}
		now := time.Now()
		if err := tx.Model(&employee).Updates(map[string]any{"department": fresh.AfterDepartment, "position": fresh.AfterPosition, "job_level": fresh.AfterJobLevel}).Error; err != nil {
			return err
		}
		return tx.Model(&fresh).Updates(map[string]any{"status": models.PersonnelChangeStatusEffective, "effective_at": now, "effective_by": userID}).Error
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "无法生效") {
			status = http.StatusConflict
		}
		respondError(w, status, "人事异动生效失败", err)
		return
	}
	h.respondPersonnelChange(w, record.ID, http.StatusOK)
}

func (h *Handler) voidPersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadPersonnelChange(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.PersonnelChangeStatusDraft && record.Status != models.PersonnelChangeStatusEffective {
		respondError(w, http.StatusConflict, "仅草稿或生效记录可作废", nil)
		return
	}
	var payload personnelChangeVoidPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if reason := strings.TrimSpace(payload.Reason); reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	} else if err := h.db.Model(record).Updates(map[string]any{"status": models.PersonnelChangeStatusVoided, "void_reason": reason, "voided_at": time.Now()}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "作废人事异动记录失败", err)
		return
	}
	h.respondPersonnelChange(w, record.ID, http.StatusOK)
}

func (h *Handler) respondPersonnelChange(w http.ResponseWriter, id uint, status int) {
	var record models.PersonnelChange
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重新加载人事异动记录失败", err)
		return
	}
	respondJSON(w, status, record)
}
