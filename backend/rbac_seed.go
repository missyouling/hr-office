package main

import (
	"log"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func seedRBAC(db *gorm.DB) error {
	roles := []models.Role{
		{Name: "user", Label: "普通用户", Description: "基本功能访问", IsSystem: true},
		{Name: "admin", Label: "管理员", Description: "系统管理权限", IsSystem: true},
		{Name: "super_admin", Label: "超级管理员", Description: "完整控制权限", IsSystem: true},
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

	// Assign all permissions to super_admin
	var superAdminRole models.Role
	if err := db.Where("name = ?", "super_admin").First(&superAdminRole).Error; err == nil {
		var allPerms []models.Permission
		db.Find(&allPerms)
		for _, p := range allPerms {
			var existing models.RolePermission
			if err := db.Where("role_id = ? AND permission_id = ?", superAdminRole.ID, p.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
				db.Create(&models.RolePermission{RoleID: superAdminRole.ID, PermissionID: p.ID})
			}
		}
	}

	// Assign limited permissions to admin
	var adminRole models.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
		adminPerms := []string{
			"employee-view", "employee-create", "employee-edit", "employee-delete",
			"insurance-view", "insurance-create", "insurance-edit",
			"dormitory-view", "dormitory-create", "dormitory-edit", "dormitory-delete",
			"archives-view", "archives-create", "archives-edit", "archives-delete",
			"settings-view", "settings-edit",
			"announcements-view", "announcements-create", "announcements-edit",
			"backups-view", "backups-create",
			"users-view",
		}
		for _, ap := range adminPerms {
			var perm models.Permission
			db.Where("module = ? AND action = ?", ap[:len(ap)-5], ap[len(ap)-4:]).First(&perm)
			if perm.ID > 0 {
				var existing models.RolePermission
				if err := db.Where("role_id = ? AND permission_id = ?", adminRole.ID, perm.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
					db.Create(&models.RolePermission{RoleID: adminRole.ID, PermissionID: perm.ID})
				}
			}
		}
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
		for _, up := range userPerms {
			var perm models.Permission
			db.Where("module = ? AND action = ?", up[:len(up)-5], up[len(up)-4:]).First(&perm)
			if perm.ID > 0 {
				var existing models.RolePermission
				if err := db.Where("role_id = ? AND permission_id = ?", userRole.ID, perm.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
					db.Create(&models.RolePermission{RoleID: userRole.ID, PermissionID: perm.ID})
				}
			}
		}
	}

	log.Println("RBAC data seeded successfully")
	return nil
}
