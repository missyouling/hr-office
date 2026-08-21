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

// setupSafetyInspectionTestEnv 初始化安全检查测试环境（admin 具备 safety 全权限）。
func setupSafetyInspectionTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.SafetyInspection{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantSafetyPermission(t, env, "admin", action)
	}
	return env
}

// grantSafetyPermission 为指定角色补充 safety 模块指定动作权限。
func grantSafetyPermission(t *testing.T, env *rbacTestEnv, role, action string) {
	t.Helper()
	var permission models.Permission
	err := env.db.Where("module = ? AND action = ?", "safety", action).First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = models.Permission{Module: "safety", Action: action, Label: action}
		require.NoError(t, env.db.Create(&permission).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, env.db.Create(&models.RolePermission{RoleID: env.roleIDs[role], PermissionID: permission.ID}).Error)
}

// safetyInspectionBody 构造合法的安全检查创建请求体。
func safetyInspectionBody() map[string]any {
	return map[string]any{
		"inspection_type":           models.SafetyInspectionTypeRoutine,
		"inspection_date":           "2026-08-19",
		"location":                  "一号车间",
		"responsible_person":        "张三",
		"issue_description":         "消防通道堆放杂物",
		"rectification_requirement": "立即清理并保持畅通",
	}
}

// TestSafetyInspectionLifecycle 完整生命周期：创建草稿 → 完成 → 完成后不可编辑/删除 → 作废 → 终态不可再作废。
func TestSafetyInspectionLifecycle(t *testing.T) {
	env := setupSafetyInspectionTestEnv(t)
	token := env.admin.SupabaseUID

	response := doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections", token, safetyInspectionBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var record models.SafetyInspection
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &record))
	require.Equal(t, models.SafetyInspectionStatusDraft, record.Status)

	// 列表与详情（view）
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/safety-inspections", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/safety-inspections/1", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// 草稿编辑（edit）
	body := safetyInspectionBody()
	body["location"] = "二号车间"
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/safety-inspections/1", token, body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// 手动完成（edit）
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections/1/complete", token, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, env.db.First(&record, record.ID).Error)
	require.Equal(t, models.SafetyInspectionStatusCompleted, record.Status)

	// 完成后不可编辑/删除
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/safety-inspections/1", token, body)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodDelete, "/api/safety-inspections/1", token, nil)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	// 作废（delete，原因必填）
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections/1/void", token, map[string]any{"reason": ""})
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections/1/void", token, map[string]any{"reason": "检查重复"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// 终态不可再作废
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections/1/void", token, map[string]any{"reason": "再次作废"})
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}

// TestSafetyInspectionValidation 必填字段与类型校验：非法输入返回 400。
func TestSafetyInspectionValidation(t *testing.T) {
	env := setupSafetyInspectionTestEnv(t)
	token := env.admin.SupabaseUID

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"类型非法", func(b map[string]any) { b["inspection_type"] = "other" }},
		{"日期缺失", func(b map[string]any) { b["inspection_date"] = "" }},
		{"日期格式错误", func(b map[string]any) { b["inspection_date"] = "2026/08/19" }},
		{"地点缺失", func(b map[string]any) { b["location"] = " " }},
		{"责任人缺失", func(b map[string]any) { b["responsible_person"] = "" }},
		{"问题描述缺失", func(b map[string]any) { b["issue_description"] = " " }},
		{"整改要求缺失", func(b map[string]any) { b["rectification_requirement"] = "" }},
	}
	for _, c := range cases {
		body := safetyInspectionBody()
		c.mutate(body)
		response := doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections", token, body)
		require.Equal(t, http.StatusBadRequest, response.Code, "%s 应返回 400: %s", c.name, response.Body.String())
	}
}

// TestSafetyInspectionRBACAndIsolation 权限与跨租户隔离：viewer 无权限 403；跨租户记录不可见 404。
func TestSafetyInspectionRBACAndIsolation(t *testing.T) {
	env := setupSafetyInspectionTestEnv(t)

	// viewer 无 safety 权限：全部端点 403
	response := doRBACRequest(t, env.router, http.MethodGet, "/api/safety-inspections", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections", env.viewer.SupabaseUID, safetyInspectionBody())
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	// admin 创建本租户记录
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections", env.admin.SupabaseUID, safetyInspectionBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	// 跨租户：viewer 即使获得 view 权限也看不到 admin 的记录（404）
	grantSafetyPermission(t, env, "viewer", "view")
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/safety-inspections/1", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	// viewer 创建自己的记录，admin 同样不可见（404）
	grantSafetyPermission(t, env, "viewer", "create")
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/safety-inspections", env.viewer.SupabaseUID, safetyInspectionBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/safety-inspections/2", env.admin.SupabaseUID, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}
