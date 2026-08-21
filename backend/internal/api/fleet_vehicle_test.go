package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupFleetVehicleTestEnv 初始化车队管理测试环境（admin 具备 fleet 全权限）。
func setupFleetVehicleTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.FleetVehicle{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantFleetPermission(t, env, "admin", action)
	}
	return env
}

// grantFleetPermission 为指定角色补充 fleet 模块指定动作权限。
func grantFleetPermission(t *testing.T, env *rbacTestEnv, role, action string) {
	t.Helper()
	var permission models.Permission
	err := env.db.Where("module = ? AND action = ?", "fleet", action).First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = models.Permission{Module: "fleet", Action: action, Label: action}
		require.NoError(t, env.db.Create(&permission).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, env.db.Create(&models.RolePermission{RoleID: env.roleIDs[role], PermissionID: permission.ID}).Error)
}

// fleetVehicleBody 构造合法的车辆档案创建请求体。
func fleetVehicleBody() map[string]any {
	seatCount := 19
	return map[string]any{
		"plate_number":  "京A12345",
		"vehicle_model": "丰田考斯特",
		"status":        models.FleetVehicleStatusActive,
		"brand":         "丰田",
		"seat_count":    seatCount,
		"purchase_date": "2024-05-01",
		"remarks":       "商务接待用车",
	}
}

// TestFleetVehicleLifecycle 完整生命周期：创建 → 列表/详情 → active 编辑 → 置为 inactive →
// inactive 不可编辑/删除 → 恢复为 active → 删除。
func TestFleetVehicleLifecycle(t *testing.T) {
	env := setupFleetVehicleTestEnv(t)
	token := env.admin.SupabaseUID

	// 创建（create）
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", token, fleetVehicleBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var record models.FleetVehicle
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &record))
	require.Equal(t, models.FleetVehicleStatusActive, record.Status)

	// 列表与详情（view）
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles/1", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// active 编辑（edit）：修改品牌
	body := fleetVehicleBody()
	body["brand"] = "一汽丰田"
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/fleet-vehicles/1", token, body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// active 置为 inactive（edit）
	body = fleetVehicleBody()
	body["status"] = models.FleetVehicleStatusInactive
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/fleet-vehicles/1", token, body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, env.db.First(&record, record.ID).Error)
	require.Equal(t, models.FleetVehicleStatusInactive, record.Status)

	// inactive 不可编辑（修改业务字段 → 409）
	body = fleetVehicleBody()
	body["status"] = models.FleetVehicleStatusActive
	body["brand"] = "改品牌"
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/fleet-vehicles/1", token, body)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	// inactive 不可删除（409）
	response = doRBACRequest(t, env.router, http.MethodDelete, "/api/fleet-vehicles/1", token, nil)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	// inactive 恢复为 active（仅 status 变更，其他字段一致 → 200）
	body = fleetVehicleBody()
	body["status"] = models.FleetVehicleStatusActive
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/fleet-vehicles/1", token, body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, env.db.First(&record, record.ID).Error)
	require.Equal(t, models.FleetVehicleStatusActive, record.Status)

	// active 可删除（delete）
	response = doRBACRequest(t, env.router, http.MethodDelete, "/api/fleet-vehicles/1", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles/1", token, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}

// TestFleetVehicleValidation 必填字段与类型校验：非法输入返回 400。
func TestFleetVehicleValidation(t *testing.T) {
	env := setupFleetVehicleTestEnv(t)
	token := env.admin.SupabaseUID

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"车牌缺失", func(b map[string]any) { b["plate_number"] = " " }},
		{"车辆型号缺失", func(b map[string]any) { b["vehicle_model"] = "" }},
		{"状态非法", func(b map[string]any) { b["status"] = "pending" }},
		{"状态缺失", func(b map[string]any) { b["status"] = "" }},
		{"座位数为零", func(b map[string]any) { b["seat_count"] = 0 }},
		{"座位数为负", func(b map[string]any) { b["seat_count"] = -1 }},
		{"购置日期格式错误", func(b map[string]any) { b["purchase_date"] = "2024/05/01" }},
	}
	for _, c := range cases {
		body := fleetVehicleBody()
		c.mutate(body)
		response := doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", token, body)
		require.Equal(t, http.StatusBadRequest, response.Code, "%s 应返回 400: %s", c.name, response.Body.String())
	}
}

// TestFleetVehiclePlateUnique 车牌号租户内唯一：重复创建返回 409。
func TestFleetVehiclePlateUnique(t *testing.T) {
	env := setupFleetVehicleTestEnv(t)
	token := env.admin.SupabaseUID

	response := doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", token, fleetVehicleBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	// 同租户重复车牌 → 409
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", token, fleetVehicleBody())
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	// 创建第二辆车（不同车牌）
	body := fleetVehicleBody()
	body["plate_number"] = "京B67890"
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", token, body)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	// 编辑第二辆车时改为已存在车牌 → 409
	editBody := fleetVehicleBody()
	editBody["plate_number"] = "京A12345"
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/fleet-vehicles/2", token, editBody)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}

// TestFleetVehicleRBACAndIsolation 权限与跨租户隔离：viewer 无权限 403；跨租户记录不可见 404。
func TestFleetVehicleRBACAndIsolation(t *testing.T) {
	env := setupFleetVehicleTestEnv(t)

	// viewer 无 fleet 权限：全部端点 403
	response := doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", env.viewer.SupabaseUID, fleetVehicleBody())
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	// admin 创建本租户记录
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", env.admin.SupabaseUID, fleetVehicleBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	// 跨租户：viewer 即使获得 view 权限也看不到 admin 的记录（404）
	grantFleetPermission(t, env, "viewer", "view")
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles/1", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	// viewer 创建自己的记录，admin 同样不可见（404）
	grantFleetPermission(t, env, "viewer", "create")
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/fleet-vehicles", env.viewer.SupabaseUID, fleetVehicleBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/fleet-vehicles/2", env.admin.SupabaseUID, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}
