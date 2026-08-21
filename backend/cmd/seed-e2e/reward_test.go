package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupRewardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Employee{}); err != nil {
		t.Fatalf("迁移 Employee 表失败: %v", err)
	}
	return db
}

// assertE2ERewardEmployeeActive 校验奖惩 E2E 员工处于目标状态（active + 固定信息）
func assertE2ERewardEmployeeActive(t *testing.T, emp models.Employee) {
	t.Helper()
	if emp.Name != e2eRewardEmployeeName {
		t.Errorf("员工姓名应为 %q，实际为 %q", e2eRewardEmployeeName, emp.Name)
	}
	if emp.IDNumber != e2eRewardEmployeeIDNumber {
		t.Errorf("员工身份证号应为 %q，实际为 %q", e2eRewardEmployeeIDNumber, emp.IDNumber)
	}
	if emp.EmployeeID != e2eRewardEmployeeID {
		t.Errorf("员工工号应为 %q，实际为 %q", e2eRewardEmployeeID, emp.EmployeeID)
	}
	if emp.Department != e2eEmployeeDepartment {
		t.Errorf("员工部门应为 %q，实际为 %q", e2eEmployeeDepartment, emp.Department)
	}
	if emp.Status != models.EmployeeStatusActive {
		t.Errorf("员工状态应为 active，实际为 %q", emp.Status)
	}
}

// TestEnsureE2ERewardEmployee_IdempotentAndRestoresActive seed 幂等且能恢复目标状态
func TestEnsureE2ERewardEmployee_IdempotentAndRestoresActive(t *testing.T) {
	db := setupRewardDB(t)
	const userID = uint(1)

	// 首次：创建 active 员工
	if err := ensureE2ERewardEmployee(db, userID); err != nil {
		t.Fatalf("首次创建奖惩 E2E 员工失败: %v", err)
	}
	var emp models.Employee
	if err := db.Where("user_id = ? AND id_number = ?", userID, e2eRewardEmployeeIDNumber).First(&emp).Error; err != nil {
		t.Fatalf("查询奖惩 E2E 员工失败: %v", err)
	}
	assertE2ERewardEmployeeActive(t, emp)

	// 再次执行：幂等，不重复创建
	if err := ensureE2ERewardEmployee(db, userID); err != nil {
		t.Fatalf("重复执行奖惩 E2E 员工 seed 失败: %v", err)
	}
	var count int64
	db.Model(&models.Employee{}).Where("user_id = ? AND id_number = ?", userID, e2eRewardEmployeeIDNumber).Count(&count)
	if count != 1 {
		t.Fatalf("重复执行后奖惩 E2E 员工数应为 1，实际为 %d", count)
	}

	// 模拟旧残留：改为 resigned
	if err := db.Model(&emp).Update("status", "resigned").Error; err != nil {
		t.Fatalf("模拟离职残留失败: %v", err)
	}

	// 再次执行：恢复为 active
	if err := ensureE2ERewardEmployee(db, userID); err != nil {
		t.Fatalf("恢复奖惩 E2E 员工失败: %v", err)
	}
	if err := db.First(&emp, emp.ID).Error; err != nil {
		t.Fatalf("重新查询奖惩 E2E 员工失败: %v", err)
	}
	assertE2ERewardEmployeeActive(t, emp)
}

// TestEnsureE2ERewardEmployee_DoesNotTouchOtherEmployees 不污染非 E2E 员工
func TestEnsureE2ERewardEmployee_DoesNotTouchOtherEmployees(t *testing.T) {
	db := setupRewardDB(t)
	const userID = uint(1)

	// 预置一名非 E2E 员工（同用户下、不同身份证号）
	other := models.Employee{
		UserID:     userID,
		Name:       "普通员工",
		IDNumber:   "110101199501011234",
		Department: "财务部",
		Position:   "会计",
		Status:     "resigned",
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("创建非 E2E 员工失败: %v", err)
	}

	if err := ensureE2ERewardEmployee(db, userID); err != nil {
		t.Fatalf("执行奖惩 E2E 员工 seed 失败: %v", err)
	}

	// 非 E2E 员工保持原样（未被恢复、未被删除）
	var reloaded models.Employee
	if err := db.First(&reloaded, other.ID).Error; err != nil {
		t.Fatalf("查询非 E2E 员工失败: %v", err)
	}
	if reloaded.Status != "resigned" {
		t.Errorf("非 E2E 员工不应被改动，实际: status=%q", reloaded.Status)
	}

	// E2E 员工正常创建
	var e2e models.Employee
	if err := db.Where("user_id = ? AND id_number = ?", userID, e2eRewardEmployeeIDNumber).First(&e2e).Error; err != nil {
		t.Fatalf("查询奖惩 E2E 员工失败: %v", err)
	}
	assertE2ERewardEmployeeActive(t, e2e)
}
