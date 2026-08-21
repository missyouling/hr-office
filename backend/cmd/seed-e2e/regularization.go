// E2E 转正流程测试 Seed —— 幂等准备转正管理 E2E 所需的最小数据底座。
//
// 转正 API 硬约束（backend/internal/api/regularization.go）：
//   - 发起人、直属上级、HR 复核人必须为三名不同用户且同租户（company_id 相同且非空）；
//   - 仅 active+trial 员工可发起转正申请；
//   - 审批编号租户内唯一（服务端生成）。
//
// 本文件仅操作精确匹配 E2E 标记（用户名/身份证/工号）的记录，绝不触碰非 E2E 数据；
// 密码仅用于本地 E2E 登录（helpers/auth.ts 的 ACCOUNTS），不写入任何生产配置。
package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// e2eRegularizationTenantID 转正 E2E 三人审批的固定同租户标识（仅本地 E2E 库使用）。
const e2eRegularizationTenantID = "E2E-TENANT-001"

// 转正 E2E 三名审批用户（均需 employee.edit 权限，seed-e2e 已按角色默认授权）：
// admin 发起（HR）、manager 直属上级、editor HR 复核。
const (
	e2eRegularizationInitiator  = "admin"
	e2eRegularizationSupervisor = "manager"
	e2eRegularizationHRReviewer = "editor"
)

// e2eTrialEmployee 描述一名稳定试用期员工（可预测工号，供前端 E2E 稳定定位）。
type e2eTrialEmployee struct {
	Name       string
	IDNumber   string // 虚构测试身份证号，稳定唯一
	EmployeeID string // 可预测工号
}

// 转正 E2E 两名稳定试用期员工：
//   - 主流程员工：走「上级通过 + HR 通过 → effective → formal」；
//   - 拒绝路径员工：走「上级通过 + HR 拒绝 → rejected」，保持 trial 并生成离职待办。
var e2eTrialEmployees = []e2eTrialEmployee{
	{Name: "E2E转正测试员工", IDNumber: "110101199203154321", EmployeeID: "E2E-REG-001"},
	{Name: "E2E转正拒绝员工", IDNumber: "110101199405154321", EmployeeID: "E2E-REG-002"},
}

// ensureE2ERegularizationUsers 幂等确保三名审批用户同租户（company_id 相同且非空）。
// 仅更新精确匹配用户名的记录，不触碰其他用户。
func ensureE2ERegularizationUsers(db *gorm.DB) error {
	for _, username := range []string{e2eRegularizationInitiator, e2eRegularizationSupervisor, e2eRegularizationHRReviewer} {
		var user models.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			return fmt.Errorf("查询转正审批用户 %s 失败: %w", username, err)
		}
		if user.CompanyID == e2eRegularizationTenantID {
			continue
		}
		if err := db.Model(&user).Update("company_id", e2eRegularizationTenantID).Error; err != nil {
			return fmt.Errorf("设置转正审批用户 %s 租户失败: %w", username, err)
		}
		fmt.Printf("  [设置] 转正审批用户 %s 租户（company_id: %s）\n", username, e2eRegularizationTenantID)
	}
	return nil
}

// ensureE2ETrialEmployees 幂等创建/恢复两名稳定试用期员工（供转正 E2E 使用）。
// 按 (user_id, id_number) 唯一索引定位：
//   - 不存在则创建（status=active, employment_status=trial, 可预测工号）；
//   - 已存在则强制恢复为 active+trial 并清空转正字段，保证重复运行安全；
//   - 同时作废该员工进行中的转正记录（避免重复运行被"已有进行中申请"拦截）。
func ensureE2ETrialEmployees(db *gorm.DB, userID uint) error {
	for _, spec := range e2eTrialEmployees {
		if err := ensureOneTrialEmployee(db, userID, spec); err != nil {
			return err
		}
	}
	return nil
}

// ensureOneTrialEmployee 幂等创建/恢复单个稳定试用期员工。
func ensureOneTrialEmployee(db *gorm.DB, userID uint, spec e2eTrialEmployee) error {
	var employee models.Employee
	err := db.Where("user_id = ? AND id_number = ?", userID, spec.IDNumber).First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		employee = models.Employee{
			UserID:           userID,
			EmployeeID:       spec.EmployeeID,
			Name:             spec.Name,
			IDNumber:         spec.IDNumber,
			Department:       e2eEmployeeDepartment,
			Position:         e2eEmployeePosition,
			Status:           models.EmployeeStatusActive,
			EmploymentStatus: models.EmploymentStatusTrial,
			ProbationEndDate: time.Now().Format("2006-01-02"),
		}
		if err := db.Create(&employee).Error; err != nil {
			return fmt.Errorf("创建转正 E2E 员工失败: %w", err)
		}
		fmt.Printf("  [创建] 转正 E2E 员工（姓名: %s，工号: %s，状态: active/trial）\n",
			spec.Name, spec.EmployeeID)
	} else if err != nil {
		return fmt.Errorf("查询转正 E2E 员工失败: %w", err)
	} else {
		updates := map[string]any{
			"employee_id":         spec.EmployeeID,
			"name":                spec.Name,
			"department":          e2eEmployeeDepartment,
			"position":            e2eEmployeePosition,
			"status":              models.EmployeeStatusActive,
			"employment_status":   models.EmploymentStatusTrial,
			"probation_end_date":  time.Now().Format("2006-01-02"),
			"actual_regular_date": "",
			"updated_at":          time.Now(),
		}
		if err := db.Model(&employee).Updates(updates).Error; err != nil {
			return fmt.Errorf("恢复转正 E2E 员工失败: %w", err)
		}
		fmt.Printf("  [恢复] 转正 E2E 员工（姓名: %s，工号: %s，状态: active/trial）\n",
			spec.Name, spec.EmployeeID)
	}

	// 作废该员工进行中的转正记录（幂等重置，避免重复运行被"已有进行中申请"拦截）
	inFlight := []string{
		models.RegularizationStatusPendingSupervisor,
		models.RegularizationStatusPendingHRReview,
		models.RegularizationStatusScheduled,
		models.RegularizationStatusPostponedScheduled,
	}
	now := time.Now()
	if err := db.Model(&models.RegularizationRecord{}).
		Where("user_id = ? AND employee_id = ? AND status IN ?", userID, employee.ID, inFlight).
		Updates(map[string]any{
			"status":      models.RegularizationStatusVoided,
			"void_reason": "E2E 种子重置（重复运行安全）",
			"voided_at":   now,
		}).Error; err != nil {
		return fmt.Errorf("作废转正 E2E 员工进行中记录失败: %w", err)
	}
	return nil
}