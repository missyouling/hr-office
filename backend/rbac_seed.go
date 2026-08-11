package main

import (
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func seedRBAC(db *gorm.DB) error {
	// 4 角色体系（P7.1）：admin / manager / editor / viewer
	// 保留 super_admin 和 user 作为向后兼容（不再作为主角色新分配）
	roles := []models.Role{
		{Name: models.RoleAdmin, Label: "管理员", Description: "系统管理权限", IsSystem: true},
		{Name: "super_admin", Label: "超级管理员（兼容）", Description: "完整控制权限（映射为 admin）", IsSystem: true},
		{Name: models.RoleManager, Label: "部门经理", Description: "本部门全模块查看、创建、编辑", IsSystem: true},
		{Name: models.RoleEditor, Label: "编辑者", Description: "全模块查看、编辑（不可创建/删除）", IsSystem: true},
		{Name: models.RoleViewer, Label: "只读用户", Description: "仅查看数据", IsSystem: true},
		{Name: "user", Label: "普通用户（兼容）", Description: "基本功能访问（旧版角色）", IsSystem: true},
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

	// manager: 所有业务模块查看、创建、编辑（不可删除）
	// 部门经理可管理本部门的完整数据生命周期
	var managerRole models.Role
	if err := db.Where("name = ?", models.RoleManager).First(&managerRole).Error; err == nil {
		managerPerms := []string{
			"employee-view", "employee-create", "employee-edit",
			"insurance-view", "insurance-create", "insurance-edit",
			"dormitory-view", "dormitory-create", "dormitory-edit",
			"archives-view", "archives-create", "archives-edit",
			"announcements-view", "announcements-create", "announcements-edit",
			"settings-view",
			"backups-view",
			"users-view",
		}
		assignPermissionsToRole(db, managerRole.ID, managerPerms)
	}

	// editor: 所有业务模块查看和编辑（不可创建、不可删除）
	var editorRole models.Role
	if err := db.Where("name = ?", models.RoleEditor).First(&editorRole).Error; err == nil {
		editorPerms := []string{
			"employee-view", "employee-edit",
			"insurance-view", "insurance-edit",
			"dormitory-view", "dormitory-edit",
			"archives-view", "archives-edit",
			"announcements-view", "announcements-edit",
			"settings-view",
			"backups-view",
		}
		assignPermissionsToRole(db, editorRole.ID, editorPerms)
	}

	// viewer: 仅查看所有业务模块
	var viewerRole models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&viewerRole).Error; err == nil {
		viewerPerms := []string{
			"employee-view",
			"insurance-view",
			"dormitory-view",
			"archives-view",
			"announcements-view",
			"settings-view",
			"backups-view",
		}
		assignPermissionsToRole(db, viewerRole.ID, viewerPerms)
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

// seedDepartments 初始化默认部门数据（总经办 / 业务部）
func seedDepartments(db *gorm.DB) error {
	depts := []models.Department{
		{Name: "总经办", Code: "HQ", ParentID: nil},
		{Name: "业务部", Code: "BIZ", ParentID: nil},
	}

	var createdIDs []uint
	for _, dept := range depts {
		var existing models.Department
		if err := db.Where("name = ?", dept.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&dept).Error; err != nil {
				log.Printf("failed to create department %s: %v", dept.Name, err)
				continue
			}
			createdIDs = append(createdIDs, dept.ID)
		}
	}

	// 将 admin 用户分配到总经办（如果存在 admin 用户）
	if len(createdIDs) > 0 {
		var adminUser models.User
		if err := db.Where("username = ?", "admin").First(&adminUser).Error; err == nil {
			var dept models.Department
			if err := db.Where("name = ?", "总经办").First(&dept).Error; err == nil {
				// 更新用户的 DepartmentID
				db.Model(&adminUser).Update("department_id", dept.ID)
				// 创建部门成员记录
				member := models.DepartmentMember{
					DepartmentID: dept.ID,
					UserID:       adminUser.ID,
					Role:         "leader",
					JoinedAt:     time.Now(),
				}
				var existingMember models.DepartmentMember
				if err := db.Where("department_id = ? AND user_id = ?", dept.ID, adminUser.ID).First(&existingMember).Error; err == gorm.ErrRecordNotFound {
					db.Create(&member)
				}
			}
		}
	}

	log.Println("Department seed completed")
	return nil
}
