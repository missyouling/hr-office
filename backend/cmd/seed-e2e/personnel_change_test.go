package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupPersonnelChangeSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Department{}, &models.Employee{}); err != nil {
		t.Fatalf("迁移人事异动 seed 表失败: %v", err)
	}
	return db
}

func TestEnsureE2EPersonnelChangeData_RestoresBaseline(t *testing.T) {
	db := setupPersonnelChangeSeedDB(t)
	const userID = uint(1)
	if err := ensureE2EPersonnelChangeData(db, userID); err != nil {
		t.Fatalf("首次准备人事异动 E2E 数据失败: %v", err)
	}
	var employee models.Employee
	if err := db.Where("user_id = ? AND id_number = ?", userID, e2ePersonnelChangeEmployeeIDNumber).First(&employee).Error; err != nil {
		t.Fatalf("查询人事异动 E2E 员工失败: %v", err)
	}
	if employee.Department != e2ePersonnelChangeBeforeDepartment || employee.Position != e2ePersonnelChangeBeforePosition || employee.JobLevel != e2ePersonnelChangeBeforeJobLevel || employee.Status != models.EmployeeStatusActive {
		t.Fatalf("员工基线资料不正确: %#v", employee)
	}
	if err := db.Model(&employee).Updates(map[string]any{"department": "已变更部门", "position": "已变更岗位", "job_level": "P9", "status": "resigned"}).Error; err != nil {
		t.Fatalf("模拟异动残留失败: %v", err)
	}
	if err := ensureE2EPersonnelChangeData(db, userID); err != nil {
		t.Fatalf("恢复人事异动 E2E 数据失败: %v", err)
	}
	if err := db.First(&employee, employee.ID).Error; err != nil {
		t.Fatalf("重新查询人事异动 E2E 员工失败: %v", err)
	}
	if employee.Department != e2ePersonnelChangeBeforeDepartment || employee.Position != e2ePersonnelChangeBeforePosition || employee.JobLevel != e2ePersonnelChangeBeforeJobLevel || employee.Status != models.EmployeeStatusActive {
		t.Fatalf("恢复后员工基线资料不正确: %#v", employee)
	}
	var departments int64
	db.Model(&models.Department{}).Where("user_id = ? AND name IN ?", userID, []string{e2ePersonnelChangeBeforeDepartment, e2ePersonnelChangeAfterDepartment}).Count(&departments)
	if departments != 2 {
		t.Fatalf("应存在两个异动 E2E 部门，实际为 %d", departments)
	}
}
