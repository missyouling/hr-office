package api

import (
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

// onboardingPayload 创建/更新待入职记录请求体。
// Offer 字段仅持久化及内部服务契约，不加外部 HTTP 同步。
type onboardingPayload struct {
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Department      string `json:"department"`
	Position        string `json:"position"`
	PlannedHireDate string `json:"planned_hire_date"`
	IDNumber        string `json:"id_number"`
	Remarks         string `json:"remarks"`
	OfferID         string `json:"offer_id"`
	OfferSource     string `json:"offer_source"`
}

// onboardingConfirmPayload 确认入职请求体（用工状态空默认 trial）
type onboardingConfirmPayload struct {
	EmploymentStatus string `json:"employment_status"`
}

// onboardingAbandonPayload 放弃入职请求体（原因/备注必填）
type onboardingAbandonPayload struct {
	Reason  string `json:"reason"`
	Remarks string `json:"remarks"`
}

// onboardingQuickPayload 快速入职请求体（创建+确认一步到位，跳过 pending）
type onboardingQuickPayload struct {
	onboardingPayload
	EmploymentStatus string `json:"employment_status"`
}

// registerOnboardingRoutes 注册入职管理路由（P12.3.2.2）。
// 权限：列表 employee.view；创建/快速入职 employee.create；编辑/确认/放弃/恢复 employee.edit。
func (h *Handler) registerOnboardingRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "employee", "view")).Get("/onboarding-records", h.listOnboardingRecords)
	r.With(middleware.RequirePermission(h.db, "employee", "create")).Post("/onboarding-records", h.createOnboardingRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "create")).Post("/onboarding-records/quick", h.quickOnboardingRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Put("/onboarding-records/{id}", h.updateOnboardingRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/onboarding-records/{id}/confirm", h.confirmOnboardingRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/onboarding-records/{id}/abandon", h.abandonOnboardingRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/onboarding-records/{id}/restore", h.restoreOnboardingRecord)
	// 入职导入（P12.3.2.3）：独立端点与独立模板，不改变旧 employees 导入
	h.registerOnboardingImportRoutes(r)
}

// validateOnboardingPayload 校验创建/更新待入职记录必填字段。
func validateOnboardingPayload(p *onboardingPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("姓名必填")
	}
	if strings.TrimSpace(p.PlannedHireDate) == "" {
		return errors.New("计划入职日期必填")
	}
	if strings.TrimSpace(p.IDNumber) == "" {
		return errors.New("身份证号必填")
	}
	return nil
}

// checkOnboardingIDNumberConflict 检查身份证号与现有员工冲突。
// 对当前租户员工全量（active/resigned）统一拒绝，禁止自动返聘/覆盖/恢复。
func checkOnboardingIDNumberConflict(db *gorm.DB, userID uint, idNumber string) error {
	if strings.TrimSpace(idNumber) == "" {
		return nil
	}
	var count int64
	if err := db.Model(&models.Employee{}).Where("user_id = ? AND id_number = ?", userID, idNumber).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该身份证号已存在员工记录，无法重复入职")
	}
	return nil
}

// loadOnboardingRecord 按用户/部门隔离加载单条入职记录。
func (h *Handler) loadOnboardingRecord(w http.ResponseWriter, r *http.Request, userID uint) (*models.OnboardingRecord, bool) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "onboarding record id is required", nil)
		return nil, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid onboarding record id", err)
		return nil, false
	}
	query := h.db.Where("id = ? AND user_id = ?", id, userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("department = ?", dept)
	}
	var record models.OnboardingRecord
	if err := query.First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load onboarding record", err)
		return nil, false
	}
	return &record, true
}

func (h *Handler) listOnboardingRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("department = ?", dept)
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidOnboardingStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的入职状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var records []models.OnboardingRecord
	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list onboarding records", err)
		return
	}
	if records == nil {
		records = []models.OnboardingRecord{}
	}
	respondJSON(w, http.StatusOK, records)
}

func (h *Handler) createOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload onboardingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateOnboardingPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := checkOnboardingIDNumberConflict(h.db, userID, payload.IDNumber); err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	record := models.OnboardingRecord{
		UserID:          userID,
		Name:            strings.TrimSpace(payload.Name),
		IDNumber:        strings.TrimSpace(payload.IDNumber),
		Phone:           strings.TrimSpace(payload.Phone),
		Department:      strings.TrimSpace(payload.Department),
		Position:        strings.TrimSpace(payload.Position),
		PlannedHireDate: strings.TrimSpace(payload.PlannedHireDate),
		Remarks:         strings.TrimSpace(payload.Remarks),
		Status:          models.OnboardingStatusPending,
		OfferID:         strings.TrimSpace(payload.OfferID),
		OfferSource:     strings.TrimSpace(payload.OfferSource),
	}
	if err := h.db.Create(&record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create onboarding record", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

func (h *Handler) updateOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOnboardingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OnboardingStatusPending {
		respondError(w, http.StatusConflict, "仅待入职记录可编辑", nil)
		return
	}
	var payload onboardingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateOnboardingPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := checkOnboardingIDNumberConflict(h.db, userID, payload.IDNumber); err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	updates := map[string]interface{}{
		"name":              strings.TrimSpace(payload.Name),
		"id_number":         strings.TrimSpace(payload.IDNumber),
		"phone":             strings.TrimSpace(payload.Phone),
		"department":        strings.TrimSpace(payload.Department),
		"position":          strings.TrimSpace(payload.Position),
		"planned_hire_date": strings.TrimSpace(payload.PlannedHireDate),
		"remarks":           strings.TrimSpace(payload.Remarks),
		"offer_id":          strings.TrimSpace(payload.OfferID),
		"offer_source":      strings.TrimSpace(payload.OfferSource),
	}
	if err := h.db.Model(&models.OnboardingRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update onboarding record", err)
		return
	}
	var updated models.OnboardingRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload onboarding record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) confirmOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOnboardingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OnboardingStatusPending {
		respondError(w, http.StatusConflict, "仅待入职记录可确认入职", nil)
		return
	}
	var payload onboardingConfirmPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	employmentStatus := strings.TrimSpace(payload.EmploymentStatus)
	if employmentStatus == "" {
		employmentStatus = models.EmploymentStatusTrial
	}
	if !models.IsValidEmploymentStatus(employmentStatus) {
		respondError(w, http.StatusBadRequest, "无效的用工状态", nil)
		return
	}
	if err := checkOnboardingIDNumberConflict(h.db, userID, record.IDNumber); err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	updated, err := h.confirmOnboardingTx(userID, record, employmentStatus)
	if err != nil {
		if errors.Is(err, ErrOnboardingEmployeeConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		if errors.Is(err, ErrOnboardingDeptCodeMissing) || errors.Is(err, ErrOnboardingAdminMissing) {
			respondError(w, http.StatusUnprocessableEntity, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "确认入职失败", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// confirmOnboardingTx 事务内调用单一入职生效服务（创建员工 + 工号 + 关联 + 待办）。
func (h *Handler) confirmOnboardingTx(userID uint, record *models.OnboardingRecord, employmentStatus string) (*models.OnboardingRecord, error) {
	today := time.Now().Format("2006-01-02")
	var updated models.OnboardingRecord
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.applyOnboardingEffect(tx, userID, record, employmentStatus, today); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (h *Handler) abandonOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOnboardingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OnboardingStatusPending {
		respondError(w, http.StatusConflict, "仅待入职记录可放弃", nil)
		return
	}
	var payload onboardingAbandonPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	remarks := strings.TrimSpace(payload.Remarks)
	if reason == "" || remarks == "" {
		respondError(w, http.StatusBadRequest, "放弃原因和备注必填", nil)
		return
	}
	now := time.Now()
	if err := h.db.Model(&models.OnboardingRecord{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
		"status":         models.OnboardingStatusAbandoned,
		"abandon_reason": reason,
		"abandoned_at":   now,
		"remarks":        remarks,
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "放弃入职失败", err)
		return
	}
	var updated models.OnboardingRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload onboarding record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) restoreOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadOnboardingRecord(w, r, userID)
	if !ok {
		return
	}
	if record.Status != models.OnboardingStatusAbandoned {
		respondError(w, http.StatusConflict, "仅已放弃记录可恢复", nil)
		return
	}
	// 恢复为待入职：保留原计划日期与放弃历史（原因/时间/备注不清空）
	if err := h.db.Model(&models.OnboardingRecord{}).Where("id = ?", record.ID).Update("status", models.OnboardingStatusPending).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "恢复入职记录失败", err)
		return
	}
	var updated models.OnboardingRecord
	if err := h.db.First(&updated, record.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload onboarding record", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

func (h *Handler) quickOnboardingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload onboardingQuickPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateOnboardingPayload(&payload.onboardingPayload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	employmentStatus := strings.TrimSpace(payload.EmploymentStatus)
	if employmentStatus == "" {
		employmentStatus = models.EmploymentStatusTrial
	}
	if !models.IsValidEmploymentStatus(employmentStatus) {
		respondError(w, http.StatusBadRequest, "无效的用工状态", nil)
		return
	}
	if err := checkOnboardingIDNumberConflict(h.db, userID, payload.IDNumber); err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	record := models.OnboardingRecord{
		UserID:          userID,
		Name:            strings.TrimSpace(payload.Name),
		IDNumber:        strings.TrimSpace(payload.IDNumber),
		Phone:           strings.TrimSpace(payload.Phone),
		Department:      strings.TrimSpace(payload.Department),
		Position:        strings.TrimSpace(payload.Position),
		PlannedHireDate: strings.TrimSpace(payload.PlannedHireDate),
		Remarks:         strings.TrimSpace(payload.Remarks),
		Status:          models.OnboardingStatusPending,
		OfferID:         strings.TrimSpace(payload.OfferID),
		OfferSource:     strings.TrimSpace(payload.OfferSource),
	}
	updated, err := h.quickOnboardingTx(userID, &record, employmentStatus)
	if err != nil {
		if errors.Is(err, ErrOnboardingEmployeeConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		if errors.Is(err, ErrOnboardingDeptCodeMissing) || errors.Is(err, ErrOnboardingAdminMissing) {
			respondError(w, http.StatusUnprocessableEntity, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "快速入职失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, updated)
}

// quickOnboardingTx 事务内调用单一入职生效服务（创建记录 + 员工 + 工号 + 关联 + 待办）。
func (h *Handler) quickOnboardingTx(userID uint, record *models.OnboardingRecord, employmentStatus string) (*models.OnboardingRecord, error) {
	today := time.Now().Format("2006-01-02")
	var updated models.OnboardingRecord
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.applyOnboardingEffect(tx, userID, record, employmentStatus, today); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
