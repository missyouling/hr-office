package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"siapp/internal/auth"
	"siapp/internal/models"
)

var callbackHeaderLabels = map[string][]string{
	"name":        {"姓名"},
	"identity":    {"证件号码", "证件号", "身份证号码", "身份证号"},
	"personal":    {"个人编号", "社保编号", "人员编号"},
	"change_type": {"减少原因", "变动原因", "办理类型", "变动类型", "状态"},
	"phone":       {"联系电话", "联系方式", "手机"},
	"remark":      {"备注", "说明"},
}

var requiredCallbackHeaders = []string{"identity", "personal", "name"}

type callbackParsedEntry struct {
	PersonalNumber string
	IdentityNumber string
	Name           string
	ChangeType     string
	Phone          string
	Remark         string
	Sequence       int
}

type callbackRecordDTO struct {
	ID             uint      `json:"id"`
	PersonalNumber string    `json:"personal_number"`
	IdentityNumber string    `json:"identity_number"`
	Name           string    `json:"name"`
	ChangeType     string    `json:"change_type"`
	Phone          string    `json:"phone"`
	Remark         string    `json:"remark"`
	Sequence       int       `json:"sequence"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type callbackRecordsResponse struct {
	Records        []callbackRecordDTO `json:"records"`
	LastUploadedAt *time.Time          `json:"last_uploaded_at,omitempty"`
	LastFileName   string              `json:"last_file_name,omitempty"`
	PersonalMap    map[string]string   `json:"personal_map"`
}

func normalizeCallbackToken(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), " ", "")
}

func normalizeCallbackID(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func callbackCellMatches(value string, labels []string) bool {
	if value == "" {
		return false
	}
	normalized := normalizeCallbackToken(value)
	for _, label := range labels {
		if strings.Contains(normalized, normalizeCallbackToken(label)) {
			return true
		}
	}
	return false
}

func findCallbackHeader(row []string) map[string]int {
	indices := make(map[string]int)
	for idx, cell := range row {
		value := strings.TrimSpace(cell)
		if value == "" {
			continue
		}
		for key, labels := range callbackHeaderLabels {
			if _, exists := indices[key]; exists {
				continue
			}
			if callbackCellMatches(value, labels) {
				indices[key] = idx
				break
			}
		}
	}
	return indices
}

func getCellValue(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseCallbackWorkbook(data []byte) ([]callbackParsedEntry, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	sheetName := file.GetSheetName(0)
	if sheetName == "" {
		return nil, errors.New("未找到工作表")
	}
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("回盘表为空")
	}

	headerIndex := -1
	var headerMap map[string]int
	for i, row := range rows {
		indices := findCallbackHeader(row)
		valid := true
		for _, key := range requiredCallbackHeaders {
			if _, ok := indices[key]; !ok {
				valid = false
				break
			}
		}
		if valid {
			headerIndex = i
			headerMap = indices
			break
		}
	}
	if headerIndex == -1 {
		return nil, errors.New("未找到包含“证件号码”表头的行")
	}

	entries := make([]callbackParsedEntry, 0, len(rows)-headerIndex-1)
	sequence := 1
	for i := headerIndex + 1; i < len(rows); i++ {
		row := rows[i]
		identity := normalizeCallbackID(getCellValue(row, headerMap["identity"]))
		name := strings.TrimSpace(getCellValue(row, headerMap["name"]))
		if identity == "" || name == "" {
			continue
		}
		entry := callbackParsedEntry{
			IdentityNumber: identity,
			PersonalNumber: strings.TrimSpace(getCellValue(row, headerMap["personal"])),
			Name:           name,
			ChangeType:     strings.TrimSpace(getCellValue(row, headerMap["change_type"])),
			Phone:          strings.TrimSpace(getCellValue(row, headerMap["phone"])),
			Remark:         strings.TrimSpace(getCellValue(row, headerMap["remark"])),
			Sequence:       sequence,
		}
		sequence++
		entries = append(entries, entry)
	}
	return entries, nil
}

func buildCallbackPersonalMapFromEntries(entries []models.CallbackRecord) map[string]string {
	result := make(map[string]string)
	for _, entry := range entries {
		identity := normalizeCallbackID(entry.IdentityNumber)
		personal := strings.TrimSpace(entry.PersonalNumber)
		if identity == "" || personal == "" {
			continue
		}
		result[identity] = personal
	}
	return result
}

func (h *Handler) fetchCallbackRecords(userID uint) ([]models.CallbackRecord, *models.CallbackUpload, error) {
	var records []models.CallbackRecord
	if err := h.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&records).Error; err != nil {
		return nil, nil, err
	}
	var upload models.CallbackUpload
	if err := h.db.Where("user_id = ?", userID).Order("uploaded_at DESC").First(&upload).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return records, nil, nil
		}
		return nil, nil, err
	}
	return records, &upload, nil
}

func buildCallbackRecordsResponse(records []models.CallbackRecord, upload *models.CallbackUpload) callbackRecordsResponse {
	resp := callbackRecordsResponse{
		Records:     make([]callbackRecordDTO, 0, len(records)),
		PersonalMap: buildCallbackPersonalMapFromEntries(records),
	}
	if upload != nil {
		resp.LastUploadedAt = &upload.UploadedAt
		resp.LastFileName = upload.FileName
	}
	for _, record := range records {
		resp.Records = append(resp.Records, callbackRecordDTO{
			ID:             record.ID,
			PersonalNumber: record.PersonalNumber,
			IdentityNumber: record.IdentityNumber,
			Name:           record.Name,
			ChangeType:     record.ChangeType,
			Phone:          record.Phone,
			Remark:         record.Remark,
			Sequence:       record.Sequence,
			UpdatedAt:      record.UpdatedAt,
		})
	}
	return resp
}

func (h *Handler) listCallbackRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	records, upload, err := h.fetchCallbackRecords(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load callback records", err)
		return
	}
	respondJSON(w, http.StatusOK, buildCallbackRecordsResponse(records, upload))
}

func (h *Handler) uploadCallbackRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
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
	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read uploaded file", err)
		return
	}
	entries, err := parseCallbackWorkbook(data)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if len(entries) == 0 {
		respondError(w, http.StatusBadRequest, "未解析到任何有效记录", nil)
		return
	}

	uploadUserID := userID
	upload := models.CallbackUpload{
		UserID:       &uploadUserID,
		FileName:     header.Filename,
		FileSize:     header.Size,
		TotalRecords: len(entries),
		UploadedAt:   time.Now(),
		RawFile:      data,
	}
	tx := h.db.Begin()
	if err := tx.Create(&upload).Error; err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to store callback upload", err)
		return
	}
	if err := h.persistCallbackEntries(tx, userID, upload.ID, entries); err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to store callback records", err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to commit callback records", err)
		return
	}
	records, latestUpload, err := h.fetchCallbackRecords(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload callback records", err)
		return
	}
	respondJSON(w, http.StatusCreated, buildCallbackRecordsResponse(records, latestUpload))
}

func (h *Handler) persistCallbackEntries(tx *gorm.DB, userID uint, uploadID uint, entries []callbackParsedEntry) error {
	userIDCopy := userID
	for _, entry := range entries {
		record := models.CallbackRecord{
			UploadID:       uploadID,
			UserID:         &userIDCopy,
			PersonalNumber: entry.PersonalNumber,
			IdentityNumber: entry.IdentityNumber,
			Name:           entry.Name,
			ChangeType:     entry.ChangeType,
			Phone:          entry.Phone,
			Remark:         entry.Remark,
			Sequence:       entry.Sequence,
		}
		assignments := map[string]interface{}{
			"personal_number": record.PersonalNumber,
			"name":            record.Name,
			"change_type":     record.ChangeType,
			"phone":           record.Phone,
			"remark":          record.Remark,
			"sequence":        record.Sequence,
			"upload_id":       uploadID,
			"updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "identity_number"}},
			DoUpdates: clause.Assignments(assignments),
		}).Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) clearCallbackRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	tx := h.db.Begin()
	if err := tx.Where("user_id = ?", userID).Delete(&models.CallbackRecord{}).Error; err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to clear callback records", err)
		return
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.CallbackUpload{}).Error; err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "failed to clear callback uploads", err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear callback data", err)
		return
	}
	respondJSON(w, http.StatusOK, callbackRecordsResponse{
		Records:     []callbackRecordDTO{},
		PersonalMap: map[string]string{},
	})
}
