package service

import (
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============================================================
// 知识库访问权限（P9.1）
// 检索层需要按"用户有权限的 KB 集合"过滤检索范围（kb_id=0 时）。
// 本实现与 api.HasAccess 语义保持一致；因 api 包依赖 service 包，
// 无法反向引用，故在此提供等价实现并保持同步演进。
// ============================================================

// AccessibleKBIDs 返回用户有权限的知识库 ID 集合（空/无权限时返回空切片）
func AccessibleKBIDs(db *gorm.DB, userID uint) []uint {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil
	}

	var kbs []models.KnowledgeBase
	if err := db.Order("id ASC").Find(&kbs).Error; err != nil {
		return nil
	}

	ids := make([]uint, 0, len(kbs))
	for _, kb := range kbs {
		if hasKBAccess(db, &user, kb) {
			ids = append(ids, kb.ID)
		}
	}
	return ids
}

// hasKBAccess 判断用户对单个知识库的访问权限（与 api.HasAccess 语义一致）
// 规则优先级：admin/super_admin 全量放行 > public+employee 全员可见 > private 仅所有者 > restricted 访问规则 OR 组合
func hasKBAccess(db *gorm.DB, user *models.User, kb models.KnowledgeBase) bool {
	// admin / super_admin 统一全量放行（不限 visibility/owner/规则）
	if userIsAdmin(db, user.ID) {
		return true
	}

	// 公开 + 员工花名册模块：全员可见
	if kb.Visibility == "public" && kb.SourceModule == "employee" {
		return true
	}

	// 私有知识库：仅所有者可访问
	if kb.Visibility == "private" {
		return kb.OwnerID != nil && *kb.OwnerID == user.ID
	}

	// 受限知识库：检查访问规则（OR 组合）
	var rules []models.KBAccessRule
	db.Where("knowledge_base_id = ?", kb.ID).Find(&rules)
	for _, r := range rules {
		if r.RoleLevel != nil && *r.RoleLevel != "" {
			if !userHasRole(db, user.ID, *r.RoleLevel) {
				continue
			}
		}
		if r.DepartmentID != nil {
			if user.DepartmentID == nil || *r.DepartmentID != *user.DepartmentID {
				continue
			}
		}
		if r.UserID != nil && *r.UserID != user.ID {
			continue
		}
		return true
	}
	return false
}

// userHasRole 判断用户是否拥有指定角色（通过 user_roles 联表查询）
func userHasRole(db *gorm.DB, userID uint, roleLevel string) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleLevel).
		Count(&count)
	return count > 0
}

// userIsAdmin 判断用户是否为管理员（admin / super_admin 兼容）
// 管理员在 kb_id=0 检索时全量可见（见 docScopeSQL）
func userIsAdmin(db *gorm.DB, userID uint) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name IN (?, ?)", userID, models.RoleAdmin, "super_admin").
		Count(&count)
	return count > 0
}
