package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func TestEnsureE2ETrainingEmployeeRestoresActive(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Employee{}); err != nil {
		t.Fatalf("迁移员工表失败: %v", err)
	}
	if err := ensureE2ETrainingEmployee(db, 1); err != nil {
		t.Fatalf("创建培训员工失败: %v", err)
	}
	var employee models.Employee
	if err := db.Where("user_id = ? AND id_number = ?", 1, e2eTrainingEmployeeIDNumber).First(&employee).Error; err != nil {
		t.Fatalf("查询培训员工失败: %v", err)
	}
	if err := db.Model(&employee).Update("status", "resigned").Error; err != nil {
		t.Fatalf("模拟状态变更失败: %v", err)
	}
	if err := ensureE2ETrainingEmployee(db, 1); err != nil {
		t.Fatalf("恢复培训员工失败: %v", err)
	}
	if err := db.First(&employee, employee.ID).Error; err != nil || employee.Status != models.EmployeeStatusActive {
		t.Fatalf("培训员工应恢复为在职: %v, %s", err, employee.Status)
	}
}
