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
	// 劳动合同管理（P12.3.2 劳动合同批次，对齐 backend/rbac_seed.go 的 SortOrder 90-93）
	{Module: "contract", Action: "view", Label: "查看", SortOrder: 90},
	{Module: "contract", Action: "create", Label: "创建", SortOrder: 91},
	{Module: "contract", Action: "edit", Label: "编辑", SortOrder: 92},
	{Module: "contract", Action: "delete", Label: "删除", SortOrder: 93},
	// 行政合同管理（P12.3.5 行政合同批次，对齐 backend/rbac_seed.go 的 SortOrder 94-97）
	{Module: "admin_contract", Action: "view", Label: "查看", SortOrder: 94},
	{Module: "admin_contract", Action: "create", Label: "创建", SortOrder: 95},
	{Module: "admin_contract", Action: "edit", Label: "编辑", SortOrder: 96},
	{Module: "admin_contract", Action: "delete", Label: "删除", SortOrder: 97},
	// 奖惩记录管理（P12.3.6 奖惩记录批次，对齐 backend/rbac_seed.go 的 SortOrder 98-101）
	{Module: "reward", Action: "view", Label: "查看", SortOrder: 98},
	{Module: "reward", Action: "create", Label: "创建", SortOrder: 99},
	{Module: "reward", Action: "edit", Label: "编辑", SortOrder: 100},
	{Module: "reward", Action: "delete", Label: "删除", SortOrder: 101},
	{Module: "training", Action: "view", Label: "查看", SortOrder: 102},
	{Module: "training", Action: "create", Label: "创建", SortOrder: 103},
	{Module: "training", Action: "edit", Label: "编辑", SortOrder: 104},
	{Module: "training", Action: "delete", Label: "删除", SortOrder: 105},
	// 安全管理（P12.3.9 安全检查批次，对齐 backend/rbac_seed.go 的 SortOrder 106-109）
	{Module: "safety", Action: "view", Label: "查看", SortOrder: 106},
	{Module: "safety", Action: "create", Label: "创建", SortOrder: 107},
	{Module: "safety", Action: "edit", Label: "编辑", SortOrder: 108},
	{Module: "safety", Action: "delete", Label: "删除", SortOrder: 109},
	// 车队管理（P12 车队管理最小真实功能，对齐 backend/rbac_seed.go 的 SortOrder 110-113）
	{Module: "fleet", Action: "view", Label: "查看", SortOrder: 110},
	{Module: "fleet", Action: "create", Label: "创建", SortOrder: 111},
	{Module: "fleet", Action: "edit", Label: "编辑", SortOrder: 112},
	{Module: "fleet", Action: "delete", Label: "删除", SortOrder: 113},
	// 职业卫生检查台账（P12 最小真实功能，对齐 backend/rbac_seed.go 的 SortOrder 114-117）
	{Module: "occupational_health", Action: "view", Label: "查看", SortOrder: 114},
	{Module: "occupational_health", Action: "create", Label: "创建", SortOrder: 115},
	{Module: "occupational_health", Action: "edit", Label: "编辑", SortOrder: 116},
	{Module: "occupational_health", Action: "delete", Label: "删除", SortOrder: 117},
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
	// E2E 验收约束：viewer 必须无 contract.view（劳动合同入口基于 contract.view）。
	// assignPermsToRole 只增不减，历史库可能残留 viewer 的 contract-view（如生产 rbac_seed.go 曾分配），
	// 此处幂等移除，保证「无 contract.view 的 viewer 无劳动合同入口」验收稳定成立。
	if err := ensureViewerNoContractPermission(db); err != nil {
		return err
	}
	// E2E 验收约束：viewer 必须无 admin_contract.view（行政合同入口基于 admin_contract.view）。
	// 与劳动合同同理：生产 rbac_seed.go 曾分配 viewer 的 admin_contract-view，此处幂等移除，
	// 保证「无 admin_contract.view 的 viewer 无行政合同入口」验收稳定成立。
	if err := ensureViewerNoAdminContractPermission(db); err != nil {
		return err
	}
	// E2E 验收约束：viewer 必须无 reward.view（奖惩记录入口基于 reward.view）。
	// 与劳动合同同理：生产 rbac_seed.go 曾分配 viewer 的 reward-view，此处幂等移除，
	// 保证「无 reward.view 的 viewer 无奖惩记录入口」验收稳定成立。
	if err := ensureViewerNoRewardPermission(db); err != nil {
		return err
	}
	if err := ensureViewerNoTrainingPermission(db); err != nil {
		return err
	}
	if err := ensureViewerNoOccupationalHealthPermission(db); err != nil {
		return err
	}
	// E2E 验收约束：viewer 必须无 safety.view（安全检查入口基于 safety.view）。
	if err := ensureViewerNoSafetyPermission(db); err != nil {
		return err
	}
	// E2E 验收约束：viewer 必须无 fleet.view（车队管理入口基于 fleet.view）。
	if err := ensureViewerNoFleetPermission(db); err != nil {
		return err
	}
	return nil
}

// ensureViewerNoRewardPermission 幂等移除 viewer 角色的 reward-view 权限。
// 仅操作精确匹配 (viewer 角色, reward-view 权限) 的关联记录，不触碰其他角色/权限。
func ensureViewerNoRewardPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "reward", "view").First(&perm).Error; err != nil {
		return fmt.Errorf("查找 reward-view 权限失败: %w", err)
	}
	res := db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Delete(&models.RolePermission{})
	if res.Error != nil {
		return fmt.Errorf("移除 viewer 的 reward-view 权限失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		fmt.Printf("  [移除] viewer 角色 reward-view 权限（E2E 验收：viewer 无奖惩记录入口）\n")
	}
	return nil
}

// ensureViewerNoTrainingPermission 幂等移除 viewer 的 training-view 权限。
func ensureViewerNoTrainingPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", "training", "view").First(&permission).Error; err != nil {
		return fmt.Errorf("查找 training-view 权限失败: %w", err)
	}
	if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).Delete(&models.RolePermission{}).Error; err != nil {
		return fmt.Errorf("移除 viewer 的 training-view 权限失败: %w", err)
	}
	return nil
}

// ensureViewerNoOccupationalHealthPermission 幂等移除 viewer 的 occupational_health-view 权限。
func ensureViewerNoOccupationalHealthPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", "occupational_health", "view").First(&permission).Error; err != nil {
		return fmt.Errorf("查找 occupational_health-view 权限失败: %w", err)
	}
	if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).Delete(&models.RolePermission{}).Error; err != nil {
		return fmt.Errorf("移除 viewer 的 occupational_health-view 权限失败: %w", err)
	}
	return nil
}

// ensureViewerNoSafetyPermission 幂等移除 viewer 的 safety-view 权限。
func ensureViewerNoSafetyPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", "safety", "view").First(&permission).Error; err != nil {
		return fmt.Errorf("查找 safety-view 权限失败: %w", err)
	}
	if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).Delete(&models.RolePermission{}).Error; err != nil {
		return fmt.Errorf("移除 viewer 的 safety-view 权限失败: %w", err)
	}
	return nil
}

// ensureViewerNoFleetPermission 幂等移除 viewer 的 fleet-view 权限。
// E2E 验收约束：viewer 必须无 fleet.view（车队管理入口基于 fleet.view）。
func ensureViewerNoFleetPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var permission models.Permission
	if err := db.Where("module = ? AND action = ?", "fleet", "view").First(&permission).Error; err != nil {
		return fmt.Errorf("查找 fleet-view 权限失败: %w", err)
	}
	if err := db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).Delete(&models.RolePermission{}).Error; err != nil {
		return fmt.Errorf("移除 viewer 的 fleet-view 权限失败: %w", err)
	}
	return nil
}

// ensureViewerNoAdminContractPermission 幂等移除 viewer 角色的 admin_contract-view 权限。
// 仅操作精确匹配 (viewer 角色, admin_contract-view 权限) 的关联记录，不触碰其他角色/权限。
func ensureViewerNoAdminContractPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "admin_contract", "view").First(&perm).Error; err != nil {
		return fmt.Errorf("查找 admin_contract-view 权限失败: %w", err)
	}
	res := db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Delete(&models.RolePermission{})
	if res.Error != nil {
		return fmt.Errorf("移除 viewer 的 admin_contract-view 权限失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		fmt.Printf("  [移除] viewer 角色 admin_contract-view 权限（E2E 验收：viewer 无行政合同入口）\n")
	}
	return nil
}

// ensureViewerNoContractPermission 幂等移除 viewer 角色的 contract-view 权限。
// 仅操作精确匹配 (viewer 角色, contract-view 权限) 的关联记录，不触碰其他角色/权限。
func ensureViewerNoContractPermission(db *gorm.DB) error {
	var role models.Role
	if err := db.Where("name = ?", models.RoleViewer).First(&role).Error; err != nil {
		return fmt.Errorf("查找 viewer 角色失败: %w", err)
	}
	var perm models.Permission
	if err := db.Where("module = ? AND action = ?", "contract", "view").First(&perm).Error; err != nil {
		return fmt.Errorf("查找 contract-view 权限失败: %w", err)
	}
	res := db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Delete(&models.RolePermission{})
	if res.Error != nil {
		return fmt.Errorf("移除 viewer 的 contract-view 权限失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		fmt.Printf("  [移除] viewer 角色 contract-view 权限（E2E 验收：viewer 无劳动合同入口）\n")
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
		// 劳动合同：manager 可查看/创建/编辑（不可作废/删除，对齐生产 rbac_seed.go）
		"contract-view", "contract-create", "contract-edit",
		// 行政合同：manager 可查看/创建/编辑（不可作废/删除，对齐生产 rbac_seed.go）
		"admin_contract-view", "admin_contract-create", "admin_contract-edit",
		// 奖惩记录：manager 可查看/创建/编辑（不可作废/删除，对齐生产 rbac_seed.go）
		"reward-view", "reward-create", "reward-edit",
		"training-view", "training-create", "training-edit",
		"occupational_health-view", "occupational_health-create", "occupational_health-edit",
		// 安全管理：manager 可查看/创建/编辑（不可作废/删除，对齐生产 rbac_seed.go）
		"safety-view", "safety-create", "safety-edit",
		// 车队管理：manager 可查看/创建/编辑（不可删除，对齐生产 rbac_seed.go）
		"fleet-view", "fleet-create", "fleet-edit",
	},
	models.RoleEditor: {
		"employee-view", "employee-edit", "insurance-view", "insurance-edit", "dormitory-view", "dormitory-edit",
		"archives-view", "archives-edit", "announcements-view", "announcements-edit", "settings-view", "backups-view",
		"invoice-view", "invoice-create", "invoice-edit", "invoice-delete", "invoice-submit",
		// 劳动合同：editor 可查看/编辑（不可创建/作废，对齐生产 rbac_seed.go）
		"contract-view", "contract-edit",
		// 行政合同：editor 可查看/编辑（不可创建/作废，对齐生产 rbac_seed.go）
		"admin_contract-view", "admin_contract-edit",
		// 奖惩记录：editor 可查看/编辑（不可创建/作废，对齐生产 rbac_seed.go）
		"reward-view", "reward-edit",
		"training-view", "training-edit",
		"occupational_health-view", "occupational_health-edit",
		// 安全管理：editor 可查看/编辑（不可创建/作废，对齐生产 rbac_seed.go）
		"safety-view", "safety-edit",
		// 车队管理：editor 可查看/编辑（不可创建/删除，对齐生产 rbac_seed.go）
		"fleet-view", "fleet-edit",
	},
	models.RoleViewer: {
		"employee-view", "insurance-view", "dormitory-view", "archives-view", "announcements-view", "invoice-view",
		// 注意：viewer 不分配 contract-view —— E2E 验收要求「无 contract.view 的 viewer 无劳动合同入口」
		// 同时不分配 admin_contract-view —— E2E 验收要求「无 admin_contract.view 的 viewer 无行政合同入口」
		// 同时不分配 reward-view —— E2E 验收要求「无 reward.view 的 viewer 无奖惩记录入口」
		// 同时不分配 occupational_health-view —— E2E 验收要求「无 occupational_health.view 的 viewer 无职业卫生检查入口」
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
