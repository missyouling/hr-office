package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

type Handler struct {
	db      *gorm.DB
	process *service.Processor
}

type batchUploadItem struct {
	FileName     string        `json:"file_name"`
	OriginalName string        `json:"original_name"`
	Scheme       models.Scheme `json:"scheme"`
	Part         models.Part   `json:"part"`
	Imported     int           `json:"imported"`
	Error        string        `json:"error,omitempty"`
}

// fileKey 用于标识文件的唯一性 (文件名 + 大小)
type fileKey struct {
	name string
	size int64
}

// deduplicateFiles 去除重复文件 (相同文件名和大小的文件只保留第一个)
func deduplicateFiles(files []*multipart.FileHeader) []*multipart.FileHeader {
	seen := make(map[fileKey]bool)
	var unique []*multipart.FileHeader

	for _, file := range files {
		key := fileKey{
			name: file.Filename,
			size: file.Size,
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, file)
		}
	}

	return unique
}

// deduplicateFilesWithMetadata 去除重复文件并保持相关元数据的同步
func deduplicateFilesWithMetadata(files []*multipart.FileHeader, schemes []string, parts []string) ([]*multipart.FileHeader, []string, []string) {
	seen := make(map[fileKey]bool)
	var uniqueFiles []*multipart.FileHeader
	var uniqueSchemes []string
	var uniqueParts []string

	for i, file := range files {
		key := fileKey{
			name: file.Filename,
			size: file.Size,
		}

		if !seen[key] {
			seen[key] = true
			uniqueFiles = append(uniqueFiles, file)
			if i < len(schemes) {
				uniqueSchemes = append(uniqueSchemes, schemes[i])
			}
			if i < len(parts) {
				uniqueParts = append(uniqueParts, parts[i])
			}
		}
	}

	return uniqueFiles, uniqueSchemes, uniqueParts
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:      db,
		process: service.NewProcessor(db),
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/periods", h.listPeriods)
	r.Post("/periods", h.createPeriod)
	r.Get("/roster-template", h.downloadRosterTemplate)

	r.Route("/periods/{periodID}", func(pr chi.Router) {
		pr.Get("/", h.getPeriod)
		pr.Delete("/", h.deletePeriod)
		pr.Post("/reset", h.resetPeriod)
		pr.Get("/files", h.listFiles)
		pr.Post("/files", h.uploadFile)
		pr.Post("/files/batch", h.uploadFilesBatch)
		pr.Post("/files/clear", h.clearFiles)
		pr.Get("/roster", h.getRoster)
		pr.Post("/roster", h.uploadRoster)
		pr.Post("/roster/import", h.importLatestRoster)
		pr.Post("/process", h.processPeriod)
		pr.Get("/summary", h.getSummary)
		pr.Get("/charges", h.getCharges)
		pr.Get("/charges/export", h.exportChargesExcel)
		pr.Get("/charges/scheme", h.getSchemeCharges)
		pr.Get("/charges/scheme/export", h.exportSchemeChargesExcel)

		// 补退功能
		pr.Post("/adjustments/batch", h.uploadAdjustmentsBatch)
		pr.Post("/adjustments/process", h.processAdjustments)
		pr.Post("/adjustments/clear", h.clearAdjustments)
	})
}

func (h *Handler) listPeriods(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var periods []models.Period
	if err := h.db.Where("user_id = ?", userID).Order("year_month DESC").Find(&periods).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list periods", err)
		return
	}
	respondJSON(w, http.StatusOK, periods)
}

type createPeriodRequest struct {
	YearMonth string `json:"year_month"`
}

func (h *Handler) createPeriod(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req createPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}
	req.YearMonth = strings.TrimSpace(req.YearMonth)
	if req.YearMonth == "" {
		respondError(w, http.StatusBadRequest, "year_month is required", nil)
		return
	}

	period := models.Period{
		UserID:    userID,
		YearMonth: req.YearMonth,
		Status:    "draft",
	}
	// Check for existing period for this user
	if err := h.db.FirstOrCreate(&period, models.Period{UserID: userID, YearMonth: req.YearMonth}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create period", err)
		return
	}
	respondJSON(w, http.StatusCreated, period)
}

func (h *Handler) getPeriod(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}
	respondJSON(w, http.StatusOK, period)
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	var files []models.SourceFile
	if err := h.db.Where("period_id = ?", period.ID).Order("created_at ASC").Find(&files).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch files", err)
		return
	}
	respondJSON(w, http.StatusOK, files)
}

func (h *Handler) getRoster(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	var roster []models.RosterEntry
	if err := h.db.Where("period_id = ?", period.ID).
		Order("id_number ASC").
		Find(&roster).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch roster", err)
		return
	}
	respondJSON(w, http.StatusOK, roster)
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	schemeStr := strings.TrimSpace(r.FormValue("scheme"))
	partStr := strings.TrimSpace(r.FormValue("part"))
	if schemeStr == "" || partStr == "" {
		respondError(w, http.StatusBadRequest, "scheme and part are required", nil)
		return
	}
	scheme := models.Scheme(schemeStr)
	part := models.Part(partStr)
	if !isValidScheme(scheme) || !isValidPart(part) {
		respondError(w, http.StatusBadRequest, "invalid scheme or part", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required", err)
		return
	}
	defer file.Close()

	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}

	targetDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create period upload dir", err)
		return
	}

	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
	storedPath := filepath.Join(targetDir, filename)

	out, err := os.Create(storedPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create file", err)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save file", err)
		return
	}

	result, err := h.process.ParseSourceFile(period.ID, storedPath, header.Filename, scheme, part)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse file", err)
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

func (h *Handler) uploadFilesBatch(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	if err := r.ParseMultipartForm(128 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	form := r.MultipartForm
	originalFiles := form.File["files"]
	originalSchemes := form.Value["scheme"]
	originalParts := form.Value["part"]

	if len(originalFiles) == 0 {
		respondError(w, http.StatusBadRequest, "files field is required", nil)
		return
	}
	if len(originalSchemes) != len(originalFiles) || len(originalParts) != len(originalFiles) {
		respondError(w, http.StatusBadRequest, "scheme and part count must match files count", nil)
		return
	}

	// 去除重复文件 (相同文件名和大小)
	files, schemes, parts := deduplicateFilesWithMetadata(originalFiles, originalSchemes, originalParts)

	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}
	targetDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create period upload dir", err)
		return
	}

	items := make([]batchUploadItem, 0, len(files))
	for idx, header := range files {
		item := batchUploadItem{
			OriginalName: header.Filename,
		}

		scheme := models.Scheme(strings.TrimSpace(schemes[idx]))
		part := models.Part(strings.TrimSpace(parts[idx]))
		item.Scheme = scheme
		item.Part = part

		if !isValidScheme(scheme) || !isValidPart(part) {
			item.Error = "invalid scheme or part"
			items = append(items, item)
			continue
		}

		src, err := header.Open()
		if err != nil {
			item.Error = fmt.Sprintf("open file: %v", err)
			items = append(items, item)
			continue
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".xlsx"
		}
		filename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
		storedPath := filepath.Join(targetDir, filename)

		out, err := os.Create(storedPath)
		if err != nil {
			item.Error = fmt.Sprintf("create file: %v", err)
			_ = src.Close()
			items = append(items, item)
			continue
		}

		if _, err := io.Copy(out, src); err != nil {
			item.Error = fmt.Sprintf("save file: %v", err)
			_ = src.Close()
			_ = out.Close()
			items = append(items, item)
			continue
		}
		_ = src.Close()
		_ = out.Close()

		result, err := h.process.ParseSourceFile(period.ID, storedPath, header.Filename, scheme, part)
		if err != nil {
			item.Error = err.Error()
			items = append(items, item)
			continue
		}

		item.FileName = filepath.Base(result.File.StoredPath)
		item.Imported = result.Imported
		items = append(items, item)
	}

	status := http.StatusOK
	for _, item := range items {
		if item.Error != "" {
			status = http.StatusMultiStatus
			break
		}
	}

	respondJSON(w, status, map[string]any{
		"items": items,
	})
}

func (h *Handler) uploadRoster(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required", err)
		return
	}
	defer file.Close()

	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}

	targetDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID), "roster")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create roster directory", err)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".xlsx"
	}
	filename := fmt.Sprintf("roster-%s%s", uuid.NewString(), ext)
	storedPath := filepath.Join(targetDir, filename)

	out, err := os.Create(storedPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create file", err)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save file", err)
		return
	}

	result, err := h.process.ParseRosterFile(period.ID, storedPath, header.Filename)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to import roster", err)
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

func (h *Handler) importLatestRoster(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	// 获取最新的花名册数据（从其他账期中找到最新的非空花名册）
	var latestRoster []models.RosterEntry
	err = h.db.Raw(`
		SELECT re.* FROM roster_entries re
		INNER JOIN (
			SELECT period_id, MAX(created_at) as max_created
			FROM roster_entries
			WHERE period_id != ?
			GROUP BY period_id
			ORDER BY max_created DESC
			LIMIT 1
		) latest ON re.period_id = latest.period_id
		ORDER BY re.id_number
	`, period.ID).Find(&latestRoster).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch latest roster", err)
		return
	}

	if len(latestRoster) == 0 {
		respondError(w, http.StatusNotFound, "no existing roster data found", nil)
		return
	}

	// 复制花名册数据到当前账期
	var newEntries []models.RosterEntry
	for _, entry := range latestRoster {
		newEntry := models.RosterEntry{
			PeriodID:   period.ID,
			Name:       entry.Name,
			IDNumber:   entry.IDNumber,
			Department: entry.Department,
			Title:      entry.Title,
			Remarks:    entry.Remarks,
		}
		newEntries = append(newEntries, newEntry)
	}

	// 在事务中删除现有数据并插入新数据
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 删除当前账期的花名册数据
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.RosterEntry{}).Error; err != nil {
			return fmt.Errorf("cleanup existing roster: %w", err)
		}

		// 插入新的花名册数据
		if err := tx.Create(&newEntries).Error; err != nil {
			return fmt.Errorf("insert new roster entries: %w", err)
		}

		return nil
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to import roster data", err)
		return
	}

	result := map[string]interface{}{
		"imported": len(newEntries),
		"message":  fmt.Sprintf("成功导入 %d 条花名册记录", len(newEntries)),
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) processPeriod(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	output, err := h.process.ProcessPeriod(period.ID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to process period", err)
		return
	}

	respondJSON(w, http.StatusOK, output)
}

func (h *Handler) getSummary(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	var summaries []models.PeriodSummary
	if err := h.db.Where("period_id = ?", period.ID).Order("scheme, part").Find(&summaries).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch summary", err)
		return
	}
	respondJSON(w, http.StatusOK, summaries)
}

func (h *Handler) getCharges(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	partStr := strings.TrimSpace(r.URL.Query().Get("part"))
	if partStr == "" {
		respondError(w, http.StatusBadRequest, "part query parameter is required", nil)
		return
	}
	part := models.Part(partStr)
	if !isValidPart(part) {
		respondError(w, http.StatusBadRequest, "invalid part value", nil)
		return
	}

	if part == models.PartPersonal {
		var charges []models.PersonalCharge
		if err := h.db.Where("period_id = ?", period.ID).Order("id_number ASC").Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch personal charges", err)
			return
		}
		respondJSON(w, http.StatusOK, charges)
		return
	}

	var charges []models.UnitCharge
	if err := h.db.Where("period_id = ?", period.ID).Order("id_number ASC").Find(&charges).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch unit charges", err)
		return
	}
	respondJSON(w, http.StatusOK, charges)
}

func (h *Handler) exportChargesExcel(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	partStr := strings.TrimSpace(r.URL.Query().Get("part"))
	if partStr == "" {
		respondError(w, http.StatusBadRequest, "part query parameter is required", nil)
		return
	}
	part := models.Part(partStr)
	if !isValidPart(part) {
		respondError(w, http.StatusBadRequest, "invalid part value", nil)
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheetName := f.GetSheetName(0)

	var (
		headers []string
		label   string
	)

	if part == models.PartPersonal {
		headers = []string{"序号", "姓名", "证件号码", "部门", "基数", "养老保险", "医疗+生育保险", "大额医疗", "失业保险", "小计"}
		label = "个人"
	} else {
		headers = []string{"序号", "姓名", "证件号码", "部门", "基数", "养老保险", "医疗+生育保险", "工伤保险", "失业保险", "小计"}
		label = "单位"
	}

	for idx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to write header", err)
			return
		}
	}

	if part == models.PartPersonal {
		var charges []models.PersonalCharge
		if err := h.db.Where("period_id = ?", period.ID).
			Order("id_number ASC").
			Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch personal charges", err)
			return
		}
		if len(charges) == 0 {
			respondError(w, http.StatusBadRequest, "no personal charges available", nil)
			return
		}

		var (
			baseTotal         float64
			pensionTotal      float64
			medicalTotal      float64
			seriousTotal      float64
			unemploymentTotal float64
			subtotalTotal     float64
		)

		for idx, row := range charges {
			baseTotal += row.Base
			pensionTotal += row.Pension
			medicalTotal += row.MedicalMaternity
			seriousTotal += row.SeriousIllness
			unemploymentTotal += row.Unemployment
			subtotalTotal += row.Subtotal

			values := []any{
				idx + 1,
				row.Name,
				row.IDNumber,
				row.Department,
				row.Base,
				row.Pension,
				row.MedicalMaternity,
				row.SeriousIllness,
				row.Unemployment,
				row.Subtotal,
			}
			for colIdx, value := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
				if err := f.SetCellValue(sheetName, cell, value); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to write data", err)
					return
				}
			}
		}

		totalRow := len(charges) + 2
		totalValues := []any{
			"合计",
			"",
			"",
			"",
			baseTotal,
			pensionTotal,
			medicalTotal,
			seriousTotal,
			unemploymentTotal,
			subtotalTotal,
		}
		for colIdx, value := range totalValues {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, totalRow)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write total row", err)
				return
			}
		}
	} else {
		var charges []models.UnitCharge
		if err := h.db.Where("period_id = ?", period.ID).
			Order("id_number ASC").
			Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch unit charges", err)
			return
		}
		if len(charges) == 0 {
			respondError(w, http.StatusBadRequest, "no unit charges available", nil)
			return
		}

		var (
			baseTotal         float64
			pensionTotal      float64
			medicalTotal      float64
			injuryTotal       float64
			unemploymentTotal float64
			subtotalTotal     float64
		)

		for idx, row := range charges {
			baseTotal += row.Base
			pensionTotal += row.Pension
			medicalTotal += row.MedicalMaternity
			injuryTotal += row.Injury
			unemploymentTotal += row.Unemployment
			subtotalTotal += row.Subtotal

			values := []any{
				idx + 1,
				row.Name,
				row.IDNumber,
				row.Department,
				row.Base,
				row.Pension,
				row.MedicalMaternity,
				row.Injury,
				row.Unemployment,
				row.Subtotal,
			}
			for colIdx, value := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
				if err := f.SetCellValue(sheetName, cell, value); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to write data", err)
					return
				}
			}
		}

		totalRow := len(charges) + 2
		totalValues := []any{
			"合计",
			"",
			"",
			"",
			baseTotal,
			pensionTotal,
			medicalTotal,
			injuryTotal,
			unemploymentTotal,
			subtotalTotal,
		}
		for colIdx, value := range totalValues {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, totalRow)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write total row", err)
				return
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to encode excel", err)
		return
	}

	filename := fmt.Sprintf("%s-%s扣款明细.xlsx", period.YearMonth, label)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(buf.Bytes()))
}
func (h *Handler) getPeriodByParam(r *http.Request) (*models.Period, error) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	param := chi.URLParam(r, "periodID")
	if param == "" {
		return nil, fmt.Errorf("missing periodID")
	}
	id, err := strconv.Atoi(param)
	if err != nil {
		return nil, fmt.Errorf("invalid periodID: %w", err)
	}
	var period models.Period
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&period).Error; err != nil {
		return nil, err
	}
	return &period, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	resp := map[string]any{
		"error":   message,
		"details": "",
	}
	if err != nil {
		resp["details"] = err.Error()
	}
	respondJSON(w, status, resp)
}

func isValidScheme(scheme models.Scheme) bool {
	switch scheme {
	case models.SchemePension,
		models.SchemeMedical,
		models.SchemeSeriousIllness,
		models.SchemeUnemployment,
		models.SchemeInjury:
		return true
	default:
		return false
	}
}

func isValidPart(part models.Part) bool {
	switch part {
	case models.PartPersonal, models.PartUnit:
		return true
	default:
		return false
	}
}

func (h *Handler) downloadRosterTemplate(w http.ResponseWriter, r *http.Request) {
	templatePath := service.GetRosterTemplatePath()

	// Check if template file exists, if not, generate it
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := service.GenerateRosterTemplate(templatePath); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate template", err)
			return
		}
	}

	// Read the template file
	file, err := os.Open(templatePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to open template file", err)
		return
	}
	defer file.Close()

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"花名册模板.xlsx\"")

	// Copy file content to response
	if _, err := io.Copy(w, file); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to send template file", err)
		return
	}
}

type SchemeChargeDetail struct {
	Name       string  `json:"name"`
	IDNumber   string  `json:"id_number"`
	Department string  `json:"department"`
	Base       float64 `json:"base"`
	Amount     float64 `json:"amount"`
}

func (h *Handler) getSchemeCharges(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	schemeStr := strings.TrimSpace(r.URL.Query().Get("scheme"))
	partStr := strings.TrimSpace(r.URL.Query().Get("part"))
	if schemeStr == "" || partStr == "" {
		respondError(w, http.StatusBadRequest, "scheme and part query parameters are required", nil)
		return
	}

	// 添加 is_adjustment 参数支持
	isAdjustmentStr := strings.TrimSpace(r.URL.Query().Get("is_adjustment"))
	var isAdjustment *bool
	if isAdjustmentStr != "" {
		if isAdjustmentStr == "true" {
			val := true
			isAdjustment = &val
		} else if isAdjustmentStr == "false" {
			val := false
			isAdjustment = &val
		} else {
			respondError(w, http.StatusBadRequest, "invalid is_adjustment value, must be 'true' or 'false'", nil)
			return
		}
	}

	scheme := models.Scheme(schemeStr)
	part := models.Part(partStr)

	if !isValidScheme(scheme) || !isValidPart(part) {
		respondError(w, http.StatusBadRequest, "invalid scheme or part value", nil)
		return
	}

	var details []SchemeChargeDetail

	if part == models.PartPersonal {
		var charges []models.PersonalCharge
		query := h.db.Where("period_id = ?", period.ID)
		if isAdjustment != nil {
			query = query.Where("is_adjustment = ?", *isAdjustment)
		}
		if err := query.Order("id_number ASC").Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch personal charges", err)
			return
		}


		for _, charge := range charges {
			var amount float64
			switch scheme {
			case models.SchemePension:
				amount = charge.Pension
			case models.SchemeMedical:
				amount = charge.MedicalMaternity
			case models.SchemeSeriousIllness:
				amount = charge.SeriousIllness
			case models.SchemeUnemployment:
				amount = charge.Unemployment
			default:
				continue
			}

			details = append(details, SchemeChargeDetail{
				Name:       charge.Name,
				IDNumber:   charge.IDNumber,
				Department: charge.Department,
				Base:       charge.Base,
				Amount:     amount,
			})
		}
	} else {
		var charges []models.UnitCharge
		query := h.db.Where("period_id = ?", period.ID)
		if isAdjustment != nil {
			query = query.Where("is_adjustment = ?", *isAdjustment)
		}
		if err := query.Order("id_number ASC").Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch unit charges", err)
			return
		}


		for _, charge := range charges {
			var amount float64
			switch scheme {
			case models.SchemePension:
				amount = charge.Pension
			case models.SchemeMedical:
				amount = charge.MedicalMaternity
			case models.SchemeSeriousIllness:
				amount = charge.SeriousIllness
			case models.SchemeUnemployment:
				amount = charge.Unemployment
			case models.SchemeInjury:
				amount = charge.Injury
			default:
				continue
			}

			details = append(details, SchemeChargeDetail{
				Name:       charge.Name,
				IDNumber:   charge.IDNumber,
				Department: charge.Department,
				Base:       charge.Base,
				Amount:     amount,
			})
		}
	}

	respondJSON(w, http.StatusOK, details)
}

func (h *Handler) exportSchemeChargesExcel(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	schemeStr := strings.TrimSpace(r.URL.Query().Get("scheme"))
	partStr := strings.TrimSpace(r.URL.Query().Get("part"))
	if schemeStr == "" || partStr == "" {
		respondError(w, http.StatusBadRequest, "scheme and part query parameters are required", nil)
		return
	}

	scheme := models.Scheme(schemeStr)
	part := models.Part(partStr)
	if !isValidScheme(scheme) || !isValidPart(part) {
		respondError(w, http.StatusBadRequest, "invalid scheme or part value", nil)
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheetName := f.GetSheetName(0)

	headers := []string{"序号", "姓名", "证件号码", "部门", "缴费基数", "应缴金额"}

	for idx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to write header", err)
			return
		}
	}

	var details []SchemeChargeDetail

	if part == models.PartPersonal {
		var charges []models.PersonalCharge
		if err := h.db.Where("period_id = ?", period.ID).Order("id_number ASC").Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch personal charges", err)
			return
		}

		for _, charge := range charges {
			var amount float64
			switch scheme {
			case models.SchemePension:
				amount = charge.Pension
			case models.SchemeMedical:
				amount = charge.MedicalMaternity
			case models.SchemeSeriousIllness:
				amount = charge.SeriousIllness
			case models.SchemeUnemployment:
				amount = charge.Unemployment
			default:
				continue
			}

			if amount > 0 {
				details = append(details, SchemeChargeDetail{
					Name:       charge.Name,
					IDNumber:   charge.IDNumber,
					Department: charge.Department,
					Base:       charge.Base,
					Amount:     amount,
				})
			}
		}
	} else {
		var charges []models.UnitCharge
		if err := h.db.Where("period_id = ?", period.ID).Order("id_number ASC").Find(&charges).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch unit charges", err)
			return
		}

		for _, charge := range charges {
			var amount float64
			switch scheme {
			case models.SchemePension:
				amount = charge.Pension
			case models.SchemeMedical:
				amount = charge.MedicalMaternity
			case models.SchemeSeriousIllness:
				amount = charge.SeriousIllness
			case models.SchemeUnemployment:
				amount = charge.Unemployment
			case models.SchemeInjury:
				amount = charge.Injury
			default:
				continue
			}

			if amount > 0 {
				details = append(details, SchemeChargeDetail{
					Name:       charge.Name,
					IDNumber:   charge.IDNumber,
					Department: charge.Department,
					Base:       charge.Base,
					Amount:     amount,
				})
			}
		}
	}

	if len(details) == 0 {
		respondError(w, http.StatusBadRequest, "no data available for this scheme and part", nil)
		return
	}

	var (
		baseTotal   float64
		amountTotal float64
	)

	for idx, detail := range details {
		baseTotal += detail.Base
		amountTotal += detail.Amount

		values := []any{
			idx + 1,
			detail.Name,
			detail.IDNumber,
			detail.Department,
			detail.Base,
			detail.Amount,
		}
		for colIdx, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, idx+2)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write data", err)
				return
			}
		}
	}

	totalRow := len(details) + 2
	totalValues := []any{
		"合计",
		"",
		"",
		"",
		baseTotal,
		amountTotal,
	}
	for colIdx, value := range totalValues {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, totalRow)
		if err := f.SetCellValue(sheetName, cell, value); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to write total row", err)
			return
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to encode excel", err)
		return
	}

	schemeLabels := map[models.Scheme]string{
		models.SchemePension:        "养老保险",
		models.SchemeMedical:        "医疗保险",
		models.SchemeSeriousIllness: "大额医疗",
		models.SchemeUnemployment:   "失业保险",
		models.SchemeInjury:         "工伤保险",
	}
	partLabels := map[models.Part]string{
		models.PartPersonal: "个人",
		models.PartUnit:     "单位",
	}

	filename := fmt.Sprintf("%s-%s-%s明细.xlsx", period.YearMonth, schemeLabels[scheme], partLabels[part])
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(buf.Bytes()))
}

func (h *Handler) resetPeriod(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	// 在事务中删除所有相关数据
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 删除花名册数据
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.RosterEntry{}).Error; err != nil {
			return fmt.Errorf("delete roster entries: %w", err)
		}

		// 删除原始记录
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.RawRecord{}).Error; err != nil {
			return fmt.Errorf("delete raw records: %w", err)
		}

		// 删除汇总数据
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("delete period summaries: %w", err)
		}

		// 删除个人扣款明细
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("delete personal charges: %w", err)
		}

		// 删除单位扣款明细
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("delete unit charges: %w", err)
		}

		// 删除源文件记录
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.SourceFile{}).Error; err != nil {
			return fmt.Errorf("delete source files: %w", err)
		}

		// 重置账期状态
		if err := tx.Model(period).Update("status", "draft").Error; err != nil {
			return fmt.Errorf("reset period status: %w", err)
		}

		return nil
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reset period", err)
		return
	}

	// 删除实际的文件存储目录
	uploadDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID))
	if _, err := os.Stat(uploadDir); err == nil {
		if err := os.RemoveAll(uploadDir); err != nil {
			// 日志记录错误，但不影响API响应
			fmt.Printf("Warning: failed to remove upload directory %s: %v\n", uploadDir, err)
		}
	}

	result := map[string]interface{}{
		"message": fmt.Sprintf("账期 %s 已重置，所有数据已清除", period.YearMonth),
		"period_id": period.ID,
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) deletePeriod(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	// 在事务中删除账期及其所有相关数据
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 删除花名册数据
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.RosterEntry{}).Error; err != nil {
			return fmt.Errorf("delete roster entries: %w", err)
		}

		// 删除原始记录
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.RawRecord{}).Error; err != nil {
			return fmt.Errorf("delete raw records: %w", err)
		}

		// 删除汇总数据
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("delete period summaries: %w", err)
		}

		// 删除个人扣款明细
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("delete personal charges: %w", err)
		}

		// 删除单位扣款明细
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("delete unit charges: %w", err)
		}

		// 删除源文件记录
		if err := tx.Where("period_id = ?", period.ID).Delete(&models.SourceFile{}).Error; err != nil {
			return fmt.Errorf("delete source files: %w", err)
		}

		// 最后删除账期本身
		if err := tx.Delete(period).Error; err != nil {
			return fmt.Errorf("delete period: %w", err)
		}

		return nil
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete period", err)
		return
	}

	// 删除实际的文件存储目录
	uploadDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID))
	if _, err := os.Stat(uploadDir); err == nil {
		if err := os.RemoveAll(uploadDir); err != nil {
			// 日志记录错误，但不影响API响应
			fmt.Printf("Warning: failed to remove upload directory %s: %v\n", uploadDir, err)
		}
	}

	result := map[string]interface{}{
		"message":   fmt.Sprintf("账期 %s 已删除", period.YearMonth),
		"period_id": period.ID,
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) uploadAdjustmentsBatch(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	if err := r.ParseMultipartForm(128 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	form := r.MultipartForm
	originalFiles := form.File["files"]

	if len(originalFiles) == 0 {
		respondError(w, http.StatusBadRequest, "files field is required", nil)
		return
	}

	// 去除重复文件 (相同文件名和大小)
	files := deduplicateFiles(originalFiles)

	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}
	targetDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID), "adjustments")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create adjustment upload dir", err)
		return
	}

	items := make([]batchUploadItem, 0, len(files))
	for _, header := range files {
		item := batchUploadItem{
			OriginalName: header.Filename,
		}

		// 从文件名解析险种和缴费部分
		scheme, part, err := parseAdjustmentFileName(header.Filename)
		if err != nil {
			item.Error = fmt.Sprintf("解析文件名失败: %v", err)
			items = append(items, item)
			continue
		}

		item.Scheme = scheme
		item.Part = part

		src, err := header.Open()
		if err != nil {
			item.Error = fmt.Sprintf("open file: %v", err)
			items = append(items, item)
			continue
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".xlsx"
		}
		filename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
		storedPath := filepath.Join(targetDir, filename)

		out, err := os.Create(storedPath)
		if err != nil {
			item.Error = fmt.Sprintf("create file: %v", err)
			_ = src.Close()
			items = append(items, item)
			continue
		}

		if _, err := io.Copy(out, src); err != nil {
			item.Error = fmt.Sprintf("save file: %v", err)
			_ = src.Close()
			_ = out.Close()
			items = append(items, item)
			continue
		}
		_ = src.Close()
		_ = out.Close()

		result, err := h.process.ParseAdjustmentFile(period.ID, storedPath, header.Filename, scheme, part)
		if err != nil {
			item.Error = err.Error()
			items = append(items, item)
			continue
		}

		item.FileName = filepath.Base(result.File.StoredPath)
		item.Imported = result.Imported
		items = append(items, item)
	}

	status := http.StatusOK
	for _, item := range items {
		if item.Error != "" {
			status = http.StatusMultiStatus
			break
		}
	}

	respondJSON(w, status, map[string]any{
		"items": items,
	})
}

func (h *Handler) processAdjustments(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	output, err := h.process.ProcessAdjustments(period.ID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to process adjustments", err)
		return
	}

	respondJSON(w, http.StatusOK, output)
}

// parseAdjustmentFileName 从补退文件名中解析险种和缴费部分
// 文件名格式: 张英俊职工基本养老保险(个人缴纳)_2025-01至2025-01_未申报信息明细.xlsx
func parseAdjustmentFileName(filename string) (models.Scheme, models.Part, error) {
	// 移除扩展名
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 提取险种信息
	var scheme models.Scheme
	var part models.Part

	if strings.Contains(name, "职工基本养老保险") || strings.Contains(name, "养老保险") {
		scheme = models.SchemePension
	} else if strings.Contains(name, "失业保险") {
		scheme = models.SchemeUnemployment
	} else if strings.Contains(name, "工伤保险") {
		scheme = models.SchemeInjury
	} else if strings.Contains(name, "医疗保险") {
		scheme = models.SchemeMedical
	} else if strings.Contains(name, "大额医疗") {
		scheme = models.SchemeSeriousIllness
	} else {
		return "", "", fmt.Errorf("无法识别险种类型")
	}

	// 提取缴费部分
	if strings.Contains(name, "(个人缴纳)") || strings.Contains(name, "个人缴纳") {
		part = models.PartPersonal
	} else if strings.Contains(name, "(单位缴纳)") || strings.Contains(name, "单位缴纳") {
		part = models.PartUnit
	} else if scheme == models.SchemeInjury {
		// 工伤保险只有单位缴纳
		part = models.PartUnit
	} else {
		return "", "", fmt.Errorf("无法识别缴费部分")
	}

	return scheme, part, nil
}

func (h *Handler) clearFiles(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	// 在事务中清除正常社保文件相关数据
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 删除正常社保文件的原始记录
		if err := tx.Where("period_id = ? AND file_type = ?", period.ID, models.FileTypeNormal).Delete(&models.RawRecord{}).Error; err != nil {
			return fmt.Errorf("delete normal raw records: %w", err)
		}

		// 删除正常社保文件的汇总数据
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, false).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("delete normal period summaries: %w", err)
		}

		// 删除正常社保文件的个人扣款明细
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, false).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("delete normal personal charges: %w", err)
		}

		// 删除正常社保文件的单位扣款明细
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, false).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("delete normal unit charges: %w", err)
		}

		// 删除正常社保文件记录
		if err := tx.Where("period_id = ? AND file_type = ?", period.ID, models.FileTypeNormal).Delete(&models.SourceFile{}).Error; err != nil {
			return fmt.Errorf("delete normal source files: %w", err)
		}

		return nil
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear files", err)
		return
	}

	result := map[string]interface{}{
		"message": "社保文件已清空",
		"cleared": "normal",
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) clearAdjustments(w http.ResponseWriter, r *http.Request) {
	period, err := h.getPeriodByParam(r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, err.Error(), nil)
		return
	}

	// 在事务中清除补退文件相关数据
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 删除补退文件的原始记录
		if err := tx.Where("period_id = ? AND file_type = ?", period.ID, models.FileTypeAdjustment).Delete(&models.RawRecord{}).Error; err != nil {
			return fmt.Errorf("delete adjustment raw records: %w", err)
		}

		// 删除补退文件的汇总数据
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, true).Delete(&models.PeriodSummary{}).Error; err != nil {
			return fmt.Errorf("delete adjustment period summaries: %w", err)
		}

		// 删除补退文件的个人扣款明细
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, true).Delete(&models.PersonalCharge{}).Error; err != nil {
			return fmt.Errorf("delete adjustment personal charges: %w", err)
		}

		// 删除补退文件的单位扣款明细
		if err := tx.Where("period_id = ? AND is_adjustment = ?", period.ID, true).Delete(&models.UnitCharge{}).Error; err != nil {
			return fmt.Errorf("delete adjustment unit charges: %w", err)
		}

		// 删除补退文件记录
		if err := tx.Where("period_id = ? AND file_type = ?", period.ID, models.FileTypeAdjustment).Delete(&models.SourceFile{}).Error; err != nil {
			return fmt.Errorf("delete adjustment source files: %w", err)
		}

		return nil
	})

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear adjustments", err)
		return
	}

	// 删除补退文件存储目录
	adjustmentDir := filepath.Join("./uploads", fmt.Sprintf("%d", period.ID), "adjustments")
	if _, err := os.Stat(adjustmentDir); err == nil {
		if err := os.RemoveAll(adjustmentDir); err != nil {
			fmt.Printf("Warning: failed to remove adjustment directory %s: %v\n", adjustmentDir, err)
		}
	}

	result := map[string]interface{}{
		"message": "补退文件已清空",
		"cleared": "adjustments",
	}

	respondJSON(w, http.StatusOK, result)
}
