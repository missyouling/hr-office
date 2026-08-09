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
