package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupDormEnergyTestEnv 初始化能耗汇总测试环境（admin 具备 dormitory.view）。
func setupDormEnergyTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(
		&models.DormSite{}, &models.DormBuilding{}, &models.DormRoom{}, &models.DormMeterReading{},
	))
	grantDormitoryPermission(t, env, "admin", "view")
	return env
}

// grantDormitoryPermission 为指定角色补充 dormitory 模块指定动作权限。
func grantDormitoryPermission(t *testing.T, env *rbacTestEnv, role, action string) {
	t.Helper()
	var permission models.Permission
	err := env.db.Where("module = ? AND action = ?", "dormitory", action).First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = models.Permission{Module: "dormitory", Action: action, Label: action}
		require.NoError(t, env.db.Create(&permission).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, env.db.Create(&models.RolePermission{RoleID: env.roleIDs[role], PermissionID: permission.ID}).Error)
}

// createDormEnergyFixture 创建 1 个园区、2 栋楼、每栋 2 个房间，返回楼栋与房间。
func createDormEnergyFixture(t *testing.T, env *rbacTestEnv, userID uint) (models.DormBuilding, models.DormBuilding, []models.DormRoom) {
	t.Helper()
	site := models.DormSite{UserID: uintPointer(userID), Name: "测试园区"}
	require.NoError(t, env.db.Create(&site).Error)
	b1 := models.DormBuilding{UserID: uintPointer(userID), SiteID: site.ID, Name: "1号楼"}
	b2 := models.DormBuilding{UserID: uintPointer(userID), SiteID: site.ID, Name: "2号楼"}
	require.NoError(t, env.db.Create(&b1).Error)
	require.NoError(t, env.db.Create(&b2).Error)
	rooms := []models.DormRoom{
		{UserID: uintPointer(userID), BuildingID: b1.ID, RoomNumber: "101"},
		{UserID: uintPointer(userID), BuildingID: b1.ID, RoomNumber: "102"},
		{UserID: uintPointer(userID), BuildingID: b2.ID, RoomNumber: "201"},
		{UserID: uintPointer(userID), BuildingID: b2.ID, RoomNumber: "202"},
	}
	for i := range rooms {
		require.NoError(t, env.db.Create(&rooms[i]).Error)
	}
	return b1, b2, rooms
}

// createDormEnergyReading 直接落库一条抄表记录（charge_details 为明细数组）。
func createDormEnergyReading(t *testing.T, env *rbacTestEnv, userID, roomID uint, meterDate string, details []map[string]any) {
	t.Helper()
	date, err := time.Parse("2006-01-02", meterDate)
	require.NoError(t, err)
	raw, err := json.Marshal(details)
	require.NoError(t, err)
	reading := models.DormMeterReading{
		UserID:        uintPointer(userID),
		RoomID:        roomID,
		MeterDate:     date,
		BillingStart:  date,
		BillingEnd:    date,
		ChargeDetails: datatypes.JSON(raw),
	}
	require.NoError(t, env.db.Create(&reading).Error)
}

// getDormEnergySummary 请求能耗汇总并解析响应。
func getDormEnergySummary(t *testing.T, env *rbacTestEnv, token, query string) (*energySummaryResponse, int) {
	t.Helper()
	path := "/api/dormitories/energy/summary"
	if query != "" {
		path += "?" + query
	}
	rec := doRBACRequest(t, env.router, http.MethodGet, path, token, nil)
	var resp energySummaryResponse
	if rec.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	}
	return &resp, rec.Code
}

// TestDormEnergySummaryAggregation 正常聚合：跨月、跨楼栋、跨房间，验证整体/按楼栋/按房间三级汇总。
func TestDormEnergySummaryAggregation(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	b1, b2, rooms := createDormEnergyFixture(t, env, env.admin.ID)
	token := env.admin.SupabaseUID

	// 2026-01：101 电 100/50 + 水 10/20；102 电 200/100
	createDormEnergyReading(t, env, env.admin.ID, rooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 100, "amount": 50},
		{"key": "water", "usage": 10, "amount": 20},
	})
	createDormEnergyReading(t, env, env.admin.ID, rooms[1].ID, "2026-01-20", []map[string]any{
		{"key": "electric", "usage": 200, "amount": 100},
	})
	// 2026-02：201 水 30/60
	createDormEnergyReading(t, env, env.admin.ID, rooms[2].ID, "2026-02-05", []map[string]any{
		{"key": "water", "usage": 30, "amount": 60},
	})

	resp, code := getDormEnergySummary(t, env, token, "")
	require.Equal(t, http.StatusOK, code)

	// 整体：电 300/150（2 条），水 40/80（2 条），合计 230
	require.Equal(t, 300.0, resp.Overall.Electric.Usage)
	require.Equal(t, 150.0, resp.Overall.Electric.Amount)
	require.Equal(t, 2, resp.Overall.Electric.Count)
	require.Equal(t, 40.0, resp.Overall.Water.Usage)
	require.Equal(t, 80.0, resp.Overall.Water.Amount)
	require.Equal(t, 2, resp.Overall.Water.Count)
	require.Equal(t, 230.0, resp.Overall.TotalAmount)

	// 按楼栋：1号楼 电 300/150 + 水 10/20 = 170；2号楼 水 30/60 = 60
	require.Len(t, resp.ByBuilding, 2)
	require.Equal(t, b1.ID, resp.ByBuilding[0].BuildingID)
	require.Equal(t, "1号楼", resp.ByBuilding[0].BuildingName)
	require.Equal(t, 300.0, resp.ByBuilding[0].Electric.Usage)
	require.Equal(t, 10.0, resp.ByBuilding[0].Water.Usage)
	require.Equal(t, 170.0, resp.ByBuilding[0].TotalAmount)
	require.Equal(t, b2.ID, resp.ByBuilding[1].BuildingID)
	require.Equal(t, 60.0, resp.ByBuilding[1].TotalAmount)

	// 按房间：101 电 100/50 + 水 10/20 = 70；102 电 200/100 = 100；201 水 30/60 = 60
	require.Len(t, resp.Rooms, 3)
	require.Equal(t, rooms[0].ID, resp.Rooms[0].RoomID)
	require.Equal(t, "101", resp.Rooms[0].RoomNumber)
	require.Equal(t, b1.ID, resp.Rooms[0].BuildingID)
	require.Equal(t, 70.0, resp.Rooms[0].TotalAmount)
	require.Equal(t, rooms[1].ID, resp.Rooms[1].RoomID)
	require.Equal(t, 100.0, resp.Rooms[1].TotalAmount)
	require.Equal(t, rooms[2].ID, resp.Rooms[2].RoomID)
	require.Equal(t, 60.0, resp.Rooms[2].TotalAmount)
}

// TestDormEnergySummaryMonthFilter month=YYYY-MM 按抄表日期所在自然月筛选。
func TestDormEnergySummaryMonthFilter(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	_, _, rooms := createDormEnergyFixture(t, env, env.admin.ID)
	token := env.admin.SupabaseUID

	createDormEnergyReading(t, env, env.admin.ID, rooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 100, "amount": 50},
	})
	createDormEnergyReading(t, env, env.admin.ID, rooms[1].ID, "2026-02-05", []map[string]any{
		{"key": "electric", "usage": 200, "amount": 100},
	})

	// 仅 2026-01
	resp, code := getDormEnergySummary(t, env, token, "month=2026-01")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 100.0, resp.Overall.Electric.Usage)
	require.Equal(t, 50.0, resp.Overall.TotalAmount)
	require.Len(t, resp.Rooms, 1)

	// 仅 2026-02
	resp, code = getDormEnergySummary(t, env, token, "month=2026-02")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 200.0, resp.Overall.Electric.Usage)
	require.Equal(t, 100.0, resp.Overall.TotalAmount)

	// 无数据月份
	resp, code = getDormEnergySummary(t, env, token, "month=2026-03")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 0.0, resp.Overall.Electric.Usage)
	require.Len(t, resp.ByBuilding, 0)
	require.Len(t, resp.Rooms, 0)

	// 非法 month 格式 → 400
	_, code = getDormEnergySummary(t, env, token, "month=2026/01")
	require.Equal(t, http.StatusBadRequest, code)
}

// TestDormEnergySummaryBuildingFilter building_id 按楼栋筛选。
func TestDormEnergySummaryBuildingFilter(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	b1, b2, rooms := createDormEnergyFixture(t, env, env.admin.ID)
	token := env.admin.SupabaseUID

	createDormEnergyReading(t, env, env.admin.ID, rooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 100, "amount": 50},
	})
	createDormEnergyReading(t, env, env.admin.ID, rooms[2].ID, "2026-01-06", []map[string]any{
		{"key": "water", "usage": 30, "amount": 60},
	})

	// 仅 1号楼
	resp, code := getDormEnergySummary(t, env, token, "building_id="+strconv.Itoa(int(b1.ID)))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 100.0, resp.Overall.Electric.Usage)
	require.Equal(t, 0.0, resp.Overall.Water.Usage)
	require.Len(t, resp.ByBuilding, 1)
	require.Len(t, resp.Rooms, 1)

	// 仅 2号楼
	resp, code = getDormEnergySummary(t, env, token, "building_id="+strconv.Itoa(int(b2.ID)))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 30.0, resp.Overall.Water.Usage)
	require.Equal(t, 60.0, resp.Overall.TotalAmount)
	require.Len(t, resp.Rooms, 1)

	// 非法 building_id → 400
	_, code = getDormEnergySummary(t, env, token, "building_id=abc")
	require.Equal(t, http.StatusBadRequest, code)
}

// TestDormEnergySummaryEmptyData 无任何抄表数据时返回空聚合。
func TestDormEnergySummaryEmptyData(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	createDormEnergyFixture(t, env, env.admin.ID)
	token := env.admin.SupabaseUID

	resp, code := getDormEnergySummary(t, env, token, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 0.0, resp.Overall.Electric.Usage)
	require.Equal(t, 0, resp.Overall.Electric.Count)
	require.Equal(t, 0.0, resp.Overall.Water.Usage)
	require.Equal(t, 0.0, resp.Overall.TotalAmount)
	require.Empty(t, resp.ByBuilding)
	require.Empty(t, resp.Rooms)
}

// TestDormEnergySummaryIgnoresInvalid 忽略燃气、空/缺失 usage 或 amount、负数等不合法数值。
func TestDormEnergySummaryIgnoresInvalid(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	_, _, rooms := createDormEnergyFixture(t, env, env.admin.ID)
	token := env.admin.SupabaseUID

	// 有效项 + 各类无效项混在同一条抄表记录中
	createDormEnergyReading(t, env, env.admin.ID, rooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 100, "amount": 50}, // 有效
		{"key": "water", "usage": 10, "amount": 20},     // 有效
		{"key": "gas", "usage": 5, "amount": 3},         // 燃气：忽略
		{"key": "electric", "usage": nil, "amount": 10}, // usage 缺失：忽略
		{"key": "water", "usage": 20, "amount": nil},    // amount 缺失：忽略
		{"key": "electric", "usage": -5, "amount": -2},  // 负数：忽略
		{"key": "electric", "usage": 0, "amount": 0},    // 零值：合法，计入
	})
	// charge_details 无法解析（usage 为字符串）：整条忽略
	createDormEnergyReading(t, env, env.admin.ID, rooms[1].ID, "2026-01-06", []map[string]any{
		{"key": "electric", "usage": "abc", "amount": 10},
	})

	resp, code := getDormEnergySummary(t, env, token, "")
	require.Equal(t, http.StatusOK, code)
	// electric 有效项 2 条：usage=100 与 usage=0（零值合法计入）
	require.Equal(t, 100.0, resp.Overall.Electric.Usage)
	require.Equal(t, 50.0, resp.Overall.Electric.Amount)
	require.Equal(t, 2, resp.Overall.Electric.Count)
	require.Equal(t, 10.0, resp.Overall.Water.Usage)
	require.Equal(t, 20.0, resp.Overall.Water.Amount)
	require.Equal(t, 1, resp.Overall.Water.Count)
	require.Equal(t, 70.0, resp.Overall.TotalAmount)
	require.Len(t, resp.Rooms, 1)
}

// TestDormEnergySummaryRBAC 无 dormitory.view 权限返回 403，授权后 200。
func TestDormEnergySummaryRBAC(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	createDormEnergyFixture(t, env, env.admin.ID)

	// viewer 无 dormitory.view → 403
	_, code := getDormEnergySummary(t, env, env.viewer.SupabaseUID, "")
	require.Equal(t, http.StatusForbidden, code)

	// 授予 viewer dormitory.view → 200
	grantDormitoryPermission(t, env, "viewer", "view")
	_, code = getDormEnergySummary(t, env, env.viewer.SupabaseUID, "")
	require.Equal(t, http.StatusOK, code)
}

// TestDormEnergySummaryTenantIsolation 跨租户隔离：只能看到本 user_id 的抄表数据。
func TestDormEnergySummaryTenantIsolation(t *testing.T) {
	env := setupDormEnergyTestEnv(t)
	_, _, rooms := createDormEnergyFixture(t, env, env.admin.ID)
	// 另一租户（manager）的数据
	_, _, otherRooms := createDormEnergyFixture(t, env, env.manager.ID)

	createDormEnergyReading(t, env, env.admin.ID, rooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 100, "amount": 50},
	})
	createDormEnergyReading(t, env, env.manager.ID, otherRooms[0].ID, "2026-01-05", []map[string]any{
		{"key": "electric", "usage": 999, "amount": 999},
	})

	// admin 只看得到自己的数据
	resp, code := getDormEnergySummary(t, env, env.admin.SupabaseUID, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 100.0, resp.Overall.Electric.Usage)
	require.Equal(t, 50.0, resp.Overall.TotalAmount)
	require.Len(t, resp.Rooms, 1)

	// manager 只看得到自己的数据
	grantDormitoryPermission(t, env, "manager", "view")
	resp, code = getDormEnergySummary(t, env, env.manager.SupabaseUID, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 999.0, resp.Overall.Electric.Usage)
	require.Len(t, resp.Rooms, 1)
}
