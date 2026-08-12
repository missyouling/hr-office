package api

import (
	"fmt"

	"gorm.io/gorm"

	"siapp/internal/models"
)

func uintPointer(v uint) *uint {
	value := v
	return &value
}

// loadUserPermissions 从 user_roles → role_permissions → permissions 联表查询用户拥有的所有权限
// 返回扁平化字符串数组，格式为 "module.action"，如 ["employee.view", "employee.edit"]
// 前端可直接使用 .includes() 做权限判断
func loadUserPermissions(db *gorm.DB, userID uint) ([]string, error) {
	type permResult struct {
		Module string
		Action string
	}
	var results []permResult
	// 通过 user_roles JOIN role_permissions JOIN permissions 三联表查询
	// 注意：PostgreSQL 要求 DISTINCT 时 ORDER BY 列必须出现在 SELECT 列表中，
	// 因此这里显式带上 sort_order（Scan 只映射 Module/Action 字段，多余列被忽略）
	err := db.Model(&models.Permission{}).
		Select("DISTINCT permissions.module, permissions.action, permissions.sort_order").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Order("permissions.sort_order ASC").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}

	perms := make([]string, 0, len(results))
	for _, r := range results {
		perms = append(perms, fmt.Sprintf("%s.%s", r.Module, r.Action))
	}
	return perms, nil
}
