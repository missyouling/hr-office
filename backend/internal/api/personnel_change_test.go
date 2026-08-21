package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupPersonnelChangeTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}, &models.Department{}, &models.PersonnelChange{}))
	grantPersonnelChangePermission(t, env)
	return env
}

func grantPersonnelChangePermission(t *testing.T, env *rbacTestEnv) {
	t.Helper()
	var permission models.Permission
	err := env.db.Where("module = ? AND action = ?", "employee", "edit").First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = models.Permission{Module: "employee", Action: "edit", Label: "编辑", SortOrder: 3}
		require.NoError(t, env.db.Create(&permission).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, env.db.Create(&models.RolePermission{RoleID: env.roleIDs["admin"], PermissionID: permission.ID}).Error)
}

func createPersonnelChangeEmployee(t *testing.T, env *rbacTestEnv) models.Employee {
	t.Helper()
	employee := models.Employee{UserID: env.admin.ID, Name: "异动测试员工", Department: "研发部", Position: "工程师", JobLevel: "P5", Status: models.EmployeeStatusActive}
	require.NoError(t, env.db.Create(&employee).Error)
	return employee
}

func TestPersonnelChangeLifecycle(t *testing.T) {
	env := setupPersonnelChangeTestEnv(t)
	employee := createPersonnelChangeEmployee(t, env)
	department := models.Department{UserID: &env.admin.ID, Name: "产品部"}
	require.NoError(t, env.db.Create(&department).Error)
	body := map[string]any{"employee_id": employee.ID, "change_type": models.PersonnelChangeTypePromotion, "effective_date": "2026-08-19", "reason": "晋升", "after_department_id": department.ID, "after_position": "高级工程师", "after_job_level": "P6"}
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var record models.PersonnelChange
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &record))
	require.Equal(t, models.PersonnelChangeStatusDraft, record.Status)
	require.Equal(t, "研发部", record.BeforeDepartment)

	response = doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes/1/activate", env.admin.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, env.db.First(&employee, employee.ID).Error)
	require.Equal(t, "产品部", employee.Department)
	require.Equal(t, "高级工程师", employee.Position)
	require.Equal(t, "P6", employee.JobLevel)

	response = doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes/1/void", env.admin.SupabaseUID, map[string]any{"reason": "录入错误"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, env.db.First(&employee, employee.ID).Error)
	require.Equal(t, "产品部", employee.Department, "生效记录作废不应回滚员工")
}

func TestPersonnelChangeActivationRejectsChangedEmployee(t *testing.T) {
	env := setupPersonnelChangeTestEnv(t)
	employee := createPersonnelChangeEmployee(t, env)
	body := map[string]any{"employee_id": employee.ID, "change_type": models.PersonnelChangeTypePromotion, "effective_date": "2026-08-19", "reason": "晋升", "after_position": "高级工程师"}
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.NoError(t, env.db.Model(&employee).Update("position", "已变更岗位").Error)
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes/1/activate", env.admin.SupabaseUID, nil)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}

func TestPersonnelChangeRejectsInactiveOrUnchanged(t *testing.T) {
	env := setupPersonnelChangeTestEnv(t)
	employee := createPersonnelChangeEmployee(t, env)
	unchanged := map[string]any{"employee_id": employee.ID, "change_type": models.PersonnelChangeTypeTransfer, "effective_date": "2026-08-19", "reason": "调整"}
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes", env.admin.SupabaseUID, unchanged)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.NoError(t, env.db.Model(&employee).Update("status", "resigned").Error)
	inactive := map[string]any{"employee_id": employee.ID, "change_type": models.PersonnelChangeTypeTransfer, "effective_date": "2026-08-19", "reason": "调整", "after_position": "新岗位"}
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/personnel-changes", env.admin.SupabaseUID, inactive)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
}
