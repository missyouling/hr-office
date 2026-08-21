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

func setupTrainingTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}, &models.TrainingRecord{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantTrainingPermission(t, env, "admin", action)
	}
	return env
}

func grantTrainingPermission(t *testing.T, env *rbacTestEnv, role, action string) {
	t.Helper()
	var permission models.Permission
	err := env.db.Where("module = ? AND action = ?", "training", action).First(&permission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = models.Permission{Module: "training", Action: action, Label: action}
		require.NoError(t, env.db.Create(&permission).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, env.db.Create(&models.RolePermission{RoleID: env.roleIDs[role], PermissionID: permission.ID}).Error)
}

func createTrainingEmployee(t *testing.T, env *rbacTestEnv, userID uint, name string) models.Employee {
	t.Helper()
	employee := models.Employee{UserID: userID, Name: name, Department: "研发部", Position: "工程师", Status: models.EmployeeStatusActive}
	require.NoError(t, env.db.Create(&employee).Error)
	return employee
}

func trainingBody(employeeID *uint) map[string]any {
	return map[string]any{"topic": "安全培训", "training_type": models.TrainingTypeInternal, "training_date": "2026-08-19", "trainer_or_institution": "人力资源部", "employee_id": employeeID, "result": "通过"}
}

func TestTrainingRecordLifecycle(t *testing.T) {
	env := setupTrainingTestEnv(t)
	employee := createTrainingEmployee(t, env, env.admin.ID, "培训员工")
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/training-records", env.admin.SupabaseUID, trainingBody(&employee.ID))
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var record models.TrainingRecord
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &record))
	require.Equal(t, "培训员工", record.SnapshotName)
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/training-records/1/complete", env.admin.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPut, "/api/training-records/1", env.admin.SupabaseUID, trainingBody(&employee.ID))
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/training-records/1/void", env.admin.SupabaseUID, map[string]any{"reason": "录入错误"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/training-records/1/void", env.admin.SupabaseUID, map[string]any{"reason": "再次作废"})
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}

func TestTrainingRecordValidationAndIsolation(t *testing.T) {
	env := setupTrainingTestEnv(t)
	employee := createTrainingEmployee(t, env, env.admin.ID, "本租户员工")
	invalid := trainingBody(&employee.ID)
	invalid["training_type"] = "other"
	response := doRBACRequest(t, env.router, http.MethodPost, "/api/training-records", env.admin.SupabaseUID, invalid)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	foreign := createTrainingEmployee(t, env, env.viewer.ID, "他租户员工")
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/training-records", env.admin.SupabaseUID, trainingBody(&foreign.ID))
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodPost, "/api/training-records", env.admin.SupabaseUID, trainingBody(&employee.ID))
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/training-records/1", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	foreignRecord := models.TrainingRecord{UserID: env.viewer.ID, Topic: "他租户培训", TrainingType: models.TrainingTypeOnline, TrainingDate: "2026-08-19", Status: models.TrainingStatusDraft}
	require.NoError(t, env.db.Create(&foreignRecord).Error)
	grantTrainingPermission(t, env, "viewer", "view")
	response = doRBACRequest(t, env.router, http.MethodGet, "/api/training-records/1", env.viewer.SupabaseUID, nil)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}
