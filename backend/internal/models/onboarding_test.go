package models

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupOnboardingTestDB 打开内存 SQLite 并迁移入职/待办相关模型。
func setupOnboardingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Employee{}, &OnboardingRecord{}, &WorkTodo{}); err != nil {
		t.Fatalf("迁移入职/待办模型失败: %v", err)
	}
	return db
}

// createOnboardingTestUser 创建测试归属用户（租户隔离载体）。
func createOnboardingTestUser(t *testing.T, db *gorm.DB, suffix string) User {
	t.Helper()
	user := User{Username: "user-" + suffix, Email: suffix + "@example.test", Password: "test-password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func TestOnboardingStatusValidators(t *testing.T) {
	for _, s := range []string{OnboardingStatusPending, OnboardingStatusOnboarded, OnboardingStatusAbandoned} {
		if !IsValidOnboardingStatus(s) {
			t.Errorf("入职状态 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "active", "resigned", "ONBOARDED", "unknown"} {
		if IsValidOnboardingStatus(s) {
			t.Errorf("入职状态 %q 应非法", s)
		}
	}
}

func TestEmploymentStatusValidators(t *testing.T) {
	for _, s := range []string{EmploymentStatusTrial, EmploymentStatusFormal} {
		if !IsValidEmploymentStatus(s) {
			t.Errorf("用工状态 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "intern", "probation", "TRIAL"} {
		if IsValidEmploymentStatus(s) {
			t.Errorf("用工状态 %q 应非法", s)
		}
	}
}

func TestWorkTodoStatusValidators(t *testing.T) {
	for _, s := range []string{WorkTodoStatusPending, WorkTodoStatusCompleted} {
		if !IsValidWorkTodoStatus(s) {
			t.Errorf("待办状态 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "done", "cancelled", "PENDING"} {
		if IsValidWorkTodoStatus(s) {
			t.Errorf("待办状态 %q 应非法", s)
		}
	}
}

func TestOnboardingRecordValidate(t *testing.T) {
	// 合法：待入职
	valid := OnboardingRecord{Name: "张三", PlannedHireDate: "2026-09-01", Status: OnboardingStatusPending}
	if err := valid.Validate(); err != nil {
		t.Fatalf("合法待入职记录校验应通过: %v", err)
	}
	// 合法：已入职 + 用工状态
	onboarded := OnboardingRecord{Name: "李四", PlannedHireDate: "2026-09-01", Status: OnboardingStatusOnboarded, EmploymentStatus: EmploymentStatusTrial}
	if err := onboarded.Validate(); err != nil {
		t.Fatalf("已入职记录设置用工状态应通过: %v", err)
	}
	// 非法生命周期状态
	badStatus := OnboardingRecord{Name: "王五", PlannedHireDate: "2026-09-01", Status: "unknown"}
	if err := badStatus.Validate(); err == nil {
		t.Error("非法生命周期状态应被拒绝")
	}
	// 计划入职日期必填
	noDate := OnboardingRecord{Name: "赵六", Status: OnboardingStatusPending}
	if err := noDate.Validate(); err == nil {
		t.Error("空计划入职日期应被拒绝")
	}
	// 非法用工状态
	badEmployment := OnboardingRecord{Name: "孙七", PlannedHireDate: "2026-09-01", Status: OnboardingStatusOnboarded, EmploymentStatus: "intern"}
	if err := badEmployment.Validate(); err == nil {
		t.Error("非法用工状态应被拒绝")
	}
	// 用工状态仅已入职记录可设置
	notOnboarded := OnboardingRecord{Name: "周八", PlannedHireDate: "2026-09-01", Status: OnboardingStatusPending, EmploymentStatus: EmploymentStatusFormal}
	if err := notOnboarded.Validate(); err == nil {
		t.Error("未入职记录设置用工状态应被拒绝")
	}
}

func TestWorkTodoValidate(t *testing.T) {
	valid := WorkTodo{BusinessType: "onboarding", BusinessID: 1, Title: "办理入职", Status: WorkTodoStatusPending}
	if err := valid.Validate(); err != nil {
		t.Fatalf("合法待办校验应通过: %v", err)
	}
	noType := WorkTodo{BusinessID: 1, Title: "办理入职", Status: WorkTodoStatusPending}
	if err := noType.Validate(); err == nil {
		t.Error("空业务类型应被拒绝")
	}
	noID := WorkTodo{BusinessType: "onboarding", Title: "办理入职", Status: WorkTodoStatusPending}
	if err := noID.Validate(); err == nil {
		t.Error("零业务ID应被拒绝")
	}
	noTitle := WorkTodo{BusinessType: "onboarding", BusinessID: 1, Status: WorkTodoStatusPending}
	if err := noTitle.Validate(); err == nil {
		t.Error("空待办标题应被拒绝")
	}
	badStatus := WorkTodo{BusinessType: "onboarding", BusinessID: 1, Title: "办理入职", Status: "done"}
	if err := badStatus.Validate(); err == nil {
		t.Error("非法待办状态应被拒绝")
	}
}

func TestOnboardingRecordDefaultsAndAbandonFieldsPersist(t *testing.T) {
	db := setupOnboardingTestDB(t)
	user := createOnboardingTestUser(t, db, "onboard-defaults")
	rec := OnboardingRecord{UserID: user.ID, Name: "张三", PlannedHireDate: "2026-09-01"}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("创建入职记录失败: %v", err)
	}
	if rec.Status != OnboardingStatusPending {
		t.Errorf("默认状态应为 pending，实际 %q", rec.Status)
	}
	// 写入放弃原因/时间/备注
	now := time.Now()
	if err := db.Model(&rec).Updates(map[string]interface{}{
		"status":         OnboardingStatusAbandoned,
		"abandon_reason": "候选人放弃",
		"abandoned_at":   now,
		"remarks":        "已电话确认",
	}).Error; err != nil {
		t.Fatalf("更新放弃字段失败: %v", err)
	}
	var saved OnboardingRecord
	if err := db.First(&saved, rec.ID).Error; err != nil {
		t.Fatalf("查询入职记录失败: %v", err)
	}
	if saved.Status != OnboardingStatusAbandoned || saved.AbandonReason != "候选人放弃" || saved.Remarks != "已电话确认" || saved.AbandonedAt == nil {
		t.Error("放弃原因/时间/备注未保留")
	}
	// 状态回退后放弃字段仍保留（永久保留语义）
	if err := db.Model(&saved).Update("status", OnboardingStatusPending).Error; err != nil {
		t.Fatalf("回退状态失败: %v", err)
	}
	var again OnboardingRecord
	if err := db.First(&again, rec.ID).Error; err != nil {
		t.Fatalf("再次查询入职记录失败: %v", err)
	}
	if again.AbandonReason != "候选人放弃" || again.AbandonedAt == nil || again.Remarks != "已电话确认" {
		t.Error("状态回退后放弃原因/时间/备注应永久保留")
	}
}

func TestWorkTodoBusinessUnique(t *testing.T) {
	db := setupOnboardingTestDB(t)
	userA := createOnboardingTestUser(t, db, "todo-a")
	userB := createOnboardingTestUser(t, db, "todo-b")
	first := WorkTodo{UserID: userA.ID, BusinessType: "onboarding", BusinessID: 1, Title: "办理入职", Status: WorkTodoStatusPending}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("创建首个待办失败: %v", err)
	}
	// 同租户同业务类型+业务ID 重复 → 唯一约束拒绝
	dup := WorkTodo{UserID: userA.ID, BusinessType: "onboarding", BusinessID: 1, Title: "重复待办", Status: WorkTodoStatusPending}
	if err := db.Create(&dup).Error; err == nil {
		t.Error("同租户重复业务待办应被唯一约束拒绝")
	}
	// 不同租户同业务类型+业务ID → 允许
	otherTenant := WorkTodo{UserID: userB.ID, BusinessType: "onboarding", BusinessID: 1, Title: "其他租户待办", Status: WorkTodoStatusPending}
	if err := db.Create(&otherTenant).Error; err != nil {
		t.Fatalf("不同租户同业务待办应允许: %v", err)
	}
	// 同租户不同业务类型 → 允许
	diffType := WorkTodo{UserID: userA.ID, BusinessType: "resignation", BusinessID: 1, Title: "办理离职", Status: WorkTodoStatusPending}
	if err := db.Create(&diffType).Error; err != nil {
		t.Fatalf("同租户不同业务类型待办应允许: %v", err)
	}
	// 同租户同业务类型不同业务ID → 允许
	diffID := WorkTodo{UserID: userA.ID, BusinessType: "onboarding", BusinessID: 2, Title: "另一条入职", Status: WorkTodoStatusPending}
	if err := db.Create(&diffID).Error; err != nil {
		t.Fatalf("同租户不同业务ID待办应允许: %v", err)
	}
}

func TestOnboardingSchemaMigratesOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开迁移测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Employee{}, &OnboardingRecord{}, &WorkTodo{}); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}
	if !db.Migrator().HasTable(&OnboardingRecord{}) {
		t.Error("onboarding_records 表未创建")
	}
	if !db.Migrator().HasTable(&WorkTodo{}) {
		t.Error("work_todos 表未创建")
	}
	if !db.Migrator().HasIndex(&WorkTodo{}, "idx_work_todos_business") {
		t.Error("work_todos 缺少业务唯一索引")
	}
	// 幂等：重复迁移不报错
	if err := db.AutoMigrate(&User{}, &Employee{}, &OnboardingRecord{}, &WorkTodo{}); err != nil {
		t.Fatalf("重复迁移失败: %v", err)
	}
}
