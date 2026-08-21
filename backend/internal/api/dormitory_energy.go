package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// energySummaryKeys 能耗汇总仅统计电/水两类能源，燃气等其他类型一律忽略
var energySummaryKeys = map[string]bool{
	"electric": true,
	"water":    true,
}

// energyMetric 单类能源聚合指标
type energyMetric struct {
	Usage  float64 `json:"usage"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// energyOverallSummary 整体汇总
type energyOverallSummary struct {
	Electric    energyMetric `json:"electric"`
	Water       energyMetric `json:"water"`
	TotalAmount float64      `json:"total_amount"`
}

// energyBuildingSummary 按楼栋汇总
type energyBuildingSummary struct {
	BuildingID   uint         `json:"building_id"`
	BuildingName string       `json:"building_name"`
	Electric     energyMetric `json:"electric"`
	Water        energyMetric `json:"water"`
	TotalAmount  float64      `json:"total_amount"`
}

// energyRoomSummary 可下钻房间数据
type energyRoomSummary struct {
	RoomID      uint         `json:"room_id"`
	RoomNumber  string       `json:"room_number"`
	BuildingID  uint         `json:"building_id"`
	Electric    energyMetric `json:"electric"`
	Water       energyMetric `json:"water"`
	TotalAmount float64      `json:"total_amount"`
}

// energySummaryResponse 能耗汇总响应体（整体 / 按楼栋 / 按房间三级）
type energySummaryResponse struct {
	Overall    energyOverallSummary    `json:"overall"`
	ByBuilding []energyBuildingSummary `json:"by_building"`
	Rooms      []energyRoomSummary     `json:"rooms"`
}

// energyAgg 聚合累加器
type energyAgg struct {
	electric energyMetric
	water    energyMetric
}

// add 累加一条合法明细
func (a *energyAgg) add(key string, usage, amount float64) {
	switch key {
	case "electric":
		a.electric.Usage += usage
		a.electric.Amount += amount
		a.electric.Count++
	case "water":
		a.water.Usage += usage
		a.water.Amount += amount
		a.water.Count++
	}
}

// totalAmount 电水金额合计
func (a *energyAgg) totalAmount() float64 {
	return a.electric.Amount + a.water.Amount
}

// isValidEnergyValue 判断能耗数值是否合法：非空、非 NaN/Inf、非负数
func isValidEnergyValue(v *float64) bool {
	if v == nil {
		return false
	}
	val := *v
	return !math.IsNaN(val) && !math.IsInf(val, 0) && val >= 0
}

// parseMonthRange 解析 YYYY-MM 为自然月起止时间（左闭右开）
func parseMonthRange(value string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 1, 0), nil
}

// parseChargeDetails 解析抄表明细 JSON，解析失败返回 false（整条忽略）
func parseChargeDetails(raw datatypes.JSON) ([]models.MeterChargeDetail, bool) {
	var details []models.MeterChargeDetail
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, false
	}
	return details, true
}

// roomMeta 提取房间所在楼栋与房间号；房间缺失时归入未知楼栋
func roomMeta(reading *models.DormMeterReading) (buildingID uint, buildingName, roomNumber string) {
	if reading.Room == nil {
		return 0, "未知楼栋", ""
	}
	if reading.Room.Building != nil {
		buildingID = reading.Room.Building.ID
		buildingName = reading.Room.Building.Name
	}
	return buildingID, buildingName, reading.Room.RoomNumber
}

// getDormEnergySummary 只读能耗汇总：聚合 charge_details 中 electric/water 的 usage 与 amount，
// 按抄表日期所在自然月（month=YYYY-MM）与楼栋（building_id）筛选，返回整体/按楼栋/按房间三级汇总。
func (h *Handler) getDormEnergySummary(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query, err := h.buildEnergySummaryQuery(r, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var readings []models.DormMeterReading
	if err := query.Preload("Room.Building").Order("meter_date ASC").Find(&readings).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load meter readings", err)
		return
	}
	respondJSON(w, http.StatusOK, aggregateEnergySummary(readings))
}

// buildEnergySummaryQuery 组装能耗汇总查询：user_id 隔离 + month/building_id 筛选
// 注意：JOIN dorm_rooms 后两表均含 user_id 列，必须显式限定表名避免歧义
func (h *Handler) buildEnergySummaryQuery(r *http.Request, userID uint) (*gorm.DB, error) {
	query := h.db.Where("dorm_meter_readings.user_id = ?", userID).Model(&models.DormMeterReading{})
	if v := strings.TrimSpace(r.URL.Query().Get("month")); v != "" {
		start, end, err := parseMonthRange(v)
		if err != nil {
			return nil, fmt.Errorf("invalid month, expected YYYY-MM")
		}
		query = query.Where("meter_date >= ? AND meter_date < ?", start, end)
	}
	if v := strings.TrimSpace(r.URL.Query().Get("building_id")); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("invalid building_id")
		}
		query = query.Joins("JOIN dorm_rooms ON dorm_rooms.id = dorm_meter_readings.room_id").
			Where("dorm_rooms.building_id = ?", id)
	}
	return query, nil
}

// aggregateEnergySummary 聚合抄表记录为整体/按楼栋/按房间三级汇总
func aggregateEnergySummary(readings []models.DormMeterReading) energySummaryResponse {
	overall := energyAgg{}
	buildings := map[uint]*energyAgg{}
	rooms := map[uint]*energyAgg{}
	buildingNames := map[uint]string{}
	roomNumbers := map[uint]string{}
	roomBuildings := map[uint]uint{}

	for i := range readings {
		reading := &readings[i]
		details, ok := parseChargeDetails(reading.ChargeDetails)
		if !ok {
			continue
		}
		buildingID, buildingName, roomNumber := roomMeta(reading)
		for _, d := range details {
			if !energySummaryKeys[d.Key] || !isValidEnergyValue(d.Usage) || !isValidEnergyValue(d.Amount) {
				continue
			}
			overall.add(d.Key, *d.Usage, *d.Amount)
			agg := buildings[buildingID]
			if agg == nil {
				agg = &energyAgg{}
				buildings[buildingID] = agg
				buildingNames[buildingID] = buildingName
			}
			agg.add(d.Key, *d.Usage, *d.Amount)
			agg = rooms[reading.RoomID]
			if agg == nil {
				agg = &energyAgg{}
				rooms[reading.RoomID] = agg
				roomNumbers[reading.RoomID] = roomNumber
				roomBuildings[reading.RoomID] = buildingID
			}
			agg.add(d.Key, *d.Usage, *d.Amount)
		}
	}
	return buildEnergySummary(overall, buildings, rooms, buildingNames, roomNumbers, roomBuildings)
}

// buildEnergySummary 组装三级汇总响应（按楼栋/房间 ID 升序，保证输出稳定）
func buildEnergySummary(overall energyAgg, buildings, rooms map[uint]*energyAgg,
	buildingNames, roomNumbers map[uint]string, roomBuildings map[uint]uint) energySummaryResponse {
	resp := energySummaryResponse{
		Overall: energyOverallSummary{
			Electric:    overall.electric,
			Water:       overall.water,
			TotalAmount: overall.totalAmount(),
		},
		ByBuilding: make([]energyBuildingSummary, 0, len(buildings)),
		Rooms:      make([]energyRoomSummary, 0, len(rooms)),
	}
	for id, agg := range buildings {
		resp.ByBuilding = append(resp.ByBuilding, energyBuildingSummary{
			BuildingID:   id,
			BuildingName: buildingNames[id],
			Electric:     agg.electric,
			Water:        agg.water,
			TotalAmount:  agg.totalAmount(),
		})
	}
	for id, agg := range rooms {
		resp.Rooms = append(resp.Rooms, energyRoomSummary{
			RoomID:      id,
			RoomNumber:  roomNumbers[id],
			BuildingID:  roomBuildings[id],
			Electric:    agg.electric,
			Water:       agg.water,
			TotalAmount: agg.totalAmount(),
		})
	}
	sort.Slice(resp.ByBuilding, func(i, j int) bool { return resp.ByBuilding[i].BuildingID < resp.ByBuilding[j].BuildingID })
	sort.Slice(resp.Rooms, func(i, j int) bool { return resp.Rooms[i].RoomID < resp.Rooms[j].RoomID })
	return resp
}
