package middleware

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// userDepartmentKey 用于在 context 中存储当前用户所属部门
type userDepartmentKey string

const (
	// UserDepartmentContextKey 部门隔离 context key
	UserDepartmentContextKey userDepartmentKey = "user_department"
)

// RequirePermission 检查用户是否有指定模块和操作的权限
func RequirePermission(db *gorm.DB, module string, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.GetUserIDFromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// 查询用户角色
			var userRoles []models.UserRole
			db.Where("user_id = ?", userID).Find(&userRoles)
			if len(userRoles) == 0 {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			roleIDs := make([]uint, len(userRoles))
			for i, ur := range userRoles {
				roleIDs[i] = ur.RoleID
			}

			// 查询角色拥有的权限
			var count int64
			db.Model(&models.RolePermission{}).
				Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
				Where("role_permissions.role_id IN ? AND permissions.module = ? AND permissions.action = ?", roleIDs, module, action).
				Count(&count)

			if count == 0 {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DepartmentContext 部门数据隔离：从 User 表查询部门，注入到 context
func DepartmentContext(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.GetUserIDFromContext(r.Context())
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			var user models.User
			if err := db.First(&user, userID).Error; err == nil && user.Department != "" {
				ctx := context.WithValue(r.Context(), UserDepartmentContextKey, user.Department)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserDepartmentFromContext 从 context 获取用户部门
func GetUserDepartmentFromContext(ctx context.Context) (string, bool) {
	dept, ok := ctx.Value(UserDepartmentContextKey).(string)
	return dept, ok
}

// RequireRole 检查用户是否拥有指定角色之一
// 通过 user_roles 联表查询角色的 name 字段，兼容 NormalizeRole 映射（super_admin → admin）
func RequireRole(db *gorm.DB, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.GetUserIDFromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// 查询用户拥有的角色名称（通过 user_roles 联表）
			var roleNames []string
			db.Model(&models.Role{}).
				Joins("JOIN user_roles ON user_roles.role_id = roles.id").
				Where("user_roles.user_id = ?", userID).
				Pluck("roles.name", &roleNames)

			// 兼容映射（super_admin → admin）
			for _, rn := range roleNames {
				normalized := models.NormalizeRole(rn)
				for _, required := range roles {
					if normalized == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

// RequireAdmin 仅允许 admin 角色访问
func RequireAdmin(db *gorm.DB) func(http.Handler) http.Handler {
	return RequireRole(db, models.RoleAdmin)
}

// RequireManagerOrAbove 允许 manager 和 admin 角色访问
func RequireManagerOrAbove(db *gorm.DB) func(http.Handler) http.Handler {
	return RequireRole(db, models.RoleManager, models.RoleAdmin)
}

// RequireSameDepartment 部门级别数据隔离中间件
// 检查当前用户与目标资源所属用户的 DepartmentID 是否一致
// admin 角色可绕过部门隔离
func RequireSameDepartment(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.GetUserIDFromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// admin 角色绕过部门隔离检查
			if isAdmin(db, userID) {
				next.ServeHTTP(w, r)
				return
			}

			// 查询当前用户的部门 ID
			var currentUser models.User
			if err := db.First(&currentUser, userID).Error; err != nil {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			// 用户无部门归属时拒绝访问（非 admin 必须属于某部门）
			if currentUser.DepartmentID == nil {
				http.Error(w, `{"error":"forbidden","message":"用户未分配部门"}`, http.StatusForbidden)
				return
			}

			// 无明确资源归属时允许通过（由业务层进一步过滤，如列表查询）
			resourceUserID := getResourceUserIDFromRequest(r)
			if resourceUserID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// 查询资源所属用户的部门
			var resourceUser models.User
			if err := db.First(&resourceUser, resourceUserID).Error; err != nil {
				http.Error(w, `{"error":"forbidden","message":"资源用户不存在"}`, http.StatusForbidden)
				return
			}

			// 比较部门 ID
			if resourceUser.DepartmentID == nil ||
				*currentUser.DepartmentID != *resourceUser.DepartmentID {
				http.Error(w, `{"error":"forbidden","message":"跨部门数据不可访问"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAdmin 判断用户是否具有 admin 角色
func isAdmin(db *gorm.DB, userID uint) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (roles.name = ? OR roles.name = ?)", userID, models.RoleAdmin, "super_admin").
		Count(&count)
	return count > 0
}

// getResourceUserIDFromRequest 从请求中提取资源所属用户的 ID
// 优先级：URL 查询参数 resource_user_id > 0（无明确资源归属）
func getResourceUserIDFromRequest(r *http.Request) uint {
	raw := r.URL.Query().Get("resource_user_id")
	if raw == "" {
		return 0
	}
	// 简单的字符串转换，忽略解析错误（返回 0 表示无归属）
	var id uint
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0
		}
		id = id*10 + uint(c-'0')
	}
	return id
}
