package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ErrOnboardingEmployeeConflict 表示导入事务内发现员工身份证冲突。
var ErrOnboardingEmployeeConflict = errors.New("该身份证号已存在员工记录，无法重复入职")

// OnboardingImportRow 入职导入单行数据（手动 JSON 与 Excel 共用）。
type OnboardingImportRow struct {
	Name             string `json:"name"`
	IDNumber         string `json:"id_number"`
	Phone            string `json:"phone"`
	Department       string `json:"department"`
	Position         string `json:"position"`
	PlannedHireDate  string `json:"planned_hire_date"`
	EmploymentStatus string `json:"employment_status"`
	Remarks          string `json:"remarks"`
}

// OnboardingImportError 导入校验错误（行号 + 原因）。
type OnboardingImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

func (e OnboardingImportError) Error() string {
	return fmt.Sprintf("第 %d 行: %s", e.Row, e.Reason)
}

// NormalizeEmploymentStatus 归一化用工状态（支持中英文），非法值原样返回由校验层拒绝。
func NormalizeEmploymentStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trial", "试用", "试用期":
		return models.EmploymentStatusTrial
	case "formal", "正式", "正式员工":
		return models.EmploymentStatusFormal
	default:
		return strings.TrimSpace(s)
	}
}

// ValidateOnboardingImportRows 全文件预校验（P12.3.2.3）：
//  1. 必填字段（姓名/身份证号/计划入职日期）
//  2. 计划入职日期格式（YYYY-MM-DD）
//  3. 用工状态合法值（trial/formal/空）
//  4. 文件内身份证重复拒绝
//  5. 身份证命中 employees 全量（active/resigned）统一拒绝
//
// 任一错误返回全部错误列表，调用方整体拒绝不落库。
func ValidateOnboardingImportRows(db *gorm.DB, userID uint, rows []OnboardingImportRow) []OnboardingImportError {
	var errs []OnboardingImportError
	seen := make(map[string]int) // 身份证 -> 首次出现行号
	for i, row := range rows {
		line := i + 2 // 表头占第 1 行
		name := strings.TrimSpace(row.Name)
		idNumber := strings.TrimSpace(row.IDNumber)
		plannedDate := strings.TrimSpace(row.PlannedHireDate)
		employmentStatus := NormalizeEmploymentStatus(row.EmploymentStatus)

		if name == "" {
			errs = append(errs, OnboardingImportError{Row: line, Reason: "姓名必填"})
		}
		if idNumber == "" {
			errs = append(errs, OnboardingImportError{Row: line, Reason: "身份证号必填"})
		}
		if plannedDate == "" {
			errs = append(errs, OnboardingImportError{Row: line, Reason: "计划入职日期必填"})
		} else if _, err := time.Parse("2006-01-02", plannedDate); err != nil {
			errs = append(errs, OnboardingImportError{Row: line, Reason: "计划入职日期格式应为 YYYY-MM-DD"})
		}
		if employmentStatus != "" && !models.IsValidEmploymentStatus(employmentStatus) {
			errs = append(errs, OnboardingImportError{Row: line, Reason: "用工状态仅支持 trial/formal（试用/正式）"})
		}
		if idNumber != "" {
			if firstLine, dup := seen[idNumber]; dup {
				errs = append(errs, OnboardingImportError{Row: line, Reason: fmt.Sprintf("文件内身份证号与第 %d 行重复", firstLine)})
			} else {
				seen[idNumber] = line
			}
		}
	}
	// 基础校验全部通过后再查员工冲突（避免无效行浪费查询）
	if len(errs) == 0 {
		for i, row := range rows {
			line := i + 2
			idNumber := strings.TrimSpace(row.IDNumber)
			if idNumber == "" {
				continue
			}
			var count int64
			if err := db.Model(&models.Employee{}).Where("user_id = ? AND id_number = ?", userID, idNumber).Count(&count).Error; err != nil {
				errs = append(errs, OnboardingImportError{Row: line, Reason: "员工冲突检查失败"})
				continue
			}
			if count > 0 {
				errs = append(errs, OnboardingImportError{Row: line, Reason: "该身份证号已存在员工记录，无法重复入职"})
			}
		}
	}
	return errs
}

// ImportOnboardingRecords 单事务批量创建待入职记录（P12.3.2.3）。
// 调用方必须先通过 ValidateOnboardingImportRows 预校验，此处不再重复校验；
// 任一创建失败整体回滚，不产生部分落库。
func ImportOnboardingRecords(db *gorm.DB, userID uint, rows []OnboardingImportRow) ([]models.OnboardingRecord, error) {
	records := make([]models.OnboardingRecord, 0, len(rows))
	err := db.Transaction(func(tx *gorm.DB) error {
		idNumbers := make([]string, 0, len(rows))
		for _, row := range rows {
			idNumbers = append(idNumbers, strings.TrimSpace(row.IDNumber))
		}
		var conflictCount int64
		if err := tx.Model(&models.Employee{}).
			Where("user_id = ? AND id_number IN ?", userID, idNumbers).
			Count(&conflictCount).Error; err != nil {
			return err
		}
		if conflictCount > 0 {
			return ErrOnboardingEmployeeConflict
		}
		for _, row := range rows {
			rec := models.OnboardingRecord{
				UserID:           userID,
				Name:             strings.TrimSpace(row.Name),
				IDNumber:         strings.TrimSpace(row.IDNumber),
				Phone:            strings.TrimSpace(row.Phone),
				Department:       strings.TrimSpace(row.Department),
				Position:         strings.TrimSpace(row.Position),
				PlannedHireDate:  strings.TrimSpace(row.PlannedHireDate),
				EmploymentStatus: NormalizeEmploymentStatus(row.EmploymentStatus),
				Remarks:          strings.TrimSpace(row.Remarks),
				Status:           models.OnboardingStatusPending,
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
			records = append(records, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// ParseOnboardingExcel 解析入职导入 Excel 文件。
// 表头（兼容中英文）：姓名、身份证号、联系电话、部门、岗位、计划入职日期、用工状态、备注。
// 返回空行会被跳过；无有效数据行返回错误。
func ParseOnboardingExcel(content []byte) ([]OnboardingImportRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("打开 Excel 失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, errors.New("Excel 无工作表")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel 至少需要表头与一行数据")
	}

	header := rows[0]
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}
	get := func(row []string, names ...string) string {
		for _, n := range names {
			if idx, ok := colIdx[n]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
		}
		return ""
	}

	var items []OnboardingImportRow
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		allEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}
		items = append(items, OnboardingImportRow{
			Name:             get(row, "姓名", "name"),
			IDNumber:         get(row, "身份证号", "身份证号码", "id_number"),
			Phone:            get(row, "联系电话", "电话", "phone"),
			Department:       get(row, "部门", "department"),
			Position:         get(row, "岗位", "position"),
			PlannedHireDate:  get(row, "计划入职日期", "预计入职日期", "planned_hire_date"),
			EmploymentStatus: get(row, "用工状态", "employment_status"),
			Remarks:          get(row, "备注", "remark", "remarks"),
		})
	}
	if len(items) == 0 {
		return nil, errors.New("Excel 中无有效数据行")
	}
	return items, nil
}
