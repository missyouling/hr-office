package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"siapp/internal/models"
)

func setupRegularizationWorkerTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRegularizationImportTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.RegularizationEffectRun{}, &models.WorkTodo{}, &SystemLog{}))
	return env
}

func seedRegularizationScheduledRecord(t *testing.T, env *rbacTestEnv, emp models.Employee, plannedDate string) models.RegularizationRecord {
	t.Helper()
	rec := models.RegularizationRecord{
		UserID:                   env.admin.ID,
		EmployeeID:               &emp.ID,
		SnapshotName:             emp.Name,
		SnapshotDepartment:       emp.Department,
		SnapshotPosition:         emp.Position,
		SnapshotEmploymentStatus: emp.EmploymentStatus,
		SnapshotProbationEndDate: emp.ProbationEndDate,
		PlannedRegularDate:       plannedDate,
		Status:                   models.RegularizationStatusScheduled,
		Source:                   models.RegularizationSourceExcelDirect,
	}
	require.NoError(t, env.db.Create(&rec).Error)
	return rec
}

func TestRegularizationWorkerSuccessAndIdempotent(t *testing.T) {
	env := setupRegularizationWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)
	emp := createRegularizationTestEmployee(t, env, "110101199001011234", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	rec := seedRegularizationScheduledRecord(t, env, emp, "2026-08-18")

	handler := NewHandler(env.db)
	handler.runRegularizationWorkerOnce(loc, now)

	var saved models.RegularizationRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.RegularizationStatusEffective, saved.Status)
	require.NotNil(t, saved.EffectAttemptedAt)
	assert.Equal(t, "2026-08-18", saved.ActualRegularDate)

	var empSaved models.Employee
	require.NoError(t, env.db.First(&empSaved, emp.ID).Error)
	assert.Equal(t, models.EmploymentStatusFormal, empSaved.EmploymentStatus)
	assert.Equal(t, "2026-08-18", empSaved.ActualRegularDate)

	// 同日重复触发不应再次处理
	handler.runRegularizationWorkerOnce(loc, now)
	var runCount int64
	require.NoError(t, env.db.Model(&models.RegularizationEffectRun{}).Where("run_date = ?", "2026-08-18").Count(&runCount).Error)
	assert.Equal(t, int64(1), runCount)

	// 运行记录统计正确：成功计为 processed，不产生失败
	var run models.RegularizationEffectRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, 1, run.Processed)
	assert.Equal(t, 0, run.Failed)
	assert.Equal(t, models.RegularizationRunStatusSuccess, run.Status)

	var todoCount int64
	require.NoError(t, env.db.Model(&models.WorkTodo{}).Where("business_type = ? AND business_id = ?", "regularization_effect_exception", rec.ID).Count(&todoCount).Error)
	assert.Zero(t, todoCount)
}

func TestRegularizationWorkerFailureCreatesTodo(t *testing.T) {
	env := setupRegularizationWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)
	emp := createRegularizationTestEmployee(t, env, "110101199001011235", "研发部", models.EmployeeStatusActive, models.EmploymentStatusFormal)
	rec := seedRegularizationScheduledRecord(t, env, emp, "2026-08-18")

	handler := NewHandler(env.db)
	handler.runRegularizationWorkerOnce(loc, now)

	var saved models.RegularizationRecord
	require.NoError(t, env.db.First(&saved, rec.ID).Error)
	assert.Equal(t, models.RegularizationStatusEffectFailed, saved.Status)
	require.NotNil(t, saved.EffectAttemptedAt)
	assert.NotEmpty(t, saved.EffectFailedReason)

	var todo models.WorkTodo
	require.NoError(t, env.db.Where("business_type = ? AND business_id = ?", "regularization_effect_exception", rec.ID).First(&todo).Error)
	assert.Equal(t, env.admin.ID, todo.UserID)
	require.NotNil(t, todo.AssigneeID)
	assert.Equal(t, env.admin.ID, *todo.AssigneeID)

	// 运行记录统计正确：失败计为 failed 而非 processed，状态 failed 且错误信息非空
	var run models.RegularizationEffectRun
	require.NoError(t, env.db.Where("run_date = ?", "2026-08-18").First(&run).Error)
	assert.Equal(t, 0, run.Processed)
	assert.Equal(t, 1, run.Failed)
	assert.Equal(t, models.RegularizationRunStatusFailed, run.Status)
	assert.NotEmpty(t, run.ErrorMsg)

	// 再跑同日不应重复
	handler.runRegularizationWorkerOnce(loc, now)
	var todoCount int64
	require.NoError(t, env.db.Model(&models.WorkTodo{}).Where("business_type = ? AND business_id = ?", "regularization_effect_exception", rec.ID).Count(&todoCount).Error)
	assert.Equal(t, int64(1), todoCount)
}

func TestRegularizationWorkerOnlyShanghaiToday(t *testing.T) {
	env := setupRegularizationWorkerTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 8, 18, 2, 0, 0, 0, loc)
	todayEmp := createRegularizationTestEmployee(t, env, "110101199001011236", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	yesterdayEmp := createRegularizationTestEmployee(t, env, "110101199001011237", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	tomorrowEmp := createRegularizationTestEmployee(t, env, "110101199001011238", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	seedRegularizationScheduledRecord(t, env, todayEmp, "2026-08-18")
	seedRegularizationScheduledRecord(t, env, yesterdayEmp, "2026-08-17")
	seedRegularizationScheduledRecord(t, env, tomorrowEmp, "2026-08-19")

	handler := NewHandler(env.db)
	handler.runRegularizationWorkerOnce(loc, now)

	var records []models.RegularizationRecord
	require.NoError(t, env.db.Order("id ASC").Find(&records).Error)
	assert.Equal(t, models.RegularizationStatusEffective, records[0].Status)
	assert.Equal(t, models.RegularizationStatusScheduled, records[1].Status)
	assert.Equal(t, models.RegularizationStatusScheduled, records[2].Status)
}
