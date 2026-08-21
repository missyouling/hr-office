package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupEmployeeTestDB 打开内存 SQLite 并迁移员工模型。
func setupEmployeeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("启用 SQLite 外键失败: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Employee{}); err != nil {
		t.Fatalf("迁移员工模型失败: %v", err)
	}
	return db
}

// createHistoricalEmployee 构造迁移前的历史员工：
// employmentStatus 为空时跳过创建 hook（模拟加列后尚未回填的数据）。
func createHistoricalEmployee(t *testing.T, db *gorm.DB, userID uint, name, idNumber, status, employmentStatus string) Employee {
	t.Helper()
	emp := Employee{
		UserID:           userID,
		Name:             name,
		IDNumber:         idNumber,
		Status:           status,
		EmploymentStatus: employmentStatus,
	}
	create := db
	if employmentStatus == "" {
		create = db.Session(&gorm.Session{SkipHooks: true})
	}
	if err := create.Create(&emp).Error; err != nil {
		t.Fatalf("创建历史员工失败: %v", err)
	}
	return emp
}

func reloadEmployee(t *testing.T, db *gorm.DB, id uint) Employee {
	t.Helper()
	var emp Employee
	if err := db.First(&emp, id).Error; err != nil {
		t.Fatalf("查询员工失败: %v", err)
	}
	return emp
}

// TestEmployeeStatusConstants 员工生命周期状态常量合法性。
func TestEmployeeStatusConstants(t *testing.T) {
	if EmployeeStatusActive != "active" {
		t.Errorf("在职状态常量应为 active，实际 %q", EmployeeStatusActive)
	}
	if EmployeeStatusResigned != "resigned" {
		t.Errorf("离职状态常量应为 resigned，实际 %q", EmployeeStatusResigned)
	}
}

// TestEmploymentStatusConstantsAndValidator 就业状态常量与校验函数合法性。
func TestEmploymentStatusConstantsAndValidator(t *testing.T) {
	if EmploymentStatusTrial != "trial" {
		t.Errorf("试用状态常量应为 trial，实际 %q", EmploymentStatusTrial)
	}
	if EmploymentStatusFormal != "formal" {
		t.Errorf("正式状态常量应为 formal，实际 %q", EmploymentStatusFormal)
	}
	for _, s := range []string{EmploymentStatusTrial, EmploymentStatusFormal} {
		if !IsValidEmploymentStatus(s) {
			t.Errorf("就业状态 %q 应合法", s)
		}
	}
	for _, s := range []string{"", "intern", "probation", "TRIAL", EmployeeStatusActive, EmployeeStatusResigned} {
		if IsValidEmploymentStatus(s) {
			t.Errorf("就业状态 %q 应非法", s)
		}
	}
}

// TestMigrateEmployeeEmploymentStatusBackfill 历史 active 员工一次性回填为 formal。
func TestMigrateEmployeeEmploymentStatusBackfill(t *testing.T) {
	db := setupEmployeeTestDB(t)
	active := createHistoricalEmployee(t, db, 1, "历史在职", "110101199001011234", EmployeeStatusActive, "")
	if err := MigrateEmployeeEmploymentStatus(db); err != nil {
		t.Fatalf("迁移回填失败: %v", err)
	}
	if got := reloadEmployee(t, db, active.ID).EmploymentStatus; got != EmploymentStatusFormal {
		t.Errorf("历史 active 员工应回填为 formal，实际 %q", got)
	}
}

// TestMigrateEmployeeEmploymentStatusIdempotent 迁移函数幂等：重复执行不报错且结果一致。
func TestMigrateEmployeeEmploymentStatusIdempotent(t *testing.T) {
	db := setupEmployeeTestDB(t)
	active := createHistoricalEmployee(t, db, 1, "幂等在职", "110101199001011234", EmployeeStatusActive, "")
	if err := MigrateEmployeeEmploymentStatus(db); err != nil {
		t.Fatalf("首次回填失败: %v", err)
	}
	if err := MigrateEmployeeEmploymentStatus(db); err != nil {
		t.Fatalf("重复回填失败: %v", err)
	}
	if got := reloadEmployee(t, db, active.ID).EmploymentStatus; got != EmploymentStatusFormal {
		t.Errorf("重复执行后应为 formal，实际 %q", got)
	}
}

// TestMigrateEmployeeEmploymentStatusKeepsExistingValues 迁移不覆盖已有 trial/formal。
func TestMigrateEmployeeEmploymentStatusKeepsExistingValues(t *testing.T) {
	db := setupEmployeeTestDB(t)
	trial := createHistoricalEmployee(t, db, 1, "试用在职", "110101199001011234", EmployeeStatusActive, EmploymentStatusTrial)
	formal := createHistoricalEmployee(t, db, 1, "正式在职", "110101199002022345", EmployeeStatusActive, EmploymentStatusFormal)
	if err := MigrateEmployeeEmploymentStatus(db); err != nil {
		t.Fatalf("迁移回填失败: %v", err)
	}
	if got := reloadEmployee(t, db, trial.ID).EmploymentStatus; got != EmploymentStatusTrial {
		t.Errorf("已有 trial 不应被覆盖，实际 %q", got)
	}
	if got := reloadEmployee(t, db, formal.ID).EmploymentStatus; got != EmploymentStatusFormal {
		t.Errorf("已有 formal 不应被覆盖，实际 %q", got)
	}
}

// TestMigrateEmployeeEmploymentStatusKeepsResignedEmpty 历史 resigned 员工保持空值。
func TestMigrateEmployeeEmploymentStatusKeepsResignedEmpty(t *testing.T) {
	db := setupEmployeeTestDB(t)
	resigned := createHistoricalEmployee(t, db, 1, "历史离职", "110101199001011234", EmployeeStatusResigned, "")
	if err := MigrateEmployeeEmploymentStatus(db); err != nil {
		t.Fatalf("迁移回填失败: %v", err)
	}
	if got := reloadEmployee(t, db, resigned.ID).EmploymentStatus; got != "" {
		t.Errorf("resigned 员工应保持空值，实际 %q", got)
	}
}

// TestMigrateEmployeeEmploymentStatusNilDB nil 数据库边界：返回 nil。
func TestMigrateEmployeeEmploymentStatusNilDB(t *testing.T) {
	if err := MigrateEmployeeEmploymentStatus(nil); err != nil {
		t.Errorf("nil db 应返回 nil，实际 %v", err)
	}
}

// TestEmployeeEmploymentFieldsPersist 新增就业字段持久化（写入后读回一致）。
func TestEmployeeEmploymentFieldsPersist(t *testing.T) {
	db := setupEmployeeTestDB(t)
	emp := Employee{
		UserID:            1,
		Name:              "字段员工",
		IDNumber:          "110101199001011234",
		Status:            EmployeeStatusActive,
		EmploymentStatus:  EmploymentStatusTrial,
		ProbationEndDate:  "2026-12-31",
		ActualRegularDate: "2027-01-01",
	}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}
	got := reloadEmployee(t, db, emp.ID)
	if got.EmploymentStatus != EmploymentStatusTrial {
		t.Errorf("就业状态未持久化，实际 %q", got.EmploymentStatus)
	}
	if got.ProbationEndDate != "2026-12-31" {
		t.Errorf("试用期结束日期未持久化，实际 %q", got.ProbationEndDate)
	}
	if got.ActualRegularDate != "2027-01-01" {
		t.Errorf("实际转正日期未持久化，实际 %q", got.ActualRegularDate)
	}
}

// TestEmployeeEmploymentStatusDefaultFormal 旧创建路径的安全默认：
// 未指定就业状态的在职员工默认 formal（不误试用化），离职员工保持空，显式 trial 不被覆盖。
func TestEmployeeEmploymentStatusDefaultFormal(t *testing.T) {
	db := setupEmployeeTestDB(t)
	// 旧创建路径：未指定就业状态 → 默认 formal（安全，不误试用化）
	active := Employee{UserID: 1, Name: "默认在职", IDNumber: "110101199001011234", Status: EmployeeStatusActive}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("创建默认在职员工失败: %v", err)
	}
	if active.EmploymentStatus != EmploymentStatusFormal {
		t.Errorf("未指定就业状态的在职员工应默认 formal，实际 %q", active.EmploymentStatus)
	}
	// 离职员工不自动填 formal（保持空值语义）
	resigned := Employee{UserID: 1, Name: "默认离职", IDNumber: "110101199002022345", Status: EmployeeStatusResigned}
	if err := db.Create(&resigned).Error; err != nil {
		t.Fatalf("创建默认离职员工失败: %v", err)
	}
	if resigned.EmploymentStatus != "" {
		t.Errorf("未指定就业状态的离职员工应保持空值，实际 %q", resigned.EmploymentStatus)
	}
	// 显式指定 trial 不被覆盖
	explicit := Employee{UserID: 1, Name: "显式试用", IDNumber: "110101199003033456", Status: EmployeeStatusActive, EmploymentStatus: EmploymentStatusTrial}
	if err := db.Create(&explicit).Error; err != nil {
		t.Fatalf("创建显式试用员工失败: %v", err)
	}
	if explicit.EmploymentStatus != EmploymentStatusTrial {
		t.Errorf("显式指定的 trial 不应被覆盖，实际 %q", explicit.EmploymentStatus)
	}
}
