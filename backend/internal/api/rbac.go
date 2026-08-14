package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// systemRoleNames 系统内置角色名，禁止通过 API 创建同名角色（防止越权伪造系统角色）
var systemRoleNames = map[string]bool{
	"admin":       true,
	"super_admin": true,
	"manager":     true,
	"editor":      true,
	"viewer":      true,
	"user":        true,
}

// isSystemRoleName 判断角色名是否为系统内置角色名
func isSystemRoleName(name string) bool {
	return systemRoleNames[strings.ToLower(strings.TrimSpace(name))]
}

// isAdminRoleName 判断角色名是否为管理员级角色（admin / super_admin）
func isAdminRoleName(name string) bool {
	return name == models.RoleAdmin || name == "super_admin"
}

// logRBACAudit 记录 RBAC 业务审计日志（含操作者与变更详情）
func logRBACAudit(r *http.Request, action models.ActionType, resource string, resourceID *string, details map[string]interface{}) {
	auditService := middleware.GetAuditServiceFromContext(r.Context())
	if auditService == nil {
		return
	}
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		userID = 0
	}
	uid := &userID
	auditService.LogAction(r.Context(), models.CreateAuditLogParams{
		UserID:     uid,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Method:     r.Method,
		Path:       r.URL.Path,
		Status:     models.StatusSuccess,
		StatusCode: http.StatusOK,
		Details:    &models.LogDetails{Custom: details},
	})
}

type rolePayload struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type permissionPayload struct {
	Module string `json:"module"`
	Action string `json:"action"`
	Label  string `json:"label"`
}

type rolePermissionPayload struct {
	PermissionIDs []uint `json:"permission_ids"`
}

type userRolePayload struct {
	RoleIDs []uint `json:"role_ids"`
}

func (h *Handler) registerRolePermissionRoutes(r chi.Router) {
	// 全部 RBAC 路由需 rbac.manage 权限（防止普通用户管理角色/权限）
	r.Use(middleware.RequirePermission(h.db, "rbac", "manage"))
	r.Get("/roles", h.listRoles)
	r.Post("/roles", h.createRole)
	r.Put("/roles/{id}", h.updateRole)
	r.Delete("/roles/{id}", h.deleteRole)
	r.Get("/roles/{id}/permissions", h.getRolePermissions)
	r.Put("/roles/{id}/permissions", h.updateRolePermissions)

	r.Get("/permissions", h.listPermissions)
	r.Post("/permissions", h.createPermission)
	r.Delete("/permissions/{id}", h.deletePermission)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	var roles []models.Role
	if err := h.db.Order("is_system DESC, id ASC").Find(&roles).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i := range roles {
		var count int64
		h.db.Model(&models.UserRole{}).Where("role_id = ?", roles[i].ID).Count(&count)
		roles[i].UserCount = int(count)
	}

	writeJSON(w, roles)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var payload rolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// 禁止创建系统内置角色名（防止越权伪造系统角色）
	if isSystemRoleName(payload.Name) {
		respondError(w, http.StatusForbidden, "cannot create system role", nil)
		return
	}

	role := models.Role{
		Name:        payload.Name,
		Label:       payload.Label,
		Description: payload.Description,
		IsSystem:    false,
	}

	if err := h.db.Create(&role).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create role", err)
		return
	}

	roleIDStr := strconv.FormatUint(uint64(role.ID), 10)
	logRBACAudit(r, models.ActionRoleCreate, "roles", &roleIDStr, map[string]interface{}{
		"name":  role.Name,
		"label": role.Label,
	})

	writeJSON(w, role)
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "role not found", err)
		return
	}

	if role.IsSystem {
		respondError(w, http.StatusForbidden, "cannot modify system role", nil)
		return
	}

	var payload rolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	updates := map[string]interface{}{
		"label":       payload.Label,
		"description": payload.Description,
	}

	if err := h.db.Model(&models.Role{}).Where("id = ? AND is_system = ?", id, false).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update role", err)
		return
	}

	h.db.First(&role, id)
	roleIDStr := strconv.FormatUint(id, 10)
	logRBACAudit(r, models.ActionRoleUpdate, "roles", &roleIDStr, map[string]interface{}{
		"label":       role.Label,
		"description": role.Description,
	})
	writeJSON(w, role)
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "role not found", err)
		return
	}

	if role.IsSystem {
		respondError(w, http.StatusForbidden, "cannot delete system role", nil)
		return
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", id).Delete(&models.RolePermission{})
		tx.Where("role_id = ?", id).Delete(&models.UserRole{})
		tx.Delete(&role)
		return nil
	})

	roleIDStr := strconv.FormatUint(id, 10)
	logRBACAudit(r, models.ActionRoleDelete, "roles", &roleIDStr, map[string]interface{}{
		"name": role.Name,
	})

	writeJSON(w, map[string]string{"message": "deleted"})
}

func (h *Handler) getRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var permissions []models.Permission
	if err := h.db.
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", id).
		Find(&permissions).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, permissions)
}

func (h *Handler) updateRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "role not found", err)
		return
	}

	if role.IsSystem {
		respondError(w, http.StatusForbidden, "cannot modify system role permissions", nil)
		return
	}

	var payload rolePermissionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// PermissionIDs 必须全部存在，防止写入无效权限关联
	if len(payload.PermissionIDs) > 0 {
		var count int64
		if err := h.db.Model(&models.Permission{}).Where("id IN ?", payload.PermissionIDs).Count(&count).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to validate permissions", err)
			return
		}
		if count != int64(len(payload.PermissionIDs)) {
			respondError(w, http.StatusBadRequest, "some permission ids do not exist", nil)
			return
		}
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", id).Delete(&models.RolePermission{})
		for _, permID := range payload.PermissionIDs {
			tx.Create(&models.RolePermission{RoleID: uint(id), PermissionID: permID})
		}
		return nil
	})

	roleIDStr := strconv.FormatUint(id, 10)
	logRBACAudit(r, models.ActionRolePermUpdate, "roles", &roleIDStr, map[string]interface{}{
		"permission_ids": payload.PermissionIDs,
	})

	writeJSON(w, map[string]string{"message": "permissions updated"})
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	var permissions []models.Permission
	if err := h.db.Order("module ASC, sort_order ASC").Find(&permissions).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, permissions)
}

func (h *Handler) createPermission(w http.ResponseWriter, r *http.Request) {
	var payload permissionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	permission := models.Permission{
		Module: payload.Module,
		Action: payload.Action,
		Label:  payload.Label,
	}

	if err := h.db.Create(&permission).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create permission", err)
		return
	}

	permIDStr := strconv.FormatUint(uint64(permission.ID), 10)
	logRBACAudit(r, models.ActionPermissionCreate, "permissions", &permIDStr, map[string]interface{}{
		"module": permission.Module,
		"action": permission.Action,
	})

	writeJSON(w, permission)
}

func (h *Handler) deletePermission(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var permission models.Permission
	if err := h.db.First(&permission, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "permission not found", err)
		return
	}

	// 系统权限保护：被任一角色引用的权限不可删除（种子权限均被 admin 角色引用）
	var refCount int64
	if err := h.db.Model(&models.RolePermission{}).Where("permission_id = ?", id).Count(&refCount).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to check permission usage", err)
		return
	}
	if refCount > 0 {
		respondError(w, http.StatusForbidden, "cannot delete permission in use by roles", nil)
		return
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("permission_id = ?", id).Delete(&models.RolePermission{})
		tx.Delete(&models.Permission{}, id)
		return nil
	})

	permIDStr := strconv.FormatUint(id, 10)
	logRBACAudit(r, models.ActionPermissionDelete, "permissions", &permIDStr, map[string]interface{}{
		"module": permission.Module,
		"action": permission.Action,
	})

	writeJSON(w, map[string]string{"message": "deleted"})
}

func (h *Handler) registerUserRoleRoutes(r chi.Router) {
	// 查看用户角色需 users.view；分配角色属 RBAC 敏感操作需 rbac.manage
	r.With(middleware.RequirePermission(h.db, "users", "view")).Get("/{id}/roles", h.getUserRoles)
	r.With(middleware.RequirePermission(h.db, "rbac", "manage")).Post("/{id}/roles", h.assignUserRoles)
}

func (h *Handler) assignUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		respondError(w, http.StatusNotFound, "user not found", err)
		return
	}

	var payload userRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// 过滤掉不存在的角色
	var validRoleIDs []uint
	if len(payload.RoleIDs) > 0 {
		var existingRoles []models.Role
		if err := h.db.Where("id IN ?", payload.RoleIDs).Find(&existingRoles).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to validate roles", err)
			return
		}
		roleIDSet := make(map[uint]struct{}, len(existingRoles))
		for _, role := range existingRoles {
			roleIDSet[role.ID] = struct{}{}
		}
		for _, id := range payload.RoleIDs {
			if _, ok := roleIDSet[id]; ok {
				validRoleIDs = append(validRoleIDs, id)
			}
		}
	}

	// 角色自锁保护：操作者不能移除自己的最后管理员角色
	operatorID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if uint(userID) == operatorID {
		hasAdminNow := userHasAdminRole(h.db, uint(userID))
		hasAdminAfter := roleIDsContainAdmin(h.db, validRoleIDs)
		if hasAdminNow && !hasAdminAfter {
			respondError(w, http.StatusForbidden, "cannot remove your own last admin role", nil)
			return
		}
	}

	// 最后管理员保护：禁止移除系统中最后一个 admin/super_admin 用户的管理员角色
	targetHasAdmin := userHasAdminRole(h.db, uint(userID))
	if targetHasAdmin && !roleIDsContainAdmin(h.db, validRoleIDs) {
		var otherAdminCount int64
		h.db.Model(&models.Role{}).
			Joins("JOIN user_roles ON user_roles.role_id = roles.id").
			Where("roles.name IN ? AND user_roles.user_id != ?", []string{models.RoleAdmin, "super_admin"}, userID).
			Distinct("user_roles.user_id").
			Count(&otherAdminCount)
		if otherAdminCount == 0 {
			respondError(w, http.StatusForbidden, "cannot remove the last admin user", nil)
			return
		}
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range validRoleIDs {
			if err := tx.Create(&models.UserRole{UserID: uint(userID), RoleID: roleID}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	userIDStr := strconv.FormatUint(userID, 10)
	logRBACAudit(r, models.ActionUserRoleUpdate, "users", &userIDStr, map[string]interface{}{
		"role_ids": validRoleIDs,
	})

	writeJSON(w, map[string]interface{}{
		"message":  "roles updated",
		"role_ids": validRoleIDs,
	})
}

// userHasAdminRole 判断用户是否拥有 admin / super_admin 角色
func userHasAdminRole(db *gorm.DB, userID uint) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name IN ?", userID, []string{models.RoleAdmin, "super_admin"}).
		Count(&count)
	return count > 0
}

// roleIDsContainAdmin 判断角色 ID 列表中是否包含 admin / super_admin 角色
func roleIDsContainAdmin(db *gorm.DB, roleIDs []uint) bool {
	if len(roleIDs) == 0 {
		return false
	}
	var count int64
	db.Model(&models.Role{}).
		Where("id IN ? AND name IN ?", roleIDs, []string{models.RoleAdmin, "super_admin"}).
		Count(&count)
	return count > 0
}

func (h *Handler) getUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		respondError(w, http.StatusNotFound, "user not found", err)
		return
	}

	var roles []models.Role
	if err := h.db.
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list user roles", err)
		return
	}

	writeJSON(w, roles)
}

// ========== 部门管理 CRUD（P7.1 新增）==========

type departmentPayload struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
	Code     string `json:"code"`
}

type departmentMemberPayload struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"` // leader / member
}

func (h *Handler) registerDepartmentRoutes(r chi.Router) {
	// 查看部门需 users.view；写操作属 RBAC 敏感操作需 rbac.manage
	r.With(middleware.RequirePermission(h.db, "users", "view")).Get("/", h.listDepartments)
	r.With(middleware.RequirePermission(h.db, "rbac", "manage")).Post("/", h.createDepartment)
	r.With(middleware.RequirePermission(h.db, "rbac", "manage")).Put("/{id}", h.updateDepartment)
	r.With(middleware.RequirePermission(h.db, "rbac", "manage")).Delete("/{id}", h.deleteDepartment)
	r.With(middleware.RequirePermission(h.db, "rbac", "manage")).Post("/{id}/members", h.assignUserToDepartment)
	r.With(middleware.RequirePermission(h.db, "users", "view")).Get("/{id}/members", h.listDepartmentMembers)
}

// listDepartments 获取部门列表（支持按 UserID 多租户过滤）
func (h *Handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	query := h.db.Order("id ASC")
	// 多租户：如果请求中包含 user_id 查询参数，按此过滤
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		query = query.Where("user_id = ?", userIDStr)
	}

	var departments []models.Department
	if err := query.Find(&departments).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list departments", err)
		return
	}
	writeJSON(w, departments)
}

// createDepartment 创建部门
func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var payload departmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if payload.Name == "" {
		respondError(w, http.StatusBadRequest, "部门名称不能为空", nil)
		return
	}

	dept := models.Department{
		Name:     payload.Name,
		ParentID: payload.ParentID,
		Code:     payload.Code,
	}

	if err := h.db.Create(&dept).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create department", err)
		return
	}
	writeJSON(w, dept)
}

// updateDepartment 更新部门信息
func (h *Handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var dept models.Department
	if err := h.db.First(&dept, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "department not found", err)
		return
	}

	var payload departmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	updates := map[string]interface{}{
		"name":      payload.Name,
		"parent_id": payload.ParentID,
		"code":      payload.Code,
	}
	if err := h.db.Model(&dept).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update department", err)
		return
	}

	h.db.First(&dept, id)
	writeJSON(w, dept)
}

// deleteDepartment 删除部门（清理关联成员）
func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("department_id = ?", id).Delete(&models.DepartmentMember{})
		tx.Delete(&models.Department{}, id)
		return nil
	})

	writeJSON(w, map[string]string{"message": "deleted"})
}

// assignUserToDepartment 将用户分配到部门
func (h *Handler) assignUserToDepartment(w http.ResponseWriter, r *http.Request) {
	deptID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid department id", err)
		return
	}

	var payload departmentMemberPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// 验证部门存在
	var dept models.Department
	if err := h.db.First(&dept, deptID).Error; err != nil {
		respondError(w, http.StatusNotFound, "department not found", err)
		return
	}

	// 验证用户存在
	var user models.User
	if err := h.db.First(&user, payload.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "user not found", err)
		return
	}

	// 更新用户的 DepartmentID
	if err := h.db.Model(&user).Update("department_id", deptID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to assign user to department", err)
		return
	}

	// 创建部门成员记录（upsert：先删后建）
	member := models.DepartmentMember{
		DepartmentID: uint(deptID),
		UserID:       payload.UserID,
		Role:         payload.Role,
	}
	h.db.Where("department_id = ? AND user_id = ?", deptID, payload.UserID).Delete(&models.DepartmentMember{})
	if err := h.db.Create(&member).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create department member", err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"message":       "user assigned to department",
		"department_id": deptID,
		"user_id":       payload.UserID,
	})
}

// listDepartmentMembers 获取部门成员列表
func (h *Handler) listDepartmentMembers(w http.ResponseWriter, r *http.Request) {
	deptID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid department id", err)
		return
	}

	var members []models.DepartmentMember
	if err := h.db.Where("department_id = ?", deptID).Find(&members).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list department members", err)
		return
	}
	writeJSON(w, members)
}
