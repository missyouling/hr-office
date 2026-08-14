package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	auditmw "siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/supabase"
)

// ---------- 测试辅助 ----------

// rbacTestEnv 保存 RBAC 测试环境（用户、角色、路由）
type rbacTestEnv struct {
	db      *gorm.DB
	router  http.Handler
	admin   models.User
	manager models.User
	editor  models.User
	viewer  models.User
	rbacOp  models.User // 拥有 rbac.manage 但非 admin 的操作者
	target  models.User // 被操作对象（viewer）
	roleIDs map[string]uint
	permIDs map[string]uint
}

// seedRBACForTest 在测试库中创建角色/权限/角色权限分配（模拟 rbac_seed.go 的种子结构）
func seedRBACForTest(t *testing.T, db *gorm.DB) (map[string]uint, map[string]uint) {
	t.Helper()

	roles := []models.Role{
		{Name: "admin", Label: "管理员", IsSystem: true},
		{Name: "manager", Label: "部门经理", IsSystem: true},
		{Name: "editor", Label: "编辑者", IsSystem: true},
		{Name: "viewer", Label: "只读用户", IsSystem: true},
		{Name: "rbac_op", Label: "RBAC操作员", IsSystem: false},
	}
	roleIDs := make(map[string]uint, len(roles))
	for i := range roles {
		require.NoError(t, db.Create(&roles[i]).Error)
		roleIDs[roles[i].Name] = roles[i].ID
	}

	perms := []models.Permission{
		{Module: "announcements", Action: "view", Label: "查看", SortOrder: 50},
		{Module: "announcements", Action: "create", Label: "创建", SortOrder: 51},
		{Module: "announcements", Action: "edit", Label: "编辑", SortOrder: 52},
		{Module: "announcements", Action: "delete", Label: "删除", SortOrder: 53},
		{Module: "settings", Action: "view", Label: "查看", SortOrder: 40},
		{Module: "settings", Action: "create", Label: "创建", SortOrder: 41},
		{Module: "settings", Action: "edit", Label: "编辑", SortOrder: 42},
		{Module: "settings", Action: "delete", Label: "删除", SortOrder: 43},
		{Module: "logs", Action: "view", Label: "查看日志", SortOrder: 74},
		{Module: "logs", Action: "manage", Label: "备份/清理/告警管理", SortOrder: 75},
		{Module: "notifications", Action: "view", Label: "查看通知配置", SortOrder: 76},
		{Module: "notifications", Action: "manage", Label: "管理通知配置与发送", SortOrder: 77},
		{Module: "rbac", Action: "manage", Label: "角色权限管理", SortOrder: 78},
		{Module: "users", Action: "view", Label: "查看", SortOrder: 70},
		{Module: "archives", Action: "view", Label: "查看", SortOrder: 30},
		{Module: "archives", Action: "create", Label: "创建", SortOrder: 31},
		{Module: "archives", Action: "edit", Label: "编辑", SortOrder: 32},
		{Module: "archives", Action: "delete", Label: "删除", SortOrder: 33},
	}
	permIDs := make(map[string]uint, len(perms))
	for i := range perms {
		require.NoError(t, db.Create(&perms[i]).Error)
		permIDs[perms[i].Module+"-"+perms[i].Action] = perms[i].ID
	}

	// admin：全量权限
	for _, p := range perms {
		require.NoError(t, db.Create(&models.RolePermission{RoleID: roleIDs["admin"], PermissionID: p.ID}).Error)
	}
	// manager：settings.view + logs.view + announcements 读/建/改 + users.view + archives 读/建/改
	managerPerms := []string{
		"announcements-view", "announcements-create", "announcements-edit",
		"settings-view", "logs-view", "users-view", "archives-view", "archives-create", "archives-edit",
	}
	for _, key := range managerPerms {
		require.NoError(t, db.Create(&models.RolePermission{RoleID: roleIDs["manager"], PermissionID: permIDs[key]}).Error)
	}
	// editor：settings.view + logs.view + announcements 读/改 + archives 读/改
	editorPerms := []string{"announcements-view", "announcements-edit", "settings-view", "logs-view", "archives-view", "archives-edit"}
	for _, key := range editorPerms {
		require.NoError(t, db.Create(&models.RolePermission{RoleID: roleIDs["editor"], PermissionID: permIDs[key]}).Error)
	}
	// viewer：仅 announcements.view + archives.view
	viewerPerms := []string{"announcements-view", "archives-view"}
	for _, key := range viewerPerms {
		require.NoError(t, db.Create(&models.RolePermission{RoleID: roleIDs["viewer"], PermissionID: permIDs[key]}).Error)
	}
	// rbac_op：仅 rbac.manage（用于最后管理员保护测试）
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleIDs["rbac_op"], PermissionID: permIDs["rbac-manage"]}).Error)

	return roleIDs, permIDs
}

// createRBACTestUser 创建用户并分配角色
func createRBACTestUser(t *testing.T, db *gorm.DB, username, supabaseUID string, roleIDs ...uint) models.User {
	t.Helper()
	user := models.User{
		Username:    username,
		Email:       username + "@test.com",
		SupabaseUID: supabaseUID,
		Active:      true,
	}
	require.NoError(t, user.SetPassword("test123"))
	require.NoError(t, db.Create(&user).Error)
	for _, roleID := range roleIDs {
		require.NoError(t, db.Create(&models.UserRole{UserID: user.ID, RoleID: roleID}).Error)
	}
	return user
}

// setupRBACTestEnv 初始化完整测试环境（DB + 种子 + 用户 + 路由）
func setupRBACTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	db := setupTestDB(t)

	// 迁移 RBAC 测试所需全部表
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Role{}, &models.Permission{},
		&models.RolePermission{}, &models.UserRole{},
		&models.Department{}, &models.DepartmentMember{},
		&models.Announcement{}, &models.ModelConfig{},
		&models.StorageConfig{}, &models.SMTPConfig{}, &models.NotificationConfig{},
		&models.AuditLog{}, &models.ArchiveConfig{}, &models.UserPreference{},
		&SystemLog{}, &LogBackup{}, &AlertRule{},
		// 档案配置相关表（供 archives 配置路由 RBAC 测试使用）
		&models.Document{}, &models.DocumentCategory{}, &models.DocumentSubCategory{},
		&models.ArchiveSharedField{}, &models.ArchiveFieldGroup{}, &models.ArchiveFieldDefinition{},
		&models.RetentionPeriod{}, &models.StorageLocation{}, &models.CodeRule{},
		&models.TypeDefaultColumn{},
	))

	roleIDs, permIDs := seedRBACForTest(t, db)

	env := &rbacTestEnv{
		db:      db,
		roleIDs: roleIDs,
		permIDs: permIDs,
	}
	env.admin = createRBACTestUser(t, db, "admin", "admin-uuid", roleIDs["admin"])
	env.manager = createRBACTestUser(t, db, "manager", "manager-uuid", roleIDs["manager"])
	env.editor = createRBACTestUser(t, db, "editor", "editor-uuid", roleIDs["editor"])
	env.viewer = createRBACTestUser(t, db, "viewer", "viewer-uuid", roleIDs["viewer"])
	env.rbacOp = createRBACTestUser(t, db, "rbacop", "rbacop-uuid", roleIDs["rbac_op"])
	env.target = createRBACTestUser(t, db, "target", "target-uuid", roleIDs["viewer"])

	env.router = buildRBACTestRouter(t, db)
	return env
}

// buildRBACTestRouter 模拟 main.go 的路由挂载（Supabase JWT + 权限中间件 + 关键路由）
func buildRBACTestRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	auditService := service.NewAuditService(db)
	handler := NewHandler(db)

	// mock 验证器：Authorization Bearer 值即 supabase_uid，直接映射本地用户
	mockValidate := func(token string) (*supabase.SupabaseJWTClaims, error) {
		return &supabase.SupabaseJWTClaims{Sub: token, Email: token + "@test.com"}, nil
	}

	r := chi.NewRouter()
	r.Use(auditmw.AuditContext(auditService))

	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Group(func(protected chi.Router) {
			protected.Use(supabase.SupabaseJWTMiddlewareWithValidator(db, mockValidate))
			protected.Use(auditmw.DepartmentContext(db))
			protected.Use(auditmw.AuditMiddleware(auditService))

			// 日志（logs.view / logs.manage）
			protected.Mount("/logs", NewLogHandler(db).Routes())

			// 通知（模拟 main.go 挂载：用户通知不加权限，配置/发送挂 notifications.*）
			notificationHandler := NewNotificationHandler(db)
			protected.Route("/notifications", func(notifRouter chi.Router) {
				notifRouter.Get("/", notificationHandler.ListNotifications)
				notifRouter.Put("/{id}/read", notificationHandler.MarkAsRead)
				notifRouter.Route("/configs", func(cr chi.Router) {
					cr.With(auditmw.RequirePermission(db, "notifications", "view")).Get("/", handler.ListNotificationConfigs)
					cr.With(auditmw.RequirePermission(db, "notifications", "manage")).Post("/", handler.CreateNotificationConfig)
				})
				notifRouter.With(auditmw.RequirePermission(db, "notifications", "manage")).Post("/test", handler.TestNotification)
			})

			// 监控（settings.view / settings.edit）
			monHandler := NewMonitoringHandler(db, service.NewMonitoringService(db))
			monHandler.RegisterProtectedMonitoringRoutes(protected)

			// 主路由（含 settings/models、admin、rbac、users、departments、announcements 权限挂载）
			handler.RegisterRoutes(protected)
		})
	})
	return r
}

// doRBACRequest 执行请求并返回响应
func doRBACRequest(t *testing.T, router http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// ---------- 测试用例 ----------

// TestRBACViewerDeniedOnAdminWriteEndpoints viewer 对系统设置/高风险管理写接口一律 403
func TestRBACViewerDeniedOnAdminWriteEndpoints(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.viewer.SupabaseUID

	cases := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"公告创建", http.MethodPost, "/api/announcements", map[string]interface{}{"title": "t", "content": "c"}},
		{"模型配置创建", http.MethodPost, "/api/settings/models", map[string]interface{}{"name": "m"}},
		{"SMTP 保存", http.MethodPut, "/api/admin/smtp", map[string]interface{}{"host": "smtp.test.com"}},
		{"通知发送测试", http.MethodPost, "/api/notifications/test", map[string]interface{}{"channel": "smtp"}},
		{"角色创建", http.MethodPost, "/api/rbac/roles", map[string]interface{}{"name": "ops", "label": "运维"}},
		{"用户角色分配", http.MethodPost, fmt.Sprintf("/api/users/%d/roles", env.target.ID), map[string]interface{}{"role_ids": []uint{env.roleIDs["viewer"]}}},
		{"日志清理", http.MethodPost, "/api/logs/cleanup", map[string]interface{}{"days": 30}},
		{"监控维护", http.MethodPost, "/api/monitoring/maintenance", map[string]interface{}{"action": "restart"}},
		{"通知配置读取", http.MethodPost, "/api/notifications/configs", nil},
		{"存储配置读取", http.MethodPost, "/api/admin/storage", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, tc.method, tc.path, token, tc.body)
			assert.Equal(t, http.StatusForbidden, rec.Code, "viewer 应被拒绝: %s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}

	// viewer 可读公告（announcements.view）
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/announcements", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestRBACSettingsViewReadSuccessWriteFailed 仅 settings.view 角色读成功、写失败
func TestRBACSettingsViewReadSuccessWriteFailed(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.manager.SupabaseUID

	// 读成功
	readCases := []struct {
		name string
		path string
	}{
		{"模型配置列表", "/api/settings/models"},
		{"SMTP 配置", "/api/admin/smtp"},
		{"监控指标", "/api/monitoring/metrics"},
		{"日志列表", "/api/logs"},
		{"存储配置列表", "/api/admin/storage"},
	}
	for _, tc := range readCases {
		t.Run("读-"+tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, http.MethodGet, tc.path, token, nil)
			assert.Equal(t, http.StatusOK, rec.Code, "%s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}

	// 写失败（manager 无 settings.create/edit/delete、logs.manage、notifications.*）
	writeCases := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"模型配置创建", http.MethodPost, "/api/settings/models", map[string]interface{}{"name": "m"}},
		{"SMTP 保存", http.MethodPut, "/api/admin/smtp", map[string]interface{}{"host": "smtp.test.com"}},
		{"监控维护", http.MethodPost, "/api/monitoring/maintenance", map[string]interface{}{"action": "restart"}},
		{"日志清理", http.MethodPost, "/api/logs/cleanup", map[string]interface{}{"days": 30}},
		{"通知配置读取", http.MethodPost, "/api/notifications/configs", nil},
	}
	for _, tc := range writeCases {
		t.Run("写-"+tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, tc.method, tc.path, token, tc.body)
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}
}

// TestRBACAdminWriteSuccess 授权管理员写操作成功
func TestRBACAdminWriteSuccess(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.admin.SupabaseUID

	// 公告创建成功
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/announcements", token,
		map[string]interface{}{"title": "公告", "content": "内容", "status": "published"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 角色创建成功
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/rbac/roles", token,
		map[string]interface{}{"name": "ops", "label": "运维", "description": "运维角色"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 通知配置读取成功
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/notifications/configs", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 日志列表读取成功
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/logs", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestRBACSupabaseUUIDPermissionChain Supabase UUID 用户权限链路（mock 验证器映射本地用户）
func TestRBACSupabaseUUIDPermissionChain(t *testing.T) {
	env := setupRBACTestEnv(t)

	// admin（UUID）写公告成功
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/announcements", env.admin.SupabaseUID,
		map[string]interface{}{"title": "t", "content": "c"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// viewer（UUID）写公告 403
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/announcements", env.viewer.SupabaseUID,
		map[string]interface{}{"title": "t", "content": "c"})
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// 未映射的 UUID 直接 401（禁止自动创建用户）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/announcements", "unknown-uuid", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestRBACRoleSelfLockAndLastAdminProtection 角色自锁 + 最后管理员保护
func TestRBACRoleSelfLockAndLastAdminProtection(t *testing.T) {
	env := setupRBACTestEnv(t)

	// 自锁：admin 尝试移除自己的 admin 角色（改为空角色）→ 403
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/users/%d/roles", env.admin.ID),
		env.admin.SupabaseUID, map[string]interface{}{"role_ids": []uint{}})
	assert.Equal(t, http.StatusForbidden, rec.Code, "自锁保护应拒绝: %s", rec.Body.String())

	// 最后管理员保护：rbacOp（有 rbac.manage 但非 admin）尝试把唯一 admin 降级为 viewer → 403
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/users/%d/roles", env.admin.ID),
		env.rbacOp.SupabaseUID, map[string]interface{}{"role_ids": []uint{env.roleIDs["viewer"]}})
	assert.Equal(t, http.StatusForbidden, rec.Code, "最后管理员保护应拒绝: %s", rec.Body.String())

	// 正常分配不受影响：rbacOp 给 target 分配 viewer 角色 → 成功
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/users/%d/roles", env.target.ID),
		env.rbacOp.SupabaseUID, map[string]interface{}{"role_ids": []uint{env.roleIDs["viewer"]}})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestRBACSystemRoleAndPermissionProtection 系统角色/系统权限保护
func TestRBACSystemRoleAndPermissionProtection(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.admin.SupabaseUID

	// 禁止创建系统角色名
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rbac/roles", token,
		map[string]interface{}{"name": "admin", "label": "伪造管理员"})
	assert.Equal(t, http.StatusForbidden, rec.Code, "系统角色名应被拒绝: %s", rec.Body.String())

	// 系统角色不可修改/删除/改权限
	adminRoleID := env.roleIDs["admin"]
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rbac/roles/%d", adminRoleID), token,
		map[string]interface{}{"label": "改名"})
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/rbac/roles/%d", adminRoleID), token, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rbac/roles/%d/permissions", adminRoleID), token,
		map[string]interface{}{"permission_ids": []uint{env.permIDs["logs-view"]}})
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// 被角色引用的权限不可删除（系统权限保护）
	rec = doRBACRequest(t, env.router, http.MethodDelete,
		fmt.Sprintf("/api/rbac/permissions/%d", env.permIDs["settings-view"]), token, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "被引用权限应禁止删除: %s", rec.Body.String())

	// 自定义角色可正常更新权限，但无效权限 ID 应 400
	rbacOpRoleID := env.roleIDs["rbac_op"]
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rbac/roles/%d/permissions", rbacOpRoleID), token,
		map[string]interface{}{"permission_ids": []uint{99999}})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "无效权限 ID 应 400: %s", rec.Body.String())
}

// TestRBACPersonalNotificationsNotAffected 个人通知读写不受系统设置权限影响
func TestRBACPersonalNotificationsNotAffected(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.viewer.SupabaseUID

	// viewer 无任何系统设置权限，但个人通知列表/已读标记可用
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/notifications", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = doRBACRequest(t, env.router, http.MethodPut, "/api/notifications/5/read", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 通知配置仍受保护（viewer 无 notifications.view）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/notifications/configs", token, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
