package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupRegularizationTestDB 打开内存 SQLite 并迁移转正记录模型。
func setupRegularizationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &RegularizationRecord{}, &RegularizationEffectRun{}); err != nil {
		t.Fatalf("迁移转正模型失败: %v", err)
	}
	return db
}

// TestRegularizationEffectRun 验证任务运行记录的状态、日期和单日唯一约束。
func TestRegularizationEffectRun(t *testing.T) {
	db := setupRegularizationTestDB(t)
	run := RegularizationEffectRun{
		RunDate:   "2026-08-18",
		Status:    RegularizationRunStatusSuccess,
		Processed: 2,
		Failed:    1,
	}
	require.NoError(t, run.Validate())
	require.NoError(t, db.Create(&run).Error)

	duplicate := RegularizationEffectRun{RunDate: run.RunDate, Status: RegularizationRunStatusSuccess}
	assert.Error(t, db.Create(&duplicate).Error, "同一业务日期只能有一条运行记录")

	for _, invalid := range []RegularizationEffectRun{
		{RunDate: "2026/08/18", Status: RegularizationRunStatusSuccess},
		{RunDate: "2026-08-18", Status: "running"},
		{RunDate: "2026-08-18", Status: RegularizationRunStatusFailed, Failed: -1},
	} {
		assert.Error(t, invalid.Validate())
	}
}

// uintPtr 返回 uint 值的指针（测试构造指针字段用）。
func uintPtr(v uint) *uint {
	return &v
}

// createRegularizationTestUser 创建测试归属用户（租户隔离载体）。
func createRegularizationTestUser(t *testing.T, db *gorm.DB, suffix string) User {
	t.Helper()
	user := User{Username: "reg-" + suffix, Email: "reg-" + suffix + "@example.test", Password: "test-password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

// TestRegularizationStatusValidators 状态常量全量合法、非法值拒绝。
func TestRegularizationStatusValidators(t *testing.T) {
	for _, s := range []string{
		RegularizationStatusPendingSupervisor,
		RegularizationStatusPendingHRReview,
		RegularizationStatusScheduled,
		RegularizationStatusEffective,
		RegularizationStatusRejected,
		RegularizationStatusPostponedScheduled,
		RegularizationStatusEffectFailed,
		RegularizationStatusCancelledByResignation,
		RegularizationStatusVoided,
	} {
		if !IsValidRegularizationStatus(s) {
			t.Errorf("转正状态 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "pending", "approved", "cancelled", "PENDING_SUPERVISOR", "unknown"} {
		if IsValidRegularizationStatus(s) {
			t.Errorf("转正状态 %q 应非法", s)
		}
	}
}

// TestRegularizationSourceValidators 来源常量全量合法、非法值拒绝。
func TestRegularizationSourceValidators(t *testing.T) {
	for _, s := range []string{RegularizationSourceManual, RegularizationSourceExcelDirect} {
		if !IsValidRegularizationSource(s) {
			t.Errorf("转正来源 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "auto", "import", "MANUAL", "unknown"} {
		if IsValidRegularizationSource(s) {
			t.Errorf("转正来源 %q 应非法", s)
		}
	}
}

// TestRegularizationValidate 状态/来源校验与三人审批必须不同。
func TestRegularizationValidate(t *testing.T) {
	// 合法：默认状态与来源
	valid := RegularizationRecord{
		Status: RegularizationStatusPendingSupervisor,
		Source: RegularizationSourceManual,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("合法转正记录校验应通过: %v", err)
	}
	// 三人审批不同（全部设置）→ 通过
	threeDistinct := RegularizationRecord{
		Status:                   RegularizationStatusPendingSupervisor,
		Source:                   RegularizationSourceManual,
		InitiatorHRUserID:        uintPtr(1),
		SupervisorApproverUserID: uintPtr(2),
		HRReviewerUserID:         uintPtr(3),
	}
	if err := threeDistinct.Validate(); err != nil {
		t.Fatalf("三人审批不同校验应通过: %v", err)
	}
	// 部分设置且不同 → 通过
	partial := RegularizationRecord{
		Status:                   RegularizationStatusPendingSupervisor,
		Source:                   RegularizationSourceManual,
		SupervisorApproverUserID: uintPtr(2),
		HRReviewerUserID:         uintPtr(3),
	}
	if err := partial.Validate(); err != nil {
		t.Fatalf("部分审批人设置且不同校验应通过: %v", err)
	}
	// 非法状态
	badStatus := RegularizationRecord{Status: "approved", Source: RegularizationSourceManual}
	if err := badStatus.Validate(); err == nil {
		t.Error("非法转正状态应被拒绝")
	}
	// 非法来源
	badSource := RegularizationRecord{Status: RegularizationStatusPendingSupervisor, Source: "import"}
	if err := badSource.Validate(); err == nil {
		t.Error("非法转正来源应被拒绝")
	}
	// 发起人与主管相同 → 拒绝
	initiatorSameAsSupervisor := RegularizationRecord{
		Status:                   RegularizationStatusPendingSupervisor,
		Source:                   RegularizationSourceManual,
		InitiatorHRUserID:        uintPtr(7),
		SupervisorApproverUserID: uintPtr(7),
	}
	if err := initiatorSameAsSupervisor.Validate(); err == nil {
		t.Error("发起人与主管相同应被拒绝")
	}
	// 主管与 HR 复核相同 → 拒绝
	supervisorSameAsHR := RegularizationRecord{
		Status:                   RegularizationStatusPendingSupervisor,
		Source:                   RegularizationSourceManual,
		SupervisorApproverUserID: uintPtr(9),
		HRReviewerUserID:         uintPtr(9),
	}
	if err := supervisorSameAsHR.Validate(); err == nil {
		t.Error("主管与 HR 复核相同应被拒绝")
	}
}

// TestRegularizationDefaults 默认值：状态 pending_supervisor、来源 manual、延期次数 0。
func TestRegularizationDefaults(t *testing.T) {
	db := setupRegularizationTestDB(t)
	user := createRegularizationTestUser(t, db, "defaults")
	rec := RegularizationRecord{UserID: user.ID}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("创建转正记录失败: %v", err)
	}
	if rec.Status != RegularizationStatusPendingSupervisor {
		t.Errorf("默认状态应为 pending_supervisor，实际 %q", rec.Status)
	}
	if rec.Source != RegularizationSourceManual {
		t.Errorf("默认来源应为 manual，实际 %q", rec.Source)
	}
	if rec.ExtensionCount != 0 {
		t.Errorf("默认延期次数应为 0，实际 %d", rec.ExtensionCount)
	}
}

// TestRegularizationApprovalNoUniqueWithinTenant 审批编号租户内唯一约束：
// 同租户重复拒绝；不同租户同编号允许；同租户不同编号允许；空编号允许多条。
func TestRegularizationApprovalNoUniqueWithinTenant(t *testing.T) {
	db := setupRegularizationTestDB(t)
	userA := createRegularizationTestUser(t, db, "unique-a")
	userB := createRegularizationTestUser(t, db, "unique-b")

	first := RegularizationRecord{UserID: userA.ID, ApprovalNo: "REG-2026-001", Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("创建首个转正记录失败: %v", err)
	}
	// 同租户同审批编号 → 唯一约束拒绝
	dup := RegularizationRecord{UserID: userA.ID, ApprovalNo: "REG-2026-001", Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	if err := db.Create(&dup).Error; err == nil {
		t.Error("同租户重复审批编号应被唯一约束拒绝")
	}
	// 不同租户同审批编号 → 允许
	otherTenant := RegularizationRecord{UserID: userB.ID, ApprovalNo: "REG-2026-001", Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	if err := db.Create(&otherTenant).Error; err != nil {
		t.Fatalf("不同租户同审批编号应允许: %v", err)
	}
	// 同租户不同审批编号 → 允许
	diffNo := RegularizationRecord{UserID: userA.ID, ApprovalNo: "REG-2026-002", Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	if err := db.Create(&diffNo).Error; err != nil {
		t.Fatalf("同租户不同审批编号应允许: %v", err)
	}
	// 空审批编号 → 允许多条（部分唯一索引不约束空值）
	empty1 := RegularizationRecord{UserID: userA.ID, Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	empty2 := RegularizationRecord{UserID: userA.ID, Status: RegularizationStatusPendingSupervisor, Source: RegularizationSourceManual}
	if err := db.Create(&empty1).Error; err != nil {
		t.Fatalf("空审批编号首条应允许: %v", err)
	}
	if err := db.Create(&empty2).Error; err != nil {
		t.Fatalf("空审批编号第二条应允许: %v", err)
	}
}

// TestRegularizationSnapshotFieldsPersist 快照字段写入后读回一致（持久化）。
func TestRegularizationSnapshotFieldsPersist(t *testing.T) {
	db := setupRegularizationTestDB(t)
	user := createRegularizationTestUser(t, db, "snapshot")
	rec := RegularizationRecord{
		UserID:                   user.ID,
		EmployeeID:               uintPtr(42),
		SnapshotName:             "张三",
		SnapshotDepartment:       "研发部",
		SnapshotPosition:         "后端工程师",
		SnapshotEmploymentStatus: EmploymentStatusTrial,
		SnapshotProbationEndDate: "2026-12-31",
		ContractTermMonths:       36,
		EmployeeSelfReview:       "试用期表现良好",
		ApprovalNo:               "REG-2026-100",
		PlannedRegularDate:       "2027-01-01",
		Status:                   RegularizationStatusPendingSupervisor,
		Source:                   RegularizationSourceManual,
		InitiatorHRUserID:        uintPtr(1),
		SupervisorApproverUserID: uintPtr(2),
		HRReviewerUserID:         uintPtr(3),
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("创建转正记录失败: %v", err)
	}
	var saved RegularizationRecord
	if err := db.First(&saved, rec.ID).Error; err != nil {
		t.Fatalf("查询转正记录失败: %v", err)
	}
	assert.Equal(t, "张三", saved.SnapshotName, "快照姓名未持久化")
	assert.Equal(t, "研发部", saved.SnapshotDepartment, "快照部门未持久化")
	assert.Equal(t, "后端工程师", saved.SnapshotPosition, "快照岗位未持久化")
	assert.Equal(t, EmploymentStatusTrial, saved.SnapshotEmploymentStatus, "快照用工状态未持久化")
	assert.Equal(t, "2026-12-31", saved.SnapshotProbationEndDate, "快照试用期结束日期未持久化")
	assert.Equal(t, 36, saved.ContractTermMonths, "合同期限月数未持久化")
	assert.Equal(t, "试用期表现良好", saved.EmployeeSelfReview, "员工自评未持久化")
	assert.Equal(t, "REG-2026-100", saved.ApprovalNo, "审批编号未持久化")
	assert.Equal(t, "2027-01-01", saved.PlannedRegularDate, "计划转正日期未持久化")
	require.NotNil(t, saved.EmployeeID, "关联员工ID未持久化")
	assert.Equal(t, uint(42), *saved.EmployeeID)
	require.NotNil(t, saved.InitiatorHRUserID, "发起人未持久化")
	assert.Equal(t, uint(1), *saved.InitiatorHRUserID)
	require.NotNil(t, saved.SupervisorApproverUserID, "主管审批人未持久化")
	assert.Equal(t, uint(2), *saved.SupervisorApproverUserID)
	require.NotNil(t, saved.HRReviewerUserID, "HR复核人未持久化")
	assert.Equal(t, uint(3), *saved.HRReviewerUserID)
}

// TestRegularizationApprovalNoIndexExists 审批编号联合唯一索引存在（模型标签层面约束）。
func TestRegularizationApprovalNoIndexExists(t *testing.T) {
	db := setupRegularizationTestDB(t)
	if !db.Migrator().HasIndex(&RegularizationRecord{}, "idx_reg_approval_no_user") {
		t.Error("regularization_records 缺少审批编号联合唯一索引 idx_reg_approval_no_user")
	}
}

// TestRegularizationWriteFieldsPersist P12.3.3-3 写接口新增字段持久化：
// 上级拒绝时间/延期原因/作废原因/作废时间。
func TestRegularizationWriteFieldsPersist(t *testing.T) {
	db := setupRegularizationTestDB(t)
	user := createRegularizationTestUser(t, db, "write-fields")
	now := time.Now()
	rec := RegularizationRecord{
		UserID:                   user.ID,
		ApprovalNo:               "REG-2026-200",
		Status:                   RegularizationStatusVoided,
		Source:                   RegularizationSourceManual,
		SupervisorRejectedAt:     &now,
		PostponedReason:          "需延长试用期",
		VoidReason:               "信息填写错误",
		VoidedAt:                 &now,
		InitiatorHRUserID:        uintPtr(1),
		SupervisorApproverUserID: uintPtr(2),
		HRReviewerUserID:         uintPtr(3),
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("创建转正记录失败: %v", err)
	}
	var saved RegularizationRecord
	if err := db.First(&saved, rec.ID).Error; err != nil {
		t.Fatalf("查询转正记录失败: %v", err)
	}
	require.NotNil(t, saved.SupervisorRejectedAt, "上级拒绝时间未持久化")
	assert.Equal(t, "需延长试用期", saved.PostponedReason, "延期原因未持久化")
	assert.Equal(t, "信息填写错误", saved.VoidReason, "作废原因未持久化")
	require.NotNil(t, saved.VoidedAt, "作废时间未持久化")
}
