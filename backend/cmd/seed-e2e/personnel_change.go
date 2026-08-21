// E2E 人事异动测试 Seed —— 幂等准备异动前员工资料和目标部门。
package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

const (
	e2ePersonnelChangeEmployeeName     = "E2E人事异动测试员工"
	e2ePersonnelChangeEmployeeID       = "E2E-CHANGE-001"
	e2ePersonnelChangeEmployeeIDNumber = "110101199708154321"
	e2ePersonnelChangeBeforeDepartment = "E2E异动原部门"
	e2ePersonnelChangeAfterDepartment  = "E2E异动目标部门"
	e2ePersonnelChangeBeforePosition   = "初级专员"
	e2ePersonnelChangeBeforeJobLevel   = "P3"
	e2ePersonnelChangeAfterPosition    = "高级专员"
	e2ePersonnelChangeAfterJobLevel    = "P4"
)

// ensureE2EPersonnelChangeData 恢复异动前员工资料，并保证目标部门属于 admin。
func ensureE2EPersonnelChangeData(db *gorm.DB, userID uint) error {
	if err := ensurePersonnelChangeDepartment(db, userID, e2ePersonnelChangeBeforeDepartment, "E2E-CHANGE-BEFORE"); err != nil {
		return err
	}
	if err := ensurePersonnelChangeDepartment(db, userID, e2ePersonnelChangeAfterDepartment, "E2E-CHANGE-AFTER"); err != nil {
		return err
	}
	return ensurePersonnelChangeEmployee(db, userID)
}

func ensurePersonnelChangeDepartment(db *gorm.DB, userID uint, name, code string) error {
	var department models.Department
	err := db.Where("name = ?", name).First(&department).Error
	if err == gorm.ErrRecordNotFound {
		return db.Create(&models.Department{UserID: &userID, Name: name, Code: code}).Error
	}
	if err != nil {
		return fmt.Errorf("查询人事异动 E2E 部门 %q 失败: %w", name, err)
	}
	if department.UserID == nil || *department.UserID != userID {
		return fmt.Errorf("同名人事异动 E2E 部门 %q 不归属用户 %d，拒绝修改", name, userID)
	}
	if department.Code == code {
		return nil
	}
	return db.Model(&department).Update("code", code).Error
}

func ensurePersonnelChangeEmployee(db *gorm.DB, userID uint) error {
	var employee models.Employee
	err := db.Where("user_id = ? AND id_number = ?", userID, e2ePersonnelChangeEmployeeIDNumber).First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		employee = personnelChangeEmployee(userID)
		if err := db.Create(&employee).Error; err != nil {
			return fmt.Errorf("创建人事异动 E2E 员工失败: %w", err)
		}
		fmt.Printf("  [创建] 人事异动 E2E 员工（姓名: %s，状态: active）\n", employee.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询人事异动 E2E 员工失败: %w", err)
	}
	updates := personnelChangeEmployeeUpdates()
	updates["updated_at"] = time.Now()
	if err := db.Model(&employee).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复人事异动 E2E 员工失败: %w", err)
	}
	fmt.Printf("  [恢复] 人事异动 E2E 员工（姓名: %s，状态: active）\n", employee.Name)
	return nil
}

func personnelChangeEmployee(userID uint) models.Employee {
	return models.Employee{
		UserID:           userID,
		EmployeeID:       e2ePersonnelChangeEmployeeID,
		Name:             e2ePersonnelChangeEmployeeName,
		IDNumber:         e2ePersonnelChangeEmployeeIDNumber,
		Department:       e2ePersonnelChangeBeforeDepartment,
		Position:         e2ePersonnelChangeBeforePosition,
		JobLevel:         e2ePersonnelChangeBeforeJobLevel,
		Status:           models.EmployeeStatusActive,
		EmploymentStatus: models.EmploymentStatusFormal,
	}
}

func personnelChangeEmployeeUpdates() map[string]any {
	return map[string]any{
		"employee_id":       e2ePersonnelChangeEmployeeID,
		"name":              e2ePersonnelChangeEmployeeName,
		"department":        e2ePersonnelChangeBeforeDepartment,
		"position":          e2ePersonnelChangeBeforePosition,
		"job_level":         e2ePersonnelChangeBeforeJobLevel,
		"status":            models.EmployeeStatusActive,
		"employment_status": models.EmploymentStatusFormal,
	}
}
