package main

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// superAdminRole 为向后兼容保留的超级管理员角色名（映射为 admin）。
const superAdminRole = "super_admin"

// defaultPermissions 定义 E2E 环境所需的全部默认 RBAC 权限。
var defaultPermissions = []models.Permission{
	{Module: "employee", Action: "view", Label: "查看", SortOrder: 1},
	{Module: "employee", Action: "create", Label: "创建", SortOrder: 2},
	{Module: "employee", Action: "edit", Label: "编辑", SortOrder: 3},
	{Module: "employee", Action: "delete", Label: "删除", SortOrder: 4},
	{Module: "insurance", Action: "view", Label: "查看", SortOrder: 10},
	{Module: "insurance", Action: "create", Label: "创建", SortOrder: 11},
	{Module: "insurance", Action: "edit", Label: "编辑", SortOrder: 12},
	{Module: "insurance", Action: "delete", Label: "删除", SortOrder: 13},
	{Module: "dormitory", Action: "view", Label: "查看", SortOrder: 20},
	{Module: "dormitory", Action: "create", Label: "创建", SortOrder: 21},
	{Module: "dormitory", Action: "edit", Label: "编辑", SortOrder: 22},
	{Module: "dormitory", Action: "delete", Label: "删除", SortOrder: 23},
	{Module: "archives", Action: "view", Label: "查看", SortOrder: 30},
	{Module: "archives", Action: "create", Label: "创建", SortOrder: 31},
	{Module: "archives", Action: "edit", Label: "编辑", SortOrder: 32},
	{Module: "archives", Action: "delete", Label: "删除", SortOrder: 33},
	{Module: "settings", Action: "view", Label: "查看", SortOrder: 40},
	{Module: "settings", Action: "create", Label: "创建", SortOrder: 41},
	{Module: "settings", Action: "edit", Label: "编辑", SortOrder: 42},
	{Module: "settings", Action: "delete", Label: "删除", SortOrder: 43},
	{Module: "announcements", Action: "view", Label: "查看", SortOrder: 50},
	{Module: "announcements", Action: "create", Label: "创建", SortOrder: 51},
	{Module: "announcements", Action: "edit", Label: "编辑", SortOrder: 52},
	{Module: "announcements", Action: "delete", Label: "删除", SortOrder: 53},
	{Module: "backups", Action: "view", Label: "查看", SortOrder: 60},
	{Module: "backups", Action: "create", Label: "创建", SortOrder: 61},
	{Module: "backups", Action: "edit", Label: "编辑", SortOrder: 62},
	{Module: "backups", Action: "delete", Label: "删除", SortOrder: 63},
	{Module: "users", Action: "view", Label: "查看", SortOrder: 70},
	{Module: "users", Action: "create", Label: "创建", SortOrder: 71},
	{Module: "users", Action: "edit", Label: "编辑", SortOrder: 72},
	{Module: "users", Action: "delete", Label: "删除", SortOrder: 73},
	{Module: "invoice", Action: "view", Label: "查看", SortOrder: 80},
	{Module: "invoice", Action: "create", Label: "创建", SortOrder: 81},
	{Module: "invoice", Action: "edit", Label: "编辑", SortOrder: 82},
	{Module: "invoice", Action: "delete", Label: "删除", SortOrder: 83},
	{Module: "invoice", Action: "submit", Label: "提交", SortOrder: 84},
	{Module: "invoice", Action: "approve", Label: "审批", SortOrder: 85},
	{Module: "invoice", Action: "reject", Label: "驳回", SortOrder: 86},
}

// ensurePermissions 幂等创建默认权限并为核心角色补齐默认授权。
func ensurePermissions(db *gorm.DB) error {
	if err := seedPermissions(db); err != nil {
		return err
	}
	return seedRolePermissions(db)
}

func seedPermissions(db *gorm.DB) error {
	for _, permission := range defaultPermissions {
		var existing models.Permission
		err := db.Where("module = ? AND action = ?", permission.Module, permission.Action).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询权限 %s-%s 失败: %w", permission.Module, permission.Action, err)
		}
		if err := db.Create(&permission).Error; err != nil {
			return fmt.Errorf("创建权限 %s-%s 失败: %w", permission.Module, permission.Action, err)
		}
	}
	return nil
}

func seedRolePermissions(db *gorm.DB) error {
	// 确保 super_admin 兼容角色存在（对齐 rbac_seed.go 的向后兼容策略）
	if err := ensureSuperAdminRole(db); err != nil {
		return err
	}
	// admin 与 super_admin 均通过 assignAllToRole 获得全部权限
	for _, roleName := range []string{models.RoleAdmin, superAdminRole} {
		if err := assignAllToRole(db, roleName); err != nil {
			return err
		}
	}
	for roleName, permissions := range rolePermissionDefaults {
		if err := assignPermsToRole(db, roleName, permissions); err != nil {
			return err
		}
	}
	return nil
}

// ensureSuperAdminRole 幂等创建 super_admin 兼容角色（映射为 admin）。
func ensureSuperAdminRole(db *gorm.DB) error {
	var existing models.Role
	err := db.Where("name = ?", superAdminRole).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询 super_admin 角色失败: %w", err)
	}
	role := models.Role{
		Name:        superAdminRole,
		Label:       "超级管理员（兼容）",
		Description: "完整控制权限（映射为 admin）",
		IsSystem:    true,
	}
	if err := db.Create(&role).Error; err != nil {
		return fmt.Errorf("创建 super_admin 角色失败: %w", err)
	}
	return nil
}

var rolePermissionDefaults = map[string][]string{
	models.RoleManager: {
		"employee-view", "employee-create", "employee-edit", "insurance-view", "insurance-create", "insurance-edit",
		"dormitory-view", "dormitory-create", "dormitory-edit", "archives-view", "archives-create", "archives-edit",
		"announcements-view", "announcements-create", "announcements-edit", "settings-view", "backups-view", "users-view",
		"invoice-view", "invoice-create", "invoice-edit", "invoice-delete", "invoice-submit", "invoice-approve", "invoice-reject",
	},
	models.RoleEditor: {
		"employee-view", "employee-edit", "insurance-view", "insurance-edit", "dormitory-view", "dormitory-edit",
		"archives-view", "archives-edit", "announcements-view", "announcements-edit", "settings-view", "backups-view",
		"invoice-view", "invoice-create", "invoice-edit", "invoice-delete", "invoice-submit",
	},
	models.RoleViewer: {
		"employee-view", "insurance-view", "dormitory-view", "archives-view", "announcements-view", "invoice-view",
	},
}

func assignAllToRole(db *gorm.DB, roleName string) error {
	var permissions []models.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return fmt.Errorf("查询权限列表失败: %w", err)
	}
	return assignPermissions(db, roleName, permissions)
}

func assignPermsToRole(db *gorm.DB, roleName string, keys []string) error {
	permissions := make([]models.Permission, 0, len(keys))
	for _, key := range keys {
		module, action, ok := strings.Cut(key, "-")
		if !ok {
			return fmt.Errorf("无效权限键 %q", key)
		}
		var permission models.Permission
		if err := db.Where("module = ? AND action = ?", module, action).First(&permission).Error; err != nil {
			return fmt.Errorf("查询权限 %s 失败: %w", key, err)
		}
		permissions = append(permissions, permission)
	}
	return assignPermissions(db, roleName, permissions)
}

func assignPermissions(db *gorm.DB, roleName string, permissions []models.Permission) error {
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("查找角色 %s 失败: %w", roleName, err)
	}
	for _, permission := range permissions {
		var existing models.RolePermission
		err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询角色 %s 的权限失败: %w", roleName, err)
		}
		if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
			return fmt.Errorf("分配权限给角色 %s 失败: %w", roleName, err)
		}
	}
	return nil
}
