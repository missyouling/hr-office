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

// setupRegularizationTestEnv 初始化转正管理测试环境（admin 具备 employee.edit + 迁移转正模型）。
func setupRegularizationTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.RegularizationRecord{}, &models.RegularizationEffectRun{}, &models.WorkTodo{}, &models.Employee{}))
	grantEmployeePermission(t, env.db, env.roleIDs["admin"], "edit")
	return env
}

// createRegularizationRecord 直接落库创建转正记录（绕过 API，本批次无写接口）。
func createRegularizationRecord(t *testing.T, db *gorm.DB, userID uint, department, approvalNo, status string) models.RegularizationRecord {
	t.Helper()
	rec := models.RegularizationRecord{
		UserID:                   userID,
		SnapshotName:             "转正员工-" + approvalNo,
		SnapshotDepartment:       department,
		SnapshotPosition:         "开发工程师",
		SnapshotEmploymentStatus: models.EmploymentStatusTrial,
		SnapshotProbationEndDate: "2026-12-31",
		ApprovalNo:               approvalNo,
		Status:                   status,
		Source:                   models.RegularizationSourceManual,
		PlannedRegularDate:       "2027-01-01",
	}
	require.NoError(t, db.Create(&rec).Error)
	return rec
}

// TestRegularizationReadUnauthorizedNoToken 无登录令牌：列表与详情一律 401。
func TestRegularizationReadUnauthorizedNoToken(t *testing.T) {
	env := setupRegularizationTestEnv(t)
	rec := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-001", models.RegularizationStatusPendingSupervisor)

	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records", "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.Code, "无 token 列表应 401: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", rec.ID), "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.Code, "无 token 详情应 401: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationReadRBACForbidden 无 employee.edit 权限：列表与详情一律 403。
func TestRegularizationReadRBACForbidden(t *testing.T) {
	env := setupRegularizationTestEnv(t)
	rec := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-001", models.RegularizationStatusPendingSupervisor)
	token := env.viewer.SupabaseUID // viewer 无 employee 模块权限

	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records", token, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code, "无权限列表应 403: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", rec.ID), token, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code, "无权限详情应 403: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationListAndDetail 同租户：列表/详情/状态过滤/非法状态参数 400。
func TestRegularizationListAndDetail(t *testing.T) {
	env := setupRegularizationTestEnv(t)
	token := env.admin.SupabaseUID
	rec1 := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-001", models.RegularizationStatusPendingSupervisor)
	rec2 := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-002", models.RegularizationStatusEffective)

	// 列表返回 2 条（created_at DESC：rec2 在前）
	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records", token, nil)
	require.Equal(t, http.StatusOK, resp.Code, "列表应成功: %d %s", resp.Code, resp.Body.String())
	var records []models.RegularizationRecord
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &records))
	assert.Len(t, records, 2)
	assert.Equal(t, rec2.ID, records[0].ID, "列表应按创建时间倒序")

	// 状态过滤：pending_supervisor 仅 1 条
	resp = doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records?status=pending_supervisor", token, nil)
	require.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &records))
	assert.Len(t, records, 1)
	assert.Equal(t, rec1.ID, records[0].ID)

	// 来源过滤：manual 全部命中
	resp = doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records?source=manual", token, nil)
	require.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &records))
	assert.Len(t, records, 2)

	// 非法状态参数 → 400
	resp = doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records?status=approved", token, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "非法状态参数应 400: %d %s", resp.Code, resp.Body.String())

	// 详情返回完整记录
	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", rec1.ID), token, nil)
	require.Equal(t, http.StatusOK, resp.Code, "详情应成功: %d %s", resp.Code, resp.Body.String())
	var detail models.RegularizationRecord
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &detail))
	assert.Equal(t, rec1.ID, detail.ID)
	assert.Equal(t, models.RegularizationStatusPendingSupervisor, detail.Status)
	assert.Equal(t, "转正员工-REG-2026-001", detail.SnapshotName, "详情应返回快照字段")

	// 不存在 ID → 404
	resp = doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records/99999", token, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code, "不存在记录应 404: %d %s", resp.Code, resp.Body.String())

	// 非法 ID → 400
	resp = doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records/abc", token, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "非法 ID 应 400: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationTenantIsolation 不同租户不可见：
// 租户 B（admin 角色、具备 employee.edit）列表为空、详情 404。
func TestRegularizationTenantIsolation(t *testing.T) {
	env := setupRegularizationTestEnv(t)
	rec := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-001", models.RegularizationStatusPendingSupervisor)
	tenantB := createRBACTestUser(t, env.db, "tenantb", "tenantb-uuid", env.roleIDs["admin"])

	// 租户 B 有 employee.edit（admin 角色），但看不到租户 A 的数据
	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records", tenantB.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, resp.Code, "租户B列表应成功: %d %s", resp.Code, resp.Body.String())
	var records []models.RegularizationRecord
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &records))
	assert.Len(t, records, 0, "不同租户列表应为空")

	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", rec.ID), tenantB.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code, "不同租户详情应 404: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationDepartmentIsolation 部门隔离复用现有规则（DepartmentContext）：
// 用户有部门时仅可见快照部门一致的记录，越部门详情 404。
func TestRegularizationDepartmentIsolation(t *testing.T) {
	env := setupRegularizationTestEnv(t)
	// 管理员归属研发部（DepartmentContext 按 User.Department 注入）
	require.NoError(t, env.db.Model(&env.admin).Update("department", "研发部").Error)
	recSame := createRegularizationRecord(t, env.db, env.admin.ID, "研发部", "REG-2026-001", models.RegularizationStatusPendingSupervisor)
	recOther := createRegularizationRecord(t, env.db, env.admin.ID, "市场部", "REG-2026-002", models.RegularizationStatusPendingSupervisor)
	token := env.admin.SupabaseUID

	// 列表仅返回本部门记录
	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records", token, nil)
	require.Equal(t, http.StatusOK, resp.Code, "部门隔离列表应成功: %d %s", resp.Code, resp.Body.String())
	var records []models.RegularizationRecord
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &records))
	assert.Len(t, records, 1, "仅应返回本部门记录")
	assert.Equal(t, recSame.ID, records[0].ID)

	// 本部门详情可见
	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", recSame.ID), token, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "本部门详情应成功: %d %s", resp.Code, resp.Body.String())

	// 跨部门详情 404
	resp = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/regularization-records/%d", recOther.ID), token, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code, "跨部门详情应 404: %d %s", resp.Code, resp.Body.String())
}

// ---------- P12.3.3-3 写接口测试 ----------

// setupRegularizationWriteTestEnv 初始化转正写接口测试环境（迁移 Employee/WorkTodo，admin 归属 tenant-a）。
func setupRegularizationWriteTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRegularizationTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.Employee{}, &models.WorkTodo{}))
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", env.admin.ID).Update("company_id", "tenant-a").Error)
	return env
}

// createRegularizationApprover 创建指定租户审批人（admin 角色，自动继承 employee.edit）。
func createRegularizationApprover(t *testing.T, env *rbacTestEnv, username, companyID string) models.User {
	t.Helper()
	user := createRBACTestUser(t, env.db, username, username+"-uuid", env.roleIDs["admin"])
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", user.ID).Update("company_id", companyID).Error)
	return user
}

// createRegularizationTestEmployee 创建转正测试员工。
func createRegularizationTestEmployee(t *testing.T, env *rbacTestEnv, idNumber, department, status, empStatus string) models.Employee {
	t.Helper()
	emp := models.Employee{
		UserID:           env.admin.ID,
		EmployeeID:       "REG-" + idNumber[len(idNumber)-4:],
		Name:             "转正员工-" + idNumber[len(idNumber)-4:],
		IDNumber:         idNumber,
		Department:       department,
		Position:         "后端工程师",
		Status:           status,
		EmploymentStatus: empStatus,
		ProbationEndDate: "2026-08-01",
	}
	require.NoError(t, env.db.Create(&emp).Error)
	return emp
}

// fixRegularizationNow 固定上海业务时间（测试用），结束后恢复原函数。
func fixRegularizationNow(t *testing.T, y int, m time.Month, d int) {
	t.Helper()
	original := regularizationNowFunc
	loc := shanghaiLoc(t)
	regularizationNowFunc = func() time.Time {
		return time.Date(y, m, d, 12, 0, 0, 0, loc)
	}
	t.Cleanup(func() { regularizationNowFunc = original })
}

// createRegularizationViaAPI 通过 API 创建转正申请并断言成功。
func createRegularizationViaAPI(t *testing.T, env *rbacTestEnv, employeeID uint, supervisor, hrReviewer models.User, plannedDate string) models.RegularizationRecord {
	t.Helper()
	body := map[string]any{
		"employee_id":                 employeeID,
		"contract_term_months":        36,
		"employee_self_review":        "试用期表现良好",
		"supervisor_approver_user_id": supervisor.ID,
		"hr_reviewer_user_id":         hrReviewer.ID,
		"planned_regular_date":        plannedDate,
	}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建转正申请应成功: %d %s", rec.Code, rec.Body.String())
	var record models.RegularizationRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &record))
	return record
}

// reloadRegularization 重载转正记录。
func reloadRegularization(t *testing.T, db *gorm.DB, id uint) models.RegularizationRecord {
	t.Helper()
	var record models.RegularizationRecord
	require.NoError(t, db.First(&record, id).Error)
	return record
}

// TestRegularizationCreateTodayEffective 正常主流程-今天计划日：创建→上级通过→HR通过立即生效。
func TestRegularizationCreateTodayEffective(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-a", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-a", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011234", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-18")
	assert.Equal(t, models.RegularizationStatusPendingSupervisor, record.Status)
	assert.Equal(t, "manual", record.Source)
	assert.NotEmpty(t, record.ApprovalNo, "审批编号应服务端生成")
	assert.Equal(t, emp.Name, record.SnapshotName)
	assert.Equal(t, "研发部", record.SnapshotDepartment)
	assert.Equal(t, emp.EmploymentStatus, record.SnapshotEmploymentStatus)
	assert.Equal(t, "2026-08-01", record.SnapshotProbationEndDate, "试用期结束日快照")
	require.NotNil(t, record.InitiatorHRUserID)
	assert.Equal(t, env.admin.ID, *record.InitiatorHRUserID, "发起人必须为当前用户")

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, map[string]any{"comment": "同意"})
	assert.Equal(t, http.StatusOK, resp.Code, "上级通过应成功: %d %s", resp.Code, resp.Body.String())
	assert.Equal(t, models.RegularizationStatusPendingHRReview, reloadRegularization(t, env.db, record.ID).Status)

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, map[string]any{"comment": "复核通过"})
	assert.Equal(t, http.StatusOK, resp.Code, "HR通过应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusEffective, saved.Status)
	assert.Equal(t, "2026-08-18", saved.ActualRegularDate)

	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, empSaved.EmploymentStatus, "员工应转正式")
	assert.Equal(t, "2026-08-18", empSaved.ActualRegularDate)
	assert.Equal(t, models.EmployeeStatusActive, empSaved.Status, "员工在职状态不变")

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, map[string]any{})
	assert.Equal(t, http.StatusConflict, resp.Code, "重复HR通过应 409")
}

// TestRegularizationCreatePastEffective 正常主流程-过去计划日：HR通过立即生效，实际日期记录计划日期。
func TestRegularizationCreatePastEffective(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-b", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-b", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011235", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-10")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)
	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "过去计划日HR通过应生效: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusEffective, saved.Status)
	assert.Equal(t, "2026-08-10", saved.ActualRegularDate, "实际转正日期记录计划日期")
	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, empSaved.EmploymentStatus)
	assert.Equal(t, "2026-08-10", empSaved.ActualRegularDate)
}

// TestRegularizationCreateFutureScheduled 正常主流程-未来计划日：HR通过仅排期，员工保持 trial。
func TestRegularizationCreateFutureScheduled(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-c", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-c", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011236", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)
	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "未来计划日HR通过应排期: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusScheduled, saved.Status)
	assert.Equal(t, "", saved.ActualRegularDate, "排期不写实际转正日期")
	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusTrial, empSaved.EmploymentStatus, "排期员工保持试用")
}

// TestRegularizationSupervisorReject 上级拒绝：原因必填，拒绝后转 rejected。
func TestRegularizationSupervisorReject(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-d", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-d", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011237", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-reject", record.ID), supervisor.SupabaseUID, map[string]any{"reason": "  "})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "拒绝原因必填应 400: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-reject", record.ID), supervisor.SupabaseUID, map[string]any{"reason": "试用期表现不达标", "comment": "需延长观察"})
	assert.Equal(t, http.StatusOK, resp.Code, "上级拒绝应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusRejected, saved.Status)
	assert.Equal(t, "试用期表现不达标", saved.RejectionReason)
	require.NotNil(t, saved.SupervisorRejectedAt)
	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusTrial, empSaved.EmploymentStatus, "拒绝不改变员工状态")
}

// TestRegularizationHRRejectCreatesUniqueTodo HR 拒绝：员工保持 active+trial，同事务创建唯一离职待办。
func TestRegularizationHRRejectCreatesUniqueTodo(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-e", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-e", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011238", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-reject", record.ID), reviewer.SupabaseUID, map[string]any{"reason": ""})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "HR拒绝原因必填应 400: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-reject", record.ID), reviewer.SupabaseUID, map[string]any{"reason": "不符合转正条件", "comment": "建议离职"})
	assert.Equal(t, http.StatusOK, resp.Code, "HR拒绝应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusRejected, saved.Status)
	assert.Equal(t, "不符合转正条件", saved.RejectionReason)

	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmployeeStatusActive, empSaved.Status, "HR拒绝不自动离职")
	assert.Equal(t, models.EmploymentStatusTrial, empSaved.EmploymentStatus, "HR拒绝员工保持试用")

	var todos []models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "regularization_rejection", record.ID).Find(&todos).Error)
	require.Len(t, todos, 1, "HR拒绝应创建唯一离职待办")
	assert.Equal(t, env.admin.ID, todos[0].UserID, "待办归属发起人租户")
	require.NotNil(t, todos[0].AssigneeID)
	assert.Equal(t, env.admin.ID, *todos[0].AssigneeID, "待办归属发起HR")

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-reject", record.ID), reviewer.SupabaseUID, map[string]any{"reason": "再次拒绝"})
	assert.Equal(t, http.StatusConflict, resp.Code, "重复HR拒绝应 409: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationApproverMismatchAndOutOfOrder 审批人不匹配/越级/重复一律 409。
func TestRegularizationApproverMismatchAndOutOfOrder(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-f", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-f", "tenant-a")
	other := createRegularizationApprover(t, env, "other-f", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011239", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), other.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "非指定上级审批应 409: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "越级HR复核应 409（未到HR步骤）: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "上级通过应成功: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "重复上级通过应 409: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-reject", record.ID), supervisor.SupabaseUID, map[string]any{"reason": "重复拒绝"})
	assert.Equal(t, http.StatusConflict, resp.Code, "上级重复拒绝应 409: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), supervisor.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "主管操作HR复核应 409: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationThreeApproversSameRejected 三人相同一律 400。
func TestRegularizationThreeApproversSameRejected(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	reviewer := createRegularizationApprover(t, env, "hrr-g", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011240", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	body := map[string]any{
		"employee_id": emp.ID, "contract_term_months": 36,
		"supervisor_approver_user_id": env.admin.ID, "hr_reviewer_user_id": reviewer.ID,
		"planned_regular_date": "2026-08-20",
	}
	resp := doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "发起人=上级应 400: %d %s", resp.Code, resp.Body.String())

	body["supervisor_approver_user_id"] = reviewer.ID
	body["hr_reviewer_user_id"] = reviewer.ID
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "上级=HR复核应 400: %d %s", resp.Code, resp.Body.String())

	body["supervisor_approver_user_id"] = 0
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "缺上级应 400: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationPostponeOnceThenReject 延期一次成功、二次拒绝、延期到过去立即生效。
func TestRegularizationPostponeOnceThenReject(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-h", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-h", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011241", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/postpone", record.ID), reviewer.SupabaseUID, map[string]any{"new_planned_regular_date": "2026-09-01"})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "延期原因必填应 400: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/postpone", record.ID), reviewer.SupabaseUID, map[string]any{"new_planned_regular_date": "2026/09/01", "reason": "延长试用"})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "延期日期格式错误应 400: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/postpone", record.ID), reviewer.SupabaseUID, map[string]any{"new_planned_regular_date": "2026-09-01", "reason": "需延长试用期", "comment": "复核同意延期"})
	assert.Equal(t, http.StatusOK, resp.Code, "延期应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusPostponedScheduled, saved.Status)
	assert.Equal(t, "2026-09-01", saved.PlannedRegularDate)
	assert.Equal(t, 1, saved.ExtensionCount)
	assert.Equal(t, "2026-08-20", saved.OriginalPlannedRegularDate, "首次延期记录原计划日期")
	assert.Equal(t, "需延长试用期", saved.PostponedReason)
	assert.Equal(t, "研发部", saved.SnapshotDepartment, "延期不覆盖快照")

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/postpone", record.ID), reviewer.SupabaseUID, map[string]any{"new_planned_regular_date": "2026-10-01", "reason": "再次延期"})
	assert.Equal(t, http.StatusConflict, resp.Code, "二次延期应 409: %d %s", resp.Code, resp.Body.String())

	// 延期到过去 → 同事务立即生效
	emp2 := createRegularizationTestEmployee(t, env, "110101199001011242", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record2 := createRegularizationViaAPI(t, env, emp2.ID, supervisor, reviewer, "2026-08-20")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record2.ID), supervisor.SupabaseUID, nil)
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/postpone", record2.ID), reviewer.SupabaseUID, map[string]any{"new_planned_regular_date": "2026-08-10", "reason": "提前转正"})
	assert.Equal(t, http.StatusOK, resp.Code, "延期到过去应立即生效: %d %s", resp.Code, resp.Body.String())
	saved2 := reloadRegularization(t, env.db, record2.ID)
	assert.Equal(t, models.RegularizationStatusEffective, saved2.Status, "延期后立即生效状态须为 effective")
	assert.Equal(t, "2026-08-10", saved2.ActualRegularDate)
	var emp2Saved models.Employee
	require.NoError(t, env.db.First(&emp2Saved, emp2.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, emp2Saved.EmploymentStatus)
}

// TestRegularizationVoid 作废：非 effective 可作废、effective 不可作废、重复作废 409、作废后可重新发起。
func TestRegularizationVoid(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-i", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-i", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011243", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")

	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/void", record.ID), env.admin.SupabaseUID, map[string]any{"reason": ""})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "作废原因必填应 400: %d %s", resp.Code, resp.Body.String())

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/void", record.ID), env.admin.SupabaseUID, map[string]any{"reason": "信息填写错误"})
	assert.Equal(t, http.StatusOK, resp.Code, "作废应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusVoided, saved.Status)
	assert.Equal(t, "信息填写错误", saved.VoidReason)
	require.NotNil(t, saved.VoidedAt)
	assert.Equal(t, "研发部", saved.SnapshotDepartment, "作废不覆盖快照")

	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/void", record.ID), env.admin.SupabaseUID, map[string]any{"reason": "重复作废"})
	assert.Equal(t, http.StatusConflict, resp.Code, "重复作废应 409: %d %s", resp.Code, resp.Body.String())

	// 作废后可重新发起（无进行中记录）
	record2 := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-25")
	assert.Equal(t, models.RegularizationStatusPendingSupervisor, record2.Status)

	// effective 记录不可作废
	emp2 := createRegularizationTestEmployee(t, env, "110101199001011244", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	rec3 := createRegularizationViaAPI(t, env, emp2.ID, supervisor, reviewer, "2026-08-18")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", rec3.ID), supervisor.SupabaseUID, nil)
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", rec3.ID), reviewer.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "rec3 HR通过应生效: %d %s", resp.Code, resp.Body.String())
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/void", rec3.ID), env.admin.SupabaseUID, map[string]any{"reason": "试图作废已生效"})
	assert.Equal(t, http.StatusConflict, resp.Code, "effective 记录不可作废应 409: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationManualEffect 人工生效：计划日已到生效成功，未来计划日 409 且状态不变，非排期 409。
func TestRegularizationManualEffect(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	emp := createRegularizationTestEmployee(t, env, "110101199001011245", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	due := models.RegularizationRecord{
		UserID: env.admin.ID, EmployeeID: &emp.ID,
		SnapshotName: emp.Name, SnapshotDepartment: emp.Department,
		SnapshotPosition: emp.Position, SnapshotEmploymentStatus: emp.EmploymentStatus,
		SnapshotProbationEndDate: emp.ProbationEndDate,
		ApprovalNo:               "REG-MANUAL-001", PlannedRegularDate: "2026-08-10",
		Status: models.RegularizationStatusScheduled, Source: models.RegularizationSourceManual,
	}
	require.NoError(t, env.db.Create(&due).Error)
	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/effect", due.ID), env.admin.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code, "计划日已到人工生效应成功: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, due.ID)
	assert.Equal(t, models.RegularizationStatusEffective, saved.Status)
	assert.Equal(t, "2026-08-10", saved.ActualRegularDate)
	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, empSaved.EmploymentStatus)

	// 未来计划日 scheduled → 409 且状态不变（不得静默改状态）
	emp2 := createRegularizationTestEmployee(t, env, "110101199001011246", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	future := models.RegularizationRecord{
		UserID: env.admin.ID, EmployeeID: &emp2.ID,
		SnapshotName: emp2.Name, SnapshotDepartment: emp2.Department,
		SnapshotPosition: emp2.Position, SnapshotEmploymentStatus: emp2.EmploymentStatus,
		SnapshotProbationEndDate: emp2.ProbationEndDate,
		ApprovalNo:               "REG-MANUAL-002", PlannedRegularDate: "2026-08-20",
		Status: models.RegularizationStatusScheduled, Source: models.RegularizationSourceManual,
	}
	require.NoError(t, env.db.Create(&future).Error)
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/effect", future.ID), env.admin.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "未来计划日人工生效应 409: %d %s", resp.Code, resp.Body.String())
	assert.Equal(t, models.RegularizationStatusScheduled, reloadRegularization(t, env.db, future.ID).Status, "失败不得改状态")
	var emp2Saved models.Employee
	require.NoError(t, env.db.First(&emp2Saved, emp2.ID).Error)
	assert.Equal(t, models.EmploymentStatusTrial, emp2Saved.EmploymentStatus, "失败员工状态不变")

	// 非排期状态不可人工生效
	emp3 := createRegularizationTestEmployee(t, env, "110101199001011252", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	supervisor := createRegularizationApprover(t, env, "sup-j", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-j", "tenant-a")
	rec := createRegularizationViaAPI(t, env, emp3.ID, supervisor, reviewer, "2026-08-25")
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/effect", rec.ID), env.admin.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "非排期状态人工生效应 409: %d %s", resp.Code, resp.Body.String())
}

// TestRegularizationHRApproveRollback 事务回滚：员工不存在时 HR 通过失败，记录状态保持 pending_hr_review。
func TestRegularizationHRApproveRollback(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-k", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-k", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011247", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-18")
	doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)

	// 破坏关联：employee_id 指向不存在员工 → HR 通过事务内加载员工失败
	require.NoError(t, env.db.Model(&models.RegularizationRecord{}).Where("id = ?", record.ID).Update("employee_id", 99999).Error)
	resp := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/hr-approve", record.ID), reviewer.SupabaseUID, nil)
	assert.Equal(t, http.StatusConflict, resp.Code, "员工不存在HR通过应 409: %d %s", resp.Code, resp.Body.String())
	saved := reloadRegularization(t, env.db, record.ID)
	assert.Equal(t, models.RegularizationStatusPendingHRReview, saved.Status, "事务回滚后记录状态不变")
	assert.Equal(t, "", saved.ActualRegularDate, "事务回滚后不写实际转正日期")
}

// TestRegularizationCrossTenantAndDepartment 跨租户/跨部门/员工状态/进行中重复发起禁止。
func TestRegularizationCrossTenantAndDepartment(t *testing.T) {
	env := setupRegularizationWriteTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	supervisor := createRegularizationApprover(t, env, "sup-l", "tenant-a")
	reviewer := createRegularizationApprover(t, env, "hrr-l", "tenant-a")
	emp := createRegularizationTestEmployee(t, env, "110101199001011248", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)

	// 跨租户审批人 → 400
	crossTenant := createRegularizationApprover(t, env, "sup-cross", "tenant-b")
	body := map[string]any{
		"employee_id": emp.ID, "contract_term_months": 36,
		"supervisor_approver_user_id": crossTenant.ID, "hr_reviewer_user_id": reviewer.ID,
		"planned_regular_date": "2026-08-20",
	}
	resp := doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "跨租户审批人应 400: %d %s", resp.Code, resp.Body.String())

	// 部门不一致：admin 归属研发部，员工归属市场部 → 404
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", env.admin.ID).Update("department", "研发部").Error)
	otherDeptEmp := createRegularizationTestEmployee(t, env, "110101199001011249", "市场部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	body["employee_id"] = otherDeptEmp.ID
	body["supervisor_approver_user_id"] = supervisor.ID
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusNotFound, resp.Code, "跨部门员工发起应 404: %d %s", resp.Code, resp.Body.String())

	// 审批时跨部门：supervisor 归属市场部，记录快照研发部 → 404
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", env.admin.ID).Update("department", "").Error)
	record := createRegularizationViaAPI(t, env, emp.ID, supervisor, reviewer, "2026-08-20")
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", supervisor.ID).Update("department", "市场部").Error)
	resp = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/regularization-records/%d/supervisor-approve", record.ID), supervisor.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code, "审批人跨部门应 404: %d %s", resp.Code, resp.Body.String())
	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", supervisor.ID).Update("department", "").Error)

	// 已有进行中记录 → 409
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, map[string]any{
		"employee_id": emp.ID, "contract_term_months": 36,
		"supervisor_approver_user_id": supervisor.ID, "hr_reviewer_user_id": reviewer.ID,
		"planned_regular_date": "2026-09-01",
	})
	assert.Equal(t, http.StatusConflict, resp.Code, "已有进行中转正申请应 409: %d %s", resp.Code, resp.Body.String())

	// formal / resigned / 不存在员工
	formalEmp := createRegularizationTestEmployee(t, env, "110101199001011250", "研发部", models.EmployeeStatusActive, models.EmploymentStatusFormal)
	body["employee_id"] = formalEmp.ID
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusConflict, resp.Code, "formal 员工应 409: %d %s", resp.Code, resp.Body.String())

	resignedEmp := createRegularizationTestEmployee(t, env, "110101199001011251", "研发部", models.EmployeeStatusResigned, "")
	body["employee_id"] = resignedEmp.ID
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusConflict, resp.Code, "resigned 员工应 409: %d %s", resp.Code, resp.Body.String())

	body["employee_id"] = 99999
	resp = doRBACRequest(t, env.router, http.MethodPost, "/api/regularization-records", env.admin.SupabaseUID, body)
	assert.Equal(t, http.StatusNotFound, resp.Code, "不存在员工应 404: %d %s", resp.Code, resp.Body.String())
}
