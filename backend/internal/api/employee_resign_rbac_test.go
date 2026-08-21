package api

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// grantEmployeeEdit 为指定角色补充 employee.edit 权限。
// 测试种子（seedRBACForTest）默认不含 employee 模块权限，此处按需补齐。
func grantEmployeeEdit(t *testing.T, db *gorm.DB, roleID uint) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "employee", "edit").First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "employee", Action: "edit", Label: "编辑", SortOrder: 3}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// createTestEmployee 创建一名挂在指定用户下的测试员工
func createTestEmployee(t *testing.T, db *gorm.DB, userID uint, status string) models.Employee {
	t.Helper()
	emp := models.Employee{
		UserID:     userID,
		Name:       "权限测试员工",
		IDNumber:   "110101199001011234",
		Department: "测试部",
		Position:   "测试岗",
		Status:     status,
	}
	if status == "resigned" {
		emp.ResignDate = "2026-01-01"
	}
	require.NoError(t, db.Create(&emp).Error)
	return emp
}

// doResignMultipartRequest 以 multipart/form-data 发送 resign 请求（无离职证明文件）
func doResignMultipartRequest(t *testing.T, router http.Handler, path, token, resignDate string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("resign_date", resignDate))
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestEmployeeResignRestoreRequirePermission 无 employee.edit 权限调用 resign/restore 必须 403
func TestEmployeeResignRestoreRequirePermission(t *testing.T) {
	env := setupRBACTestEnv(t)
	// 测试种子环境未迁移 employees 表，此处按需补齐
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}))
	token := env.viewer.SupabaseUID // viewer 无 employee.edit

	// 先创建一名员工（挂在 admin 下），确保请求能走到权限检查之后的业务逻辑
	emp := createTestEmployee(t, env.db, env.admin.ID, "resigned")

	// resign：无 employee.edit → 403
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/employees/%d/resign", emp.ID), token, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer 调用 resign 应 403: %d %s", rec.Code, rec.Body.String())

	// restore：无 employee.edit → 403
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/employees/restore", token, map[string]interface{}{"ids": []uint{emp.ID}})
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer 调用 restore 应 403: %d %s", rec.Code, rec.Body.String())

	// 员工数据未被无权限请求改动（仍为 resigned）
	var reloaded models.Employee
	require.NoError(t, env.db.First(&reloaded, emp.ID).Error)
	assert.Equal(t, "resigned", reloaded.Status)
}

// TestEmployeeResignRestoreWithPermission 有 employee.edit 权限时 resign/restore 行为维持成功
func TestEmployeeResignRestoreWithPermission(t *testing.T) {
	env := setupRBACTestEnv(t)
	// 测试种子环境未迁移 employees 表，此处按需补齐
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}))
	// 给 admin 补充 employee.edit 权限（测试种子默认不含 employee 模块）
	grantEmployeeEdit(t, env.db, env.roleIDs["admin"])
	token := env.admin.SupabaseUID

	emp := createTestEmployee(t, env.db, env.admin.ID, "resigned")

	// restore：有 employee.edit → 成功恢复为 active
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/employees/restore", token, map[string]interface{}{"ids": []uint{emp.ID}})
	assert.Equal(t, http.StatusOK, rec.Code, "admin 调用 restore 应成功: %d %s", rec.Code, rec.Body.String())

	var reloaded models.Employee
	require.NoError(t, env.db.First(&reloaded, emp.ID).Error)
	assert.Equal(t, "active", reloaded.Status)
	assert.Empty(t, reloaded.ResignDate)

	// resign：有 employee.edit → 成功（multipart 表单，无离职证明文件）
	rec = doResignMultipartRequest(t, env.router, fmt.Sprintf("/api/employees/%d/resign", emp.ID), token, "2026-02-01")
	assert.Equal(t, http.StatusOK, rec.Code, "admin 调用 resign 应成功: %d %s", rec.Code, rec.Body.String())

	require.NoError(t, env.db.First(&reloaded, emp.ID).Error)
	assert.Equal(t, "resigned", reloaded.Status)
	assert.Equal(t, "2026-02-01", reloaded.ResignDate)
}
