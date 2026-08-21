// E2E 入职流程测试部门 Seed —— 幂等确保 admin 所属的测试部门存在。
//
// 部门名称与既有离职员工 seed（e2eEmployeeDepartment）保持一致，
// 保证全 E2E 数据（员工/入职/部门）使用同一稳定名称；
// 部门编码使用稳定非空 E2E 前缀，供前端入职 E2E 与后端查询稳定定位。
// 仅操作精确匹配 Name 的部门记录，绝不触碰其他部门；
// 本文件不预建任何 onboarding 记录（OnboardingRecord/WorkTodo/OnboardingImportRun），
// 避免与入职 E2E 流程的创建步骤冲突。
package main

import (
	"fmt"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// e2eDepartmentName 复用离职员工 seed 的部门名，保证全 E2E 数据使用同一稳定名称
const e2eDepartmentName = e2eEmployeeDepartment

// e2eDepartmentCode E2E 测试部门编码（稳定非空、E2E 前缀，供入职 E2E 稳定定位）
const e2eDepartmentCode = "E2E-TEST"

// ensureE2EDepartment 幂等确保 admin 所属的 E2E 测试部门存在。
// 按 Name（全局唯一索引）定位：
//   - 不存在：创建归属 admin 的部门（Name=e2eDepartmentName，Code=e2eDepartmentCode）；
//   - 已存在且归属 admin：幂等收敛 Code 为目标值（编码纠正），不重复创建；
//   - 已存在但归属非 admin（含 user_id 为 NULL 的全局部门）：返回明确错误，
//     绝不修改非 E2E 数据，由操作者人工处理归属冲突。
func ensureE2EDepartment(db *gorm.DB, userID uint) error {
	var dept models.Department
	err := db.Where("name = ?", e2eDepartmentName).First(&dept).Error
	if err == gorm.ErrRecordNotFound {
		dept = models.Department{
			UserID: &userID,
			Name:   e2eDepartmentName,
			Code:   e2eDepartmentCode,
		}
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建 E2E 测试部门失败: %w", err)
		}
		fmt.Printf("  [创建] E2E 测试部门（名称: %s，编码: %s，归属用户: %d）\n",
			e2eDepartmentName, e2eDepartmentCode, userID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询 E2E 测试部门失败: %w", err)
	}

	// 已存在：先校验归属，避免触碰非 admin 数据
	if dept.UserID == nil || *dept.UserID != userID {
		return fmt.Errorf("已存在同名部门 %q（user_id=%v）但不归属用户 %d，为避免触碰非 E2E 数据已拒绝修改，请人工处理",
			dept.Name, dept.UserID, userID)
	}

	// 归属正确：编码与目标不一致则纠正（幂等收敛）
	if dept.Code == e2eDepartmentCode {
		fmt.Printf("  [跳过] E2E 测试部门已存在（名称: %s，编码: %s）\n", dept.Name, dept.Code)
		return nil
	}
	if err := db.Model(&dept).Update("code", e2eDepartmentCode).Error; err != nil {
		return fmt.Errorf("纠正 E2E 测试部门编码失败: %w", err)
	}
	fmt.Printf("  [纠正] E2E 测试部门编码（名称: %s，编码: %s → %s）\n",
		dept.Name, dept.Code, e2eDepartmentCode)
	return nil
}
