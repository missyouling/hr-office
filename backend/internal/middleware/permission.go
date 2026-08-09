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
