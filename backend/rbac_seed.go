package main

import (
	"log"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func seedRBAC(db *gorm.DB) error {
	roles := []models.Role{
		{Name: "user", Label: "普通用户", Description: "基本功能访问", IsSystem: true},
		{Name: "admin", Label: "管理员", Description: "系统管理权限", IsSystem: true},
		{Name: "super_admin", Label: "超级管理员", Description: "完整控制权限", IsSystem: true},
		{Name: "manager", Label: "部门经理", Description: "查看和编辑员工、档案数据", IsSystem: true},
		{Name: "editor", Label: "编辑者", Description: "查看和编辑业务数据", IsSystem: true},
		{Name: "viewer", Label: "只读用户", Description: "仅查看数据", IsSystem: true},
	}

	for _, role := range roles {
		var existing models.Role
		if err := db.Where("name = ?", role.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&role).Error; err != nil {
				log.Printf("failed to create role %s: %v", role.Name, err)
			}
		}
	}

	permissions := []models.Permission{
		// 员工管理
		{Module: "employee", Action: "view", Label: "查看", SortOrder: 1},
		{Module: "employee", Action: "create", Label: "创建", SortOrder: 2},
		{Module: "employee", Action: "edit", Label: "编辑", SortOrder: 3},
		{Module: "employee", Action: "delete", Label: "删除", SortOrder: 4},
		// 社保管理
		{Module: "insurance", Action: "view", Label: "查看", SortOrder: 10},
		{Module: "insurance", Action: "create", Label: "创建", SortOrder: 11},
		{Module: "insurance", Action: "edit", Label: "编辑", SortOrder: 12},
		{Module: "insurance", Action: "delete", Label: "删除", SortOrder: 13},
		// 宿舍管理
		{Module: "dormitory", Action: "view", Label: "查看", SortOrder: 20},
		{Module: "dormitory", Action: "create", Label: "创建", SortOrder: 21},
		{Module: "dormitory", Action: "edit", Label: "编辑", SortOrder: 22},
		{Module: "dormitory", Action: "delete", Label: "删除", SortOrder: 23},
		// 档案管理
		{Module: "archives", Action: "view", Label: "查看", SortOrder: 30},
		{Module: "archives", Action: "create", Label: "创建", SortOrder: 31},
		{Module: "archives", Action: "edit", Label: "编辑", SortOrder: 32},
		{Module: "archives", Action: "delete", Label: "删除", SortOrder: 33},
		// 系统设置
		{Module: "settings", Action: "view", Label: "查看", SortOrder: 40},
		{Module: "settings", Action: "create", Label: "创建", SortOrder: 41},
		{Module: "settings", Action: "edit", Label: "编辑", SortOrder: 42},
		{Module: "settings", Action: "delete", Label: "删除", SortOrder: 43},
		// 公告管理
		{Module: "announcements", Action: "view", Label: "查看", SortOrder: 50},
		{Module: "announcements", Action: "create", Label: "创建", SortOrder: 51},
		{Module: "announcements", Action: "edit", Label: "编辑", SortOrder: 52},
		{Module: "announcements", Action: "delete", Label: "删除", SortOrder: 53},
		// 备份管理
		{Module: "backups", Action: "view", Label: "查看", SortOrder: 60},
		{Module: "backups", Action: "create", Label: "创建", SortOrder: 61},
		{Module: "backups", Action: "edit", Label: "编辑", SortOrder: 62},
		{Module: "backups", Action: "delete", Label: "删除", SortOrder: 63},
		// 用户管理
		{Module: "users", Action: "view", Label: "查看", SortOrder: 70},
		{Module: "users", Action: "create", Label: "创建", SortOrder: 71},
		{Module: "users", Action: "edit", Label: "编辑", SortOrder: 72},
		{Module: "users", Action: "delete", Label: "删除", SortOrder: 73},
	}

	for _, perm := range permissions {
		var existing models.Permission
		if err := db.Where("module = ? AND action = ?", perm.Module, perm.Action).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&perm).Error; err != nil {
				log.Printf("failed to create permission %s-%s: %v", perm.Module, perm.Action, err)
			}
		}
	}

	// Assign all permissions to super_admin and admin
	for _, roleName := range []string{"super_admin", "admin"} {
		var role models.Role
		if err := db.Where("name = ?", roleName).First(&role).Error; err == nil {
			var allPerms []models.Permission
			db.Find(&allPerms)
			for _, p := range allPerms {
				var existing models.RolePermission
				if err := db.Where("role_id = ? AND permission_id = ?", role.ID, p.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
					db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: p.ID})
				}
			}
		}
	}

	// manager: 查看/编辑员工和档案
	var managerRole models.Role
	if err := db.Where("name = ?", "manager").First(&managerRole).Error; err == nil {
		managerPerms := []string{
			"employee-view", "employee-create", "employee-edit",
			"archives-view", "archives-create", "archives-edit",
		}
		assignPermissionsToRole(db, managerRole.ID, managerPerms)
	}

	// editor: 仅查看和编辑（所有业务模块的 view + edit）
	var editorRole models.Role
	if err := db.Where("name = ?", "editor").First(&editorRole).Error; err == nil {
		editorPerms := []string{
			"employee-view", "employee-edit",
			"insurance-view", "insurance-edit",
			"dormitory-view", "dormitory-edit",
			"archives-view", "archives-edit",
			"announcements-view", "announcements-edit",
			"settings-view", "settings-edit",
		}
		assignPermissionsToRole(db, editorRole.ID, editorPerms)
	}

	// viewer: 仅查看（所有业务模块的 view）
	var viewerRole models.Role
	if err := db.Where("name = ?", "viewer").First(&viewerRole).Error; err == nil {
		viewerPerms := []string{
			"employee-view",
			"insurance-view",
			"dormitory-view",
			"archives-view",
			"announcements-view",
			"settings-view",
		}
		assignPermissionsToRole(db, viewerRole.ID, viewerPerms)
	}

	// Assign limited permissions to user
	var userRole models.Role
	if err := db.Where("name = ?", "user").First(&userRole).Error; err == nil {
		userPerms := []string{
			"employee-view",
			"insurance-view",
			"dormitory-view",
			"archives-view", "archives-create",
			"announcements-view",
		}
		assignPermissionsToRole(db, userRole.ID, userPerms)
	}

	log.Println("RBAC data seeded successfully")
	return nil
}

// assignPermissionsToRole 根据 module-action 字符串列表为角色分配权限
func assignPermissionsToRole(db *gorm.DB, roleID uint, perms []string) {
	for _, permKey := range perms {
		parts := strings.SplitN(permKey, "-", 2)
		if len(parts) != 2 {
			continue
		}
		module, action := parts[0], parts[1]
		var perm models.Permission
		db.Where("module = ? AND action = ?", module, action).First(&perm)
		if perm.ID > 0 {
			var existing models.RolePermission
			if err := db.Where("role_id = ? AND permission_id = ?", roleID, perm.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
				db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID})
			}
		}
	}
}
