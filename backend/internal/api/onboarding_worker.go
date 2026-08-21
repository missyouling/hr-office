package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// onboardingWorkerTimeZone 入职定时任务时区（Asia/Shanghai）。
const onboardingWorkerTimeZone = "Asia/Shanghai"

const employeeIDCreateMaxAttempts = 3

// ErrOnboardingDeptCodeMissing 部门编码缺失错误。
// 手动确认/快速入职返回业务错误；worker 路径转异常待办并保留 pending。
var ErrOnboardingDeptCodeMissing = errors.New("部门未配置编码，无法生成工号")

// ErrOnboardingEmployeeConflict 表示身份证已存在于员工表。
var ErrOnboardingEmployeeConflict = errors.New("该身份证号已存在员工记录，无法重复入职")

// ErrOnboardingAdminMissing 表示记录创建者所属公司没有管理员。
var ErrOnboardingAdminMissing = errors.New("所属公司未配置管理员，无法创建入职待办")

// StartOnboardingWorker 启动入职定时任务（P12.3.2.3）。
// 每日 Asia/Shanghai 02:00 运行一次；错过窗口不补跑（下次运行按自然日重新计算）；
// 同日幂等（onboarding_import_runs.run_date 唯一索引兜底并发）。
func (h *Handler) StartOnboardingWorker(ctx context.Context) {
	go func() {
		loc, err := time.LoadLocation(onboardingWorkerTimeZone)
		if err != nil {
			log.Printf("[onboarding-worker] 加载时区失败: %v", err)
			return
		}
		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
			if !now.Before(next) {
				next = next.AddDate(0, 0, 1)
			}
			timer := time.NewTimer(next.Sub(now))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				h.runOnboardingWorkerOnce(loc, time.Now())
			}
		}
	}()
}

// runOnboardingWorkerOnce 执行一次入职扫描（同日幂等，并发安全）。
// 只扫描 pending 且 planned_hire_date 等于当日的记录；失败保留 pending、写日志，次日不自动重试。
func (h *Handler) runOnboardingWorkerOnce(loc *time.Location, now time.Time) {
	today := now.In(loc).Format("2006-01-02")

	// 同日幂等：运行记录已存在则跳过（错过窗口不补跑）
	var existing models.OnboardingImportRun
	err := h.db.Where("run_date = ?", today).First(&existing).Error
	if err == nil {
		log.Printf("[onboarding-worker] %s 已运行，跳过", today)
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[onboarding-worker] 检查运行记录失败: %v", err)
		return
	}

	// 创建运行记录（唯一索引兜底并发：插入失败即视为当日已运行）
	run := models.OnboardingImportRun{RunDate: today, Status: models.OnboardingRunStatusSuccess}
	if err := h.db.Create(&run).Error; err != nil {
		log.Printf("[onboarding-worker] 创建运行记录失败（可能同日已运行）: %v", err)
		return
	}

	// 只扫描上海当日，历史失败记录次日不会再次命中。
	var records []models.OnboardingRecord
	if err := h.db.Where("status = ? AND planned_hire_date = ?", models.OnboardingStatusPending, today).Find(&records).Error; err != nil {
		run.Status = models.OnboardingRunStatusFailed
		run.ErrorMsg = "扫描待入职记录失败: " + err.Error()
		_ = h.db.Save(&run).Error
		h.writeOnboardingWorkerLog("error", "扫描待入职记录失败", err.Error())
		return
	}

	processed, failed := 0, 0
	for i := range records {
		if err := h.onboardOneRecord(&records[i], today); err != nil {
			failed++
			// 错误详情只记录业务记录ID，不写身份证号。
			h.writeOnboardingWorkerLog("error", "入职处理失败",
				"record_id="+strconv.FormatUint(uint64(records[i].ID), 10)+" 原因="+err.Error())
		} else {
			processed++
		}
	}

	run.Processed = processed
	run.Failed = failed
	if failed > 0 {
		run.Status = models.OnboardingRunStatusFailed
		run.ErrorMsg = strconv.Itoa(failed) + " 条记录处理失败"
	}
	if err := h.db.Save(&run).Error; err != nil {
		log.Printf("[onboarding-worker] 更新运行记录失败: %v", err)
	}
	log.Printf("[onboarding-worker] %s 运行完成: 成功 %d 失败 %d", today, processed, failed)
}

// onboardOneRecord 单条记录入职处理（事务内调用单一入职生效服务）。
// 成功：Employee active + 实际日期 + 工号 + OnboardingRecord 关联 + 通用待办（归属租户管理员）。
// 部门编码缺失：保留 pending + 写日志 + 创建唯一异常待办，返回错误计入失败。
// 其他失败：返回错误，调用方保留 pending 并写日志。
func (h *Handler) onboardOneRecord(record *models.OnboardingRecord, today string) error {
	employmentStatus := record.EmploymentStatus
	if employmentStatus == "" {
		employmentStatus = models.EmploymentStatusTrial
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		return h.applyOnboardingEffect(tx, record.UserID, record, employmentStatus, today)
	})
	if err != nil {
		adminID, adminErr := findTenantAdminID(h.db, record.UserID)
		if adminErr != nil {
			h.writeOnboardingWorkerLog("error", "创建入职异常待办失败",
				"record_id="+strconv.FormatUint(uint64(record.ID), 10)+" 原因="+adminErr.Error())
		} else {
			h.handleOnboardingException(record, adminID, err)
		}
		return err
	}
	return nil
}

// applyOnboardingEffect 单一内部入职生效服务（必须在事务内调用，三条路径共用）：
// 部门Code检查 → 工号生成 → 创建Employee(active) → 关联OnboardingRecord → 幂等通用待办。
// record.ID==0（快速入职）时创建新记录；否则（手动确认/worker）更新已有记录。
// 仅关联入职登记，绝不创建档案记录/文件。
func (h *Handler) applyOnboardingEffect(tx *gorm.DB, userID uint, record *models.OnboardingRecord, employmentStatus, today string) error {
	var conflictCount int64
	if err := tx.Model(&models.Employee{}).Where("user_id = ? AND id_number = ?", userID, record.IDNumber).Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return ErrOnboardingEmployeeConflict
	}

	// 部门编码：按当前租户 + 记录部门名称查找，缺失返回业务错误
	code, err := findDepartmentCode(tx, userID, record.Department)
	if err != nil {
		return err
	}
	if code == "" {
		return ErrOnboardingDeptCodeMissing
	}

	// 租户管理员（待办归属）
	adminID, err := findTenantAdminID(tx, userID)
	if err != nil {
		return err
	}

	// 创建员工；工号唯一冲突时有限次数重新计算并重试。
	emp := models.Employee{
		UserID:           userID,
		Name:             record.Name,
		IDNumber:         record.IDNumber,
		Department:       record.Department,
		Position:         record.Position,
		HireDate:         today,
		Status:           "active",
		EmploymentStatus: employmentStatus,
	}
	if err := createEmployeeWithIDRetry(tx, &emp, code); err != nil {
		return err
	}

	// 关联入职记录
	if record.ID == 0 {
		// 快速入职：创建新记录（跳过 pending 直接 onboarded）
		record.Status = models.OnboardingStatusOnboarded
		record.ActualHireDate = &today
		record.EmployeeID = &emp.ID
		record.EmploymentStatus = employmentStatus
		if err := tx.Create(record).Error; err != nil {
			return err
		}
	} else {
		// 手动确认/worker：更新已有记录
		if err := tx.Model(&models.OnboardingRecord{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
			"status":            models.OnboardingStatusOnboarded,
			"actual_hire_date":  today,
			"employee_id":       emp.ID,
			"employment_status": employmentStatus,
		}).Error; err != nil {
			return err
		}
	}

	// 幂等创建通用待办（唯一索引兜底：同租户同业务类型+业务ID 仅一条）
	todo := models.WorkTodo{
		UserID:       adminID,
		BusinessType: "onboarding",
		BusinessID:   record.ID,
		Title:        "办理入职手续：" + record.Name,
		Description:  "计划入职日期 " + record.PlannedHireDate + "，请及时办理入职手续",
		Status:       models.WorkTodoStatusPending,
		AssigneeID:   &adminID,
	}
	if err := tx.Create(&todo).Error; err != nil && !isUniqueViolation(err) {
		return err
	}
	return nil
}

// handleOnboardingException 持久化单条自动生效异常，待办按业务键幂等。
func (h *Handler) handleOnboardingException(record *models.OnboardingRecord, adminID uint, cause error) {
	todo := models.WorkTodo{
		UserID:       adminID,
		BusinessType: "onboarding_exception",
		BusinessID:   record.ID,
		Title:        "入职异常：" + record.Name,
		Description:  "自动入职失败，请检查入职资料后手动处理。原因：" + cause.Error(),
		Status:       models.WorkTodoStatusPending,
		AssigneeID:   &adminID,
	}
	if err := h.db.Create(&todo).Error; err != nil && !isUniqueViolation(err) {
		log.Printf("[onboarding-worker] 创建异常待办失败: %v", err)
	}
}

// findDepartmentCode 按当前租户 + 部门名称查找部门编码。
// 部门为全局（user_id IS NULL）或属于当前租户；找不到或编码为空返回空串。
func findDepartmentCode(db *gorm.DB, userID uint, deptName string) (string, error) {
	name := strings.TrimSpace(deptName)
	if name == "" {
		return "", nil
	}
	var dept models.Department
	err := db.Where("name = ? AND (user_id IS NULL OR user_id = ?)", name, userID).First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(dept.Code), nil
}

// findTenantAdminID 查询记录创建者同公司最早创建的 admin/super_admin。
func findTenantAdminID(db *gorm.DB, creatorID uint) (uint, error) {
	var creator models.User
	if err := db.Select("id", "company_id").First(&creator, creatorID).Error; err != nil {
		return 0, err
	}
	var adminUser models.User
	err := db.Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("users.company_id = ? AND roles.name IN ?", creator.CompanyID, []string{models.RoleAdmin, "super_admin"}).
		Order("users.created_at ASC").Order("users.id ASC").
		First(&adminUser).Error
	if err == nil {
		return adminUser.ID, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrOnboardingAdminMissing
	}
	return 0, err
}

// createEmployeeWithIDRetry 创建员工，工号并发冲突时最多重新计算三次。
func createEmployeeWithIDRetry(tx *gorm.DB, emp *models.Employee, deptCode string) error {
	for attempt := 0; attempt < employeeIDCreateMaxAttempts; attempt++ {
		employeeID, err := generateEmployeeID(tx, emp.UserID, deptCode)
		if err != nil {
			return err
		}
		emp.EmployeeID = employeeID
		savepoint := fmt.Sprintf("employee_id_%d", attempt)
		if err := tx.SavePoint(savepoint).Error; err != nil {
			return err
		}
		err = tx.Create(emp).Error
		if err == nil {
			return nil
		}
		if rollbackErr := tx.RollbackTo(savepoint).Error; rollbackErr != nil {
			return rollbackErr
		}
		if !isUniqueViolation(err) {
			return err
		}
		emp.ID = 0
	}
	return errors.New("工号生成冲突，请稍后重试")
}

// generateEmployeeID 生成工号：部门编码 + 三位序号（如 DEV001）。
// 在当前租户 + 部门前缀范围内顺序递增；非匹配历史工号忽略；
// 事务内调用，并对候选工号做存在性检查（数据库检查，避免明显重复）。
func generateEmployeeID(db *gorm.DB, userID uint, deptCode string) (string, error) {
	prefix := strings.TrimSpace(deptCode)
	if prefix == "" {
		return "", errors.New("部门编码为空，无法生成工号")
	}
	var ids []string
	if err := db.Model(&models.Employee{}).Where("user_id = ? AND employee_id != ''", userID).Pluck("employee_id", &ids).Error; err != nil {
		return "", err
	}
	maxSeq := 0
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if !strings.HasPrefix(trimmed, prefix) {
			continue // 非匹配历史工号忽略
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(trimmed, prefix)); err == nil && n > maxSeq {
			maxSeq = n
		}
	}
	// 数据库存在性检查：候选工号已存在（并发写入）则递增重试
	for {
		candidate := fmt.Sprintf("%s%03d", prefix, maxSeq+1)
		var count int64
		if err := db.Model(&models.Employee{}).Where("user_id = ? AND employee_id = ?", userID, candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		maxSeq++
	}
}

// writeOnboardingWorkerLog 写入系统日志（SystemLog），供运维排查。
func (h *Handler) writeOnboardingWorkerLog(level, message, detail string) {
	entry := SystemLog{Level: level, Source: "onboarding-worker", Message: message}
	if detail != "" {
		if data, err := json.Marshal(map[string]string{"detail": detail}); err == nil {
			entry.Details = datatypes.JSON(data)
		}
	}
	if err := h.db.Create(&entry).Error; err != nil {
		log.Printf("[onboarding-worker] 写日志失败: %v", err)
	}
}

// isUniqueViolation 判断是否为唯一约束冲突（SQLite/PostgreSQL 通用）。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
