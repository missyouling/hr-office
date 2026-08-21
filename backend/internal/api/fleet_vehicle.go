package api

import (
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

// fleetVehiclePayload 创建/更新车辆档案请求体。
// 必填：plate_number / vehicle_model / status；可选：brand / seat_count / purchase_date / remarks。
type fleetVehiclePayload struct {
	PlateNumber  string `json:"plate_number"`
	VehicleModel string `json:"vehicle_model"`
	Status       string `json:"status"`
	Brand        string `json:"brand"`
	SeatCount    *int   `json:"seat_count"`
	PurchaseDate string `json:"purchase_date"`
	Remarks      string `json:"remarks"`
}

// registerFleetVehicleRoutes 注册车队管理路由（P12）。
// 契约：GET/POST /fleet-vehicles；GET/PUT/DELETE /fleet-vehicles/{id}。
// 权限：列表/详情 fleet.view；创建 fleet.create；编辑 fleet.edit；删除 fleet.delete。
func (h *Handler) registerFleetVehicleRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "fleet", "view")).Get("/fleet-vehicles", h.listFleetVehicles)
	r.With(middleware.RequirePermission(h.db, "fleet", "create")).Post("/fleet-vehicles", h.createFleetVehicle)
	r.With(middleware.RequirePermission(h.db, "fleet", "view")).Get("/fleet-vehicles/{id}", h.getFleetVehicle)
	r.With(middleware.RequirePermission(h.db, "fleet", "edit")).Put("/fleet-vehicles/{id}", h.updateFleetVehicle)
	r.With(middleware.RequirePermission(h.db, "fleet", "delete")).Delete("/fleet-vehicles/{id}", h.deleteFleetVehicle)
}

// loadFleetVehicle 按租户隔离加载单条车辆档案。
func (h *Handler) loadFleetVehicle(w http.ResponseWriter, r *http.Request, userID uint) (*models.FleetVehicle, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的车辆 ID", err)
		return nil, false
	}
	var record models.FleetVehicle
	err = h.db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "未找到车辆档案", err)
		return nil, false
	}
	return &record, true
}

// validateFleetVehiclePayload 校验创建/更新车辆档案必填字段与状态合法性。
func validateFleetVehiclePayload(p *fleetVehiclePayload) error {
	record := models.FleetVehicle{
		PlateNumber:  p.PlateNumber,
		VehicleModel: p.VehicleModel,
		Status:       p.Status,
		Brand:        p.Brand,
		SeatCount:    p.SeatCount,
		PurchaseDate: p.PurchaseDate,
		Remarks:      p.Remarks,
	}
	return record.Validate()
}

// fleetVehicleFromPayload 组装车辆档案（去除首尾空白）。
func fleetVehicleFromPayload(userID uint, p *fleetVehiclePayload) *models.FleetVehicle {
	return &models.FleetVehicle{
		UserID:       userID,
		PlateNumber:  strings.TrimSpace(p.PlateNumber),
		VehicleModel: strings.TrimSpace(p.VehicleModel),
		Status:       p.Status,
		Brand:        strings.TrimSpace(p.Brand),
		SeatCount:    p.SeatCount,
		PurchaseDate: strings.TrimSpace(p.PurchaseDate),
		Remarks:      strings.TrimSpace(p.Remarks),
	}
}

// fleetPlateNumberExists 判断租户内车牌号是否已存在（excludeID>0 时排除自身，用于编辑场景）。
func (h *Handler) fleetPlateNumberExists(userID uint, plateNumber string, excludeID uint) (bool, error) {
	query := h.db.Model(&models.FleetVehicle{}).Where("user_id = ? AND plate_number = ?", userID, plateNumber)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// listFleetVehicles 列表查询车辆档案（fleet.view）：支持 status 过滤，按车牌号排序。
func (h *Handler) listFleetVehicles(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.db.Where("user_id = ?", userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidFleetVehicleStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的车辆状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var records []models.FleetVehicle
	if err := query.Order("plate_number ASC, id ASC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询车辆档案失败", err)
		return
	}
	if records == nil {
		records = []models.FleetVehicle{}
	}
	respondJSON(w, http.StatusOK, records)
}

// getFleetVehicle 查询单条车辆档案详情（fleet.view）。
func (h *Handler) getFleetVehicle(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadFleetVehicle(w, r, userID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// createFleetVehicle 创建车辆档案（fleet.create）。
func (h *Handler) createFleetVehicle(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload fleetVehiclePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateFleetVehiclePayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	record := fleetVehicleFromPayload(userID, &payload)
	exists, err := h.fleetPlateNumberExists(userID, record.PlateNumber, 0)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "检查车牌号失败", err)
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "车牌号在当前租户内已存在", nil)
		return
	}
	if err := h.db.Create(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建车辆档案失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

// fleetVehicleFieldsEqual 比较请求体业务字段与现有记录是否完全一致（恢复场景禁止修改其他字段）。
func fleetVehicleFieldsEqual(record *models.FleetVehicle, p *fleetVehiclePayload) bool {
	if record.PlateNumber != strings.TrimSpace(p.PlateNumber) ||
		record.VehicleModel != strings.TrimSpace(p.VehicleModel) ||
		record.Brand != strings.TrimSpace(p.Brand) ||
		record.PurchaseDate != strings.TrimSpace(p.PurchaseDate) ||
		record.Remarks != strings.TrimSpace(p.Remarks) {
		return false
	}
	if record.SeatCount == nil && p.SeatCount != nil {
		return false
	}
	if record.SeatCount != nil && (p.SeatCount == nil || *record.SeatCount != *p.SeatCount) {
		return false
	}
	return true
}

// updateFleetVehicle 更新车辆档案（fleet.edit）。
// 状态边界：active 可编辑（含置为 inactive）；inactive 仅允许恢复为 active，不得修改其他字段。
func (h *Handler) updateFleetVehicle(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload fleetVehiclePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求体", err)
		return
	}
	if err := validateFleetVehiclePayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	record, ok := h.loadFleetVehicle(w, r, userID)
	if !ok {
		return
	}

	// inactive 状态边界：仅允许恢复为 active，其他字段禁止变更
	if record.Status == models.FleetVehicleStatusInactive {
		if payload.Status != models.FleetVehicleStatusActive {
			respondError(w, http.StatusConflict, "inactive 车辆不可编辑，仅可用 PUT 更新为 active 恢复", nil)
			return
		}
		if !fleetVehicleFieldsEqual(record, &payload) {
			respondError(w, http.StatusConflict, "inactive 车辆不可编辑业务字段，仅可恢复为 active", nil)
			return
		}
		if err := h.db.Model(record).Update("status", models.FleetVehicleStatusActive).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "恢复车辆档案失败", err)
			return
		}
		h.respondFleetVehicle(w, record.ID)
		return
	}

	// active 状态：全量更新
	updated := fleetVehicleFromPayload(userID, &payload)
	exists, err := h.fleetPlateNumberExists(userID, updated.PlateNumber, record.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "检查车牌号失败", err)
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "车牌号在当前租户内已存在", nil)
		return
	}
	updates := map[string]any{
		"plate_number":  updated.PlateNumber,
		"vehicle_model": updated.VehicleModel,
		"status":        updated.Status,
		"brand":         updated.Brand,
		"seat_count":    updated.SeatCount,
		"purchase_date": updated.PurchaseDate,
		"remarks":       updated.Remarks,
	}
	if err := h.db.Model(record).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新车辆档案失败", err)
		return
	}
	h.respondFleetVehicle(w, record.ID)
}

// deleteFleetVehicle 删除车辆档案（fleet.delete）：仅 active 可删除。
func (h *Handler) deleteFleetVehicle(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadFleetVehicle(w, r, userID)
	if !ok {
		return
	}
	if record.Status == models.FleetVehicleStatusInactive {
		respondError(w, http.StatusConflict, "仅 active 车辆可删除", nil)
		return
	}
	if err := h.db.Delete(record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除车辆档案失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": record.ID})
}

// respondFleetVehicle 重新加载并返回单条车辆档案。
func (h *Handler) respondFleetVehicle(w http.ResponseWriter, id uint) {
	var record models.FleetVehicle
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重新加载车辆档案失败", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}
