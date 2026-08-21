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

// rewardPayload 创建/更新奖惩记录请求体。
// 必填：employee_id/record_type/occurred_date/reason/level；选填：score/amount/owner/document_id/remarks。
type rewardPayload struct {
	EmployeeID   uint     `json:"employee_id"`
	RecordType   string   `json:"record_type"`
	OccurredDate string   `json:"occurred_date"`
	Reason       string   `json:"reason"`
	Level        string   `json:"level"`
	Score        *float64 `json:"score"`
	Amount       *float64 `json:"amount"`
	Owner        string   `json:"owner"`
	DocumentID   *uint    `json:"document_id"`
	Remarks      string   `json:"remarks"`
}

// rewardVoidPayload 作废奖惩记录请求体（原因必填）。
type rewardVoidPayload struct {
	Reason string `json:"reason"`
}

// registerRewardRoutes 注册奖惩记录路由（P12.3.6）。
// 权限：列表/详情 reward.view；创建 reward.create；
// 编辑/生效 reward.edit；删除/作废 reward.delete。
func (h *Handler) registerRewardRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "reward", "view")).Get("/rewards", h.listRewards)
	r.With(middleware.RequirePermission(h.db, "reward", "create")).Post("/rewards", h.createReward)
	r.With(middleware.RequirePermission(h.db, "reward", "view")).Get("/rewards/{id}", h.getReward)
	r.With(middleware.RequirePermission(h.db, "reward", "edit")).Put("/rewards/{id}", h.updateReward)
	r.With(middleware.RequirePermission(h.db, "reward", "delete")).Delete("/rewards/{id}", h.deleteReward)
	r.With(middleware.RequirePermission(h.db, "reward", "edit")).Post("/rewards/{id}/activate", h.activateReward)
	r.With(middleware.RequirePermission(h.db, "reward", "delete")).Post("/rewards/{id}/void", h.voidReward)
}

// applyRewardDepartmentFilter 根据当前用户所属部门过滤奖惩记录（按创建时部门快照）。
func applyRewardDepartmentFilter(ctx context.Context, db *gorm.DB) *gorm.DB {
	if dept, ok := middleware.GetUserDepartmentFromContext(ctx); ok && dept != "" {
		return db.Where("snapshot_department = ?", dept)
	}
	return db
}

// loadReward 按用户/部门隔离加载单条奖惩记录。
func (h *Handler) loadReward(w http.ResponseWriter, r *http.Request, userID uint) (*models.RewardRecord, bool) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "reward id is required", nil)
		return nil, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid reward id", err)
		return nil, false
	}
	query := h.db.Where("id = ? AND user_id = ?", id, userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("snapshot_department = ?", dept)
	}
	var record models.RewardRecord
	if err := query.First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load reward record", err)
		return nil, false
	}
	return &record, true
}

// validateRewardPayload 校验创建/更新奖惩记录必填字段。
func validateRewardPayload(p *rewardPayload) error {
	if p.EmployeeID == 0 {
		return errors.New("关联员工必填")
	}
	if !models.IsValidRewardType(p.RecordType) {
		return errors.New("记录类型必须为 reward 或 punishment")
	}
	if strings.TrimSpace(p.OccurredDate) == "" {
		return errors.New("发生日期必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.OccurredDate)); err != nil {
		return errors.New("发生日期格式必须为 YYYY-MM-DD")
	}
	if strings.TrimSpace(p.Reason) == "" {
		return errors.New("事由必填")
	}
	if strings.TrimSpace(p.Level) == "" {
		return errors.New("等级必填")
	}
	return nil
}

// checkRewardDocument 校验关联档案文档存在且属于当前租户。
func (h *Handler) checkRewardDocument(userID uint, documentID *uint) error {
	if documentID == nil {
		return nil
	}
	var count int64
	if err := h.db.Model(&models.Document{}).Where("id = ? AND user_id = ?", *documentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("关联的档案文档不存在或不属于当前租户")
	}
	return nil
}

// buildRewardFromPayload 组装奖惩记录（初始草稿态），并从员工主表拷贝快照。
func (h *Handler) buildRewardFromPayload(userID uint, p *rewardPayload) (*models.RewardRecord, error) {
	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", p.EmployeeID, userID).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("关联的员工不存在或不属于当前租户")
		}
		return nil, err
	}
	return &models.RewardRecord{
		UserID:             userID,
		EmployeeID:         p.EmployeeID,
		SnapshotName:       employee.Name,
		SnapshotDepartment: employee.Department,
		SnapshotPosition:   employee.Position,
		RecordType:         p.RecordType,
		OccurredDate:       strings.TrimSpace(p.OccurredDate),
		Reason:             strings.TrimSpace(p.Reason),
		Level:              strings.TrimSpace(p.Level),
		Score:              p.Score,
		Amount:             p.Amount,
		Owner:              strings.TrimSpace(p.Owner),
		DocumentID:         p.DocumentID,
		Remarks:            strings.TrimSpace(p.Remarks),
		Status:             models.RewardStatusDraft,
	}, nil
}

// listRewards 列表查询奖惩记录（reward.view）：支持 status / record_type 过滤，按发生日期倒序。
func (h *Handler) listRewards(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := applyRewardDepartmentFilter(r.Context(), h.db.Where("user_id = ?", userID))
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidRewardStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的记录状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	if recordType := strings.TrimSpace(r.URL.Query().Get("record_type")); recordType != "" {
		if !models.IsValidRewardType(recordType) {
			respondError(w, http.StatusBadRequest, "无效的记录类型", nil)
			return
		}
		query = query.Where("record_type = ?", recordType)
	}
	var records []models.RewardRecord
	if err := query.Order("occurred_date DESC, created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list reward records", err)
		return
	}
	if records == nil {
		records = []models.RewardRecord{}
	}
	respondJSON(w, http.StatusOK, records)
}

// getReward 查询单条奖惩记录详情（reward.view）。
func (h *Handler) getReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadReward(w, r, userID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// createReward 创建奖惩记录草稿（reward.create）。
func (h *Handler) createReward(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
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
	record, err := h.buildRewardFromPayload(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create reward record", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}
