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

type personnelChangePayload struct {
	EmployeeID        uint    `json:"employee_id"`
	ChangeType        string  `json:"change_type"`
	EffectiveDate     string  `json:"effective_date"`
	Reason            string  `json:"reason"`
	AfterDepartmentID *uint   `json:"after_department_id"`
	AfterPosition     *string `json:"after_position"`
	AfterJobLevel     *string `json:"after_job_level"`
}

type personnelChangeVoidPayload struct {
	Reason string `json:"reason"`
}

func (h *Handler) registerPersonnelChangeRoutes(r chi.Router) {
	guard := middleware.RequirePermission(h.db, "employee", "edit")
	r.With(guard).Get("/personnel-changes", h.listPersonnelChanges)
	r.With(guard).Post("/personnel-changes", h.createPersonnelChange)
	r.With(guard).Get("/personnel-changes/{id}", h.getPersonnelChange)
	r.With(guard).Put("/personnel-changes/{id}", h.updatePersonnelChange)
	r.With(guard).Delete("/personnel-changes/{id}", h.deletePersonnelChange)
	r.With(guard).Post("/personnel-changes/{id}/activate", h.activatePersonnelChange)
	r.With(guard).Post("/personnel-changes/{id}/void", h.voidPersonnelChange)
}

func personnelChangeQuery(ctx context.Context, db *gorm.DB, userID uint) *gorm.DB {
	query := db.Where("user_id = ?", userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(ctx); ok && dept != "" {
		query = query.Where("before_department = ?", dept)
	}
	return query
}

func (h *Handler) loadPersonnelChange(w http.ResponseWriter, r *http.Request, userID uint) (*models.PersonnelChange, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的人事异动 ID", err)
		return nil, false
	}
	var record models.PersonnelChange
	err = personnelChangeQuery(r.Context(), h.db.Where("id = ?", id), userID).First(&record).Error
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "未找到人事异动记录", err)
		return nil, false
	}
	return &record, true
}

func validatePersonnelChangePayload(p *personnelChangePayload) error {
	if p.EmployeeID == 0 {
		return errors.New("关联员工必填")
	}
	if !models.IsValidPersonnelChangeType(p.ChangeType) {
		return errors.New("无效的异动类型")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.EffectiveDate)); err != nil {
		return errors.New("生效日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(p.Reason) == "" {
		return errors.New("异动原因必填")
	}
	return nil
}

func (h *Handler) resolvePersonnelChangeTarget(userID uint, employee models.Employee, p *personnelChangePayload) (*uint, string, string, string, error) {
	departmentID, department := (*uint)(nil), employee.Department
	if p.AfterDepartmentID != nil {
		var target models.Department
		if err := h.db.Where("id = ? AND user_id = ?", *p.AfterDepartmentID, userID).First(&target).Error; err != nil {
			return nil, "", "", "", errors.New("异动后部门不存在或不属于当前租户")
		}
		departmentID, department = p.AfterDepartmentID, target.Name
	}
	position, level := employee.Position, employee.JobLevel
	if p.AfterPosition != nil {
		position = strings.TrimSpace(*p.AfterPosition)
	}
	if p.AfterJobLevel != nil {
		level = strings.TrimSpace(*p.AfterJobLevel)
	}
	if department == employee.Department && position == employee.Position && level == employee.JobLevel {
		return nil, "", "", "", errors.New("异动后部门、岗位或职级至少应有一项不同")
	}
	return departmentID, department, position, level, nil
}

func (h *Handler) buildPersonnelChange(userID uint, p *personnelChangePayload) (*models.PersonnelChange, error) {
	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", p.EmployeeID, userID).First(&employee).Error; err != nil {
		return nil, errors.New("关联员工不存在或不属于当前租户")
	}
	if employee.Status != models.EmployeeStatusActive {
		return nil, errors.New("仅在职员工可发起人事异动")
	}
	departmentID, department, position, level, err := h.resolvePersonnelChangeTarget(userID, employee, p)
	if err != nil {
		return nil, err
	}
	beforeDepartmentID := h.findPersonnelChangeDepartmentID(userID, employee.Department)
	return &models.PersonnelChange{UserID: userID, CreatedBy: userID, EmployeeID: employee.ID, ChangeType: p.ChangeType, EffectiveDate: strings.TrimSpace(p.EffectiveDate), Reason: strings.TrimSpace(p.Reason), BeforeDepartmentID: beforeDepartmentID, BeforeDepartment: employee.Department, BeforePosition: employee.Position, BeforeJobLevel: employee.JobLevel, AfterDepartmentID: departmentID, AfterDepartment: department, AfterPosition: position, AfterJobLevel: level, Status: models.PersonnelChangeStatusDraft}, nil
}

func (h *Handler) findPersonnelChangeDepartmentID(userID uint, name string) *uint {
	var department models.Department
	if h.db.Where("user_id = ? AND name = ?", userID, name).First(&department).Error != nil {
		return nil
	}
	return &department.ID
}

func (h *Handler) listPersonnelChanges(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := personnelChangeQuery(r.Context(), h.db, userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidPersonnelChangeStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的异动状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var records []models.PersonnelChange
	if err := query.Order("effective_date DESC, created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询人事异动记录失败", err)
		return
	}
	if records == nil {
		records = []models.PersonnelChange{}
	}
	respondJSON(w, http.StatusOK, records)
}

func (h *Handler) getPersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadPersonnelChange(w, r, userID)
	if ok {
		respondJSON(w, http.StatusOK, record)
	}
}

func (h *Handler) createPersonnelChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
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
	record, err := h.buildPersonnelChange(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建人事异动记录失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}
