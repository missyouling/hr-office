package main

import (
	"testing"

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
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("查询角色 %s 失败: %v", roleName, err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", "invoice", action).First(&permission).Error; err != nil {
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
