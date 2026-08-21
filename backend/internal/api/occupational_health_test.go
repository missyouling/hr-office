package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func grantOccupationalHealthPermission(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	sortOrders := map[string]int{
		"view":   114,
		"create": 115,
		"edit":   116,
		"delete": 117,
	}
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "occupational_health", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "occupational_health", Action: action, Label: action, SortOrder: sortOrders[action]}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

func setupOccupationalHealthTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.OccupationalHealthCheck{}, &models.Employee{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantOccupationalHealthPermission(t, env.db, env.roleIDs["admin"], action)
	}
	return env
}

func createTestOccupationalHealthEmployee(t *testing.T, db *gorm.DB, userID uint, name, department, position, idNumber string) models.Employee {
	t.Helper()
	emp := models.Employee{UserID: userID, Name: name, Department: department, Position: position, IDNumber: idNumber, Status: models.EmployeeStatusActive}
	require.NoError(t, db.Create(&emp).Error)
	return emp
}

func createOccupationalHealthViaAPI(t *testing.T, env *rbacTestEnv, token string, body map[string]interface{}) models.OccupationalHealthCheck {
	t.Helper()
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/occupational-health-checks", token, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建职业健康检查记录应成功: %d %s", rec.Code, rec.Body.String())
	var record models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &record))
	return record
}

func TestOccupationalHealthRBACForbidden(t *testing.T) {
	env := setupOccupationalHealthTestEnv(t)
	token := env.viewer.SupabaseUID
	emp := createTestOccupationalHealthEmployee(t, env.db, env.admin.ID, "张三", "业务部", "专员", "110101199001019999")
	record := createOccupationalHealthViaAPI(t, env, env.admin.SupabaseUID, map[string]interface{}{
		"employee_id":       emp.ID,
		"check_date":        "2026-08-01",
		"medical_institution": "市职业病防治院",
		"check_category":    "上岗前",
	})
	cases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/occupational-health-checks", nil},
		{http.MethodGet, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), nil},
		{http.MethodPost, "/api/occupational-health-checks", map[string]interface{}{"employee_id": emp.ID, "check_date": "2026-08-01", "medical_institution": "机", "check_category": "类"}},
		{http.MethodPut, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), map[string]interface{}{"employee_id": emp.ID, "check_date": "2026-08-02", "medical_institution": "机", "check_category": "类"}},
		{http.MethodDelete, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/complete", record.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/void", record.ID), map[string]interface{}{"reason": "作废"}},
	}
	for _, c := range cases {
		rec := doRBACRequest(t, env.router, c.method, c.path, token, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s 应 403: %d %s", c.method, c.path, rec.Code, rec.Body.String())
	}
}

func TestOccupationalHealthLifecycle(t *testing.T) {
	env := setupOccupationalHealthTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestOccupationalHealthEmployee(t, env.db, env.admin.ID, "李四", "总经办", "主管", "110101199001018888")
	record := createOccupationalHealthViaAPI(t, env, token, map[string]interface{}{
		"employee_id":       emp.ID,
		"check_date":        "2026-08-01",
		"medical_institution": "市职业病防治院",
		"check_category":    "上岗前",
		"check_conclusion":   "正常",
		"next_check_date":   "2027-08-01",
		"remarks":           "首次建档",
	})
	assert.Equal(t, models.OccupationalHealthStatusDraft, record.Status)
	assert.Equal(t, "李四", record.SnapshotName)
	assert.Equal(t, "总经办", record.SnapshotDepartment)
	assert.Equal(t, "主管", record.SnapshotPosition)

	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/occupational-health-checks", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed, 1)

	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), token, map[string]interface{}{
		"employee_id":       emp.ID,
		"check_date":        "2026-08-02",
		"medical_institution": "市职业病防治院",
		"check_category":    "在岗期间",
		"check_conclusion":   "需复查",
		"remarks":           "补录",
	})
	require.Equal(t, http.StatusOK, rec.Code, "编辑草稿应成功: %s", rec.Body.String())
	var updated models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, "2026-08-02", updated.CheckDate)
	assert.Equal(t, "在岗期间", updated.CheckCategory)
	assert.Equal(t, models.OccupationalHealthStatusDraft, updated.Status)
	assert.Equal(t, "市职业病防治院", updated.CheckInstitution)
	assert.Equal(t, "需复查", updated.Conclusion)

	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/complete", record.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code, "完成应成功: %s", rec.Body.String())
	var completed models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &completed))
	assert.Equal(t, models.OccupationalHealthStatusCompleted, completed.Status)
	assert.NotNil(t, completed.CompletedAt)
	assert.Equal(t, "需复查", completed.Conclusion)

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), token, map[string]interface{}{
		"employee_id":       emp.ID,
		"check_date":        "2026-08-03",
		"medical_institution": "市职业病防治院",
		"check_category":    "离岗时",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/void", record.ID), token, map[string]interface{}{"reason": "  "})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/void", record.ID), token, map[string]interface{}{"reason": "重复建档"})
	require.Equal(t, http.StatusOK, rec.Code, "作废应成功: %s", rec.Body.String())
	var voided models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &voided))
	assert.Equal(t, models.OccupationalHealthStatusVoided, voided.Status)
	assert.Equal(t, "重复建档", voided.VoidReason)
	assert.NotNil(t, voided.VoidedAt)

	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/occupational-health-checks/%d/void", record.ID), token, map[string]interface{}{"reason": "再次作废"})
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestOccupationalHealthTenantAndDepartmentIsolation(t *testing.T) {
	env := setupOccupationalHealthTestEnv(t)
	for _, action := range []string{"view", "create", "edit"} {
		grantOccupationalHealthPermission(t, env.db, env.roleIDs["viewer"], action)
	}
	admin := createRBACTestUser(t, env.db, "ohAdmin", "oh-admin-uuid", env.roleIDs["admin"])
	require.NoError(t, env.db.Model(&admin).Update("department", "总经办").Error)
	other := createRBACTestUser(t, env.db, "ohOther", "oh-other-uuid", env.roleIDs["viewer"])
	require.NoError(t, env.db.Model(&other).Update("department", "业务部").Error)

	adminEmp := createTestOccupationalHealthEmployee(t, env.db, admin.ID, "郑一", "总经办", "经理", "110101199001010001")
	otherEmp := createTestOccupationalHealthEmployee(t, env.db, admin.ID, "吴二", "业务部", "专员", "110101199001010002")
	record := createOccupationalHealthViaAPI(t, env, admin.SupabaseUID, map[string]interface{}{
		"employee_id":       adminEmp.ID,
		"check_date":        "2026-08-01",
		"medical_institution": "市职业病防治院",
		"check_category":    "上岗前",
	})
	_ = createOccupationalHealthViaAPI(t, env, admin.SupabaseUID, map[string]interface{}{
		"employee_id":       otherEmp.ID,
		"check_date":        "2026-08-02",
		"medical_institution": "市职业病防治院",
		"check_category":    "在岗期间",
	})

	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/occupational-health-checks", other.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Len(t, listed, 0)

	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), other.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/occupational-health-checks/%d", record.ID), other.SupabaseUID, map[string]interface{}{
		"employee_id":       otherEmp.ID,
		"check_date":        "2026-08-02",
		"medical_institution": "市职业病防治院",
		"check_category":    "在岗期间",
	})
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/occupational-health-checks", other.SupabaseUID, map[string]interface{}{
		"employee_id":       adminEmp.ID,
		"check_date":        "2026-08-01",
		"medical_institution": "市职业病防治院",
		"check_category":    "上岗前",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	require.NoError(t, env.db.Model(&admin).Update("department", "总经办").Error)
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/occupational-health-checks", admin.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var sameDeptListed []models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sameDeptListed))
	assert.Len(t, sameDeptListed, 1)
	require.Equal(t, "总经办", sameDeptListed[0].SnapshotDepartment)

	require.NoError(t, env.db.Model(&admin).Update("department", "业务部").Error)
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/occupational-health-checks", admin.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var otherDeptListed []models.OccupationalHealthCheck
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &otherDeptListed))
	assert.Len(t, otherDeptListed, 1)
	require.Equal(t, "业务部", otherDeptListed[0].SnapshotDepartment)
}
