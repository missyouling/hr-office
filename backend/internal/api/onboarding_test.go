package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// grantEmployeePermission 为指定角色补充 employee 模块指定动作权限。
// 测试种子（seedRBACForTest）默认不含 employee 模块权限，此处按需补齐。
func grantEmployeePermission(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "employee", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "employee", Action: action, Label: action, SortOrder: 3}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// setupOnboardingTestEnv 初始化入职管理测试环境（admin 具备 employee 全权限 + 测试部门）
func setupOnboardingTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}, &models.OnboardingRecord{}, &models.Department{}, &models.WorkTodo{}))
	grantEmployeePermission(t, env.db, env.roleIDs["admin"], "view")
	grantEmployeePermission(t, env.db, env.roleIDs["admin"], "create")
	grantEmployeePermission(t, env.db, env.roleIDs["admin"], "edit")
	// 测试部门（含编码 DEV），供工号生成
	createTestDepartment(t, env.db, env.admin.ID, "测试部", "DEV")
	return env
}

// createTestDepartment 创建测试部门（含编码）。
func createTestDepartment(t *testing.T, db *gorm.DB, userID uint, name, code string) models.Department {
	t.Helper()
	dept := models.Department{UserID: &userID, Name: name, Code: code}
	require.NoError(t, db.Create(&dept).Error)
	return dept
}

// createOnboardingViaAPI 通过 API 创建待入职记录
func createOnboardingViaAPI(t *testing.T, env *rbacTestEnv, idNumber string) models.OnboardingRecord {
	t.Helper()
	body := map[string]interface{}{
		"name":              "入职测试员工",
		"phone":             "13800000000",
		"department":        "测试部",
		"position":          "测试岗",
		"planned_hire_date": "2026-09-01",
		"id_number":         idNumber,
		"remarks":           "备注",
	}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建待入职应成功: %d %s", rec.Code, rec.Body.String())
	var record models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &record))
	return record
}

// createTestEmployeeWithID 创建指定身份证号的测试员工
func createTestEmployeeWithID(t *testing.T, db *gorm.DB, userID uint, idNumber, status string) models.Employee {
	t.Helper()
	emp := models.Employee{
		UserID:     userID,
		Name:       "测试员工-" + idNumber[len(idNumber)-4:],
		IDNumber:   idNumber,
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

// TestOnboardingRBACForbidden 无 employee 权限调用全部入职端点必须 403
func TestOnboardingRBACForbidden(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.viewer.SupabaseUID // viewer 无 employee 权限
	record := createOnboardingViaAPI(t, env, "110101199001011234")

	cases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/onboarding-records", nil},
		{http.MethodPost, "/api/onboarding-records", map[string]interface{}{"name": "x", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235"}},
		{http.MethodPost, "/api/onboarding-records/quick", map[string]interface{}{"name": "x", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235"}},
		{http.MethodPut, fmt.Sprintf("/api/onboarding-records/%d", record.ID), map[string]interface{}{"name": "x", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235"}},
		{http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), map[string]interface{}{}},
		{http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/abandon", record.ID), map[string]interface{}{"reason": "原因", "remarks": "备注"}},
		{http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/restore", record.ID), nil},
	}
	for _, c := range cases {
		rec := doRBACRequest(t, env.router, c.method, c.path, token, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s 应 403: %d %s", c.method, c.path, rec.Code, rec.Body.String())
	}
}

// TestOnboardingCreateListUpdate 创建/列表/更新待入职记录
func TestOnboardingCreateListUpdate(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID

	record := createOnboardingViaAPI(t, env, "110101199001011234")
	assert.Equal(t, models.OnboardingStatusPending, record.Status)
	assert.Equal(t, "2026-09-01", record.PlannedHireDate)

	// 列表
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/onboarding-records", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "列表应成功: %d %s", rec.Code, rec.Body.String())
	var records []models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &records))
	assert.Len(t, records, 1)

	// 更新 pending
	body := map[string]interface{}{
		"name": "更新姓名", "phone": "13900000000", "department": "测试部",
		"position": "测试岗", "planned_hire_date": "2026-09-15", "id_number": "110101199001011234",
	}
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/onboarding-records/%d", record.ID), token, body)
	assert.Equal(t, http.StatusOK, rec.Code, "更新待入职应成功: %d %s", rec.Code, rec.Body.String())
	var updated models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	assert.Equal(t, "更新姓名", updated.Name)
	assert.Equal(t, "2026-09-15", updated.PlannedHireDate)
	assert.Equal(t, models.OnboardingStatusPending, updated.Status)
}

// TestOnboardingStateTransitions 状态合法转换与非法转换拒绝
func TestOnboardingStateTransitions(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID

	// pending → confirm → onboarded
	record := createOnboardingViaAPI(t, env, "110101199001011234")
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusOK, rec.Code, "确认入职应成功: %d %s", rec.Code, rec.Body.String())
	var onboarded models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &onboarded))
	assert.Equal(t, models.OnboardingStatusOnboarded, onboarded.Status)

	// onboarded 不可再确认/放弃/恢复
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusConflict, rec.Code, "已入职记录不可重复确认")
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/abandon", record.ID), token, map[string]interface{}{"reason": "原因", "remarks": "备注"})
	assert.Equal(t, http.StatusConflict, rec.Code, "已入职记录不可放弃")
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/restore", record.ID), token, nil)
	assert.Equal(t, http.StatusConflict, rec.Code, "已入职记录不可恢复")

	// pending → abandon → abandoned
	record2 := createOnboardingViaAPI(t, env, "110101199001011235")
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/abandon", record2.ID), token, map[string]interface{}{"reason": "候选人放弃", "remarks": "已电话确认"})
	assert.Equal(t, http.StatusOK, rec.Code, "放弃应成功: %d %s", rec.Code, rec.Body.String())
	var abandoned models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &abandoned))
	assert.Equal(t, models.OnboardingStatusAbandoned, abandoned.Status)

	// abandoned 不可确认，可恢复
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record2.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusConflict, rec.Code, "已放弃记录不可确认")
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/restore", record2.ID), token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "恢复应成功: %d %s", rec.Code, rec.Body.String())
	var restored models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &restored))
	assert.Equal(t, models.OnboardingStatusPending, restored.Status)

	// pending 不可恢复
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/restore", record2.ID), token, nil)
	assert.Equal(t, http.StatusConflict, rec.Code, "待入职记录不可恢复")
}

// TestOnboardingIDNumberConflict 身份证冲突统一拒绝（active/resigned 全量）
func TestOnboardingIDNumberConflict(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID

	activeEmp := createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011234", "active")
	resignedEmp := createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011235", "resigned")

	// 创建时与 active 冲突 → 409
	body := map[string]interface{}{"name": "冲突", "planned_hire_date": "2026-09-01", "id_number": activeEmp.IDNumber}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records", token, body)
	assert.Equal(t, http.StatusConflict, rec.Code, "与 active 员工身份证冲突应拒绝: %d %s", rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "该身份证号已存在员工记录")

	// 创建时与 resigned 冲突 → 409（禁止自动返聘/覆盖/恢复）
	body["id_number"] = resignedEmp.IDNumber
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records", token, body)
	assert.Equal(t, http.StatusConflict, rec.Code, "与 resigned 员工身份证冲突应拒绝: %d %s", rec.Code, rec.Body.String())

	// 确认时冲突 → 409（先创建无冲突记录，再新增冲突员工）
	record := createOnboardingViaAPI(t, env, "110101199001011236")
	conflictEmp := models.Employee{UserID: env.admin.ID, Name: "后到员工", IDNumber: "110101199001011236", Status: "active"}
	require.NoError(t, env.db.Create(&conflictEmp).Error)
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusConflict, rec.Code, "确认时身份证冲突应拒绝: %d %s", rec.Code, rec.Body.String())
}

// TestOnboardingUpdateIDNumberConflict 编辑 pending 记录时，新身份证命中员工全量应返回 409。
func TestOnboardingUpdateIDNumberConflict(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	record := createOnboardingViaAPI(t, env, "110101199001011234")
	conflict := createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199002022345", "resigned")
	body := map[string]interface{}{
		"name": record.Name, "planned_hire_date": record.PlannedHireDate,
		"id_number": conflict.IDNumber, "department": record.Department,
	}
	resp := doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/onboarding-records/%d", record.ID), env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusConflict, resp.Code, "编辑为冲突身份证应返回 409: %s", resp.Body.String())
}

// TestOnboardingConfirmCreatesEmployee 确认入职事务内创建员工并关联
func TestOnboardingConfirmCreatesEmployee(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID
	record := createOnboardingViaAPI(t, env, "110101199001011234")

	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusOK, rec.Code, "确认入职应成功: %d %s", rec.Code, rec.Body.String())

	var onboarded models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &onboarded))
	assert.Equal(t, models.OnboardingStatusOnboarded, onboarded.Status)
	require.NotNil(t, onboarded.EmployeeID, "确认后应关联员工ID")
	assert.Equal(t, models.EmploymentStatusTrial, onboarded.EmploymentStatus, "默认用工状态应为 trial")
	today := time.Now().Format("2006-01-02")
	require.NotNil(t, onboarded.ActualHireDate)
	assert.Equal(t, today, *onboarded.ActualHireDate, "实际入职日期应为今天")

	var emp models.Employee
	require.NoError(t, env.db.First(&emp, *onboarded.EmployeeID).Error)
	assert.Equal(t, "active", emp.Status)
	assert.Equal(t, today, emp.HireDate)
	assert.Equal(t, models.EmploymentStatusTrial, emp.EmploymentStatus, "确认入职创建的员工主表用工状态应保持 trial")
	assert.Equal(t, "DEV001", emp.EmployeeID, "工号应为部门编码+三位序号")
	assert.Equal(t, record.Name, emp.Name)
	assert.Equal(t, record.IDNumber, emp.IDNumber)
	assert.Equal(t, record.Department, emp.Department)

	// 确认入职创建通用待办（归属租户管理员）
	var todo models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "onboarding", record.ID).First(&todo).Error)
	assert.Equal(t, env.admin.ID, todo.UserID, "待办应归属租户管理员")
	require.NotNil(t, todo.AssigneeID)
	assert.Equal(t, env.admin.ID, *todo.AssigneeID)
}

// TestOnboardingAbandonDoesNotCreateEmployee 放弃不创建员工
func TestOnboardingAbandonDoesNotCreateEmployee(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID
	record := createOnboardingViaAPI(t, env, "110101199001011234")

	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/abandon", record.ID), token, map[string]interface{}{"reason": "候选人放弃", "remarks": "已电话确认"})
	assert.Equal(t, http.StatusOK, rec.Code, "放弃应成功: %d %s", rec.Code, rec.Body.String())

	var count int64
	require.NoError(t, env.db.Model(&models.Employee{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count, "放弃不应创建员工")
}

// TestOnboardingRestoreKeepsHistory 恢复保留原计划日期与放弃历史
func TestOnboardingRestoreKeepsHistory(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID
	record := createOnboardingViaAPI(t, env, "110101199001011234")

	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/abandon", record.ID), token, map[string]interface{}{"reason": "候选人放弃", "remarks": "已电话确认"})
	assert.Equal(t, http.StatusOK, rec.Code, "放弃应成功: %d %s", rec.Code, rec.Body.String())
	var abandoned models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &abandoned))
	require.NotNil(t, abandoned.AbandonedAt)

	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/restore", record.ID), token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "恢复应成功: %d %s", rec.Code, rec.Body.String())
	var restored models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &restored))
	assert.Equal(t, models.OnboardingStatusPending, restored.Status)
	assert.Equal(t, "2026-09-01", restored.PlannedHireDate, "恢复应保留原计划日期")
	assert.Equal(t, "候选人放弃", restored.AbandonReason, "恢复应保留放弃原因")
	assert.Equal(t, "已电话确认", restored.Remarks, "恢复应保留备注")
	require.NotNil(t, restored.AbandonedAt, "恢复应保留放弃时间")
}

// TestOnboardingQuickTrialFormal 快速入职默认 trial、可选 formal，工号递增且创建待办
func TestOnboardingQuickTrialFormal(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID

	// 默认 trial
	body := map[string]interface{}{"name": "快速入职A", "planned_hire_date": "2026-09-01", "id_number": "110101199001011234", "department": "测试部"}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records/quick", token, body)
	assert.Equal(t, http.StatusCreated, rec.Code, "快速入职应成功: %d %s", rec.Code, rec.Body.String())
	var quickTrial models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &quickTrial))
	assert.Equal(t, models.OnboardingStatusOnboarded, quickTrial.Status)
	assert.Equal(t, models.EmploymentStatusTrial, quickTrial.EmploymentStatus)
	require.NotNil(t, quickTrial.EmployeeID)

	// 指定 formal
	body = map[string]interface{}{"name": "快速入职B", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235", "department": "测试部", "employment_status": "formal"}
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records/quick", token, body)
	assert.Equal(t, http.StatusCreated, rec.Code, "快速入职应成功: %d %s", rec.Code, rec.Body.String())
	var quickFormal models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &quickFormal))
	assert.Equal(t, models.EmploymentStatusFormal, quickFormal.EmploymentStatus)
	require.NotNil(t, quickFormal.EmployeeID)

	// 快速入职创建了员工，工号按部门前缀递增
	var emp1, emp2 models.Employee
	require.NoError(t, env.db.First(&emp1, *quickTrial.EmployeeID).Error)
	require.NoError(t, env.db.First(&emp2, *quickFormal.EmployeeID).Error)
	assert.Equal(t, models.EmploymentStatusTrial, emp1.EmploymentStatus, "快速入职默认 trial 时员工主表应保持 trial")
	assert.Equal(t, models.EmploymentStatusFormal, emp2.EmploymentStatus, "快速入职显式 formal 时员工主表应保持 formal")
	assert.Equal(t, "DEV001", emp1.EmployeeID)
	assert.Equal(t, "DEV002", emp2.EmployeeID)

	// 快速入职创建通用待办（归属租户管理员）
	var todos []models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ?", "onboarding").Find(&todos).Error)
	assert.Len(t, todos, 2, "两条快速入职应各创建一条待办")
	for _, todo := range todos {
		assert.Equal(t, env.admin.ID, todo.UserID)
		require.NotNil(t, todo.AssigneeID)
		assert.Equal(t, env.admin.ID, *todo.AssigneeID)
	}
}

// TestOnboardingManualDeptCodeMissing 部门无编码：手动确认/快速返回业务错误且不创建员工
func TestOnboardingManualDeptCodeMissing(t *testing.T) {
	env := setupOnboardingTestEnv(t)
	token := env.admin.SupabaseUID
	createTestDepartment(t, env.db, env.admin.ID, "无码部门", "")

	// 创建记录（部门=无码部门）
	body := map[string]interface{}{"name": "无码", "planned_hire_date": "2026-09-01", "id_number": "110101199001011234", "department": "无码部门"}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records", token, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建待入职应成功: %d %s", rec.Code, rec.Body.String())
	var record models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &record))

	// 确认 → 业务错误（422）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", record.ID), token, map[string]interface{}{})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "部门无编码确认应返回业务错误: %d %s", rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "部门未配置编码")

	// 快速 → 业务错误（422）
	body = map[string]interface{}{"name": "无码快速", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235", "department": "无码部门"}
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records/quick", token, body)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "部门无编码快速入职应返回业务错误: %d %s", rec.Code, rec.Body.String())

	// 未创建任何员工
	var count int64
	require.NoError(t, env.db.Model(&models.Employee{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count, "部门无编码不应创建员工")
}

// TestOnboardingThreePathsConsistent 手动确认/快速/worker 三条路径工号与待办一致
func TestOnboardingThreePathsConsistent(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	token := env.admin.SupabaseUID

	// 1. 手动确认 → DEV001
	rec1 := createOnboardingViaAPI(t, env, "110101199001011234")
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/onboarding-records/%d/confirm", rec1.ID), token, map[string]interface{}{})
	require.Equal(t, http.StatusOK, rec.Code, "手动确认应成功: %d %s", rec.Code, rec.Body.String())
	var onboarded models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &onboarded))
	var emp1 models.Employee
	require.NoError(t, env.db.First(&emp1, *onboarded.EmployeeID).Error)
	assert.Equal(t, "DEV001", emp1.EmployeeID)

	// 2. 快速入职 → DEV002
	body := map[string]interface{}{"name": "快速", "planned_hire_date": "2026-09-01", "id_number": "110101199001011235", "department": "测试部"}
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records/quick", token, body)
	require.Equal(t, http.StatusCreated, rec.Code, "快速入职应成功: %d %s", rec.Code, rec.Body.String())
	var quick models.OnboardingRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &quick))
	var emp2 models.Employee
	require.NoError(t, env.db.First(&emp2, *quick.EmployeeID).Error)
	assert.Equal(t, "DEV002", emp2.EmployeeID)

	// 3. worker 自动入职 → DEV003
	rec3 := createPendingOnboarding(t, env.db, env.admin.ID, "自动入职", "110101199001011236", "2026-08-18")
	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(shanghaiLoc(t), time.Date(2026, 8, 18, 2, 0, 0, 0, shanghaiLoc(t)))
	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec3.ID).Error)
	assert.Equal(t, models.OnboardingStatusOnboarded, saved.Status)
	var emp3 models.Employee
	require.NoError(t, env.db.First(&emp3, *saved.EmployeeID).Error)
	assert.Equal(t, "DEV003", emp3.EmployeeID)

	// 三条路径待办一致：business_type=onboarding、归属租户管理员
	var todos []models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ?", "onboarding").Find(&todos).Error)
	assert.Len(t, todos, 3, "三条路径应各创建一条通用待办")
	for _, todo := range todos {
		assert.Equal(t, env.admin.ID, todo.UserID)
		require.NotNil(t, todo.AssigneeID)
		assert.Equal(t, env.admin.ID, *todo.AssigneeID)
	}
}
