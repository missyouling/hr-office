package main

import (
	"testing"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

var invoiceActions = []string{"view", "create", "edit", "delete", "submit", "approve", "reject"}

func setupRBACDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.Permission{}, &models.RolePermission{}); err != nil {
		t.Fatalf("迁移 RBAC 表失败: %v", err)
	}
	return db
}

func TestEnsurePermissions_InvoicePermissionsAreIdempotent(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("首次初始化权限失败: %v", err)
	}

	assertInvoicePermissions(t, db)
	before := countRolePermissions(t, db, models.RoleManager)
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("重复初始化权限失败: %v", err)
	}

	var invoiceCount int64
	db.Model(&models.Permission{}).Where("module = ?", "invoice").Count(&invoiceCount)
	if invoiceCount != int64(len(invoiceActions)) {
		t.Errorf("重复初始化后 invoice 权限数应为 %d，实际为 %d", len(invoiceActions), invoiceCount)
	}
	if after := countRolePermissions(t, db, models.RoleManager); after != before {
		t.Errorf("重复初始化后 manager 权限数应为 %d，实际为 %d", before, after)
	}
	assertInvoiceRoleMatrix(t, db)
}

func TestSeedRolePermissions_SuperAdminGetsAllPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	for _, action := range invoiceActions {
		if !roleHasPermission(t, db, superAdminRole, action) {
			t.Errorf("super_admin 应拥有 invoice-%s，实际未分配", action)
		}
	}
}

// TestEnsurePermissions_AdminContractPermissions 行政合同权限（P12.3.5）应注册且幂等：
// admin/super_admin 全量（view/create/edit/delete），manager 不含 delete，
// editor 不含 create/delete，viewer 不得持有 view（E2E 验收：viewer 无行政合同入口）。
func TestEnsurePermissions_AdminContractPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("首次初始化权限失败: %v", err)
	}
	assertAdminContractPermissions(t, db)
	before := countRolePermissions(t, db, models.RoleManager)
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("重复初始化权限失败: %v", err)
	}
	if after := countRolePermissions(t, db, models.RoleManager); after != before {
		t.Errorf("重复初始化后 manager 权限数应为 %d，实际为 %d", before, after)
	}
	assertAdminContractPermissions(t, db)
}

// assertAdminContractPermissions 校验 admin_contract 权限注册与角色分配矩阵。
func assertAdminContractPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "admin_contract", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 admin_contract-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "admin_contract", action) {
			t.Errorf("admin 应拥有 admin_contract-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "admin_contract", action) {
			t.Errorf("super_admin 应拥有 admin_contract-%s", action)
		}
	}
	// manager：view/create/edit，无 delete
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "admin_contract", action) {
			t.Errorf("manager 应拥有 admin_contract-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "admin_contract", "delete") {
		t.Error("manager 不应拥有 admin_contract-delete")
	}
	// editor：view/edit，无 create/delete
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "admin_contract", action) {
			t.Errorf("editor 应拥有 admin_contract-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleEditor, "admin_contract", "create") || roleHasModulePermission(t, db, models.RoleEditor, "admin_contract", "delete") {
		t.Error("editor 不应拥有 admin_contract-create/delete")
	}
	// viewer：必须无 view（E2E 验收：viewer 无行政合同入口）
	if roleHasModulePermission(t, db, models.RoleViewer, "admin_contract", "view") {
		t.Error("viewer 不应拥有 admin_contract-view")
	}
}

// TestViewerRoleDefaults_NoContractView 回归测试：viewer 角色默认权限矩阵不得包含
// contract-view / admin_contract-view / reward-view / occupational_health-view（与 backend/rbac_seed.go 的 viewer 分配保持一致）。
// 防止未来在 seed-e2e 或 rbac_seed.go 中重新给 viewer 分配这些权限，
// 破坏「无对应 view 权限的 viewer 无入口」的 E2E 验收。
func TestViewerRoleDefaults_NoContractView(t *testing.T) {
	viewerPerms := rolePermissionDefaults[models.RoleViewer]
	for _, key := range []string{"contract-view", "admin_contract-view", "reward-view", "occupational_health-view"} {
		for _, perm := range viewerPerms {
			if perm == key {
				t.Errorf("viewer 默认权限矩阵不应包含 %s（E2E 验收：viewer 无对应入口）", key)
			}
		}
	}
}

// TestEnsureViewerNoAdminContractPermission_RemovesLegacy 历史残留的 viewer admin_contract-view 应被幂等移除。
func TestEnsureViewerNoAdminContractPermission_RemovesLegacy(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	// 模拟生产 rbac_seed.go 曾给 viewer 分配的 admin_contract-view（历史残留）
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		t.Fatalf("查找 viewer 角色失败: %v", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "admin_contract", "view").First(&perm).Error; err != nil {
		t.Fatalf("查找 admin_contract-view 权限失败: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("构造历史残留失败: %v", err)
	}
	if err := ensureViewerNoAdminContractPermission(db); err != nil {
		t.Fatalf("移除 viewer admin_contract-view 失败: %v", err)
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "admin_contract", "view") {
		t.Error("移除后 viewer 不应再拥有 admin_contract-view")
	}
	// 幂等：再次执行不报错
	if err := ensureViewerNoAdminContractPermission(db); err != nil {
		t.Fatalf("重复移除应幂等，实际报错: %v", err)
	}
}

// TestEnsurePermissions_OccupationalHealthPermissions 职业卫生检查权限（P12 最小真实功能）应注册且幂等：
// admin/super_admin 全量（view/create/edit/delete），manager 不含 delete，
// editor 不含 create/delete，viewer 不得持有 view（E2E 验收：viewer 无职业卫生检查入口）。
func TestEnsurePermissions_OccupationalHealthPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("首次初始化权限失败: %v", err)
	}
	assertOccupationalHealthPermissions(t, db)
	before := countRolePermissions(t, db, models.RoleManager)
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("重复初始化权限失败: %v", err)
	}
	if after := countRolePermissions(t, db, models.RoleManager); after != before {
		t.Errorf("重复初始化后 manager 权限数应为 %d，实际为 %d", before, after)
	}
	assertOccupationalHealthPermissions(t, db)
}

// assertOccupationalHealthPermissions 校验 occupational_health 权限注册与角色分配矩阵。
func assertOccupationalHealthPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "occupational_health", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 occupational_health-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "occupational_health", action) {
			t.Errorf("admin 应拥有 occupational_health-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "occupational_health", action) {
			t.Errorf("super_admin 应拥有 occupational_health-%s", action)
		}
	}
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "occupational_health", action) {
			t.Errorf("manager 应拥有 occupational_health-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "occupational_health", "delete") {
		t.Error("manager 不应拥有 occupational_health-delete")
	}
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "occupational_health", action) {
			t.Errorf("editor 应拥有 occupational_health-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleEditor, "occupational_health", "create") || roleHasModulePermission(t, db, models.RoleEditor, "occupational_health", "delete") {
		t.Error("editor 不应拥有 occupational_health-create/delete")
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "occupational_health", "view") {
		t.Error("viewer 不应拥有 occupational_health-view")
	}
}

// TestEnsureViewerNoOccupationalHealthPermission_RemovesLegacy 历史残留的 viewer occupational_health-view 应被幂等移除。
func TestEnsureViewerNoOccupationalHealthPermission_RemovesLegacy(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		t.Fatalf("查找 viewer 角色失败: %v", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "occupational_health", "view").First(&perm).Error; err != nil {
		t.Fatalf("查找 occupational_health-view 权限失败: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("构造历史残留失败: %v", err)
	}
	if err := ensureViewerNoOccupationalHealthPermission(db); err != nil {
		t.Fatalf("移除 viewer occupational_health-view 失败: %v", err)
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "occupational_health", "view") {
		t.Error("移除后 viewer 不应再拥有 occupational_health-view")
	}
	if err := ensureViewerNoOccupationalHealthPermission(db); err != nil {
		t.Fatalf("重复移除应幂等，实际报错: %v", err)
	}
}

func TestOccupationalHealthPermissionSortOrder(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	expected := map[string]int{
		"safety-view":                 106,
		"safety-create":               107,
		"safety-edit":                 108,
		"safety-delete":               109,
		"fleet-view":                  110,
		"fleet-create":                111,
		"fleet-edit":                  112,
		"fleet-delete":                113,
		"occupational_health-view":    114,
		"occupational_health-create":  115,
		"occupational_health-edit":    116,
		"occupational_health-delete":  117,
	}
	for key, sortOrder := range expected {
		module, action, _ := strings.Cut(key, "-")
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", module, action).First(&permission).Error; err != nil {
			t.Fatalf("查询权限 %s 失败: %v", key, err)
		}
		if permission.SortOrder != sortOrder {
			t.Fatalf("权限 %s 的排序应为 %d，实际为 %d", key, sortOrder, permission.SortOrder)
		}
	}
}

// TestEnsurePermissions_RewardPermissions 奖惩记录权限（P12.3.6）应注册且幂等：
// admin/super_admin 全量（view/create/edit/delete），manager 不含 delete，
// editor 不含 create/delete，viewer 不得持有 view（E2E 验收：viewer 无奖惩记录入口）。
func TestEnsurePermissions_RewardPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("首次初始化权限失败: %v", err)
	}
	assertRewardPermissions(t, db)
	before := countRolePermissions(t, db, models.RoleManager)
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("重复初始化权限失败: %v", err)
	}
	if after := countRolePermissions(t, db, models.RoleManager); after != before {
		t.Errorf("重复初始化后 manager 权限数应为 %d，实际为 %d", before, after)
	}
	assertRewardPermissions(t, db)
}

// assertRewardPermissions 校验 reward 权限注册与角色分配矩阵。
func assertRewardPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "reward", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 reward-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "reward", action) {
			t.Errorf("admin 应拥有 reward-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "reward", action) {
			t.Errorf("super_admin 应拥有 reward-%s", action)
		}
	}
	// manager：view/create/edit，无 delete
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "reward", action) {
			t.Errorf("manager 应拥有 reward-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "reward", "delete") {
		t.Error("manager 不应拥有 reward-delete")
	}
	// editor：view/edit，无 create/delete
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "reward", action) {
			t.Errorf("editor 应拥有 reward-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleEditor, "reward", "create") || roleHasModulePermission(t, db, models.RoleEditor, "reward", "delete") {
		t.Error("editor 不应拥有 reward-create/delete")
	}
	// viewer：必须无 view（E2E 验收：viewer 无奖惩记录入口）
	if roleHasModulePermission(t, db, models.RoleViewer, "reward", "view") {
		t.Error("viewer 不应拥有 reward-view")
	}
}

// assertTrainingPermissions 校验 training 权限注册与角色分配矩阵。
func assertTrainingPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "training", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 training-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "training", action) {
			t.Errorf("admin 应拥有 training-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "training", action) {
			t.Errorf("super_admin 应拥有 training-%s", action)
		}
	}
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "training", action) {
			t.Errorf("manager 应拥有 training-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "training", "delete") {
		t.Error("manager 不应拥有 training-delete")
	}
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "training", action) {
			t.Errorf("editor 应拥有 training-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "training", "view") {
		t.Error("viewer 不应拥有 training-view")
	}
}

func TestTrainingPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	assertTrainingPermissions(t, db)
}

// assertSafetyPermissions 校验 safety 权限注册与角色分配矩阵。
func assertSafetyPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "safety", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 safety-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "safety", action) {
			t.Errorf("admin 应拥有 safety-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "safety", action) {
			t.Errorf("super_admin 应拥有 safety-%s", action)
		}
	}
	// manager：view/create/edit，无 delete
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "safety", action) {
			t.Errorf("manager 应拥有 safety-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "safety", "delete") {
		t.Error("manager 不应拥有 safety-delete")
	}
	// editor：view/edit，无 create/delete
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "safety", action) {
			t.Errorf("editor 应拥有 safety-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleEditor, "safety", "create") || roleHasModulePermission(t, db, models.RoleEditor, "safety", "delete") {
		t.Error("editor 不应拥有 safety-create/delete")
	}
	// viewer：必须无 view（E2E 验收：viewer 无安全检查入口）
	if roleHasModulePermission(t, db, models.RoleViewer, "safety", "view") {
		t.Error("viewer 不应拥有 safety-view")
	}
}

// TestSafetyPermissions 校验 safety 权限注册与角色分配矩阵。
func TestSafetyPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	assertSafetyPermissions(t, db)
}

// assertFleetPermissions 校验 fleet 权限注册与角色分配矩阵。
func assertFleetPermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "fleet", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 fleet-%s 权限: %v", action, err)
		}
		if !roleHasModulePermission(t, db, models.RoleAdmin, "fleet", action) {
			t.Errorf("admin 应拥有 fleet-%s", action)
		}
		if !roleHasModulePermission(t, db, superAdminRole, "fleet", action) {
			t.Errorf("super_admin 应拥有 fleet-%s", action)
		}
	}
	// manager：view/create/edit，无 delete
	for _, action := range []string{"view", "create", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleManager, "fleet", action) {
			t.Errorf("manager 应拥有 fleet-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleManager, "fleet", "delete") {
		t.Error("manager 不应拥有 fleet-delete")
	}
	// editor：view/edit，无 create/delete
	for _, action := range []string{"view", "edit"} {
		if !roleHasModulePermission(t, db, models.RoleEditor, "fleet", action) {
			t.Errorf("editor 应拥有 fleet-%s", action)
		}
	}
	if roleHasModulePermission(t, db, models.RoleEditor, "fleet", "create") || roleHasModulePermission(t, db, models.RoleEditor, "fleet", "delete") {
		t.Error("editor 不应拥有 fleet-create/delete")
	}
	// viewer：必须无 view（E2E 验收：viewer 无车队管理入口）
	if roleHasModulePermission(t, db, models.RoleViewer, "fleet", "view") {
		t.Error("viewer 不应拥有 fleet-view")
	}
}

// TestFleetPermissions 校验 fleet 权限注册与角色分配矩阵。
func TestFleetPermissions(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	assertFleetPermissions(t, db)
}

// TestEnsureViewerNoFleetPermission_RemovesLegacy 历史残留的 viewer fleet-view 应被幂等移除。
func TestEnsureViewerNoFleetPermission_RemovesLegacy(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	// 模拟历史残留：给 viewer 分配 fleet-view
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		t.Fatalf("查找 viewer 角色失败: %v", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "fleet", "view").First(&perm).Error; err != nil {
		t.Fatalf("查找 fleet-view 权限失败: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("构造历史残留失败: %v", err)
	}
	if err := ensureViewerNoFleetPermission(db); err != nil {
		t.Fatalf("移除 viewer fleet-view 失败: %v", err)
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "fleet", "view") {
		t.Error("移除后 viewer 不应再拥有 fleet-view")
	}
	// 幂等：再次执行不报错
	if err := ensureViewerNoFleetPermission(db); err != nil {
		t.Fatalf("重复移除应幂等，实际报错: %v", err)
	}
}

// TestEnsureViewerNoSafetyPermission_RemovesLegacy 历史残留的 viewer safety-view 应被幂等移除。
func TestEnsureViewerNoSafetyPermission_RemovesLegacy(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	// 模拟历史残留：给 viewer 分配 safety-view
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		t.Fatalf("查找 viewer 角色失败: %v", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "safety", "view").First(&perm).Error; err != nil {
		t.Fatalf("查找 safety-view 权限失败: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("构造历史残留失败: %v", err)
	}
	if err := ensureViewerNoSafetyPermission(db); err != nil {
		t.Fatalf("移除 viewer safety-view 失败: %v", err)
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "safety", "view") {
		t.Error("移除后 viewer 不应再拥有 safety-view")
	}
	// 幂等：再次执行不报错
	if err := ensureViewerNoSafetyPermission(db); err != nil {
		t.Fatalf("重复移除应幂等，实际报错: %v", err)
	}
}

// TestEnsureViewerNoRewardPermission_RemovesLegacy 历史残留的 viewer reward-view 应被幂等移除。
func TestEnsureViewerNoRewardPermission_RemovesLegacy(t *testing.T) {
	db := setupRBACDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("初始化角色失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("初始化权限失败: %v", err)
	}
	// 模拟生产 rbac_seed.go 曾给 viewer 分配的 reward-view（历史残留）
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		t.Fatalf("查找 viewer 角色失败: %v", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "reward", "view").First(&perm).Error; err != nil {
		t.Fatalf("查找 reward-view 权限失败: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("构造历史残留失败: %v", err)
	}
	if err := ensureViewerNoRewardPermission(db); err != nil {
		t.Fatalf("移除 viewer reward-view 失败: %v", err)
	}
	if roleHasModulePermission(t, db, models.RoleViewer, "reward", "view") {
		t.Error("移除后 viewer 不应再拥有 reward-view")
	}
	// 幂等：再次执行不报错
	if err := ensureViewerNoRewardPermission(db); err != nil {
		t.Fatalf("重复移除应幂等，实际报错: %v", err)
	}
}

func assertInvoicePermissions(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range invoiceActions {
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", "invoice", action).First(&permission).Error; err != nil {
			t.Errorf("缺少 invoice-%s 权限: %v", action, err)
		}
	}
}

func assertInvoiceRoleMatrix(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, action := range invoiceActions {
		if !roleHasPermission(t, db, models.RoleAdmin, action) {
			t.Errorf("admin 应拥有 invoice-%s", action)
		}
		if !roleHasPermission(t, db, models.RoleManager, action) {
			t.Errorf("manager 应拥有 invoice-%s", action)
		}
	}
	for _, action := range []string{"view", "create", "edit", "delete", "submit"} {
		if !roleHasPermission(t, db, models.RoleEditor, action) {
			t.Errorf("editor 应拥有 invoice-%s", action)
		}
	}
	if roleHasPermission(t, db, models.RoleEditor, "approve") || roleHasPermission(t, db, models.RoleEditor, "reject") {
		t.Error("editor 不应拥有 invoice 审批或驳回权限")
	}
	if !roleHasPermission(t, db, models.RoleViewer, "view") {
		t.Error("viewer 应拥有 invoice-view")
	}
	if roleHasPermission(t, db, models.RoleViewer, "create") {
		t.Error("viewer 不应拥有 invoice-create")
	}
}

func roleHasPermission(t *testing.T, db *gorm.DB, roleName, action string) bool {
	t.Helper()
	return roleHasModulePermission(t, db, roleName, "invoice", action)
}

// roleHasModulePermission 按 (角色, 模块, 动作) 精确判断角色是否持有指定权限关联。
func roleHasModulePermission(t *testing.T, db *gorm.DB, roleName, module, action string) bool {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("查询角色 %s 失败: %v", roleName, err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", module, action).First(&permission).Error; err != nil {
		return false
	}
	var assignment models.RolePermission
	return db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).First(&assignment).Error == nil
}

func countRolePermissions(t *testing.T, db *gorm.DB, roleName string) int64 {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("查询角色 %s 失败: %v", roleName, err)
	}
	var count int64
	db.Model(&models.RolePermission{}).Where("role_id = ?", role.ID).Count(&count)
	return count
}
