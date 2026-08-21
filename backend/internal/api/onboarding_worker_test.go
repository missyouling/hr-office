package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupOnboardingWorkerTestEnv 初始化 worker 测试环境（含 WorkTodo/OnboardingImportRun/SystemLog 迁移）。
func setupOnboardingWorkerTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupOnboardingTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.WorkTodo{}, &models.OnboardingImportRun{}, &SystemLog{}))
	return env
}

// shanghaiLoc 返回 Asia/Shanghai 时区。
func shanghaiLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	return loc
}

// createPendingOnboarding 直接创建 pending 入职记录。
func createPendingOnboarding(t *testing.T, db *gorm.DB, userID uint, name, idNumber, plannedDate string) models.OnboardingRecord {
	t.Helper()
	rec := models.OnboardingRecord{
		UserID:          userID,
		Name:            name,
		IDNumber:        idNumber,
		Department:      "测试部",
		Position:        "测试岗",
		PlannedHireDate: plannedDate,
		Status:          models.OnboardingStatusPending,
	}
	require.NoError(t, db.Create(&rec).Error)
	return rec
}

// TestOnboardingWorkerSuccess worker 成功：pending → onboarded + Employee + WorkTodo（归属租户管理员）。
func TestOnboardingWorkerSuccess(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	rec := createPendingOnboarding(t, env.db, env.admin.ID, "入职成功", "110101199001011234", "2026-08-18")

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	// 记录转 onboarded
	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusOnboarded, saved.Status)
	require.NotNil(t, saved.ActualHireDate)
	assert.Equal(t, "2026-08-18", *saved.ActualHireDate)
	require.NotNil(t, saved.EmployeeID)
	assert.Equal(t, models.EmploymentStatusTrial, saved.EmploymentStatus, "未指定用工状态默认 trial")

	// Employee 创建：active + 实际日期 + 工号（部门编码+三位序号）
	var emp models.Employee
	require.NoError(t, env.db.First(&emp, *saved.EmployeeID).Error)
	assert.Equal(t, "active", emp.Status)
	assert.Equal(t, "2026-08-18", emp.HireDate)
	assert.Equal(t, "DEV001", emp.EmployeeID, "首个工号应为部门编码+001")
	assert.Equal(t, rec.Name, emp.Name)
	assert.Equal(t, rec.IDNumber, emp.IDNumber)

	// WorkTodo 归属租户管理员（admin 用户）
	var todo models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "onboarding", rec.ID).First(&todo).Error)
	assert.Equal(t, env.admin.ID, todo.UserID, "待办应归属租户管理员")
	require.NotNil(t, todo.AssigneeID)
	assert.Equal(t, env.admin.ID, *todo.AssigneeID)
	assert.Equal(t, models.WorkTodoStatusPending, todo.Status)

	// 运行记录
	var run models.OnboardingImportRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, models.OnboardingRunStatusSuccess, run.Status)
	assert.Equal(t, 1, run.Processed)
	assert.Zero(t, run.Failed)
}

// TestOnboardingWorkerEmploymentStatusFromImport worker 使用导入时指定的用工状态。
func TestOnboardingWorkerEmploymentStatusFromImport(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	rec := models.OnboardingRecord{
		UserID:           env.admin.ID,
		Name:             "正式入职",
		IDNumber:         "110101199001011234",
		Department:       "测试部",
		PlannedHireDate:  "2026-08-18",
		Status:           models.OnboardingStatusPending,
		EmploymentStatus: models.EmploymentStatusFormal,
	}
	require.NoError(t, env.db.Create(&rec).Error)

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, saved.EmploymentStatus, "应使用导入时指定的正式状态")
}

// TestOnboardingWorkerFailure 失败保留 pending + 写日志 + 异常待办，不自动重试。
func TestOnboardingWorkerFailure(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	// 预置冲突员工 → worker 处理时身份证冲突失败
	createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011234", "active")
	rec := createPendingOnboarding(t, env.db, env.admin.ID, "冲突入职", "110101199001011234", "2026-08-18")

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	// 记录保留 pending
	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusPending, saved.Status, "失败应保留 pending")

	// 未创建新员工（仅预置的 1 个冲突员工）
	var empCount int64
	require.NoError(t, env.db.Model(&models.Employee{}).Where("user_id = ?", env.admin.ID).Count(&empCount).Error)
	assert.Equal(t, int64(1), empCount, "失败不应创建新员工")

	// 创建唯一异常待办
	var todo models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "onboarding_exception", rec.ID).First(&todo).Error)
	assert.Equal(t, env.admin.ID, todo.UserID)

	// 写日志
	var logs []SystemLog
	require.NoError(t, env.db.Where("source = ? AND level = ?", "onboarding-worker", "error").Find(&logs).Error)
	assert.NotEmpty(t, logs, "失败应写系统日志")
	for _, entry := range logs {
		assert.NotContains(t, string(entry.Details), rec.IDNumber, "日志不得泄露身份证号")
	}

	// 运行记录标记失败
	var run models.OnboardingImportRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, models.OnboardingRunStatusFailed, run.Status)
	assert.Equal(t, 1, run.Failed)
	assert.Zero(t, run.Processed)
}

// TestOnboardingWorkerIdempotent 同日运行记录存在则跳过（幂等不重试）。
func TestOnboardingWorkerIdempotent(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	// 预置当日运行记录
	require.NoError(t, env.db.Create(&models.OnboardingImportRun{
		RunDate: "2026-08-18", Status: models.OnboardingRunStatusSuccess,
	}).Error)

	rec := createPendingOnboarding(t, env.db, env.admin.ID, "不应处理", "110101199001011234", "2026-08-18")

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	// 记录仍为 pending（未被处理）
	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusPending, saved.Status, "同日已运行应跳过处理")

	// 运行记录仍只有一条
	var runCount int64
	require.NoError(t, env.db.Model(&models.OnboardingImportRun{}).Where("run_date = ?", "2026-08-18").Count(&runCount).Error)
	assert.Equal(t, int64(1), runCount, "同日运行记录不应重复创建")
}

// TestOnboardingWorkerTodoDedup 待办唯一去重：同租户同业务类型+业务ID 仅一条。
func TestOnboardingWorkerTodoDedup(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	rec := createPendingOnboarding(t, env.db, env.admin.ID, "去重测试", "110101199001011234", "2026-08-18")

	handler := NewHandler(env.db)
	// 连续两次运行（第二次因同日幂等会跳过，直接验证唯一索引兜底）
	handler.runOnboardingWorkerOnce(loc, now)

	// 直接再创建一条同业务待办 → 唯一约束拒绝
	dup := models.WorkTodo{
		UserID:       env.admin.ID,
		BusinessType: "onboarding",
		BusinessID:   rec.ID,
		Title:        "重复待办",
		Status:       models.WorkTodoStatusPending,
	}
	err := env.db.Create(&dup).Error
	assert.Error(t, err, "同租户同业务待办应被唯一约束拒绝")
	assert.True(t, isUniqueViolation(err), "应为唯一约束冲突")

	var todoCount int64
	require.NoError(t, env.db.Model(&models.WorkTodo{}).Where("business_type = ? AND business_id = ?", "onboarding", rec.ID).Count(&todoCount).Error)
	assert.Equal(t, int64(1), todoCount, "待办应唯一")
}

// TestOnboardingWorkerAsiaShanghaiBoundary Asia/Shanghai 边界：只处理 planned_date 等于当日的记录。
func TestOnboardingWorkerAsiaShanghaiBoundary(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)

	// 当日（处理）
	recToday := createPendingOnboarding(t, env.db, env.admin.ID, "当日入职", "110101199001011234", "2026-08-18")
	// 未来（不处理）
	recFuture := createPendingOnboarding(t, env.db, env.admin.ID, "未来入职", "110101199002022345", "2026-08-19")
	// 过去（不处理）
	recPast := createPendingOnboarding(t, env.db, env.admin.ID, "过去入职", "110101199003033456", "2026-08-17")

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	var s1, s2, s3 models.OnboardingRecord
	require.NoError(t, env.db.First(&s1, recToday.ID).Error)
	require.NoError(t, env.db.First(&s2, recFuture.ID).Error)
	require.NoError(t, env.db.First(&s3, recPast.ID).Error)
	assert.Equal(t, models.OnboardingStatusOnboarded, s1.Status, "当日记录应处理")
	assert.Equal(t, models.OnboardingStatusPending, s2.Status, "未来记录不应处理")
	assert.Equal(t, models.OnboardingStatusPending, s3.Status, "历史记录不应处理")

	// 运行记录统计
	var run models.OnboardingImportRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, 1, run.Processed, "只应处理当日记录")
}

// TestOnboardingWorkerTimezoneCrossDay 时区跨日：UTC 时间对应上海次日时按上海日期运行。
func TestOnboardingWorkerTimezoneCrossDay(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	// UTC 2026-08-17 18:30 = 上海 2026-08-18 02:30（已过 02:00 窗口，调度器会顺延次日）
	// 直接验证 runOnboardingWorkerOnce 使用上海日期
	now := time.Date(2026, 8, 17, 18, 30, 0, 0, time.UTC)

	rec := createPendingOnboarding(t, env.db, env.admin.ID, "跨日入职", "110101199001011234", "2026-08-18")

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	// 上海日期为 2026-08-18，运行记录按上海日期
	var run models.OnboardingImportRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, models.OnboardingRunStatusSuccess, run.Status)

	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusOnboarded, saved.Status)
	require.NotNil(t, saved.ActualHireDate)
	assert.Equal(t, "2026-08-18", *saved.ActualHireDate, "实际入职日期应为上海当日")
}

// TestGenerateEmployeeID 工号生成：部门编码+三位序号，非匹配历史工号忽略。
func TestGenerateEmployeeID(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)

	// 无记录 → DEV001
	id, err := generateEmployeeID(env.db, env.admin.ID, "DEV")
	require.NoError(t, err)
	assert.Equal(t, "DEV001", id)

	// 已有 DEV001 → DEV002
	require.NoError(t, env.db.Create(&models.Employee{UserID: env.admin.ID, EmployeeID: "DEV001", Name: "已有", IDNumber: "110101199001011234", Status: "active"}).Error)
	id, err = generateEmployeeID(env.db, env.admin.ID, "DEV")
	require.NoError(t, err)
	assert.Equal(t, "DEV002", id)

	// 非匹配历史工号忽略（EMP-X、纯数字 1005 不影响 DEV 前缀）
	require.NoError(t, env.db.Create(&models.Employee{UserID: env.admin.ID, EmployeeID: "EMP-X", Name: "非数字", IDNumber: "110101199002022345", Status: "active"}).Error)
	require.NoError(t, env.db.Create(&models.Employee{UserID: env.admin.ID, EmployeeID: "1005", Name: "数字", IDNumber: "110101199003033456", Status: "active"}).Error)
	id, err = generateEmployeeID(env.db, env.admin.ID, "DEV")
	require.NoError(t, err)
	assert.Equal(t, "DEV002", id, "非匹配历史工号应被忽略")

	// 已有 DEV002 → DEV003
	require.NoError(t, env.db.Create(&models.Employee{UserID: env.admin.ID, EmployeeID: "DEV002", Name: "已有2", IDNumber: "110101199004044567", Status: "active"}).Error)
	id, err = generateEmployeeID(env.db, env.admin.ID, "DEV")
	require.NoError(t, err)
	assert.Equal(t, "DEV003", id)

	// 不同部门前缀互不影响
	id, err = generateEmployeeID(env.db, env.admin.ID, "HR")
	require.NoError(t, err)
	assert.Equal(t, "HR001", id, "不同部门前缀应独立递增")
}

// TestEmployeeIDUniqueAndNextSequence 复合唯一索引拒绝重复工号，重新计算后取得下一工号。
func TestEmployeeIDUniqueAndNextSequence(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	first := models.Employee{UserID: env.admin.ID, EmployeeID: "DEV001", Name: "甲", IDNumber: "110101199001011234", Status: "active"}
	require.NoError(t, env.db.Create(&first).Error)

	duplicate := models.Employee{UserID: env.admin.ID, EmployeeID: "DEV001", Name: "乙", IDNumber: "110101199002022345", Status: "active"}
	err := env.db.Create(&duplicate).Error
	require.Error(t, err)
	assert.True(t, isUniqueViolation(err), "重复工号应被数据库拒绝")

	nextID, err := generateEmployeeID(env.db, env.admin.ID, "DEV")
	require.NoError(t, err)
	assert.Equal(t, "DEV002", nextID)
}

// TestOnboardingWorkerFailureNotRetriedNextDay 当日失败后，次日扫描不再自动重试历史记录。
func TestOnboardingWorkerFailureNotRetriedNextDay(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011234", "active")
	rec := createPendingOnboarding(t, env.db, env.admin.ID, "冲突入职", "110101199001011234", "2026-08-18")
	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, time.Date(2026, 8, 18, 2, 0, 0, 0, loc))

	require.NoError(t, env.db.Delete(&models.Employee{}, "user_id = ?", env.admin.ID).Error)
	handler.runOnboardingWorkerOnce(loc, time.Date(2026, 8, 19, 2, 0, 0, 0, loc))

	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusPending, saved.Status)
}

// TestFindTenantAdminIDByCompany 管理员按创建者公司选择最早创建者，不回退创建者本人。
func TestFindTenantAdminIDByCompany(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	require.NoError(t, env.db.Model(&models.User{}).Where("id IN ?", []uint{env.admin.ID, env.manager.ID}).Update("company_id", "company-a").Error)

	adminID, err := findTenantAdminID(env.db, env.manager.ID)
	require.NoError(t, err)
	assert.Equal(t, env.admin.ID, adminID)

	require.NoError(t, env.db.Model(&models.User{}).Where("id = ?", env.manager.ID).Update("company_id", "company-without-admin").Error)
	_, err = findTenantAdminID(env.db, env.manager.ID)
	assert.ErrorIs(t, err, ErrOnboardingAdminMissing)
}

// TestOnboardingWorkerDeptCodeMissing worker 部门无编码：保留 pending + 写日志 + 唯一异常待办 + 不创建员工
func TestOnboardingWorkerDeptCodeMissing(t *testing.T) {
	env := setupOnboardingWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)
	createTestDepartment(t, env.db, env.admin.ID, "无码部门", "")

	rec := models.OnboardingRecord{
		UserID:          env.admin.ID,
		Name:            "异常入职",
		IDNumber:        "110101199001011234",
		Department:      "无码部门",
		Position:        "测试岗",
		PlannedHireDate: "2026-08-18",
		Status:          models.OnboardingStatusPending,
	}
	require.NoError(t, env.db.Create(&rec).Error)

	handler := NewHandler(env.db)
	handler.runOnboardingWorkerOnce(loc, now)

	// 保留 pending
	var saved models.OnboardingRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.OnboardingStatusPending, saved.Status, "部门无编码应保留 pending")

	// 未创建员工
	var empCount int64
	require.NoError(t, env.db.Model(&models.Employee{}).Where("user_id = ?", env.admin.ID).Count(&empCount).Error)
	assert.Zero(t, empCount, "部门无编码不应创建员工")

	// 唯一异常待办（business_type=onboarding_exception，归属租户管理员）
	var todo models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "onboarding_exception", rec.ID).First(&todo).Error)
	assert.Equal(t, env.admin.ID, todo.UserID)
	require.NotNil(t, todo.AssigneeID)
	assert.Equal(t, env.admin.ID, *todo.AssigneeID)
	assert.Contains(t, todo.Title, "入职异常")

	// 写系统日志
	var logs []SystemLog
	require.NoError(t, env.db.Where("source = ? AND level = ?", "onboarding-worker", "error").Find(&logs).Error)
	assert.NotEmpty(t, logs, "部门无编码应写系统日志")

	// 运行记录标记失败
	var run models.OnboardingImportRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, models.OnboardingRunStatusFailed, run.Status)
	assert.Equal(t, 1, run.Failed)
	assert.Zero(t, run.Processed)
}
