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

type trainingPayload struct {
	Topic                string `json:"topic"`
	TrainingType         string `json:"training_type"`
	TrainingDate         string `json:"training_date"`
	TrainerOrInstitution string `json:"trainer_or_institution"`
	EmployeeID           *uint  `json:"employee_id"`
	Result               string `json:"result"`
	Remarks              string `json:"remarks"`
}
type trainingVoidPayload struct {
	Reason string `json:"reason"`
}

func (h *Handler) registerTrainingRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "training", "view")).Get("/training-records", h.listTrainingRecords)
	r.With(middleware.RequirePermission(h.db, "training", "create")).Post("/training-records", h.createTrainingRecord)
	r.With(middleware.RequirePermission(h.db, "training", "view")).Get("/training-records/{id}", h.getTrainingRecord)
	r.With(middleware.RequirePermission(h.db, "training", "edit")).Put("/training-records/{id}", h.updateTrainingRecord)
	r.With(middleware.RequirePermission(h.db, "training", "delete")).Delete("/training-records/{id}", h.deleteTrainingRecord)
	r.With(middleware.RequirePermission(h.db, "training", "edit")).Post("/training-records/{id}/complete", h.completeTrainingRecord)
	r.With(middleware.RequirePermission(h.db, "training", "delete")).Post("/training-records/{id}/void", h.voidTrainingRecord)
}

func trainingQuery(ctx context.Context, db *gorm.DB, userID uint) *gorm.DB {
	query := db.Where("user_id = ?", userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(ctx); ok && dept != "" {
		query = query.Where("snapshot_department = ?", dept)
	}
	return query
}

func (h *Handler) loadTrainingRecord(w http.ResponseWriter, r *http.Request, userID uint) (*models.TrainingRecord, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的培训记录 ID", err)
		return nil, false
	}
	var record models.TrainingRecord
	err = trainingQuery(r.Context(), h.db.Where("id = ?", id), userID).First(&record).Error
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "未找到培训记录", err)
		return nil, false
	}
	return &record, true
}

func validateTrainingPayload(p *trainingPayload) error {
	record := models.TrainingRecord{Topic: p.Topic, TrainingType: p.TrainingType, TrainingDate: p.TrainingDate, Status: models.TrainingStatusDraft}
	return record.Validate()
}

func (h *Handler) buildTrainingRecord(userID uint, p *trainingPayload) (*models.TrainingRecord, error) {
	record := &models.TrainingRecord{UserID: userID, EmployeeID: p.EmployeeID, Topic: strings.TrimSpace(p.Topic), TrainingType: p.TrainingType, TrainingDate: strings.TrimSpace(p.TrainingDate), TrainerOrInstitution: strings.TrimSpace(p.TrainerOrInstitution), Result: strings.TrimSpace(p.Result), Remarks: strings.TrimSpace(p.Remarks), Status: models.TrainingStatusDraft}
	if p.EmployeeID == nil {
		return record, nil
	}
	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", *p.EmployeeID, userID).First(&employee).Error; err != nil {
		return nil, errors.New("关联员工不存在或不属于当前租户")
	}
	record.SnapshotName, record.SnapshotDepartment, record.SnapshotPosition = employee.Name, employee.Department, employee.Position
	return record, nil
}

func (h *Handler) listTrainingRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := trainingQuery(r.Context(), h.db, userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidTrainingStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的培训状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	if kind := strings.TrimSpace(r.URL.Query().Get("training_type")); kind != "" {
		if !models.IsValidTrainingType(kind) {
			respondError(w, http.StatusBadRequest, "无效的培训类型", nil)
			return
		}
		query = query.Where("training_type = ?", kind)
	}
	var records []models.TrainingRecord
	if err := query.Order("training_date DESC, created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询培训记录失败", err)
		return
	}
	if records == nil {
		records = []models.TrainingRecord{}
	}
	respondJSON(w, http.StatusOK, records)
}

func (h *Handler) getTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadTrainingRecord(w, r, userID)
	if ok {
		respondJSON(w, http.StatusOK, record)
	}
}

func (h *Handler) createTrainingRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
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
	record, err := h.buildTrainingRecord(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建培训记录失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

func (h *Handler) respondTrainingRecord(w http.ResponseWriter, id uint) {
	var record models.TrainingRecord
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重新加载培训记录失败", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}
