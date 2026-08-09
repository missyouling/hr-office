package api

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/service/storage"
)

// writeJSON 统一的 JSON 响应方法
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// decodeJSON 统一的 JSON 解析方法
func decodeJSON(r *http.Request, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}

// getUserIDFromContext 获取用户 ID（临时实现）
func getUserIDFromContext(r *http.Request) uint {
	// 从 JWT token 中获取用户 ID
	// 实际应该从 context 中获取，这里临时返回 1（admin）
	return 1
}

// generateDocumentCode 生成档案编号
// 格式: 一级代码-二级代码-年度-顺序号（不再添加保管期限后缀，与编码规则保持一致）
func generateDocumentCode(db *gorm.DB, categoryCode, subCategoryCode, year string, _ string) (string, int, error) {
	var maxSeq int
	yearInt, _ := strconv.Atoi(year)

	// 获取该分类该年度的最大序号
	if err := db.Model(&models.Document{}).
		Where("category_code = ? AND sub_category_code = ? AND year = ?", categoryCode, subCategoryCode, yearInt).
		Select("COALESCE(MAX(sequence), 0)").
		Scan(&maxSeq).Error; err != nil {
		return "", 0, err
	}

	seq := maxSeq + 1
	// 格式: 一级代码-二级代码-年度-顺序号（如 WS-01-2026-001）
	code := fmt.Sprintf("%s-%s-%s-%03d", categoryCode, subCategoryCode, year, seq)

	return code, seq, nil
}

// listDocumentCategories 获取分类目录
func (h *Handler) listDocumentCategories(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var categories []models.DocumentCategory
	if err := h.db.Where("user_id = ? OR user_id IS NULL", userID).
		Preload("SubCategories").
		Order("sort_order").
		Find(&categories).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, categories)
}

// listDocuments 获取档案列表
func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	// 分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 筛选参数
	categoryCode := r.URL.Query().Get("category_code")
	subCategoryCode := r.URL.Query().Get("sub_category_code")
	keyword := r.URL.Query().Get("keyword")
	retentionPeriod := r.URL.Query().Get("retention_period")
	status := r.URL.Query().Get("status")
	folderPath := r.URL.Query().Get("folder_path")

	// 排序参数
	sortField := r.URL.Query().Get("sort_field")
	sortDirection := r.URL.Query().Get("sort_direction")
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	query := h.db.Model(&models.Document{}).Where("user_id = ?", userID)

	if categoryCode != "" {
		query = query.Where("category_code = ?", categoryCode)
	}
	if subCategoryCode != "" {
		query = query.Where("sub_category_code = ?", subCategoryCode)
	}
	if keyword != "" {
		query = query.Where("file_name LIKE ? OR summary LIKE ? OR document_code LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if retentionPeriod != "" {
		query = query.Where("retention_period = ?", retentionPeriod)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if folderPath != "" {
		query = query.Where("folder_path = ?", folderPath)
	}

	// 标签筛选（多标签为 OR 关系：包含任一标签的文档）
	tagNames := r.URL.Query()["tag_names"]
	if len(tagNames) > 0 {
		subQuery := h.db.Model(&models.DocumentTagLink{}).
			Select("document_id").
			Joins("JOIN archive_tags ON archive_tags.id = document_tag_links.tag_id").
			Where("archive_tags.name IN ?", tagNames)
		query = query.Where("documents.id IN (?)", subQuery)
	}

	// 排序
	orderClause := "created_at DESC"
	if sortField != "" {
		// 验证排序字段，防止 SQL 注入
		allowedFields := map[string]bool{
			"document_code":    true,
			"file_name":         true,
			"retention_period":  true,
			"status":            true,
			"created_at":        true,
			"updated_at":        true,
			"signed_date":       true,
			"year":              true,
			"sequence":          true,
		}
		if allowedFields[sortField] {
			orderClause = sortField + " " + sortDirection
		}
	}

	// 统计总数
	var total int64
	query.Count(&total)

	// 分页查询
	var documents []models.Document
	offset := (page - 1) * pageSize
	if err := query.Order(orderClause).Offset(offset).Limit(pageSize).Find(&documents).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"items":     documents,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// createDocument 创建档案
func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		CategoryCode    string `json:"category_code"`
		SubCategoryCode string `json:"sub_category_code"`
		FileName        string `json:"file_name"`
		Summary         string `json:"summary"`
		Tags            string `json:"tags"` // 标签数组的 JSON 字符串
		SignedDate      string `json:"signed_date"`
		ExpirationDate  string `json:"expiration_date"`
		RetentionPeriod string `json:"retention_period"`

		// 文书档案
		PartyA          string  `json:"party_a"`
		PartyB          string  `json:"party_b"`
		Amount          float64 `json:"amount"`
		PaymentProgress string  `json:"payment_progress"`

		// 科技档案
		ProjectName    string `json:"project_name"`
		DesignUnit     string `json:"design_unit"`
		Designer       string `json:"designer"`
		ProjectLeader  string `json:"project_leader"`
		EquipmentName  string `json:"equipment_name"`
		EquipmentModel string `json:"equipment_model"`
		PurchaseDate   string `json:"purchase_date"`

		// 电子档案
		ContentDescription string `json:"content_description"`
		CaptureDate        string `json:"capture_date"`
		Capturer           string `json:"capturer"`
		ActivityName       string `json:"activity_name"`
		CarrierType        string `json:"carrier_type"`
		StorageLocation    string `json:"storage_location"`
		FileFormat         string `json:"file_format"`

		// 专门档案
		Petitioner     string `json:"petitioner"`
		Respondent     string `json:"respondent"`
		AuditUnit      string `json:"audit_unit"`
		AuditPeriod    string `json:"audit_period"`
		Counterparty   string `json:"counterparty"`
		Lawyer         string `json:"lawyer"`
		Winner         string `json:"winner"`
		StartDate      string `json:"start_date"`
		CompletionDate string `json:"completion_date"`

		Remarks string `json:"remarks"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取分类名称
	var category models.DocumentCategory
	var subCategory models.DocumentSubCategory

	h.db.Where("code = ?", payload.CategoryCode).First(&category)
	h.db.Where("category_code = ? AND code = ?", payload.CategoryCode, payload.SubCategoryCode).First(&subCategory)

	// 生成编号
	year := strconv.Itoa(time.Now().Year())
	docCode, seq, err := generateDocumentCode(h.db, payload.CategoryCode, payload.SubCategoryCode, year, payload.RetentionPeriod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	doc := models.Document{
		UserID:             userID,
		DocumentCode:       docCode,
		CategoryCode:       payload.CategoryCode,
		SubCategoryCode:    payload.SubCategoryCode,
		Year:               time.Now().Year(),
		Sequence:           seq,
		FileName:           payload.FileName,
		DocumentType:       category.Name,
		SubType:            subCategory.Name,
		Summary:            payload.Summary,
		Tags:               payload.Tags,
		RetentionPeriod:    payload.RetentionPeriod,
		PartyA:             payload.PartyA,
		PartyB:             payload.PartyB,
		Amount:             payload.Amount,
		PaymentProgress:    payload.PaymentProgress,
		ProjectName:        payload.ProjectName,
		DesignUnit:         payload.DesignUnit,
		Designer:           payload.Designer,
		ProjectLeader:      payload.ProjectLeader,
		EquipmentName:      payload.EquipmentName,
		EquipmentModel:     payload.EquipmentModel,
		ContentDescription: payload.ContentDescription,
		ActivityName:       payload.ActivityName,
		CarrierType:        payload.CarrierType,
		StorageLocation:    payload.StorageLocation,
		FileFormat:         payload.FileFormat,
		Petitioner:         payload.Petitioner,
		Respondent:         payload.Respondent,
		AuditUnit:          payload.AuditUnit,
		AuditPeriod:        payload.AuditPeriod,
		Counterparty:       payload.Counterparty,
		Lawyer:             payload.Lawyer,
		Winner:             payload.Winner,
		Remarks:            payload.Remarks,
		Status:             "active",
	}

	// 解析日期
	if payload.SignedDate != "" {
		if t, err := time.Parse("2006-01-02", payload.SignedDate); err == nil {
			doc.SignedDate = &t
		}
	}
	if payload.ExpirationDate != "" {
		if t, err := time.Parse("2006-01-02", payload.ExpirationDate); err == nil {
			doc.ExpirationDate = &t
		}
	}
	if payload.PurchaseDate != "" {
		if t, err := time.Parse("2006-01-02", payload.PurchaseDate); err == nil {
			doc.PurchaseDate = &t
		}
	}
	if payload.CaptureDate != "" {
		if t, err := time.Parse("2006-01-02", payload.CaptureDate); err == nil {
			doc.CaptureDate = &t
		}
	}
	if payload.StartDate != "" {
		if t, err := time.Parse("2006-01-02", payload.StartDate); err == nil {
			doc.StartDate = &t
		}
	}
	if payload.CompletionDate != "" {
		if t, err := time.Parse("2006-01-02", payload.CompletionDate); err == nil {
			doc.CompletionDate = &t
		}
	}

	if err := h.db.Create(&doc).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, doc)
}

// updateDocument 更新档案
func (h *Handler) updateDocument(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	docID := chi.URLParam(r, "docID")

	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		http.Error(w, "档案不存在", http.StatusNotFound)
		return
	}

	var payload map[string]interface{}
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 更新字段
	allowedFields := []string{
		"file_name", "summary", "tags", "signed_date", "expiration_date", "retention_period",
		"party_a", "party_b", "amount", "payment_progress",
		"project_name", "design_unit", "designer", "project_leader", "equipment_name", "equipment_model", "purchase_date",
		"content_description", "capture_date", "capturer", "activity_name", "carrier_type", "storage_location", "file_format",
		"petitioner", "respondent", "audit_unit", "audit_period", "counterparty", "lawyer", "winner", "start_date", "completion_date",
		"remarks", "status",
	}

	for _, field := range allowedFields {
		if v, ok := payload[field]; ok {
			h.db.Model(&doc).Update(field, v)
		}
	}

	h.db.First(&doc, docID)
	writeJSON(w, doc)
}

// deleteDocument 删除档案
func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	docID := chi.URLParam(r, "docID")

	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		http.Error(w, "档案不存在", http.StatusNotFound)
		return
	}

	if err := h.db.Delete(&doc).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// uploadDocumentFile 上传档案文件
func (h *Handler) uploadDocumentFile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	docID := chi.URLParam(r, "docID")

	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		http.Error(w, "档案不存在", http.StatusNotFound)
		return
	}

	// 解析 multipart form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "请选择文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 生成存储路径
	ext := filepath.Ext(header.Filename)
	newFilename := uuid.New().String() + ext

	// Resolve storage path using StorageRouter
	resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "document_file",
		Filename:     newFilename,
	})
	if err != nil {
		http.Error(w, "failed to resolve storage path: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Upload file using GlobalManager
	_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, newFilename, file, header.Size)
	if err != nil {
		http.Error(w, "failed to upload file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// For backward compatibility, also store the file path locally
	relativePath := fmt.Sprintf("documents/%d/%s/%s", userID, time.Now().Format("2006-01"), newFilename)
	fullPath := filepath.Join(h.uploadBaseDir, relativePath)
	if err := ensureDir(filepath.Dir(fullPath)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-read the file from storage for local backup
	if _, err := file.Seek(0, io.SeekStart); err == nil {
		if err := saveFile(file, fullPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 更新档案记录
	doc.FilePath = relativePath
	doc.FileNameOriginal = header.Filename
	doc.FileSize = header.Size
	doc.FileType = getFileType(ext)
	doc.FileFormat = strings.TrimPrefix(ext, ".") // 设置文件格式为扩展名
	h.db.Save(&doc)

	writeJSON(w, map[string]interface{}{
		"file_path": relativePath,
		"file_name": header.Filename,
		"file_size": header.Size,
	})
}

// uploadDocumentWithOCR 上传文件并自动OCR提取字段
// POST /api/archives/documents/upload-with-ocr
func (h *Handler) uploadDocumentWithOCR(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	// 解析 multipart form
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	subCategoryCode := r.FormValue("sub_category_code")

	// 调用OCR提取服务
	extractService := service.NewOCRExtractService(h.db, h.ocrService)
	result, err := extractService.ExtractFieldsFromFile(file, header, subCategoryCode, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

// downloadDocumentFile 下载档案文件
func (h *Handler) downloadDocumentFile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	docID := chi.URLParam(r, "docID")

	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		http.Error(w, "档案不存在", http.StatusNotFound)
		return
	}

	if doc.FilePath == "" {
		http.Error(w, "暂无附件", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(h.uploadBaseDir, doc.FilePath)
	http.ServeFile(w, r, fullPath)
}

// getExpiringDocuments 获取到期提醒
func (h *Handler) getExpiringDocuments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 {
		days = 30
	}

	expiryDate := time.Now().AddDate(0, 0, days)

	var documents []models.Document
	if err := h.db.Where("user_id = ? AND expiration_date IS NOT NULL AND expiration_date <= ? AND status = ?",
		userID, expiryDate, "active").
		Order("expiration_date").
		Find(&documents).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, documents)
}

// getExpirationReminderSettings 获取到期提醒设置
func (h *Handler) getExpirationReminderSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var settings []models.ExpirationReminder
	if err := h.db.Where("user_id = ?", userID).Find(&settings).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, settings)
}

// updateExpirationReminderSettings 更新到期提醒设置
func (h *Handler) updateExpirationReminderSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		DocumentCategory string `json:"document_category"`
		RemindDays       int    `json:"remind_days"`
		Enabled          bool   `json:"enabled"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var setting models.ExpirationReminder
	if err := h.db.Where("user_id = ? AND document_category = ?", userID, payload.DocumentCategory).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			setting = models.ExpirationReminder{
				UserID:           userID,
				DocumentCategory: payload.DocumentCategory,
				RemindDays:       payload.RemindDays,
				Enabled:          payload.Enabled,
			}
			h.db.Create(&setting)
		}
	} else {
		setting.RemindDays = payload.RemindDays
		setting.Enabled = payload.Enabled
		h.db.Save(&setting)
	}

	writeJSON(w, setting)
}

// getFileType 根据扩展名获取文件类型
func getFileType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".pdf":
		return "pdf"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv":
		return "video"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg":
		return "audio"
	case ".doc", ".docx", ".txt", ".xls", ".xlsx", ".ppt", ".pptx":
		return "document"
	default:
		return "other"
	}
}

// ensureDir 确保目录存在
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// saveFile 保存文件
func saveFile(src io.Reader, dst string) error {
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, src)
	return err
}

// ============ 档案字段定义 API ============

// listFieldGroups 获取字段分组列表
func (h *Handler) listFieldGroups(w http.ResponseWriter, r *http.Request) {
	subCategoryID := r.URL.Query().Get("sub_category_id")

	query := h.db.Model(&models.ArchiveFieldGroup{})
	if subCategoryID != "" {
		query = query.Where("sub_category_id = ?", subCategoryID)
	}

	var groups []models.ArchiveFieldGroup
	if err := query.Preload("Fields").Order("sort_order").Find(&groups).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, groups)
}

// createFieldGroup 创建字段分组
func (h *Handler) createFieldGroup(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SubCategoryID uint   `json:"sub_category_id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		SortOrder     int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group := models.ArchiveFieldGroup{
		SubCategoryID: payload.SubCategoryID,
		Name:          payload.Name,
		Description:   payload.Description,
		SortOrder:     payload.SortOrder,
	}

	if err := h.db.Create(&group).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, group)
}

// updateFieldGroup 更新字段分组
func (h *Handler) updateFieldGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")

	var group models.ArchiveFieldGroup
	if err := h.db.Where("id = ?", groupID).First(&group).Error; err != nil {
		http.Error(w, "分组不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group.Name = payload.Name
	group.Description = payload.Description
	group.SortOrder = payload.SortOrder

	if err := h.db.Save(&group).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, group)
}

// deleteFieldGroup 删除字段分组
func (h *Handler) deleteFieldGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")

	var group models.ArchiveFieldGroup
	if err := h.db.Where("id = ?", groupID).First(&group).Error; err != nil {
		http.Error(w, "分组不存在", http.StatusNotFound)
		return
	}

	// 将该分组下的字段移到无分组状态
	h.db.Model(&models.ArchiveFieldDefinition{}).Where("group_id = ?", groupID).Update("group_id", nil)

	if err := h.db.Delete(&group).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// listFieldDefinitions 获取字段定义列表
func (h *Handler) listFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	subCategoryID := r.URL.Query().Get("sub_category_id")

	query := h.db.Model(&models.ArchiveFieldDefinition{})
	if subCategoryID != "" {
		query = query.Where("sub_category_id = ?", subCategoryID)
	}

	var fields []models.ArchiveFieldDefinition
	if err := query.Preload("Group").Order("sort_order").Find(&fields).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, fields)
}

// createFieldDefinition 创建字段定义
func (h *Handler) createFieldDefinition(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SubCategoryID  uint   `json:"sub_category_id" binding:"required"`
		GroupID        *uint  `json:"group_id"`
		FieldName      string `json:"field_name" binding:"required"`
		FieldLabel     string `json:"field_label" binding:"required"`
		FieldType      string `json:"field_type" binding:"required"`
		Required       bool   `json:"required"`
		DefaultValue   string `json:"default_value"`
		Options        string `json:"options"`
		Placeholder    string `json:"placeholder"`
		SortOrder      int    `json:"sort_order"`
		Visible        bool   `json:"visible"`
		Editable       bool   `json:"editable"`
		ConditionConfig *struct {
			FieldName string `json:"field_name"`
			Operator  string `json:"operator"`
			Value     string `json:"value"`
		} `json:"condition_config"`
		HelpText string `json:"help_text"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, "解析请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if payload.SubCategoryID == 0 {
		http.Error(w, "二级分类ID不能为空", http.StatusBadRequest)
		return
	}
	if payload.FieldName == "" {
		http.Error(w, "字段名不能为空", http.StatusBadRequest)
		return
	}
	if payload.FieldLabel == "" {
		http.Error(w, "显示名称不能为空", http.StatusBadRequest)
		return
	}

	// 检查二级分类是否存在
	var subCategory models.DocumentSubCategory
	if err := h.db.Where("id = ?", payload.SubCategoryID).First(&subCategory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "二级分类不存在", http.StatusBadRequest)
			return
		}
		http.Error(w, "检查二级分类失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 检查字段名是否已存在
	var count int64
	h.db.Model(&models.ArchiveFieldDefinition{}).
		Where("sub_category_id = ? AND field_name = ?", payload.SubCategoryID, payload.FieldName).
		Count(&count)
	if count > 0 {
		http.Error(w, "该字段名已存在", http.StatusBadRequest)
		return
	}

	field := models.ArchiveFieldDefinition{
		SubCategoryID: payload.SubCategoryID,
		GroupID:       payload.GroupID,
		FieldName:     payload.FieldName,
		FieldLabel:    payload.FieldLabel,
		FieldType:     payload.FieldType,
		Required:      payload.Required,
		DefaultValue:  payload.DefaultValue,
		Options:       payload.Options,
		Placeholder:   payload.Placeholder,
		SortOrder:     payload.SortOrder,
		Visible:       payload.Visible,
		Editable:      payload.Editable,
		HelpText:      payload.HelpText,
	}

	// 处理条件配置
	if payload.ConditionConfig != nil && payload.ConditionConfig.FieldName != "" {
		field.ConditionConfig = &models.ConditionConfig{
			FieldName: payload.ConditionConfig.FieldName,
			Operator:  payload.ConditionConfig.Operator,
			Value:     payload.ConditionConfig.Value,
		}
	}

	if err := h.db.Create(&field).Error; err != nil {
		http.Error(w, "创建字段失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, field)
}

// updateFieldDefinition 更新字段定义
func (h *Handler) updateFieldDefinition(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")

	var field models.ArchiveFieldDefinition
	if err := h.db.Where("id = ?", fieldID).First(&field).Error; err != nil {
		http.Error(w, "字段不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		GroupID        *uint  `json:"group_id"`
		FieldLabel     string `json:"field_label"`
		FieldType      string `json:"field_type"`
		Required       bool   `json:"required"`
		DefaultValue   string `json:"default_value"`
		Options        string `json:"options"`
		Placeholder    string `json:"placeholder"`
		SortOrder      int    `json:"sort_order"`
		Visible        bool   `json:"visible"`
		Editable       bool   `json:"editable"`
		ConditionConfig *struct {
			FieldName string `json:"field_name"`
			Operator  string `json:"operator"`
			Value     string `json:"value"`
		} `json:"condition_config"`
		HelpText string `json:"help_text"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	field.GroupID = payload.GroupID
	field.FieldLabel = payload.FieldLabel
	field.FieldType = payload.FieldType
	field.Required = payload.Required
	field.DefaultValue = payload.DefaultValue
	field.Options = payload.Options
	field.Placeholder = payload.Placeholder
	field.SortOrder = payload.SortOrder
	field.Visible = payload.Visible
	field.Editable = payload.Editable
	field.HelpText = payload.HelpText

	// 处理条件配置
	if payload.ConditionConfig != nil {
		field.ConditionConfig = &models.ConditionConfig{
			FieldName: payload.ConditionConfig.FieldName,
			Operator:  payload.ConditionConfig.Operator,
			Value:     payload.ConditionConfig.Value,
		}
	}

	if err := h.db.Save(&field).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, field)
}

// deleteFieldDefinition 删除字段定义
func (h *Handler) deleteFieldDefinition(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "fieldID")

	var field models.ArchiveFieldDefinition
	if err := h.db.Where("id = ?", fieldID).First(&field).Error; err != nil {
		http.Error(w, "字段不存在", http.StatusNotFound)
		return
	}

	if err := h.db.Delete(&field).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// getFieldsBySubCategory 获取某二级分类的所有字段（含分组）
func (h *Handler) getFieldsBySubCategory(w http.ResponseWriter, r *http.Request) {
	subCategoryID := chi.URLParam(r, "subCategoryID")

	// 获取所有字段
	var fields []models.ArchiveFieldDefinition
	if err := h.db.Where("sub_category_id = ? AND visible = ?", subCategoryID, true).
		Preload("Group").
		Order("COALESCE(group_id, 0), sort_order").
		Find(&fields).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 按分组整理
	type FieldGroup struct {
		ID     uint                         `json:"id"`
		Name   string                       `json:"name"`
		Fields []models.ArchiveFieldDefinition `json:"fields"`
	}

	var result struct {
		Groups   []FieldGroup               `json:"groups"`
		Ungrouped []models.ArchiveFieldDefinition `json:"ungrouped"`
	}

	for _, f := range fields {
		if f.GroupID != nil {
			found := false
			for i := range result.Groups {
				if result.Groups[i].ID == *f.GroupID {
					result.Groups[i].Fields = append(result.Groups[i].Fields, f)
					found = true
					break
				}
			}
			if !found {
				result.Groups = append(result.Groups, FieldGroup{
					ID:     *f.GroupID,
					Name:   f.Group.Name,
					Fields: []models.ArchiveFieldDefinition{f},
				})
			}
		} else {
			result.Ungrouped = append(result.Ungrouped, f)
		}
	}

	writeJSON(w, result)
}

// ============ 档案批量操作 API ============

// BatchUploadRequest 批量上传请求
type BatchUploadRequest struct {
	CategoryCode    string                 `json:"category_code"`
	SubCategoryCode string                 `json:"sub_category_code"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// BatchUploadItem 批量上传结果项
type BatchUploadItem struct {
	ID            uint   `json:"id"`
	DocumentCode  string `json:"document_code"`
	FileName      string `json:"file_name"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

// batchUploadDocuments 批量上传档案文件
func (h *Handler) batchUploadDocuments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	// 解析 form data
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
		http.Error(w, "解析表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	categoryCode := r.FormValue("category_code")
	subCategoryCode := r.FormValue("sub_category_code")

	if categoryCode == "" || subCategoryCode == "" {
		http.Error(w, "请提供分类代码", http.StatusBadRequest)
		return
	}

	// 获取分类信息
	var category models.DocumentCategory
	var subCategory models.DocumentSubCategory
	h.db.Where("code = ?", categoryCode).First(&category)
	h.db.Where("category_code = ? AND code = ?", categoryCode, subCategoryCode).First(&subCategory)

	// 获取文件
	form := r.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		http.Error(w, "请选择文件", http.StatusBadRequest)
		return
	}

	year := strconv.Itoa(time.Now().Year())
	results := make([]BatchUploadItem, 0, len(files))
	successCount := 0
	failCount := 0

	for _, fileHeader := range files {
		// 生成档案编号
		docCode, seq, err := generateDocumentCode(h.db, categoryCode, subCategoryCode, year, "")
		if err != nil {
			results = append(results, BatchUploadItem{
				FileName: fileHeader.Filename,
				Status:   "error",
				Error:    "生成编号失败",
			})
			failCount++
			continue
		}

		// 创建档案记录
		doc := models.Document{
			UserID:          userID,
			DocumentCode:    docCode,
			CategoryCode:    categoryCode,
			SubCategoryCode: subCategoryCode,
			Year:            time.Now().Year(),
			Sequence:        seq,
			FileName:        fileHeader.Filename,
			DocumentType:    category.Name,
			SubType:         subCategory.Name,
			Status:          "active",
		}

		if err := h.db.Create(&doc).Error; err != nil {
			results = append(results, BatchUploadItem{
				FileName: fileHeader.Filename,
				Status:   "error",
				Error:    "创建记录失败",
			})
			failCount++
			continue
		}

		// 上传文件
		file, err := fileHeader.Open()
		if err != nil {
			results = append(results, BatchUploadItem{
				ID:           doc.ID,
				DocumentCode: docCode,
				FileName:     fileHeader.Filename,
				Status:       "error",
				Error:        "打开文件失败",
			})
			failCount++
			continue
		}

		ext := filepath.Ext(fileHeader.Filename)
		newFilename := uuid.New().String() + ext

		// Resolve storage path using StorageRouter
		resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
			ModuleCode:   "archives",
			ResourceType: "document_batch",
			Filename:     newFilename,
		})
		if err != nil {
			file.Close()
			results = append(results, BatchUploadItem{
				ID:           doc.ID,
				DocumentCode: docCode,
				FileName:     fileHeader.Filename,
				Status:       "error",
				Error:        "failed to resolve storage path",
			})
			failCount++
			continue
		}

		// Upload file using GlobalManager
		_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, newFilename, file, fileHeader.Size)
		if err != nil {
			file.Close()
			results = append(results, BatchUploadItem{
				ID:           doc.ID,
				DocumentCode: docCode,
				FileName:     fileHeader.Filename,
				Status:       "error",
				Error:        "failed to upload file",
			})
			failCount++
			continue
		}

		// For backward compatibility, also store the file path locally
		relativePath := fmt.Sprintf("documents/%d/%s/%s", userID, time.Now().Format("2006-01"), newFilename)
		fullPath := filepath.Join(h.uploadBaseDir, relativePath)

		if err := ensureDir(filepath.Dir(fullPath)); err != nil {
			file.Close()
			results = append(results, BatchUploadItem{
				ID:           doc.ID,
				DocumentCode: docCode,
				FileName:     fileHeader.Filename,
				Status:       "error",
				Error:        "创建目录失败",
			})
			failCount++
			continue
		}

		// Re-read the file from storage for local backup
		if _, err := file.Seek(0, io.SeekStart); err == nil {
			if err := saveFile(file, fullPath); err != nil {
				file.Close()
				results = append(results, BatchUploadItem{
					ID:           doc.ID,
					DocumentCode: docCode,
					FileName:     fileHeader.Filename,
					Status:       "error",
					Error:        "保存文件失败",
				})
				failCount++
				continue
			}
		}
		file.Close()

		// 更新档案记录
		doc.FilePath = relativePath
		doc.FileNameOriginal = fileHeader.Filename
		doc.FileSize = fileHeader.Size
		doc.FileType = getFileType(ext)
		doc.FileFormat = strings.TrimPrefix(ext, ".") // 设置文件格式为扩展名
		h.db.Save(&doc)

		results = append(results, BatchUploadItem{
			ID:           doc.ID,
			DocumentCode: docCode,
			FileName:     fileHeader.Filename,
			Status:       "success",
		})
		successCount++
	}

	writeJSON(w, map[string]interface{}{
		"items":   results,
		"total":   len(files),
		"success": successCount,
		"failed":  failCount,
	})
}

// batchDownloadDocuments 批量下载档案文件（打包成ZIP）
func (h *Handler) batchDownloadDocuments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		IDs []uint `json:"ids"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.IDs) == 0 {
		http.Error(w, "请选择要下载的文件", http.StatusBadRequest)
		return
	}

	// 获取文档列表
	var documents []models.Document
	if err := h.db.Where("id IN ? AND user_id = ?", payload.IDs, userID).Find(&documents).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建 ZIP 文件
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=archives_%s.zip", time.Now().Format("20060102_150405")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, doc := range documents {
		if doc.FilePath == "" {
			continue
		}

		fullPath := filepath.Join(h.uploadBaseDir, doc.FilePath)
		file, err := os.Open(fullPath)
		if err != nil {
			continue // 跳过不存在的文件
		}

		// 使用档案编号作为文件名
		fileNameInZip := fmt.Sprintf("%s_%s%s", doc.DocumentCode, doc.FileName, filepath.Ext(doc.FilePath))
		// 清理文件名
		fileNameInZip = strings.ReplaceAll(fileNameInZip, "/", "_")
		fileNameInZip = strings.ReplaceAll(fileNameInZip, "\\", "_")

		writer, err := zipWriter.Create(fileNameInZip)
		if err != nil {
			file.Close()
			continue
		}

		io.Copy(writer, file)
		file.Close()
	}
}

// ShareLink 分享链接
type ShareLink struct {
	ID        uint   `json:"id"`
	FileName  string `json:"file_name"`
	Link      string `json:"link"`
	ExpiresAt string `json:"expires_at"`
}

// generateShareLink 生成分享链接
func (h *Handler) generateShareLink(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		IDs        []uint `json:"ids"`
		ExpiryHours int   `json:"expiry_hours"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.IDs) == 0 {
		http.Error(w, "请选择要分享的文件", http.StatusBadRequest)
		return
	}

	if payload.ExpiryHours <= 0 {
		payload.ExpiryHours = 24
	}

	// 获取文档列表
	var documents []models.Document
	if err := h.db.Where("id IN ? AND user_id = ?", payload.IDs, userID).Find(&documents).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取基础URL
	baseURL := h.uploadBaseURL
	if baseURL == "" {
		baseURL = "/api"
	}

	links := make([]ShareLink, 0, len(documents))
	expiresAt := time.Now().Add(time.Duration(payload.ExpiryHours) * time.Hour)

	for _, doc := range documents {
		// 生成随机token
		tokenBytes := make([]byte, 32)
		rand.Read(tokenBytes)
		token := base64.URLEncoding.EncodeToString(tokenBytes)

		// 保存到数据库
		shareLink := models.ShareLink{
			Token:      token,
			UserID:     userID,
			DocumentID: doc.ID,
			ExpiresAt:  expiresAt,
		}
		if err := h.db.Create(&shareLink).Error; err != nil {
			http.Error(w, "创建分享链接失败", http.StatusInternalServerError)
			return
		}

		// 生成分享链接
		shareURL := fmt.Sprintf("%s/archives/shared/%s", baseURL, token)

		links = append(links, ShareLink{
			ID:        shareLink.ID,
			FileName:  doc.FileName,
			Link:      shareURL,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, map[string]interface{}{
		"links": links,
	})
}

// AccessSharedDocument 通过分享链接访问文档
func (h *Handler) AccessSharedDocument(w http.ResponseWriter, r *http.Request) {
	// 获取token
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "无效的分享链接", http.StatusBadRequest)
		return
	}

	// 查询分享链接
	var shareLink models.ShareLink
	if err := h.db.Preload("Document").Where("token = ?", token).First(&shareLink).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "分享链接不存在或已失效", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// 检查是否过期
	if shareLink.IsExpired() {
		http.Error(w, "分享链接已过期", http.StatusGone)
		return
	}

	// 返回文档信息
	if shareLink.Document == nil {
		http.Error(w, "文档不存在", http.StatusNotFound)
		return
	}

	doc := shareLink.Document

	// 如果是下载请求，返回文件
	if r.URL.Query().Get("download") == "1" {
		if doc.FilePath == "" {
			http.Error(w, "文件不存在", http.StatusNotFound)
			return
		}
		fullPath := filepath.Join(h.uploadBaseDir, doc.FilePath)
		http.ServeFile(w, r, fullPath)
		return
	}

	// 返回文档信息
	writeJSON(w, map[string]interface{}{
		"id":             doc.ID,
		"document_code":  doc.DocumentCode,
		"file_name":      doc.FileName,
		"document_type":  doc.DocumentType,
		"sub_type":       doc.SubType,
		"file_type":      doc.FileType,
		"expires_at":     shareLink.ExpiresAt.Format(time.RFC3339),
	})
}

// ============ 档案配置 API ============

// listRetentionPeriods 获取保管期限列表
func (h *Handler) listRetentionPeriods(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var periods []models.RetentionPeriod
	if err := h.db.Where("user_id = ? OR user_id IS NULL", userID).
		Order("sort_order, id").
		Find(&periods).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, periods)
}

// createRetentionPeriod 创建保管期限
func (h *Handler) createRetentionPeriod(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		Name      string `json:"name" binding:"required"`
		Years     int    `json:"years"`
		SortOrder int    `json:"sort_order"`
		IsDefault bool   `json:"is_default"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	period := models.RetentionPeriod{
		UserID:    &userID,
		Name:      payload.Name,
		Years:     payload.Years,
		SortOrder: payload.SortOrder,
		IsDefault: payload.IsDefault,
	}

	if err := h.db.Create(&period).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, period)
}

// updateRetentionPeriod 更新保管期限
func (h *Handler) updateRetentionPeriod(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	periodID := chi.URLParam(r, "periodID")

	var period models.RetentionPeriod
	if err := h.db.Where("id = ? AND user_id = ?", periodID, userID).First(&period).Error; err != nil {
		http.Error(w, "保管期限不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Name      string `json:"name"`
		Years     int    `json:"years"`
		SortOrder int    `json:"sort_order"`
		IsDefault bool   `json:"is_default"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	period.Name = payload.Name
	period.Years = payload.Years
	period.SortOrder = payload.SortOrder
	period.IsDefault = payload.IsDefault

	if err := h.db.Save(&period).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, period)
}

// deleteRetentionPeriod 删除保管期限
func (h *Handler) deleteRetentionPeriod(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	periodID := chi.URLParam(r, "periodID")

	var period models.RetentionPeriod
	if err := h.db.Where("id = ? AND user_id = ?", periodID, userID).First(&period).Error; err != nil {
		http.Error(w, "保管期限不存在", http.StatusNotFound)
		return
	}

	if err := h.db.Delete(&period).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// listStorageLocations 获取存档地点列表
func (h *Handler) listStorageLocations(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var locations []models.StorageLocation
	if err := h.db.Where("user_id = ? OR user_id IS NULL", userID).
		Order("sort_order, id").
		Find(&locations).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, locations)
}

// createStorageLocation 创建存档地点
func (h *Handler) createStorageLocation(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	location := models.StorageLocation{
		UserID:      &userID,
		Name:        payload.Name,
		Description: payload.Description,
		SortOrder:   payload.SortOrder,
	}

	if err := h.db.Create(&location).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, location)
}

// updateStorageLocation 更新存档地点
func (h *Handler) updateStorageLocation(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	locationID := chi.URLParam(r, "locationID")

	var location models.StorageLocation
	if err := h.db.Where("id = ? AND user_id = ?", locationID, userID).First(&location).Error; err != nil {
		http.Error(w, "存档地点不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	location.Name = payload.Name
	location.Description = payload.Description
	location.SortOrder = payload.SortOrder

	if err := h.db.Save(&location).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, location)
}

// deleteStorageLocation 删除存档地点
func (h *Handler) deleteStorageLocation(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	locationID := chi.URLParam(r, "locationID")

	var location models.StorageLocation
	if err := h.db.Where("id = ? AND user_id = ?", locationID, userID).First(&location).Error; err != nil {
		http.Error(w, "存档地点不存在", http.StatusNotFound)
		return
	}

	if err := h.db.Delete(&location).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// listCodeRules 获取编码规则列表
func (h *Handler) listCodeRules(w http.ResponseWriter, r *http.Request) {
	log.Printf("[listCodeRules] request received")

	// 先检查表结构是否有 user_id 字段
	var rules []models.CodeRule
	result := h.db.Order("id").Find(&rules)
	
	// 如果查询失败，尝试添加 user_id 字段后重试
	if result.Error != nil && strings.Contains(result.Error.Error(), "user_id") {
		log.Printf("[listCodeRules] 尝试添加 user_id 字段")
		h.db.Exec("ALTER TABLE code_rules ADD COLUMN IF NOT EXISTS user_id INTEGER DEFAULT NULL")
		h.db.Exec("CREATE INDEX IF NOT EXISTS idx_code_rules_user_id ON code_rules(user_id)")
		rules = []models.CodeRule{}
		result = h.db.Order("id").Find(&rules)
	}

	if result.Error != nil {
		log.Printf("[listCodeRules] query error: %v", result.Error)
		http.Error(w, fmt.Sprintf("查询失败: %v", result.Error), http.StatusInternalServerError)
		return
	}

	log.Printf("[listCodeRules] found %d rules", len(rules))
	writeJSON(w, rules)
}

// createCodeRule 创建编码规则
func (h *Handler) createCodeRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	log.Printf("[createCodeRule] userID=%d", userID)

	var payload struct {
		Name      string `json:"name" binding:"required"`
		CodeFormat string `json:"code_format" binding:"required"`
		Separator string `json:"separator"`
		Description string `json:"description"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[createCodeRule] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[createCodeRule] payload: name=%s, code_format=%s", payload.Name, payload.CodeFormat)

	rule := models.CodeRule{
		Name:        payload.Name,
		CodeFormat:  payload.CodeFormat,
		Separator:   payload.Separator,
		Description: payload.Description,
	}

	if rule.Separator == "" {
		rule.Separator = "-"
	}

	if err := h.db.Create(&rule).Error; err != nil {
		log.Printf("[createCodeRule] create error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[createCodeRule] created rule id=%d", rule.ID)
	writeJSON(w, rule)
}

// updateCodeRule 更新编码规则
func (h *Handler) updateCodeRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	ruleID := chi.URLParam(r, "ruleID")
	log.Printf("[updateCodeRule] userID=%d, ruleID=%s", userID, ruleID)

	var rule models.CodeRule
	if err := h.db.Where("id = ? AND user_id = ?", ruleID, userID).First(&rule).Error; err != nil {
		log.Printf("[updateCodeRule] rule not found: %v", err)
		http.Error(w, "编码规则不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Name        string `json:"name"`
		CodeFormat  string `json:"code_format"`
		Separator   string `json:"separator"`
		Description string `json:"description"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[updateCodeRule] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Name != "" {
		rule.Name = payload.Name
	}
	if payload.CodeFormat != "" {
		rule.CodeFormat = payload.CodeFormat
	}
	if payload.Separator != "" {
		rule.Separator = payload.Separator
	}
	if payload.Description != "" {
		rule.Description = payload.Description
	}

	if err := h.db.Save(&rule).Error; err != nil {
		log.Printf("[updateCodeRule] save error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[updateCodeRule] updated rule id=%d", rule.ID)
	writeJSON(w, rule)
}

// deleteCodeRule 删除编码规则
func (h *Handler) deleteCodeRule(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	ruleID := chi.URLParam(r, "ruleID")

	var rule models.CodeRule
	if err := h.db.Where("id = ? AND user_id = ?", ruleID, userID).First(&rule).Error; err != nil {
		http.Error(w, "编码规则不存在", http.StatusNotFound)
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// generateDocumentCodeByRule 根据编码规则生成档案编号
// 支持占位符: {CATEGORY}, {SUBCATEGORY}, {YEAR}, {MONTH}, {SEQ}
// 格式示例: WS-GW-2026-01, WS-2026-04-001
func generateDocumentCodeByRule(db *gorm.DB, categoryCode, subCategoryCode string, year int) (string, int, error) {
	var maxSeq int

	// 获取该二级分类该年度的最大序号（每个二级分类独立递增）
	if err := db.Model(&models.Document{}).
		Where("sub_category_code = ? AND year = ?", subCategoryCode, year).
		Select("COALESCE(MAX(sequence), 0)").
		Scan(&maxSeq).Error; err != nil {
		return "", 0, err
	}

	seq := maxSeq + 1
	// 默认格式: 一级分类-二级分类-年份-序号
	code := fmt.Sprintf("%s-%s-%d-%03d", categoryCode, subCategoryCode, year, seq)

	return code, seq, nil
}

// generateDocumentCodeByPattern 根据编码规则Pattern生成档案编号
// 支持占位符: {CATEGORY}, {SUBCATEGORY}, {YEAR}, {MONTH}, {SEQ}
func generateDocumentCodeByPattern(pattern string, categoryCode, subCategoryCode string, year, month int, seq int) string {
	now := time.Now()
	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}

	code := pattern
	code = strings.ReplaceAll(code, "{CATEGORY}", categoryCode)
	code = strings.ReplaceAll(code, "{SUBCATEGORY}", subCategoryCode)
	code = strings.ReplaceAll(code, "{YEAR}", strconv.Itoa(year))
	code = strings.ReplaceAll(code, "{MONTH}", fmt.Sprintf("%02d", month))
	code = strings.ReplaceAll(code, "{SEQ}", fmt.Sprintf("%03d", seq))

	return code
}

// getCodeRulePreview 获取编码规则预览
func (h *Handler) getCodeRulePreview(w http.ResponseWriter, r *http.Request) {
	categoryCode := r.URL.Query().Get("category_code")
	subCategoryCode := r.URL.Query().Get("sub_category_code")
	year := r.URL.Query().Get("year")

	if categoryCode == "" || subCategoryCode == "" {
		http.Error(w, "category_code 和 sub_category_code 不能为空", http.StatusBadRequest)
		return
	}

	if year == "" {
		year = strconv.Itoa(time.Now().Year())
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		http.Error(w, "无效的年份", http.StatusBadRequest)
		return
	}

	// 生成示例编号
	sampleCode, seq, err := generateDocumentCodeByRule(h.db, categoryCode, subCategoryCode, yearInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"sample_code": sampleCode,
		"next_sequence": seq,
		"year": yearInt,
		"format": fmt.Sprintf("%s-%s-{年份}-{序号}", categoryCode, subCategoryCode),
	})
}

// listArchiveConfig 获取档案全局配置
func (h *Handler) listArchiveConfig(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var config models.ArchiveConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			config = models.ArchiveConfig{
				UserID:           &userID,
				AutoGenerateCode: true,
				RequireCodePrefix: true,
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, config)
}

// updateArchiveConfig 更新档案全局配置
func (h *Handler) updateArchiveConfig(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var payload struct {
		DefaultCodeRuleID  *uint `json:"default_code_rule_id"`
		AutoGenerateCode   bool  `json:"auto_generate_code"`
		RequireCodePrefix  bool  `json:"require_code_prefix"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var config models.ArchiveConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			config = models.ArchiveConfig{
				UserID:             &userID,
				DefaultCodeRuleID:  payload.DefaultCodeRuleID,
				AutoGenerateCode:   payload.AutoGenerateCode,
				RequireCodePrefix:  payload.RequireCodePrefix,
			}
			if err := h.db.Create(&config).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		config.DefaultCodeRuleID = payload.DefaultCodeRuleID
		config.AutoGenerateCode = payload.AutoGenerateCode
		config.RequireCodePrefix = payload.RequireCodePrefix
		if err := h.db.Save(&config).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, config)
}

// ============ 一级/二级分类编码规则 API ============

// createCategoryCode 创建一级分类编码
func (h *Handler) createCategoryCode(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录

	var payload struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查编码是否已被使用
	var existing models.DocumentCategory
	if err := h.db.Where("code = ?", payload.Code).First(&existing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "该分类编码已存在", http.StatusBadRequest)
		return
	}

	category := models.DocumentCategory{
		Code:        payload.Code,
		Name:        payload.Name,
		Description: payload.Description,
		SortOrder:   payload.SortOrder,
	}

	if err := h.db.Create(&category).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, category)
}

// updateCategoryCode 更新一级分类编码
func (h *Handler) updateCategoryCode(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录
	categoryID := chi.URLParam(r, "categoryID")

	var category models.DocumentCategory
	if err := h.db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		http.Error(w, "分类不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查编码是否已被其他分类使用
	var existing models.DocumentCategory
	if err := h.db.Where("code = ? AND id != ?", payload.Code, categoryID).First(&existing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	category.Code = payload.Code
	if payload.Name != "" {
		category.Name = payload.Name
	}
	category.Description = payload.Description
	category.SortOrder = payload.SortOrder

	if err := h.db.Save(&category).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, category)
}

// deleteCategory 删除一级分类
func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录
	categoryID := chi.URLParam(r, "categoryID")

	var category models.DocumentCategory
	if err := h.db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		http.Error(w, "分类不存在", http.StatusNotFound)
		return
	}

	// 检查是否有档案使用该分类
	var count int64
	h.db.Model(&models.Document{}).Where("category_code = ?", category.Code).Count(&count)
	if count > 0 {
		http.Error(w, fmt.Sprintf("该分类下有 %d 个档案，无法删除", count), http.StatusBadRequest)
		return
	}

	// 检查是否有二级分类
	var subCount int64
	h.db.Model(&models.DocumentSubCategory{}).Where("category_id = ?", categoryID).Count(&subCount)
	if subCount > 0 {
		http.Error(w, fmt.Sprintf("该分类下有 %d 个二级分类，请先删除二级分类", subCount), http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(&category).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// createSubCategory 创建二级分类
func (h *Handler) createSubCategory(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录

	var payload struct {
		CategoryID  uint   `json:"category_id" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查一级分类是否存在
	var category models.DocumentCategory
	if err := h.db.Where("id = ?", payload.CategoryID).First(&category).Error; err != nil {
		http.Error(w, "一级分类不存在", http.StatusNotFound)
		return
	}

	// 检查编码是否已在该一级分类下被使用
	var existing models.DocumentSubCategory
	if err := h.db.Where("category_id = ? AND code = ?", payload.CategoryID, payload.Code).First(&existing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	subCategory := models.DocumentSubCategory{
		CategoryID:  payload.CategoryID,
		Code:        payload.Code,
		Name:        payload.Name,
		Description: payload.Description,
		SortOrder:   payload.SortOrder,
	}

	if err := h.db.Create(&subCategory).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 为新创建的二级分类创建默认字段
	createDefaultFieldsForSubCategory(h.db, &subCategory, category.Code)

	writeJSON(w, subCategory)
}

// createDefaultFieldsForSubCategory 为二级分类创建默认字段
func createDefaultFieldsForSubCategory(db *gorm.DB, subCategory *models.DocumentSubCategory, categoryCode string) {
	// 默认字段定义
	defaultFields := []struct {
		FieldName  string
		FieldLabel string
		FieldType  string
		Required   bool
		Visible    bool
		Editable   bool
		SortOrder  int
	}{
		{"file_name", "文件名称", "text", true, true, true, 1},
		{"summary", "摘要描述", "textarea", false, true, true, 2},
		{"signed_date", "签署日期", "date", true, true, true, 3},
		{"retention_period", "保管期限", "select", true, true, true, 4},
		{"storage_location", "存放位置", "text", false, true, true, 5},
		{"remarks", "备注说明", "textarea", false, true, true, 6},
	}

	// 根据一级分类添加特有字段
	extraFields := []struct {
		FieldName  string
		FieldLabel string
		FieldType  string
		Required   bool
		Visible    bool
		Editable   bool
		SortOrder  int
	}{}

	switch categoryCode {
	case "WS": // 文书档案特有字段
		extraFields = []struct {
			FieldName  string
			FieldLabel string
			FieldType  string
			Required   bool
			Visible    bool
			Editable   bool
			SortOrder  int
		}{
			{"party_a", "甲方", "text", false, true, true, 10},
			{"party_b", "乙方", "text", false, true, true, 11},
			{"amount", "金额", "number", false, true, true, 12},
		}
	case "KJ": // 科技档案特有字段
		extraFields = []struct {
			FieldName  string
			FieldLabel string
			FieldType  string
			Required   bool
			Visible    bool
			Editable   bool
			SortOrder  int
		}{
			{"project_name", "项目名称", "text", false, true, true, 10},
			{"design_unit", "设计单位", "text", false, true, true, 11},
			{"designer", "设计人员", "text", false, true, true, 12},
		}
	case "DZ": // 电子档案特有字段
		extraFields = []struct {
			FieldName  string
			FieldLabel string
			FieldType  string
			Required   bool
			Visible    bool
			Editable   bool
			SortOrder  int
		}{
			{"capture_date", "拍摄日期", "date", false, true, true, 10},
			{"capturer", "拍摄人", "text", false, true, true, 11},
			{"content_description", "内容描述", "textarea", false, true, true, 12},
		}
	}

	// 创建默认字段
	for _, f := range defaultFields {
		field := models.ArchiveFieldDefinition{
			SubCategoryID: subCategory.ID,
			FieldName:     f.FieldName,
			FieldLabel:    f.FieldLabel,
			FieldType:     f.FieldType,
			Required:      f.Required,
			Visible:       f.Visible,
			Editable:      f.Editable,
			SortOrder:     f.SortOrder,
		}
		db.Create(&field)
	}

	// 创建特有字段
	for _, f := range extraFields {
		field := models.ArchiveFieldDefinition{
			SubCategoryID: subCategory.ID,
			FieldName:     f.FieldName,
			FieldLabel:    f.FieldLabel,
			FieldType:     f.FieldType,
			Required:      f.Required,
			Visible:       f.Visible,
			Editable:      f.Editable,
			SortOrder:     f.SortOrder,
		}
		db.Create(&field)
	}
}

// updateSubCategoryCode 更新二级分类编码
func (h *Handler) updateSubCategoryCode(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录
	subCategoryID := chi.URLParam(r, "subCategoryID")

	var subCategory models.DocumentSubCategory
	if err := h.db.Where("id = ?", subCategoryID).First(&subCategory).Error; err != nil {
		http.Error(w, "二级分类不存在", http.StatusNotFound)
		return
	}

	var payload struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查编码是否已在该一级分类下被其他二级分类使用
	var existing models.DocumentSubCategory
	if err := h.db.Where("category_id = ? AND code = ? AND id != ?", subCategory.CategoryID, payload.Code, subCategoryID).First(&existing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	subCategory.Code = payload.Code
	if payload.Name != "" {
		subCategory.Name = payload.Name
	}
	subCategory.Description = payload.Description
	subCategory.SortOrder = payload.SortOrder

	if err := h.db.Save(&subCategory).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, subCategory)
}

// deleteSubCategory 删除二级分类
func (h *Handler) deleteSubCategory(w http.ResponseWriter, r *http.Request) {
	_ = getUserIDFromContext(r) // 验证用户登录
	subCategoryID := chi.URLParam(r, "subCategoryID")

	var subCategory models.DocumentSubCategory
	if err := h.db.Where("id = ?", subCategoryID).First(&subCategory).Error; err != nil {
		http.Error(w, "二级分类不存在", http.StatusNotFound)
		return
	}

	// 检查是否有档案使用该分类
	var count int64
	h.db.Model(&models.Document{}).Where("category_code = ? AND sub_category_code = ?", subCategory.CategoryID, subCategory.Code).Count(&count)
	if count > 0 {
		http.Error(w, fmt.Sprintf("该分类下有 %d 个档案，无法删除", count), http.StatusBadRequest)
		return
	}

	if err := h.db.Where("sub_category_id = ?", subCategoryID).Delete(&models.ArchiveFieldDefinition{}).Error; err != nil {
		http.Error(w, fmt.Sprintf("级联删除关联字段失败: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.db.Delete(&subCategory).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "删除成功"})
}

// listSharedFields 获取所有共用字段定义
// GET /api/archives/shared-fields
func (h *Handler) listSharedFields(w http.ResponseWriter, r *http.Request) {
	var fields []models.ArchiveSharedField
	if err := h.db.Order("sort_order ASC").Find(&fields).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, fields)
}

// listColumnConfig 获取用户的列配置
// GET /api/archives/column-config?sub_category_id=xxx
func (h *Handler) listColumnConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	subCategoryIDStr := r.URL.Query().Get("sub_category_id")
	if subCategoryIDStr == "" {
		respondError(w, http.StatusBadRequest, "sub_category_id is required", nil)
		return
	}

	subCategoryID, err := strconv.ParseUint(subCategoryIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sub_category_id", err)
		return
	}

	var config models.TypeDefaultColumn
	result := h.db.Where("sub_category_id = ? AND user_id = ?", uint(subCategoryID), userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 返回空配置，前端可使用默认列
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"column_keys": []string{},
				"is_default":  true,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load config", result.Error)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// saveColumnConfig 保存用户的列配置
// POST /api/archives/column-config
// PUT /api/archives/column-config
func (h *Handler) saveColumnConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req struct {
		SubCategoryID uint            `json:"sub_category_id"`
		ColumnKeys    datatypes.JSON  `json:"column_keys"` // JSON array
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.SubCategoryID == 0 {
		respondError(w, http.StatusBadRequest, "sub_category_id is required", nil)
		return
	}

	if len(req.ColumnKeys) == 0 {
		respondError(w, http.StatusBadRequest, "column_keys is required", nil)
		return
	}

	// upsert: 先查询是否存在
	var config models.TypeDefaultColumn
	result := h.db.Where("sub_category_id = ? AND user_id = ?", req.SubCategoryID, userID).First(&config)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 创建新记录
			config.SubCategoryID = req.SubCategoryID
			config.UserID = userID
			config.ColumnKeys = req.ColumnKeys
			if err := h.db.Create(&config).Error; err != nil {
				respondError(w, http.StatusInternalServerError, "failed to save config", err)
				return
			}
		} else {
			respondError(w, http.StatusInternalServerError, "failed to query config", result.Error)
			return
		}
	} else {
		// 更新现有记录
		config.ColumnKeys = req.ColumnKeys
		if err := h.db.Save(&config).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update config", err)
			return
		}
	}

	respondJSON(w, http.StatusOK, config)
}

// ============================================================
// 标签管理 API（P1 — 知识组织）
// ============================================================

// listArchiveTags GET /api/archives/tags
func (h *Handler) listArchiveTags(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	tags, err := h.tagService.ListTags(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list tags", err)
		return
	}
	if tags == nil {
		tags = []models.TagWithCount{}
	}
	respondJSON(w, http.StatusOK, tags)
}

// createArchiveTag POST /api/archives/tags
func (h *Handler) createArchiveTag(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, "tag name is required", nil)
		return
	}
	if req.Color == "" {
		req.Color = "#3b82f6"
	}

	tag, err := h.tagService.CreateTag(userID, req.Name, req.Color)
	if err != nil {
		respondError(w, http.StatusConflict, "failed to create tag", err)
		return
	}
	respondJSON(w, http.StatusCreated, tag)
}

// deleteArchiveTag DELETE /api/archives/tags/{tagID}
func (h *Handler) deleteArchiveTag(w http.ResponseWriter, r *http.Request) {
	tagIDStr := chi.URLParam(r, "tagID")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID", err)
		return
	}
	if err := h.tagService.DeleteTag(uint(tagID)); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete tag", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "tag deleted"})
}

// setDocumentTags POST /api/archives/documents/{docID}/tags
func (h *Handler) setDocumentTags(w http.ResponseWriter, r *http.Request) {
	docIDStr := chi.URLParam(r, "docID")
	docID, err := strconv.ParseUint(docIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID", err)
		return
	}

	var req struct {
		TagNames []string `json:"tag_names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.tagService.SetDocumentTags(uint(docID), req.TagNames); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to set tags", err)
		return
	}

	// 同步更新旧 tags JSON 字段（向后兼容）
	tagsJSON, _ := json.Marshal(req.TagNames)
	h.db.Model(&models.Document{}).Where("id = ?", docID).Update("tags", string(tagsJSON))

	respondJSON(w, http.StatusOK, map[string]interface{}{"document_id": docID, "tag_names": req.TagNames})
}

// getDocumentTags GET /api/archives/documents/{docID}/tags
func (h *Handler) getDocumentTags(w http.ResponseWriter, r *http.Request) {
	docIDStr := chi.URLParam(r, "docID")
	docID, err := strconv.ParseUint(docIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID", err)
		return
	}
	tags, err := h.tagService.GetDocumentTags(uint(docID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get tags", err)
		return
	}
	respondJSON(w, http.StatusOK, tags)
}

// ============================================================
// 文件夹树 API（P1 — 知识组织）
// ============================================================

// getFolderTree GET /api/archives/folders?category_code=WS
func (h *Handler) getFolderTree(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	categoryCode := r.URL.Query().Get("category_code")

	tree, err := service.BuildFolderTree(h.db, userID, categoryCode)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to build folder tree", err)
		return
	}
	respondJSON(w, http.StatusOK, tree)
}

// updateDocumentFolder PUT /api/archives/documents/{docID}/folder
func (h *Handler) updateDocumentFolder(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	docIDStr := chi.URLParam(r, "docID")
	docID, err := strconv.ParseUint(docIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID", err)
		return
	}

	var req struct {
		FolderPath string `json:"folder_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// 验证文档存在且属于用户
	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		respondError(w, http.StatusNotFound, "document not found", err)
		return
	}

	// 更新 folder_path
	if err := h.db.Model(&doc).Update("folder_path", req.FolderPath).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update folder", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"document_id": doc.ID,
		"folder_path": req.FolderPath,
	})
}

// ============================================================
// 知识库配置 API（P1 — Category 扩展字段）
// ============================================================

// updateCategoryKBConfig PUT /api/archives/categories/{categoryID}/kb-config
func (h *Handler) updateCategoryKBConfig(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := chi.URLParam(r, "categoryID")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid category ID", err)
		return
	}

	var req struct {
		EmbeddingModelID *uint           `json:"embedding_model_id"`
		IndexingStrategy *datatypes.JSON `json:"indexing_strategy"`
		ChunkingConfig   *datatypes.JSON `json:"chunking_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	var category models.DocumentCategory
	if err := h.db.First(&category, categoryID).Error; err != nil {
		respondError(w, http.StatusNotFound, "category not found", err)
		return
	}

	updates := map[string]interface{}{}
	if req.EmbeddingModelID != nil {
		updates["embedding_model_id"] = *req.EmbeddingModelID
	}
	if req.IndexingStrategy != nil {
		updates["indexing_strategy"] = *req.IndexingStrategy
	}
	if req.ChunkingConfig != nil {
		updates["chunking_config"] = *req.ChunkingConfig
	}

	if len(updates) == 0 {
		respondError(w, http.StatusBadRequest, "no fields to update", nil)
		return
	}

	if err := h.db.Model(&category).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update KB config", err)
		return
	}

	// 返回更新后的分类
	h.db.First(&category, categoryID)
	respondJSON(w, http.StatusOK, category)
}

// ============ 分块管理 API ============

// listDocumentChunks 获取指定文档的分块列表
// GET /api/archives/documents/{docID}/chunks
func (h *Handler) listDocumentChunks(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	docIDStr := chi.URLParam(r, "docID")
	docID, err := strconv.ParseUint(docIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID", err)
		return
	}

	// 验证文档属于当前用户
	var doc models.Document
	if err := h.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "document not found", nil)
		} else {
			respondError(w, http.StatusInternalServerError, "failed to verify document", err)
		}
		return
	}

	var chunks []models.DocumentChunk
	if err := h.db.Where("doc_id = ?", docID).Order("chunk_index ASC").Find(&chunks).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list chunks", err)
		return
	}

	respondJSON(w, http.StatusOK, chunks)
}

// updateDocumentChunk 更新分块内容并触发向量重索引
// PUT /api/archives/chunks/{chunkID}
func (h *Handler) updateDocumentChunk(w http.ResponseWriter, r *http.Request) {
	editorID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	chunkIDStr := chi.URLParam(r, "chunkID")
	chunkID, err := strconv.ParseUint(chunkIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chunk ID", err)
		return
	}

	var req struct {
		Content          string `json:"content"`
		ExpectedRevision int    `json:"expected_revision"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := h.chunkService.UpdateChunk(uint(chunkID), editorID, req.Content, req.ExpectedRevision); err != nil {
		if strings.Contains(err.Error(), "revision conflict") {
			respondError(w, http.StatusConflict, err.Error(), err)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update chunk", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "chunk updated"})
}

// listChunkRevisions 获取分块的所有版本快照
// GET /api/archives/chunks/{chunkID}/revisions
func (h *Handler) listChunkRevisions(w http.ResponseWriter, r *http.Request) {
	chunkIDStr := chi.URLParam(r, "chunkID")
	chunkID, err := strconv.ParseUint(chunkIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chunk ID", err)
		return
	}

	revisions, err := h.chunkService.ListChunkRevisions(uint(chunkID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list chunk revisions", err)
		return
	}

	respondJSON(w, http.StatusOK, revisions)
}

// revertDocumentChunk 将分块回滚到指定历史版本
// POST /api/archives/chunks/{chunkID}/revert
func (h *Handler) revertDocumentChunk(w http.ResponseWriter, r *http.Request) {
	editorID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	chunkIDStr := chi.URLParam(r, "chunkID")
	chunkID, err := strconv.ParseUint(chunkIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chunk ID", err)
		return
	}

	var req struct {
		TargetRevision int `json:"target_revision"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := h.chunkService.RevertChunk(uint(chunkID), editorID, req.TargetRevision); err != nil {
		if strings.Contains(err.Error(), "revision conflict") {
			respondError(w, http.StatusConflict, err.Error(), err)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revert chunk", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "chunk reverted"})
}
