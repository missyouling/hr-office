package service

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"siapp/internal/models"
)

type Processor struct {
	db *gorm.DB
}

func NewProcessor(db *gorm.DB) *Processor {
	return &Processor{db: db}
}

type ParseResult struct {
	File     models.SourceFile `json:"file"`
	Imported int               `json:"imported"`
}

type RosterParseResult struct {
	Imported int `json:"imported"`
}

var headerMap = map[string]string{
	"序号":      "seq",
	"姓名":      "name",
	"证件类型":    "id_type",
	"证件号码":    "id_number",
	"部门":      "department",
	"缴费工资":    "salary",
	"缴费基数":    "base",
	"费率":      "rate",
	"应缴费额":    "amount_due",
	"减免费额":    "deduction",
	"应补(退)费额": "amount_adjust",
	"人员编号":    "person_code",
}

var rosterHeaderMap = map[string]string{
	"姓名":    "name",
	"证件号码":  "id_number",
	"身份证号码": "id_number",
	"身份证号":  "id_number",
	"部门":    "department",
	"岗位":    "title",
	"职务":    "title",
	"备注":    "remarks",
}

func (p *Processor) ParseSourceFile(periodID uint, userID *uint, storedPath, originalName string, scheme models.Scheme, part models.Part) (*ParseResult, error) {
	return p.parseSourceFileWithType(periodID, userID, storedPath, originalName, scheme, part, models.FileTypeNormal)
}

func (p *Processor) ParseAdjustmentFile(periodID uint, userID *uint, storedPath, originalName string, scheme models.Scheme, part models.Part) (*ParseResult, error) {
	return p.parseSourceFileWithType(periodID, userID, storedPath, originalName, scheme, part, models.FileTypeAdjustment)
}

func (p *Processor) parseSourceFileWithType(periodID uint, userID *uint, storedPath, originalName string, scheme models.Scheme, part models.Part, fileType models.FileType) (*ParseResult, error) {
	f, err := excelize.OpenFile(storedPath)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel文件中没有找到工作表")
	}
	sheetName := sheets[0]

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel文件中没有数据行，请检查文件内容是否正确")
	}

	header := rows[0]
	indexMap := map[string]int{}
	for idx, cell := range header {
		cell = strings.TrimSpace(cell)
		if key, ok := headerMap[cell]; ok {
			indexMap[key] = idx
		}
	}
	required := []string{"seq", "name", "id_number", "salary", "base", "rate", "amount_due"}
	for _, key := range required {
		if _, ok := indexMap[key]; !ok {
			return nil, fmt.Errorf("missing required column: %s", key)
		}
	}

	var records []models.RawRecord
	now := time.Now()

	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		seq := getCell(row, indexMap["seq"])
		name := strings.TrimSpace(getCell(row, indexMap["name"]))
		idNumber := strings.TrimSpace(getCell(row, indexMap["id_number"]))
		if seq == "" || name == "" || idNumber == "" {
			continue
		}

		record := models.RawRecord{
			UserID:     userID,
			PeriodID:   periodID,
			Sequence:   toInt(seq),
			Name:       name,
			IDType:     strings.TrimSpace(getCell(row, indexMap["id_type"])),
			IDNumber:   idNumber,
			Department: strings.TrimSpace(getCell(row, indexMap["department"])),
			PaySalary:  toFloat(getCell(row, indexMap["salary"])),
			PayBase:    toFloat(getCell(row, indexMap["base"])),
			RateText:   strings.TrimSpace(getCell(row, indexMap["rate"])),
			AmountDue:  toFloat(getCell(row, indexMap["amount_due"])),
			Scheme:     scheme,
			Part:       part,
			FileType:   fileType,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if idx, ok := indexMap["amount_adjust"]; ok {
			record.AmountAdjust = toFloat(getCell(row, idx))
		} else {
			record.AmountAdjust = record.AmountDue
		}
		if idx, ok := indexMap["person_code"]; ok {
			record.PersonCode = strings.TrimSpace(getCell(row, idx))
		}

		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, errors.New("Excel文件中没有找到有效的数据行，请检查文件格式和内容")
	}

	var savedSource models.SourceFile
	txErr := p.db.Transaction(func(tx *gorm.DB) error {
		// 对于正常文件，删除同类旧记录（覆盖模式）
		// 对于补退文件，不删除旧记录（累加模式）
		if fileType == models.FileTypeNormal {
			if err := tx.Where("period_id = ? AND scheme = ? AND part = ? AND file_type = ?", periodID, scheme, part, fileType).Delete(&models.RawRecord{}).Error; err != nil {
				return fmt.Errorf("cleanup existing raw records: %w", err)
			}
			if err := tx.Where("period_id = ? AND scheme = ? AND part = ? AND file_type = ?", periodID, scheme, part, fileType).Delete(&models.SourceFile{}).Error; err != nil {
				return fmt.Errorf("cleanup existing source files: %w", err)
			}
		}

		source := models.SourceFile{
			UserID:       userID,
			PeriodID:     periodID,
			FileName:     filepath.Base(storedPath),
			StoredPath:   storedPath,
			Scheme:       scheme,
			Part:         part,
			FileType:     fileType,
			Rows:         len(records),
			Status:       "parsed",
			OriginalName: originalName,
			UploadedAt:   now,
		}
		if err := tx.Create(&source).Error; err != nil {
			return fmt.Errorf("save source file: %w", err)
		}
		savedSource = source

		for i := range records {
			records[i].SourceFileID = source.ID
		}
		if err := tx.Create(&records).Error; err != nil {
			return fmt.Errorf("insert raw records: %w", err)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return &ParseResult{
		File:     savedSource,
		Imported: len(records),
	}, nil
}

func (p *Processor) ParseRosterFile(periodID uint, userID *uint, storedPath, originalName string) (*RosterParseResult, error) {
	fmt.Printf("ParseRosterFile: starting to parse file %s (original: %s)\n", storedPath, originalName)

	f, err := excelize.OpenFile(storedPath)
	if err != nil {
		fmt.Printf("ParseRosterFile: failed to open excel file: %v\n", err)
		return nil, fmt.Errorf("open roster excel: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	fmt.Printf("ParseRosterFile: found %d sheets: %v\n", len(sheets), sheets)
	if len(sheets) == 0 {
		return nil, errors.New("花名册Excel文件中没有找到工作表")
	}
	sheetName := sheets[0]

	rows, err := f.GetRows(sheetName)
	if err != nil {
		fmt.Printf("ParseRosterFile: failed to read rows: %v\n", err)
		return nil, fmt.Errorf("read rows: %w", err)
	}
	fmt.Printf("ParseRosterFile: found %d rows\n", len(rows))
	if len(rows) < 2 {
		return nil, errors.New("花名册Excel文件中没有数据行，请检查文件内容是否正确")
	}

	header := rows[0]
	fmt.Printf("ParseRosterFile: header row: %v\n", header)
	indexMap := map[string]int{}
	for idx, cell := range header {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		if key, ok := rosterHeaderMap[cell]; ok {
			indexMap[key] = idx
			continue
		}
		switch strings.ToLower(cell) {
		case "name":
			indexMap["name"] = idx
		case "idnumber", "id_no", "id", "身份证号":
			indexMap["id_number"] = idx
		case "department", "dept":
			indexMap["department"] = idx
		case "title", "position":
			indexMap["title"] = idx
		case "remarks", "remark", "note":
			indexMap["remarks"] = idx
		}
	}

	fmt.Printf("ParseRosterFile: indexMap: %v\n", indexMap)

	if _, ok := indexMap["id_number"]; !ok {
		fmt.Printf("ParseRosterFile: missing id_number column in indexMap\n")
		return nil, errors.New("花名册文件缺少必需的列：证件号码")
	}
	if _, ok := indexMap["department"]; !ok {
		fmt.Printf("ParseRosterFile: missing department column in indexMap\n")
		return nil, errors.New("花名册文件缺少必需的列：部门")
	}

	now := time.Now()
	var entries []models.RosterEntry
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		idNumber := strings.TrimSpace(getCell(row, indexMap["id_number"]))
		if idNumber == "" {
			continue
		}
		entry := models.RosterEntry{
			UserID:     userID,
			PeriodID:   periodID,
			IDNumber:   idNumber,
			Department: strings.TrimSpace(getCell(row, indexMap["department"])),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if idx, ok := indexMap["name"]; ok {
			entry.Name = strings.TrimSpace(getCell(row, idx))
		}
		if idx, ok := indexMap["title"]; ok {
			entry.Title = strings.TrimSpace(getCell(row, idx))
		}
		if idx, ok := indexMap["remarks"]; ok {
			entry.Remarks = strings.TrimSpace(getCell(row, idx))
		}
		if entry.Department == "" {
			continue
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, errors.New("花名册文件中没有找到有效的数据行，请检查文件格式和内容")
	}

	if err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("period_id = ?", periodID).Delete(&models.RosterEntry{}).Error; err != nil {
			return fmt.Errorf("cleanup roster: %w", err)
		}
		if err := tx.Create(&entries).Error; err != nil {
			return fmt.Errorf("insert roster: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &RosterParseResult{Imported: len(entries)}, nil
}

type ProcessOutput struct {
	PeriodID uint                    `json:"period_id"`
	Summary  []models.PeriodSummary  `json:"summary"`
	Personal []models.PersonalCharge `json:"personal"`
	Unit     []models.UnitCharge     `json:"unit"`
}

var requiredUploads = map[models.Part][]models.Scheme{
	models.PartPersonal: {
		models.SchemePension,
		models.SchemeMedical,
		models.SchemeSeriousIllness,
		models.SchemeUnemployment,
	},
	models.PartUnit: {
		models.SchemePension,
		models.SchemeMedical,
		models.SchemeSeriousIllness,
		models.SchemeUnemployment,
		models.SchemeInjury,
	},
}

func (p *Processor) ProcessPeriod(periodID uint) (*ProcessOutput, error) {
	var period models.Period
	if err := p.db.First(&period, periodID).Error; err != nil {
		return nil, fmt.Errorf("load period: %w", err)
	}

	var records []models.RawRecord
	if err := p.db.Where("period_id = ? AND file_type = ?", periodID, models.FileTypeNormal).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load raw records: %w", err)
	}
	if len(records) == 0 {
		return nil, errors.New("no raw records found for period")
	}

	if err := validateRequired(records); err != nil {
		return nil, err
	}

	var rosterEntries []models.RosterEntry
	if err := p.db.Where("period_id = ?", periodID).Find(&rosterEntries).Error; err != nil {
		return nil, fmt.Errorf("load roster entries: %w", err)
	}
	rosterMap := make(map[string]models.RosterEntry, len(rosterEntries))
	for _, entry := range rosterEntries {
		if entry.IDNumber == "" {
			continue
		}
		rosterMap[entry.IDNumber] = entry
	}

	result := buildAggregates(records, rosterMap)

	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("period_id = ?", periodID).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("cleanup summary: %w", err)
		}
		if err := tx.Where("period_id = ?", periodID).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("cleanup personal charges: %w", err)
		}
		if err := tx.Where("period_id = ?", periodID).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("cleanup unit charges: %w", err)
		}

		if err := tx.Create(&result.summaries).Error; err != nil {
			return fmt.Errorf("insert summaries: %w", err)
		}
		if err := tx.Create(&result.personalCharges).Error; err != nil {
			return fmt.Errorf("insert personal charges: %w", err)
		}
		if err := tx.Create(&result.unitCharges).Error; err != nil {
			return fmt.Errorf("insert unit charges: %w", err)
		}

		period.Status = "processed"
		period.UpdatedAt = time.Now()
		if err := tx.Save(&period).Error; err != nil {
			return fmt.Errorf("update period status: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ProcessOutput{
		PeriodID: periodID,
		Summary:  result.summaries,
		Personal: result.personalCharges,
		Unit:     result.unitCharges,
	}, nil
}

func validateRequired(records []models.RawRecord) error {
	found := map[models.Part]map[models.Scheme]bool{}
	for _, rec := range records {
		if _, ok := found[rec.Part]; !ok {
			found[rec.Part] = map[models.Scheme]bool{}
		}
		found[rec.Part][rec.Scheme] = true
	}

	for part, schemes := range requiredUploads {
		for _, scheme := range schemes {
			if !found[part][scheme] {
				return fmt.Errorf("missing required data for part=%s scheme=%s", part, scheme)
			}
		}
	}
	return nil
}

type aggregateResult struct {
	summaries       []models.PeriodSummary
	personalCharges []models.PersonalCharge
	unitCharges     []models.UnitCharge
}

type personAccumulator struct {
	Name         string
	IDNumber     string
	Department   string
	PersonalBase float64
	UnitBase     float64

	Personal struct {
		Pension        float64
		Medical        float64
		SeriousIllness float64
		Unemployment   float64
	}
	Unit struct {
		Pension        float64
		Medical        float64
		SeriousIllness float64
		Injury         float64
		Unemployment   float64
	}
}

func buildAggregates(records []models.RawRecord, roster map[string]models.RosterEntry) aggregateResult {
	now := time.Now()
	summaryMap := map[string]*models.PeriodSummary{}
	headcountSet := map[string]map[string]struct{}{}
	personMap := map[string]*personAccumulator{}

	// deterministic order by IDNumber
	for _, rec := range records {
		sumKey := fmt.Sprintf("%s_%s", rec.Scheme, rec.Part)
		if _, ok := summaryMap[sumKey]; !ok {
			summaryMap[sumKey] = &models.PeriodSummary{
				UserID:    rec.UserID,
				PeriodID:  rec.PeriodID,
				Scheme:    rec.Scheme,
				Part:      rec.Part,
				CreatedAt: now,
				UpdatedAt: now,
			}
		}
		sum := summaryMap[sumKey]
		sum.BaseTotal += rec.PayBase
		sum.AmountTotal += rec.AmountDue

		if _, ok := headcountSet[sumKey]; !ok {
			headcountSet[sumKey] = map[string]struct{}{}
		}
		headcountSet[sumKey][rec.IDNumber] = struct{}{}

		person, ok := personMap[rec.IDNumber]
		if !ok {
			var name, department string
			if entry, exists := roster[rec.IDNumber]; exists {
				name = entry.Name
				department = entry.Department
			}
			if name == "" {
				name = rec.Name
			}
			if department == "" {
				department = rec.Department
			}

			person = &personAccumulator{
				Name:       name,
				IDNumber:   rec.IDNumber,
				Department: department,
			}
			personMap[rec.IDNumber] = person
		}
		switch rec.Part {
		case models.PartPersonal:
			if person.PersonalBase == 0 {
				person.PersonalBase = rec.PayBase
			}
			switch rec.Scheme {
			case models.SchemePension:
				person.Personal.Pension += rec.AmountDue
			case models.SchemeMedical:
				person.Personal.Medical += rec.AmountDue
			case models.SchemeSeriousIllness:
				person.Personal.SeriousIllness += rec.AmountDue
			case models.SchemeUnemployment:
				person.Personal.Unemployment += rec.AmountDue
			}
		case models.PartUnit:
			if person.UnitBase == 0 {
				person.UnitBase = rec.PayBase
			}
			switch rec.Scheme {
			case models.SchemePension:
				person.Unit.Pension += rec.AmountDue
			case models.SchemeMedical:
				person.Unit.Medical += rec.AmountDue
			case models.SchemeSeriousIllness:
				person.Unit.SeriousIllness += rec.AmountDue
			case models.SchemeUnemployment:
				person.Unit.Unemployment += rec.AmountDue
			case models.SchemeInjury:
				person.Unit.Injury += rec.AmountDue
			}
		}
	}

	summaries := make([]models.PeriodSummary, 0, len(summaryMap))
	for _, value := range summaryMap {
		key := fmt.Sprintf("%s_%s", value.Scheme, value.Part)
		if set, ok := headcountSet[key]; ok {
			value.Headcount = len(set)
		}
		value.BaseTotal = round2(value.BaseTotal)
		value.AmountTotal = round2(value.AmountTotal)
		summaries = append(summaries, *value)
	}

	var personalCharges []models.PersonalCharge
	var unitCharges []models.UnitCharge
	for _, person := range personMap {
		personalSubtotal := round2(person.Personal.Pension + person.Personal.Medical + person.Personal.SeriousIllness + person.Personal.Unemployment)
		personalCharges = append(personalCharges, models.PersonalCharge{
			UserID:           records[0].UserID,
			PeriodID:         records[0].PeriodID,
			Name:             person.Name,
			IDNumber:         person.IDNumber,
			Department:       person.Department,
			Base:             round2(person.PersonalBase),
			Pension:          round2(person.Personal.Pension),
			MedicalMaternity: round2(person.Personal.Medical),
			SeriousIllness:   round2(person.Personal.SeriousIllness),
			Unemployment:     round2(person.Personal.Unemployment),
			Subtotal:         personalSubtotal,
			CreatedAt:        now,
			UpdatedAt:        now,
		})

		unitMedicalTotal := person.Unit.Medical + person.Unit.SeriousIllness
		unitSubtotal := round2(person.Unit.Pension + unitMedicalTotal + person.Unit.Injury + person.Unit.Unemployment)
		unitCharges = append(unitCharges, models.UnitCharge{
			UserID:           records[0].UserID,
			PeriodID:         records[0].PeriodID,
			Name:             person.Name,
			IDNumber:         person.IDNumber,
			Department:       person.Department,
			Base:             round2(maxFloat(person.UnitBase, person.PersonalBase)), // fallback to personal base if unit missing
			Pension:          round2(person.Unit.Pension),
			MedicalMaternity: round2(unitMedicalTotal),
			SeriousIllness:   round2(person.Unit.SeriousIllness),
			Injury:           round2(person.Unit.Injury),
			Unemployment:     round2(person.Unit.Unemployment),
			Subtotal:         unitSubtotal,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Part == summaries[j].Part {
			return summaries[i].Scheme < summaries[j].Scheme
		}
		return summaries[i].Part < summaries[j].Part
	})
	sort.Slice(personalCharges, func(i, j int) bool {
		return personalCharges[i].IDNumber < personalCharges[j].IDNumber
	})
	sort.Slice(unitCharges, func(i, j int) bool {
		return unitCharges[i].IDNumber < unitCharges[j].IDNumber
	})

	return aggregateResult{
		summaries:       summaries,
		personalCharges: personalCharges,
		unitCharges:     unitCharges,
	}
}

func getCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func toInt(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return 0
	}
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return int(f)
	}
	return 0
}

func toFloat(value string) float64 {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.TrimSuffix(value, "%")
	if value == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return 0
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}

func maxFloat(a, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}

// ProcessAdjustments 处理补退数据，将其累加到现有的扣款明细中
func (p *Processor) ProcessAdjustments(periodID uint) (*ProcessOutput, error) {
	var period models.Period
	if err := p.db.First(&period, periodID).Error; err != nil {
		return nil, fmt.Errorf("load period: %w", err)
	}

	// 检查是否有补退数据
	var adjustmentRecords []models.RawRecord
	if err := p.db.Where("period_id = ? AND file_type = ?", periodID, models.FileTypeAdjustment).Find(&adjustmentRecords).Error; err != nil {
		return nil, fmt.Errorf("load adjustment records: %w", err)
	}
	if len(adjustmentRecords) == 0 {
		return nil, errors.New("no adjustment records found for period")
	}

	// 获取花名册数据
	var rosterEntries []models.RosterEntry
	if err := p.db.Where("period_id = ?", periodID).Find(&rosterEntries).Error; err != nil {
		return nil, fmt.Errorf("load roster entries: %w", err)
	}
	rosterMap := make(map[string]models.RosterEntry, len(rosterEntries))
	for _, entry := range rosterEntries {
		if entry.IDNumber == "" {
			continue
		}
		rosterMap[entry.IDNumber] = entry
	}

	// 构建补退数据的聚合结果
	adjustmentResult := buildAdjustments(adjustmentRecords, rosterMap)

	// 获取现有的扣款明细
	var existingPersonal []models.PersonalCharge
	var existingUnit []models.UnitCharge

	if err := p.db.Where("period_id = ?", periodID).Find(&existingPersonal).Error; err != nil {
		return nil, fmt.Errorf("load existing personal charges: %w", err)
	}
	if err := p.db.Where("period_id = ?", periodID).Find(&existingUnit).Error; err != nil {
		return nil, fmt.Errorf("load existing unit charges: %w", err)
	}

	// 为补退数据标记为补退记录
	for i := range adjustmentResult.personalCharges {
		adjustmentResult.personalCharges[i].IsAdjustment = true
	}
	for i := range adjustmentResult.unitCharges {
		adjustmentResult.unitCharges[i].IsAdjustment = true
	}

	// 在事务中插入补退数据
	err := p.db.Transaction(func(tx *gorm.DB) error {
		// 删除已存在的补退记录（如果有的话）
		if err := tx.Where("period_id = ? AND is_adjustment = ?", periodID, true).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("cleanup existing adjustment personal charges: %w", err)
		}
		if err := tx.Where("period_id = ? AND is_adjustment = ?", periodID, true).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("cleanup existing adjustment unit charges: %w", err)
		}

		// 插入新的补退数据
		if len(adjustmentResult.personalCharges) > 0 {
			if err := tx.Create(&adjustmentResult.personalCharges).Error; err != nil {
				return fmt.Errorf("insert adjustment personal charges: %w", err)
			}
		}
		if len(adjustmentResult.unitCharges) > 0 {
			if err := tx.Create(&adjustmentResult.unitCharges).Error; err != nil {
				return fmt.Errorf("insert adjustment unit charges: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 为补退数据创建汇总记录
	adjustmentSummaryResult := buildSummaryFromRecords(adjustmentRecords)

	// 为补退汇总数据标记为补退记录
	for i := range adjustmentSummaryResult {
		adjustmentSummaryResult[i].IsAdjustment = true
	}

	err = p.db.Transaction(func(tx *gorm.DB) error {
		// 删除已存在的补退汇总记录
		if err := tx.Where("period_id = ? AND is_adjustment = ?", periodID, true).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("cleanup adjustment summary: %w", err)
		}

		// 插入新的补退汇总数据
		if len(adjustmentSummaryResult) > 0 {
			if err := tx.Create(&adjustmentSummaryResult).Error; err != nil {
				return fmt.Errorf("insert adjustment summaries: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 获取所有记录（正常记录+补退记录）
	var allPersonalCharges []models.PersonalCharge
	var allUnitCharges []models.UnitCharge
	var allSummaries []models.PeriodSummary

	if err := p.db.Where("period_id = ?", periodID).Find(&allPersonalCharges).Error; err != nil {
		return nil, fmt.Errorf("load all personal charges: %w", err)
	}
	if err := p.db.Where("period_id = ?", periodID).Find(&allUnitCharges).Error; err != nil {
		return nil, fmt.Errorf("load all unit charges: %w", err)
	}
	if err := p.db.Where("period_id = ?", periodID).Find(&allSummaries).Error; err != nil {
		return nil, fmt.Errorf("load all summaries: %w", err)
	}

	return &ProcessOutput{
		PeriodID: periodID,
		Summary:  allSummaries,
		Personal: allPersonalCharges,
		Unit:     allUnitCharges,
	}, nil
}

// buildAdjustments 构建补退数据的聚合结果（类似buildAggregates，但只处理补退数据）
func buildAdjustments(records []models.RawRecord, roster map[string]models.RosterEntry) aggregateResult {
	now := time.Now()
	personMap := map[string]*personAccumulator{}

	for _, rec := range records {
		person, ok := personMap[rec.IDNumber]
		if !ok {
			var name, department string
			if entry, exists := roster[rec.IDNumber]; exists {
				name = entry.Name
				department = entry.Department
			}
			if name == "" {
				name = rec.Name
			}
			if department == "" {
				department = rec.Department
			}

			person = &personAccumulator{
				Name:       name,
				IDNumber:   rec.IDNumber,
				Department: department,
			}
			personMap[rec.IDNumber] = person
		}

		switch rec.Part {
		case models.PartPersonal:
			if person.PersonalBase == 0 {
				person.PersonalBase = rec.PayBase
			}
			switch rec.Scheme {
			case models.SchemePension:
				person.Personal.Pension += rec.AmountDue
			case models.SchemeMedical:
				person.Personal.Medical += rec.AmountDue
			case models.SchemeSeriousIllness:
				person.Personal.SeriousIllness += rec.AmountDue
			case models.SchemeUnemployment:
				person.Personal.Unemployment += rec.AmountDue
			}
		case models.PartUnit:
			if person.UnitBase == 0 {
				person.UnitBase = rec.PayBase
			}
			switch rec.Scheme {
			case models.SchemePension:
				person.Unit.Pension += rec.AmountDue
			case models.SchemeMedical:
				person.Unit.Medical += rec.AmountDue
			case models.SchemeSeriousIllness:
				person.Unit.SeriousIllness += rec.AmountDue
			case models.SchemeUnemployment:
				person.Unit.Unemployment += rec.AmountDue
			case models.SchemeInjury:
				person.Unit.Injury += rec.AmountDue
			}
		}
	}

	var personalCharges []models.PersonalCharge
	var unitCharges []models.UnitCharge
	for _, person := range personMap {
		personalSubtotal := round2(person.Personal.Pension + person.Personal.Medical + person.Personal.SeriousIllness + person.Personal.Unemployment)
		personalCharges = append(personalCharges, models.PersonalCharge{
			UserID:           records[0].UserID,
			PeriodID:         records[0].PeriodID,
			Name:             person.Name,
			IDNumber:         person.IDNumber,
			Department:       person.Department,
			Base:             round2(person.PersonalBase),
			Pension:          round2(person.Personal.Pension),
			MedicalMaternity: round2(person.Personal.Medical),
			SeriousIllness:   round2(person.Personal.SeriousIllness),
			Unemployment:     round2(person.Personal.Unemployment),
			Subtotal:         personalSubtotal,
			CreatedAt:        now,
			UpdatedAt:        now,
		})

		unitMedicalTotal := person.Unit.Medical + person.Unit.SeriousIllness
		unitSubtotal := round2(person.Unit.Pension + unitMedicalTotal + person.Unit.Injury + person.Unit.Unemployment)
		unitCharges = append(unitCharges, models.UnitCharge{
			PeriodID:         records[0].PeriodID,
			Name:             person.Name,
			IDNumber:         person.IDNumber,
			Department:       person.Department,
			Base:             round2(maxFloat(person.UnitBase, person.PersonalBase)),
			Pension:          round2(person.Unit.Pension),
			MedicalMaternity: round2(unitMedicalTotal),
			SeriousIllness:   round2(person.Unit.SeriousIllness),
			Injury:           round2(person.Unit.Injury),
			Unemployment:     round2(person.Unit.Unemployment),
			Subtotal:         unitSubtotal,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	return aggregateResult{
		summaries:       []models.PeriodSummary{}, // 汇总会在最后重新计算
		personalCharges: personalCharges,
		unitCharges:     unitCharges,
	}
}

// mergePersonalCharges 合并现有个人扣款明细和补退明细
func mergePersonalCharges(existing []models.PersonalCharge, adjustments []models.PersonalCharge) []models.PersonalCharge {
	existingMap := make(map[string]*models.PersonalCharge)
	for i := range existing {
		existingMap[existing[i].IDNumber] = &existing[i]
	}

	now := time.Now()
	for _, adj := range adjustments {
		if existingCharge, ok := existingMap[adj.IDNumber]; ok {
			// 累加到现有记录
			existingCharge.Base += adj.Base
			existingCharge.Pension += adj.Pension
			existingCharge.MedicalMaternity += adj.MedicalMaternity
			existingCharge.SeriousIllness += adj.SeriousIllness
			existingCharge.Unemployment += adj.Unemployment
			existingCharge.Subtotal = round2(existingCharge.Pension + existingCharge.MedicalMaternity + existingCharge.SeriousIllness + existingCharge.Unemployment)
			existingCharge.UpdatedAt = now

			// 如果补退数据有部门信息而现有数据没有，则更新部门信息
			if existingCharge.Department == "" && adj.Department != "" {
				existingCharge.Department = adj.Department
			}
			if existingCharge.Name == "" && adj.Name != "" {
				existingCharge.Name = adj.Name
			}
		} else {
			// 新增记录
			adj.CreatedAt = now
			adj.UpdatedAt = now
			existing = append(existing, adj)
		}
	}

	// 重新排序
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].IDNumber < existing[j].IDNumber
	})

	return existing
}

// mergeUnitCharges 合并现有单位扣款明细和补退明细
func mergeUnitCharges(existing []models.UnitCharge, adjustments []models.UnitCharge) []models.UnitCharge {
	existingMap := make(map[string]*models.UnitCharge)
	for i := range existing {
		existingMap[existing[i].IDNumber] = &existing[i]
	}

	now := time.Now()
	for _, adj := range adjustments {
		if existingCharge, ok := existingMap[adj.IDNumber]; ok {
			// 累加到现有记录
			existingCharge.Base += adj.Base
			existingCharge.Pension += adj.Pension
			existingCharge.MedicalMaternity += adj.MedicalMaternity
			existingCharge.SeriousIllness += adj.SeriousIllness
			existingCharge.Injury += adj.Injury
			existingCharge.Unemployment += adj.Unemployment
			existingCharge.Subtotal = round2(existingCharge.Pension + existingCharge.MedicalMaternity + existingCharge.Injury + existingCharge.Unemployment)
			existingCharge.UpdatedAt = now

			// 如果补退数据有部门信息而现有数据没有，则更新部门信息
			if existingCharge.Department == "" && adj.Department != "" {
				existingCharge.Department = adj.Department
			}
			if existingCharge.Name == "" && adj.Name != "" {
				existingCharge.Name = adj.Name
			}
		} else {
			// 新增记录
			adj.CreatedAt = now
			adj.UpdatedAt = now
			existing = append(existing, adj)
		}
	}

	// 重新排序
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].IDNumber < existing[j].IDNumber
	})

	return existing
}

// buildSummaryFromRecords 从记录构建汇总数据
func buildSummaryFromRecords(records []models.RawRecord) []models.PeriodSummary {
	now := time.Now()
	summaryMap := map[string]*models.PeriodSummary{}
	headcountSet := map[string]map[string]struct{}{}

	for _, rec := range records {
		sumKey := fmt.Sprintf("%s_%s", rec.Scheme, rec.Part)
		if _, ok := summaryMap[sumKey]; !ok {
			summaryMap[sumKey] = &models.PeriodSummary{
				UserID:    rec.UserID,
				PeriodID:  rec.PeriodID,
				Scheme:    rec.Scheme,
				Part:      rec.Part,
				CreatedAt: now,
				UpdatedAt: now,
			}
		}
		sum := summaryMap[sumKey]
		sum.BaseTotal += rec.PayBase
		sum.AmountTotal += rec.AmountDue

		if _, ok := headcountSet[sumKey]; !ok {
			headcountSet[sumKey] = map[string]struct{}{}
		}
		headcountSet[sumKey][rec.IDNumber] = struct{}{}
	}

	summaries := make([]models.PeriodSummary, 0, len(summaryMap))
	for _, value := range summaryMap {
		key := fmt.Sprintf("%s_%s", value.Scheme, value.Part)
		if set, ok := headcountSet[key]; ok {
			value.Headcount = len(set)
		}
		value.BaseTotal = round2(value.BaseTotal)
		value.AmountTotal = round2(value.AmountTotal)
		summaries = append(summaries, *value)
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Part == summaries[j].Part {
			return summaries[i].Scheme < summaries[j].Scheme
		}
		return summaries[i].Part < summaries[j].Part
	})

	return summaries
}
