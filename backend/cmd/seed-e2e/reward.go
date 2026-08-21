// E2E 奖惩记录流程测试 Seed —— 幂等准备奖惩管理 E2E 所需的最小数据底座。
//
// 奖惩 API 硬约束（backend/internal/api/reward.go）：
//   - 创建奖惩记录必须关联一名在职员工（employee_id 必填，且属于当前租户）；
//   - 创建即草稿（draft），reward.edit 手动生效（effective），
//     draft/effective 均可由 reward.delete 填写原因作废（voided 终态）；
//   - 奖惩记录不改变员工状态或薪资，仅作台账记录。
//
// 本文件仅操作精确匹配 E2E 标记（姓名/身份证/工号）的记录，绝不触碰非 E2E 数据。
package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// 奖惩 E2E 固定在职员工（可预测工号，供前端 E2E 稳定定位）。
// 独立于离职/转正员工，避免被其他流程 E2E 变更状态，保证「验证员工仍 active」验收稳定成立。
const (
	e2eRewardEmployeeName     = "E2E奖惩测试员工"
	e2eRewardEmployeeIDNumber = "110101199607154321" // 虚构测试身份证号，稳定唯一
	e2eRewardEmployeeID       = "E2E-REWARD-001"
)

// ensureE2ERewardEmployee 幂等创建/恢复一名固定在职员工（供奖惩记录 E2E 使用）。
// 按 (user_id, id_number) 唯一索引定位：
//   - 不存在则创建（status=active）；
//   - 已存在则强制恢复为 active，保证「奖惩不改变员工状态」验收稳定成立。
func ensureE2ERewardEmployee(db *gorm.DB, userID uint) error {
	var employee models.Employee
	err := db.Where("user_id = ? AND id_number = ?", userID, e2eRewardEmployeeIDNumber).First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		employee = models.Employee{
			UserID:           userID,
			EmployeeID:       e2eRewardEmployeeID,
			Name:             e2eRewardEmployeeName,
			IDNumber:         e2eRewardEmployeeIDNumber,
			Department:       e2eEmployeeDepartment,
			Position:         e2eEmployeePosition,
			Status:           models.EmployeeStatusActive,
			EmploymentStatus: models.EmploymentStatusFormal,
		}
		if err := db.Create(&employee).Error; err != nil {
			return fmt.Errorf("创建奖惩 E2E 员工失败: %w", err)
		}
		fmt.Printf("  [创建] 奖惩 E2E 员工（姓名: %s，工号: %s，状态: active）\n",
			e2eRewardEmployeeName, e2eRewardEmployeeID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询奖惩 E2E 员工失败: %w", err)
	}

	// 已存在：恢复为 active（奖惩不改变员工状态，重复运行安全）
	updates := map[string]any{
		"employee_id":       e2eRewardEmployeeID,
		"name":              e2eRewardEmployeeName,
		"department":        e2eEmployeeDepartment,
		"position":          e2eEmployeePosition,
		"status":            models.EmployeeStatusActive,
		"employment_status": models.EmploymentStatusFormal,
		"updated_at":        time.Now(),
	}
	if err := db.Model(&employee).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复奖惩 E2E 员工失败: %w", err)
	}
	fmt.Printf("  [恢复] 奖惩 E2E 员工（姓名: %s，工号: %s，状态: active）\n",
		e2eRewardEmployeeName, e2eRewardEmployeeID)
	return nil
}
