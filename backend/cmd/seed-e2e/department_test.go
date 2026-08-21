package main

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupDepartmentDB 建立内存 SQLite 并迁移 Department 表
func setupDepartmentDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Department{}); err != nil {
		t.Fatalf("迁移 Department 表失败: %v", err)
	}
	return db
}

// findE2EDepartment 按 Name 查询 E2E 测试部门
func findE2EDepartment(t *testing.T, db *gorm.DB) models.Department {
	t.Helper()
	var dept models.Department
	if err := db.Where("name = ?", e2eDepartmentName).First(&dept).Error; err != nil {
		t.Fatalf("查询 E2E 测试部门失败: %v", err)
	}
	return dept
}

// assertE2EDepartment 校验部门处于目标状态（名称 + 编码 + 归属 admin）
func assertE2EDepartment(t *testing.T, dept models.Department, userID uint) {
	t.Helper()
	if dept.Name != e2eDepartmentName {
		t.Errorf("部门名称应为 %q，实际为 %q", e2eDepartmentName, dept.Name)
	}
	if dept.Code != e2eDepartmentCode {
		t.Errorf("部门编码应为 %q，实际为 %q", e2eDepartmentCode, dept.Code)
	}
	if dept.UserID == nil || *dept.UserID != userID {
		t.Errorf("部门归属用户应为 %d，实际为 %v", userID, dept.UserID)
	}
}

// TestEnsureE2EDepartment_CreatesForAdmin 首次运行应创建归属 admin 的测试部门
func TestEnsureE2EDepartment_CreatesForAdmin(t *testing.T) {
	db := setupDepartmentDB(t)
	const userID = uint(1)

	if err := ensureE2EDepartment(db, userID); err != nil {
		t.Fatalf("首次创建 E2E 测试部门失败: %v", err)
	}
	assertE2EDepartment(t, findE2EDepartment(t, db), userID)
}

// TestEnsureE2EDepartment_Idempotent 重复运行不重复创建、复用同一部门
func TestEnsureE2EDepartment_Idempotent(t *testing.T) {
	db := setupDepartmentDB(t)
	const userID = uint(1)

	if err := ensureE2EDepartment(db, userID); err != nil {
		t.Fatalf("首次创建 E2E 测试部门失败: %v", err)
	}
	first := findE2EDepartment(t, db)

	if err := ensureE2EDepartment(db, userID); err != nil {
		t.Fatalf("重复执行 E2E 测试部门 seed 失败: %v", err)
	}
	var count int64
	db.Model(&models.Department{}).Where("name = ?", e2eDepartmentName).Count(&count)
	if count != 1 {
		t.Fatalf("重复执行后部门数应为 1，实际为 %d", count)
	}
	second := findE2EDepartment(t, db)
	if second.ID != first.ID {
		t.Errorf("重复执行应复用同一部门 ID，首次=%d 再次=%d", first.ID, second.ID)
	}
	assertE2EDepartment(t, second, userID)
}

// TestEnsureE2EDepartment_CorrectsCode 归属 admin 的同名部门编码缺失/错误时应纠正为目标编码且不新建
func TestEnsureE2EDepartment_CorrectsCode(t *testing.T) {
	db := setupDepartmentDB(t)
	userID := uint(1)

	// 预置归属 admin 的同名部门，编码为空（模拟历史残留）
	pre := models.Department{UserID: &userID, Name: e2eDepartmentName, Code: ""}
	if err := db.Create(&pre).Error; err != nil {
		t.Fatalf("预置同名部门失败: %v", err)
	}

	if err := ensureE2EDepartment(db, userID); err != nil {
		t.Fatalf("纠正 E2E 测试部门编码失败: %v", err)
	}
	dept := findE2EDepartment(t, db)
	assertE2EDepartment(t, dept, userID)
	if dept.ID != pre.ID {
		t.Errorf("编码纠正不应新建部门，原 ID=%d 实际=%d", pre.ID, dept.ID)
	}
}

// TestEnsureE2EDepartment_ConflictWithNonAdmin 同名部门归属非 admin（含全局）时应返回明确错误且不改动记录
func TestEnsureE2EDepartment_ConflictWithNonAdmin(t *testing.T) {
	db := setupDepartmentDB(t)
	const adminID = uint(1)

	// 预置全局（user_id NULL）同名部门，编码为 E2E 前缀但归属不符
	pre := models.Department{UserID: nil, Name: e2eDepartmentName, Code: "E2E-LEGACY"}
	if err := db.Create(&pre).Error; err != nil {
		t.Fatalf("预置同名部门失败: %v", err)
	}

	err := ensureE2EDepartment(db, adminID)
	if err == nil {
		t.Fatal("存在非 admin 归属的同名部门时应返回明确错误，实际返回 nil")
	}
	// 错误信息应明确指出归属冲突，便于人工处理
	if !strings.Contains(err.Error(), "不归属") {
		t.Errorf("错误信息应明确指出归属冲突，实际为: %v", err)
	}

	// 记录未被修改
	var reloaded models.Department
	if err := db.First(&reloaded, pre.ID).Error; err != nil {
		t.Fatalf("重新查询同名部门失败: %v", err)
	}
	if reloaded.UserID != nil || reloaded.Code != "E2E-LEGACY" {
		t.Errorf("非 admin 归属的同名部门不应被改动，实际: user_id=%v code=%q", reloaded.UserID, reloaded.Code)
	}
}

// TestEnsureE2EDepartment_DoesNotTouchOthers 其他部门不应被创建/修改
func TestEnsureE2EDepartment_DoesNotTouchOthers(t *testing.T) {
	db := setupDepartmentDB(t)
	userID := uint(1)

	// 预置其他部门（不同 Name，含 admin 归属与全局归属）
	other := models.Department{UserID: &userID, Name: "财务部", Code: "FIN"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("创建其他部门失败: %v", err)
	}
	global := models.Department{UserID: nil, Name: "全局部门", Code: "GLOBAL"}
	if err := db.Create(&global).Error; err != nil {
		t.Fatalf("创建全局部门失败: %v", err)
	}

	if err := ensureE2EDepartment(db, userID); err != nil {
		t.Fatalf("执行 E2E 测试部门 seed 失败: %v", err)
	}

	// 其他部门保持原样
	var reloadedOther models.Department
	if err := db.First(&reloadedOther, other.ID).Error; err != nil {
		t.Fatalf("查询其他部门失败: %v", err)
	}
	if reloadedOther.Name != "财务部" || reloadedOther.Code != "FIN" {
		t.Errorf("其他部门不应被改动，实际: name=%q code=%q", reloadedOther.Name, reloadedOther.Code)
	}
	var reloadedGlobal models.Department
	if err := db.First(&reloadedGlobal, global.ID).Error; err != nil {
		t.Fatalf("查询全局部门失败: %v", err)
	}
	if reloadedGlobal.Name != "全局部门" || reloadedGlobal.Code != "GLOBAL" {
		t.Errorf("全局部门不应被改动，实际: name=%q code=%q", reloadedGlobal.Name, reloadedGlobal.Code)
	}

	// E2E 部门正常创建
	assertE2EDepartment(t, findE2EDepartment(t, db), userID)
}
