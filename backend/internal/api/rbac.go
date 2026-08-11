package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/models"
)

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

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", id).Delete(&models.RolePermission{})
		for _, permID := range payload.PermissionIDs {
			tx.Create(&models.RolePermission{RoleID: uint(id), PermissionID: permID})
		}
		return nil
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

	writeJSON(w, permission)
}

func (h *Handler) deletePermission(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("permission_id = ?", id).Delete(&models.RolePermission{})
		tx.Delete(&models.Permission{}, id)
		return nil
	})

	writeJSON(w, map[string]string{"message": "deleted"})
}

func (h *Handler) registerUserRoleRoutes(r chi.Router) {
	r.Post("/{id}/roles", h.assignUserRoles)
	r.Get("/{id}/roles", h.getUserRoles)
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

	writeJSON(w, map[string]interface{}{
		"message":  "roles updated",
		"role_ids": validRoleIDs,
	})
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
	r.Get("/", h.listDepartments)
	r.Post("/", h.createDepartment)
	r.Put("/{id}", h.updateDepartment)
	r.Delete("/{id}", h.deleteDepartment)
	r.Post("/{id}/members", h.assignUserToDepartment)
	r.Get("/{id}/members", h.listDepartmentMembers)
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
