package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupTestDB 创建内存 SQLite 用于测试，并迁移 RBAC 相关表
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.Permission{}, &models.RolePermission{}, &models.UserRole{}); err != nil {
		t.Fatalf("自动建表失败: %v", err)
	}
	return db
}

// roleHasPerm 检查角色是否拥有指定 module-action 权限
func roleHasPerm(t *testing.T, db *gorm.DB, roleName, module, action string) bool {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("查找角色 %s 失败: %v", roleName, err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", module, action).First(&perm).Error; err != nil {
		return false
	}
	var rp models.RolePermission
	err := db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&rp).Error
	return err == nil
}

// countPerms 统计某角色的权限数量
func countPerms(t *testing.T, db *gorm.DB, roleName string) int64 {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("查找角色 %s 失败: %v", roleName, err)
	}
	var n int64
	db.Model(&models.RolePermission{}).Where("role_id = ?", role.ID).Count(&n)
	return n
}

// 发票模块权限动作清单（与前端权限矩阵一致）
var invoiceActions = []string{"view", "create", "edit", "delete", "submit", "approve", "reject"}

func TestSeedPermissions_CreatesInvoicePermissions(t *testing.T) {
	db := setupTestDB(t)
	if err := seedPermissions(db); err != nil {
		t.Fatalf("seedPermissions 失败: %v", err)
	}

	// 发票模块的 7 个动作权限都必须存在
	for _, action := range invoiceActions {
		var perm models.Permission
		if err := db.Where("module = ? AND action = ?", "invoice", action).First(&perm).Error; err != nil {
			t.Errorf("缺少权限 invoice-%s: %v", action, err)
		}
	}
}

func TestSeedPermissions_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	if err := seedPermissions(db); err != nil {
		t.Fatalf("首次 seedPermissions 失败: %v", err)
	}
	if err := seedPermissions(db); err != nil {
		t.Fatalf("重复 seedPermissions 应幂等无副作用，实际报错: %v", err)
	}

	// 重复执行后 invoice 权限不应重复创建
	var n int64
	db.Model(&models.Permission{}).Where("module = ?", "invoice").Count(&n)
	if n != int64(len(invoiceActions)) {
		t.Errorf("期望 invoice 权限数为 %d，实际为 %d（不幂等）", len(invoiceActions), n)
	}
}

func TestSeedRolePermissions_InvoiceMatrix(t *testing.T) {
	db := setupTestDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("ensureRoles 失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("ensurePermissions 失败: %v", err)
	}

	// 各角色应获得的发票权限（与前端权限矩阵一致）
	cases := []struct {
		role    string
		allowed []string
	}{
		{models.RoleAdmin, invoiceActions},
		{models.RoleManager, invoiceActions},
		{models.RoleEditor, []string{"view", "create", "edit", "delete", "submit"}},
		{models.RoleViewer, []string{"view"}},
	}
	for _, c := range cases {
		for _, action := range c.allowed {
			if !roleHasPerm(t, db, c.role, "invoice", action) {
				t.Errorf("角色 %s 应拥有 invoice-%s，实际未分配", c.role, action)
			}
		}
	}

	// 负例：不应被越权授予
	forbidden := []struct {
		role   string
		action string
	}{
		{models.RoleEditor, "approve"},
		{models.RoleEditor, "reject"},
		{models.RoleViewer, "create"},
		{models.RoleViewer, "approve"},
	}
	for _, f := range forbidden {
		if roleHasPerm(t, db, f.role, "invoice", f.action) {
			t.Errorf("角色 %s 不应拥有 invoice-%s，实际被分配", f.role, f.action)
		}
	}
}

func TestSeedRolePermissions_AdminGetsAllViaAssignAll(t *testing.T) {
	db := setupTestDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("ensureRoles 失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("ensurePermissions 失败: %v", err)
	}

	// admin 通过 assignAllToRole 自动获得全部权限（含 invoice）
	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if got := countPerms(t, db, models.RoleAdmin); got != permCount {
		t.Errorf("admin 应获得全部 %d 个权限，实际 %d 个", permCount, got)
	}
}

func TestSeedRolePermissions_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	if err := ensureRoles(db); err != nil {
		t.Fatalf("ensureRoles 失败: %v", err)
	}
	if err := ensurePermissions(db); err != nil {
		t.Fatalf("首次 ensurePermissions 失败: %v", err)
	}
	before := countPerms(t, db, models.RoleManager)

	if err := ensurePermissions(db); err != nil {
		t.Fatalf("重复 ensurePermissions 应幂等无副作用，实际报错: %v", err)
	}
	after := countPerms(t, db, models.RoleManager)
	if after != before {
		t.Errorf("重复执行后 manager 权限数应不变（%d），实际为 %d（不幂等）", before, after)
	}
}
