package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

const (
	e2eTrainingEmployeeName     = "E2E培训测试员工"
	e2eTrainingEmployeeID       = "E2E-TRAINING-001"
	e2eTrainingEmployeeIDNumber = "110101199609154321"
)

// ensureE2ETrainingEmployee 幂等创建或恢复培训流程使用的固定在职员工。
func ensureE2ETrainingEmployee(db *gorm.DB, userID uint) error {
	var employee models.Employee
	err := db.Where("user_id = ? AND id_number = ?", userID, e2eTrainingEmployeeIDNumber).First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		employee = models.Employee{UserID: userID, EmployeeID: e2eTrainingEmployeeID, Name: e2eTrainingEmployeeName, IDNumber: e2eTrainingEmployeeIDNumber, Department: e2eEmployeeDepartment, Position: e2eEmployeePosition, Status: models.EmployeeStatusActive, EmploymentStatus: models.EmploymentStatusFormal}
		if err := db.Create(&employee).Error; err != nil {
			return fmt.Errorf("创建培训 E2E 员工失败: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询培训 E2E 员工失败: %w", err)
	}
	updates := map[string]any{"employee_id": e2eTrainingEmployeeID, "name": e2eTrainingEmployeeName, "department": e2eEmployeeDepartment, "position": e2eEmployeePosition, "status": models.EmployeeStatusActive, "employment_status": models.EmploymentStatusFormal, "updated_at": time.Now()}
	if err := db.Model(&employee).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复培训 E2E 员工失败: %w", err)
	}
	return nil
}
