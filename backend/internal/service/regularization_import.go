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

// ErrRegularizationEmployeeConflict 表示导入事务内发现员工已有进行中转正申请。
var ErrRegularizationEmployeeConflict = errors.New("该员工已有进行中的转正申请")

// RegularizationImportRow 转正批量导入单行数据。
type RegularizationImportRow struct {
	EmployeeJobNumber  string `json:"employee_job_number"`
	EmployeeIDNumber   string `json:"employee_id_number"`
	PlannedRegularDate string `json:"planned_regular_date"`
	ContractTermMonths int    `json:"contract_term_months"`
	ProbationEndDate   string `json:"probation_end_date"`
	EmployeeSelfReview string `json:"employee_self_review"`
	Remarks            string `json:"remarks"`
}

// RegularizationImportWarning 导入软提示。
type RegularizationImportWarning struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// RegularizationImportError 导入硬错误（行号 + 原因）。
type RegularizationImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

func (e RegularizationImportError) Error() string {
	return fmt.Sprintf("第 %d 行: %s", e.Row, e.Reason)
}

// ValidateRegularizationImportRows 全文件预校验：任一错误整文件拒绝，不落库。
func ValidateRegularizationImportRows(db *gorm.DB, userID uint, rows []RegularizationImportRow) ([]RegularizationImportWarning, []RegularizationImportError) {
	var warnings []RegularizationImportWarning
	var errs []RegularizationImportError
	seen := make(map[string]int)
	for i, row := range rows {
		line := i + 2
		job := strings.TrimSpace(row.EmployeeJobNumber)
		idNumber := strings.TrimSpace(row.EmployeeIDNumber)
		plannedDate := strings.TrimSpace(row.PlannedRegularDate)
		contractMonths := row.ContractTermMonths
		probationEndDate := strings.TrimSpace(row.ProbationEndDate)

		if job == "" {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "员工工号必填"})
		}
		if idNumber == "" {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "身份证号必填"})
		}
		if plannedDate == "" {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "计划转正日期必填"})
		} else if _, err := time.Parse("2006-01-02", plannedDate); err != nil {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "计划转正日期格式应为 YYYY-MM-DD"})
		}
		if contractMonths <= 0 {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "劳动合同期限（月）必须为正整数"})
		}
		if probationEndDate != "" {
			if _, err := time.Parse("2006-01-02", probationEndDate); err != nil {
				errs = append(errs, RegularizationImportError{Row: line, Reason: "试用期结束日期格式应为 YYYY-MM-DD"})
			}
		}
		if idNumber != "" {
			if firstLine, dup := seen[idNumber]; dup {
				errs = append(errs, RegularizationImportError{Row: line, Reason: fmt.Sprintf("文件内身份证号与第 %d 行重复", firstLine)})
			} else {
				seen[idNumber] = line
			}
		}
	}
	if len(errs) > 0 {
		return warnings, errs
	}

	// 1. 工号+身份证匹配员工
	for i, row := range rows {
		line := i + 2
		job := strings.TrimSpace(row.EmployeeJobNumber)
		idNumber := strings.TrimSpace(row.EmployeeIDNumber)
		var emp models.Employee
		if err := db.Where("user_id = ? AND employee_id = ? AND id_number = ?", userID, job, idNumber).First(&emp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				errs = append(errs, RegularizationImportError{Row: line, Reason: "员工工号和身份证号无法匹配到同租户员工"})
				continue
			}
			errs = append(errs, RegularizationImportError{Row: line, Reason: "员工匹配校验失败"})
			continue
		}
		if emp.Status != models.EmployeeStatusActive || emp.EmploymentStatus != models.EmploymentStatusTrial {
			errs = append(errs, RegularizationImportError{Row: line, Reason: "仅在职试用期员工可导入转正申请"})
			continue
		}
		if err := checkRegularizationInFlight(db, userID, emp.ID); err != nil {
			errs = append(errs, RegularizationImportError{Row: line, Reason: err.Error()})
			continue
		}
		probationEndDate := strings.TrimSpace(row.ProbationEndDate)
		if probationEndDate == "" {
			probationEndDate = strings.TrimSpace(emp.ProbationEndDate)
		}
		warnings = append(warnings, buildRegularizationWarnings(line, emp, row, probationEndDate)...)
	}
	return warnings, errs
}

// BuildRegularizationRecords 仅构造批量导入记录（不落库）。
func BuildRegularizationRecords(db *gorm.DB, userID uint, rows []RegularizationImportRow) ([]models.RegularizationRecord, []RegularizationImportWarning, error) {
	warnings, errs := ValidateRegularizationImportRows(db, userID, rows)
	if len(errs) > 0 {
		return nil, warnings, errs[0]
	}
	records := make([]models.RegularizationRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := buildRegularizationRecord(db, userID, row)
		if err != nil {
			return nil, warnings, err
		}
		records = append(records, rec)
	}
	return records, warnings, nil
}

// ParseRegularizationExcel 解析转正导入 Excel 文件。
func ParseRegularizationExcel(content []byte) ([]RegularizationImportRow, error) {
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

	var items []RegularizationImportRow
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
		items = append(items, RegularizationImportRow{
			EmployeeJobNumber:  get(row, "员工工号", "工号", "employee_job_number"),
			EmployeeIDNumber:   get(row, "身份证号", "身份证号码", "id_number"),
			PlannedRegularDate: get(row, "计划转正日期", "planned_regular_date"),
			ContractTermMonths: parseIntCell(get(row, "劳动合同期限（月）", "合同期限（月）", "contract_term_months")),
			ProbationEndDate:   get(row, "试用期结束日期", "probation_end_date"),
			EmployeeSelfReview: get(row, "员工自评", "employee_self_review"),
			Remarks:            get(row, "备注", "remarks"),
		})
	}
	if len(items) == 0 {
		return nil, errors.New("Excel 中无有效数据行")
	}
	return items, nil
}

func parseIntCell(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(value, "%d", &n)
	return n
}

func checkRegularizationInFlight(db *gorm.DB, userID, employeeID uint) error {
	var count int64
	if err := db.Model(&models.RegularizationRecord{}).Where("user_id = ? AND employee_id = ? AND status IN ?", userID, employeeID,
		[]string{models.RegularizationStatusPendingSupervisor, models.RegularizationStatusPendingHRReview, models.RegularizationStatusScheduled, models.RegularizationStatusPostponedScheduled}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrRegularizationEmployeeConflict
	}
	return nil
}

func buildRegularizationRecord(tx *gorm.DB, userID uint, row RegularizationImportRow) (models.RegularizationRecord, error) {
	var emp models.Employee
	if err := tx.Where("user_id = ? AND employee_id = ? AND id_number = ?", userID, strings.TrimSpace(row.EmployeeJobNumber), strings.TrimSpace(row.EmployeeIDNumber)).First(&emp).Error; err != nil {
		return models.RegularizationRecord{}, err
	}
	probationEndDate := strings.TrimSpace(row.ProbationEndDate)
	if probationEndDate == "" {
		probationEndDate = strings.TrimSpace(emp.ProbationEndDate)
	}
	rec := models.RegularizationRecord{
		UserID:                   userID,
		EmployeeID:               &emp.ID,
		SnapshotName:             emp.Name,
		SnapshotDepartment:       emp.Department,
		SnapshotPosition:         emp.Position,
		SnapshotEmploymentStatus: emp.EmploymentStatus,
		SnapshotProbationEndDate: probationEndDate,
		ContractTermMonths:       row.ContractTermMonths,
		EmployeeSelfReview:       strings.TrimSpace(row.EmployeeSelfReview),
		PlannedRegularDate:       strings.TrimSpace(row.PlannedRegularDate),
		Status:                   models.RegularizationStatusPendingSupervisor,
		Source:                   models.RegularizationSourceExcelDirect,
	}
	return rec, nil
}

// IsImmediateEffect 判断计划日期是否已经到达（按字符串 YYYY-MM-DD 字典序比较）。
func (r RegularizationImportRow) IsImmediateEffect(today string) bool {
	return strings.TrimSpace(r.PlannedRegularDate) <= today
}

func buildRegularizationWarnings(rowIndex int, emp models.Employee, row RegularizationImportRow, probationEndDate string) []RegularizationImportWarning {
	if probationEndDate == "" || strings.TrimSpace(emp.HireDate) == "" {
		return []RegularizationImportWarning{{Row: rowIndex, Reason: "缺少入职日期或试用期结束日期，无法校验试用期合规"}}
	}
	hire, err1 := time.Parse("2006-01-02", strings.TrimSpace(emp.HireDate))
	probation, err2 := time.Parse("2006-01-02", probationEndDate)
	if err1 != nil || err2 != nil || probation.Before(hire) {
		return []RegularizationImportWarning{{Row: rowIndex, Reason: "试用期日期不合法，无法校验试用期合规"}}
	}
	limit, reason := probationLimitByMonths(row.ContractTermMonths, hire)
	if limit == nil {
		return nil
	}
	if probation.After(*limit) {
		return []RegularizationImportWarning{{Row: rowIndex, Reason: reason}}
	}
	return nil
}

func probationLimitByMonths(months int, hire time.Time) (*time.Time, string) {
	switch {
	case months < 3:
		return &hire, "不得约定试用期"
	case months < 12:
		limit := hire.AddDate(0, 1, 0)
		return &limit, "试用期最长1月"
	case months < 36:
		limit := hire.AddDate(0, 2, 0)
		return &limit, "试用期最长2月"
	default:
		limit := hire.AddDate(0, 6, 0)
		return &limit, "试用期最长6月"
	}
}
