package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/service/storage"
)

type Handler struct {
	db               *gorm.DB
	process          *service.Processor
	ocrService       *service.OCRService
	embeddingService *service.EmbeddingService
	retrievalService *service.RetrievalService
	chatService      *service.ChatService
	storageRouter    *storage.StorageRouter
	uploadBaseDir    string
	uploadBaseURL    string
}

type batchUploadItem struct {
	FileName     string        `json:"file_name"`
	OriginalName string        `json:"original_name"`
	Scheme       models.Scheme `json:"scheme"`
	Part         models.Part   `json:"part"`
	Imported     int           `json:"imported"`
	Error        string        `json:"error,omitempty"`
}

const (
	socialInsuranceChangeTypeIncrease = "increase"
	socialInsuranceChangeTypeDecrease = "decrease"
)

var insuranceTemplateFiles = map[string]string{
	"increase": "社保缴费人员增加申报（企业职工批量新参保）模版.xls",
	"decrease": "社保缴费人员减少申报（企业职工批量减少参保）模板.xls",
}

type socialInsuranceImportRecord struct {
	EmployeeName   string            `json:"employee_name"`
	Department     string            `json:"department"`
	IdentityNumber string            `json:"identity_number"`
	PersonalNumber string            `json:"personal_number"`
	EffectiveDate  string            `json:"effective_date"`
	Reason         string            `json:"reason"`
	TemplateValues map[string]string `json:"template_values"`
}

type socialInsuranceImportPayload struct {
	Records []socialInsuranceImportRecord `json:"records"`
}

type socialInsuranceBatchResponse struct {
	ID               uint      `json:"id"`
	ChangeType       string    `json:"change_type"`
	OriginalFileName string    `json:"original_file_name"`
	StoredFileName   string    `json:"stored_file_name"`
	StoredFilePath   string    `json:"stored_file_path"`
	CreatedAt        time.Time `json:"created_at"`
}

type socialInsuranceRecordResponse struct {
	ID               uint              `json:"id"`
	BatchID          uint              `json:"batch_id"`
	ChangeType       string            `json:"change_type"`
	EmployeeName     string            `json:"employee_name"`
	Department       string            `json:"department"`
	IdentityNumber   string            `json:"identity_number"`
	PersonalNumber   string            `json:"personal_number"`
	EffectiveDate    string            `json:"effective_date"`
	Reason           string            `json:"reason"`
	TemplateValues   map[string]string `json:"template_values"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	OriginalFileName string            `json:"original_file_name"`
}

type socialInsuranceImportResponse struct {
	Batch   socialInsuranceBatchResponse    `json:"batch"`
	Records []socialInsuranceRecordResponse `json:"records"`
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

func getMultipartFileHeader(r *http.Request, field string) (*multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
	}
	files := r.MultipartForm.File[field]
	if len(files) == 0 {
		return nil, errors.New("missing file")
	}
	return files[0], nil
}

func NewHandler(db *gorm.DB) *Handler {
	uploadDir := resolveUploadBaseDir()
	uploadURL := os.Getenv("SIAPP_UPLOAD_BASE_URL")
	if uploadURL == "" {
		uploadURL = "/api"
	}

	embSvc := service.NewEmbeddingService(db)
	retSvc := service.NewRetrievalService(db, embSvc)
	chatSvc := service.NewChatService(db, retSvc)

	return &Handler{
		db:               db,
		process:          service.NewProcessor(db),
		ocrService:       service.NewOCRService(db),
		embeddingService: embSvc,
		retrievalService: retSvc,
		chatService:      chatSvc,
		storageRouter:    storage.NewStorageRouter(db),
		uploadBaseDir:    uploadDir,
		uploadBaseURL:    uploadURL,
	}
}

func resolveUploadBaseDir() string {
	if dir := strings.TrimSpace(os.Getenv("SIAPP_UPLOAD_DIR")); dir != "" {
		return filepath.Clean(dir)
	}

	if dbPath := strings.TrimSpace(os.Getenv("SIAPP_DATABASE_PATH")); dbPath != "" {
		if !filepath.IsAbs(dbPath) {
			if abs, err := filepath.Abs(dbPath); err == nil {
				dbPath = abs
			}
		}
		base := filepath.Dir(dbPath)
		return filepath.Join(base, "uploads")
	}

	return "./uploads"
}

func resolveTemplateBaseDir() string {
	if dir := strings.TrimSpace(os.Getenv("TEMPLATE_BASE_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	return "./templates"
}

func (h *Handler) ensureUploadDir(subdirs ...string) (string, error) {
	segments := append([]string{h.uploadBaseDir}, subdirs...)
	dir := filepath.Join(segments...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (h *Handler) uploadPath(subdirs ...string) string {
	segments := append([]string{h.uploadBaseDir}, subdirs...)
	return filepath.Join(segments...)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/periods", h.listPeriods)
	r.Post("/periods", h.createPeriod)
	r.Get("/roster-template", h.downloadRosterTemplate)
	r.Get("/employees", h.listEmployees)
	r.Post("/employees/import", h.importEmployees)
	r.Post("/employees/delete", h.deleteEmployees)
	r.Post("/employees/resigned/import", h.importResignedEmployees)
	r.Get("/employees/template", h.downloadEmployeeTemplate)
	r.Get("/employees/resigned/template", h.downloadResignedEmployeeTemplate)
	r.Post("/employees/export", h.exportEmployees)
	r.Post("/employees/{employeeID}/resign", h.resignEmployee)
	r.Get("/employees/{employeeID}/resign-proof", h.downloadResignProof)
	r.Post("/employees/restore", h.restoreEmployees)
	r.Get("/social-insurance/changes", h.listSocialInsuranceChanges)
	r.Get("/social-insurance/options", h.getSocialInsuranceOptions)
	r.Post("/social-insurance/changes", h.createSocialInsuranceChange)
	r.Put("/social-insurance/changes/{changeID}", h.updateSocialInsuranceChange)
	r.Post("/social-insurance/changes/import", h.importSocialInsuranceChanges)
	r.Post("/social-insurance/changes/delete", h.deleteSocialInsuranceChanges)
	r.Get("/callback-records", h.listCallbackRecords)
	r.Post("/callback-records/upload", h.uploadCallbackRecords)
	r.Delete("/callback-records", h.clearCallbackRecords)
	r.Get("/insurance-template", h.downloadInsuranceTemplate)

	r.Route("/dormitories", func(dr chi.Router) {
		dr.Get("/sites", h.listDormSites)
		dr.Post("/sites", h.createDormSite)
		dr.Put("/sites/{siteID}", h.updateDormSite)
		dr.Delete("/sites/{siteID}", h.deleteDormSite)

		dr.Get("/buildings", h.listDormBuildings)
		dr.Post("/buildings", h.createDormBuilding)
		dr.Put("/buildings/{buildingID}", h.updateDormBuilding)
		dr.Delete("/buildings/{buildingID}", h.deleteDormBuilding)

		dr.Get("/rooms", h.listDormRooms)
		dr.Post("/rooms", h.createDormRoom)
		dr.Put("/rooms/{roomID}", h.updateDormRoom)
		dr.Delete("/rooms/{roomID}", h.deleteDormRoom)

		dr.Get("/beds", h.listDormBeds)
		dr.Post("/beds", h.createDormBed)
		dr.Put("/beds/{bedID}", h.updateDormBed)
		dr.Delete("/beds/{bedID}", h.deleteDormBed)

		dr.Get("/assets", h.listDormAssets)
		dr.Post("/assets", h.createDormAsset)
		dr.Put("/assets/{assetID}", h.updateDormAsset)
		dr.Delete("/assets/{assetID}", h.deleteDormAsset)

		dr.Get("/contracts", h.listDormContracts)
		dr.Post("/contracts", h.createDormContract)
		dr.Put("/contracts/{contractID}", h.updateDormContract)
		dr.Delete("/contracts/{contractID}", h.deleteDormContract)
		dr.Post("/contracts/{contractID}/checkout", h.createDormCheckout)
		dr.Get("/checkouts", h.listDormCheckouts)

		dr.Get("/meter-items", h.listDormMeterItems)
		dr.Post("/meter-items", h.createDormMeterItem)
		dr.Put("/meter-items/{itemID}", h.updateDormMeterItem)
		dr.Delete("/meter-items/{itemID}", h.deleteDormMeterItem)

		dr.Get("/meter-readings", h.listDormMeterReadings)
		dr.Post("/meter-readings", h.createDormMeterReading)
		dr.Put("/meter-readings/{readingID}", h.updateDormMeterReading)
		dr.Delete("/meter-readings/{readingID}", h.deleteDormMeterReading)

		dr.Get("/bills", h.listDormBills)
		dr.Post("/bills", h.createDormBill)
		dr.Put("/bills/{billID}", h.updateDormBillStatus)
	})

	// 档案管理
	r.Route("/archives", func(ar chi.Router) {
		ar.Get("/categories", h.listDocumentCategories)
		ar.Post("/categories", h.createCategoryCode)
		ar.Put("/categories/{categoryID}", h.updateCategoryCode)
		ar.Delete("/categories/{categoryID}", h.deleteCategory)
		ar.Get("/documents", h.listDocuments)
		ar.Post("/documents", h.createDocument)
		ar.Put("/documents/{docID}", h.updateDocument)
		ar.Delete("/documents/{docID}", h.deleteDocument)
		ar.Post("/documents/{docID}/upload", h.uploadDocumentFile)
		ar.Post("/documents/upload-with-ocr", h.uploadDocumentWithOCR)
		ar.Get("/documents/{docID}/download", h.downloadDocumentFile)
		ar.Get("/documents/expiring", h.getExpiringDocuments)
		ar.Post("/documents/batch-upload", h.batchUploadDocuments)
		ar.Post("/documents/batch-download", h.batchDownloadDocuments)
		ar.Post("/documents/share", h.generateShareLink)
		ar.Get("/reminder-settings", h.getExpirationReminderSettings)
		ar.Put("/reminder-settings", h.updateExpirationReminderSettings)

		// 档案字段定义管理
		ar.Get("/field-groups", h.listFieldGroups)
		ar.Post("/field-groups", h.createFieldGroup)
		ar.Put("/field-groups/{groupID}", h.updateFieldGroup)
		ar.Delete("/field-groups/{groupID}", h.deleteFieldGroup)
		ar.Get("/field-definitions", h.listFieldDefinitions)
		ar.Post("/field-definitions", h.createFieldDefinition)
		ar.Put("/field-definitions/{fieldID}", h.updateFieldDefinition)
		ar.Delete("/field-definitions/{fieldID}", h.deleteFieldDefinition)
		ar.Get("/shared-fields", h.listSharedFields)
		ar.Get("/sub-categories/{subCategoryID}/fields", h.getFieldsBySubCategory)

		// 二级分类管理
		ar.Post("/sub-categories", h.createSubCategory)
		ar.Put("/sub-categories/{subCategoryID}", h.updateSubCategoryCode)
		ar.Delete("/sub-categories/{subCategoryID}", h.deleteSubCategory)

		// 保管期限配置
		ar.Get("/retention-periods", h.listRetentionPeriods)
		ar.Post("/retention-periods", h.createRetentionPeriod)
		ar.Put("/retention-periods/{periodID}", h.updateRetentionPeriod)
		ar.Delete("/retention-periods/{periodID}", h.deleteRetentionPeriod)

		// 存档地点配置
		ar.Get("/storage-locations", h.listStorageLocations)
		ar.Post("/storage-locations", h.createStorageLocation)
		ar.Put("/storage-locations/{locationID}", h.updateStorageLocation)
		ar.Delete("/storage-locations/{locationID}", h.deleteStorageLocation)

		// 编码规则配置
		ar.Get("/code-rules", h.listCodeRules)
		ar.Post("/code-rules", h.createCodeRule)
		ar.Put("/code-rules/{ruleID}", h.updateCodeRule)
		ar.Delete("/code-rules/{ruleID}", h.deleteCodeRule)
		ar.Get("/code-rules/preview", h.getCodeRulePreview)

		// 档案全局配置
		ar.Get("/config", h.listArchiveConfig)
		ar.Put("/config", h.updateArchiveConfig)

		// 列配置管理
		ar.Get("/column-config", h.listColumnConfig)
		ar.Post("/column-config", h.saveColumnConfig)
		ar.Put("/column-config", h.saveColumnConfig)
	})

	// OCR 服务
	r.Route("/ocr", func(or chi.Router) {
		or.Post("/extract", h.extractOCR)
		or.Post("/extract-async", h.extractOCRAsync)
		or.Get("/jobs/{jobId}", h.getOCRJobStatus)
		or.Get("/jobs/{jobId}/result", h.getOCRJobResult)
	})

	// 知识库
	r.Route("/knowledge", func(kr chi.Router) {
		kr.Post("/ingest", h.ingestDocument)
		kr.Get("/search", h.searchKnowledge)
		kr.Post("/chat", h.chatKnowledge)
		kr.Get("/stats", h.knowledgeStats)
	})

	// 模型配置
	r.Route("/settings/models", func(mr chi.Router) {
		mr.Get("/", h.ListModelConfigs)
		mr.Post("/", h.CreateModelConfig)
		mr.Get("/providers", h.ListBuiltInProviders)
		mr.Get("/fetch-models", h.FetchModelsByEndpoint)
		mr.Get("/usage", h.GetModelUsageStats)
		mr.Get("/usage/trend", h.GetModelUsageTrend)
		mr.Get("/usage/by-model", h.GetModelUsageByModel)
		mr.Delete("/usage/cleanup", h.CleanupOldUsageLogs)
		mr.Put("/{configId}", h.UpdateModelConfig)
		mr.Delete("/{configId}", h.DeleteModelConfig)
		mr.Post("/{configId}/test", h.TestModelConfig)
		mr.Get("/{configId}/available-models", h.ListAvailableModels)
	})

	// 全局搜索
	r.Get("/search/global", h.globalSearch)

	r.Route("/user", h.registerPreferenceRoutes)

	r.Route("/provident-fund", func(pr chi.Router) {
		pr.Get("/records", h.listProvidentRecords)
		pr.Post("/records", h.createProvidentRecord)
		pr.Put("/records/{recordID}", h.updateProvidentRecord)
		pr.Post("/records/{recordID}/seal", h.sealProvidentRecord)
		pr.Post("/records/{recordID}/unseal", h.unsealProvidentRecord)
		pr.Get("/settings", h.getProvidentSettings)
		pr.Put("/settings", h.updateProvidentSettings)
		pr.Post("/bills", h.createProvidentBill)
		pr.Get("/bills", h.listProvidentBills)
		pr.Get("/bills/{billID}", h.getProvidentBill)
		pr.Delete("/bills/{billID}", h.deleteProvidentBill)
	})

	r.Route("/announcements", h.registerAnnouncementRoutes)

	r.Route("/rbac", h.registerRolePermissionRoutes)

	// 系统配置 (管理员)
	r.Route("/admin", func(ar chi.Router) {
		// 存储配置
		ar.Get("/storage", h.getStorageConfig)
		ar.Put("/storage", h.saveStorageConfig)
		ar.Post("/storage/test", h.testStorageConnection)

		// SMTP配置
		ar.Get("/smtp", h.getSMTPConfig)
		ar.Put("/smtp", h.saveSMTPConfig)
		ar.Post("/smtp/test", h.testSMTPConnection)
	})

	r.Route("/admin/storage", func(sr chi.Router) {
		sr.Get("/", h.listStorageConfigs)
		sr.Post("/", h.createStorageConfig)
		sr.Put("/{id}", h.updateStorageConfig)
		sr.Delete("/{id}", h.deleteStorageConfig)
		sr.Get("/{id}/status", h.getStorageStatus)
		sr.Get("/{id}/capacity", h.getStorageCapacity)
		sr.Post("/{id}/set-primary", h.setStoragePrimary)
		sr.Post("/test", h.testStorageConnectionNew)
		sr.Get("/directories", h.listStorageDirectories)
		// 模块配置
		sr.Get("/modules", h.listStorageModules)
		sr.Post("/modules", h.createStorageModule)
		sr.Put("/modules/{id}", h.updateStorageModule)
		sr.Delete("/modules/{id}", h.deleteStorageModule)
		sr.Get("/rules", h.listStorageRules)
		sr.Post("/rules", h.createStorageRule)
		sr.Put("/rules", h.updateStorageRules)
		sr.Put("/rules/{id}", h.updateStorageRule)
		sr.Delete("/rules/{id}", h.deleteStorageRule)
	})
	// /notifications moved to main.go
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

func (h *Handler) downloadInsuranceTemplate(w http.ResponseWriter, r *http.Request) {
	templateType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("type")))
	fileName, ok := insuranceTemplateFiles[templateType]
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid template type", nil)
		return
	}
	baseDir := resolveTemplateBaseDir()
	filePath := filepath.Join(baseDir, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		respondError(w, http.StatusNotFound, "template not found", err)
		return
	}
	escapedName := url.PathEscape(fileName)
	w.Header().Set("Content-Type", "application/vnd.ms-excel")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", escapedName))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("[insurance-template] failed to write response: %v", err)
	}
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

func (h *Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var employees []models.Employee
	if err := h.db.Where("user_id = ?", userID).Order("name ASC, id_number ASC").Find(&employees).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list employees", err)
		return
	}

	for idx := range employees {
		if employees[idx].ResignProofPath != "" {
			employees[idx].ResignProofURL = fmt.Sprintf("/employees/%d/resign-proof", employees[idx].ID)
		} else {
			employees[idx].ResignProofURL = ""
		}
	}

	respondJSON(w, http.StatusOK, employees)
}

func (h *Handler) importEmployees(w http.ResponseWriter, r *http.Request) {
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

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".xlsx"
	}
	filename := fmt.Sprintf("employees-%d-%d%s", userID, time.Now().UnixNano(), ext)

	// Resolve storage path using StorageRouter
	resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employees",
		Filename:     filename,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to resolve storage path", err)
		return
	}

	// Upload file using GlobalManager
	_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, filename, file, header.Size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to upload file", err)
		return
	}

	// For backward compatibility, also store the file path locally
	storedPath := filepath.Join(h.uploadBaseDir, "employees", fmt.Sprintf("%d", userID), filename)
	if err := os.MkdirAll(filepath.Dir(storedPath), 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare employee directory", err)
		return
	}

	// Re-read the file from storage for local backup
	if _, err := file.Seek(0, io.SeekStart); err == nil {
		out, err := os.Create(storedPath)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create file", err)
			return
		}
		if _, err := io.Copy(out, file); err != nil {
			_ = out.Close()
			respondError(w, http.StatusInternalServerError, "failed to save file", err)
			return
		}
		if err := out.Close(); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to finalize file", err)
			return
		}
	}

	mode := service.ParseEmployeeImportMode(r.FormValue("mode"))

	result, err := h.process.ParseEmployeeFile(userID, storedPath, header.Filename, service.EmployeeImportOptions{
		Mode: mode,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to import employees", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) importResignedEmployees(w http.ResponseWriter, r *http.Request) {
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

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".xlsx"
	}
	filename := fmt.Sprintf("resigned-employees-%d-%d%s", userID, time.Now().UnixNano(), ext)

	// Resolve storage path using StorageRouter
	resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employees",
		Filename:     filename,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to resolve storage path", err)
		return
	}

	// Upload file using GlobalManager
	_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, filename, file, header.Size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to upload file", err)
		return
	}

	// For backward compatibility, also store the file path locally
	storedPath := filepath.Join(h.uploadBaseDir, "employees", fmt.Sprintf("%d", userID), filename)
	if err := os.MkdirAll(filepath.Dir(storedPath), 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare employee directory", err)
		return
	}

	// Re-read the file from storage for local backup
	if _, err := file.Seek(0, io.SeekStart); err == nil {
		out, err := os.Create(storedPath)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create file", err)
			return
		}
		if _, err := io.Copy(out, file); err != nil {
			_ = out.Close()
			respondError(w, http.StatusInternalServerError, "failed to save file", err)
			return
		}
		if err := out.Close(); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to finalize file", err)
			return
		}
	}

	mode := service.ParseEmployeeImportMode(r.FormValue("mode"))
	forceTransition := strings.EqualFold(r.FormValue("force_transition"), "true") || strings.EqualFold(r.URL.Query().Get("force_transition"), "true")
	result, err := h.process.ParseEmployeeFile(userID, storedPath, header.Filename, service.EmployeeImportOptions{
		Mode:               mode,
		DefaultStatus:      "resigned",
		ForceDefaultStatus: forceTransition,
	})
	if err != nil {
		var conflictErr *service.EmployeeImportConflictError
		if errors.As(err, &conflictErr) {
			respondJSON(w, http.StatusConflict, map[string]any{
				"error":      "employee_conflicts",
				"message":    "存在与在职员工身份证重复的记录，是否将其转为离职员工？",
				"conflicts":  conflictErr.Conflicts,
				"need_force": true,
			})
			return
		}
		respondError(w, http.StatusBadRequest, "failed to import resigned employees", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) resignEmployee(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	employeeIDStr := strings.TrimSpace(chi.URLParam(r, "employeeID"))
	if employeeIDStr == "" {
		respondError(w, http.StatusBadRequest, "employee id is required", nil)
		return
	}

	employeeID, err := strconv.ParseUint(employeeIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid employee id", err)
		return
	}

	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", employeeID, userID).First(&employee).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load employee", err)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	resignDateRaw := strings.TrimSpace(r.FormValue("resign_date"))
	resignDate, err := normalizeResignDate(resignDateRaw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid resign date", err)
		return
	}

	resignReasonsRaw := strings.TrimSpace(r.FormValue("resign_reasons"))
	var resignReasons []string
	if resignReasonsRaw != "" {
		decoder := json.NewDecoder(strings.NewReader(resignReasonsRaw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&resignReasons); err != nil {
			respondError(w, http.StatusBadRequest, "invalid resign reasons", err)
			return
		}
		normalized := make([]string, 0, len(resignReasons))
		for _, value := range resignReasons {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			normalized = append(normalized, trimmed)
		}
		resignReasons = normalized
	}

	proofFile, header, err := r.FormFile("resign_proof")
	hasProof := err == nil
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		respondError(w, http.StatusBadRequest, "failed to read resign proof", err)
		return
	}

	var originalName string
	var storedPath string
	if hasProof {
		defer proofFile.Close()

		originalName = filepath.Base(header.Filename)
		if originalName == "" {
			respondError(w, http.StatusBadRequest, "resign proof file name is required", nil)
			return
		}

		allowedExtensions := map[string]bool{
			".pdf":  true,
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
			".bmp":  true,
			".webp": true,
		}

		ext := strings.ToLower(filepath.Ext(originalName))
		if !allowedExtensions[ext] {
			respondError(w, http.StatusBadRequest, "离职证明仅支持 PDF 或常见图片格式", nil)
			return
		}

		contentType := ""
		buffer := make([]byte, 512)
		if n, readErr := proofFile.Read(buffer); readErr == nil || errors.Is(readErr, io.EOF) {
			contentType = http.DetectContentType(buffer[:n])
			if seeker, ok := proofFile.(io.Seeker); ok {
				_, _ = seeker.Seek(0, io.SeekStart)
			}
		} else {
			respondError(w, http.StatusBadRequest, "failed to read resign proof", readErr)
			return
		}

		allowedMimePrefixes := []string{"application/pdf", "image/"}
		validMime := false
		for _, prefix := range allowedMimePrefixes {
			if strings.HasPrefix(contentType, prefix) {
				validMime = true
				break
			}
		}
		if !validMime {
			respondError(w, http.StatusBadRequest, "离职证明仅支持 PDF 或常见图片格式", nil)
			return
		}

		storedName := fmt.Sprintf("proof-%d-%d%s", employee.ID, time.Now().UnixNano(), ext)

		// Resolve storage path using StorageRouter
		resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
			ModuleCode:   "archives",
			ResourceType: "resign_proof",
			Filename:     storedName,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to resolve storage path", err)
			return
		}

		// Upload file using GlobalManager
		_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, storedName, proofFile, header.Size)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to upload resign proof", err)
			return
		}

		// For backward compatibility, also store the file path locally
		targetDir := filepath.Join(h.uploadBaseDir, "employees", fmt.Sprintf("%d", userID), "resign", fmt.Sprintf("%d", employee.ID))
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to prepare resign proof directory", err)
			return
		}

		if employee.ResignProofPath != "" {
			if err := os.Remove(employee.ResignProofPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("[resign] remove old proof failed: %v\n", err)
			}
		}

		storedPath = filepath.Join(targetDir, storedName)

		// Re-read the file from storage for local backup
		if _, err := proofFile.Seek(0, io.SeekStart); err == nil {
			out, err := os.Create(storedPath)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to create resign proof", err)
				return
			}
			if _, err := io.Copy(out, proofFile); err != nil {
				_ = out.Close()
				respondError(w, http.StatusInternalServerError, "failed to save resign proof", err)
				return
			}
			if err := out.Close(); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to finalize resign proof", err)
				return
			}
		}
	}

	resignReasonsJSON := ""
	if len(resignReasons) > 0 {
		if data, marshalErr := json.Marshal(resignReasons); marshalErr == nil {
			resignReasonsJSON = string(data)
		}
	}

	updates := map[string]any{
		"status":         "resigned",
		"resign_date":    resignDate,
		"resign_reasons": resignReasonsJSON,
		"updated_at":     time.Now(),
	}

	if hasProof {
		updates["resign_proof_path"] = storedPath
		updates["resign_proof_name"] = originalName
	}

	if err := h.db.Model(&employee).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update employee", err)
		return
	}

	if err := h.db.Where("id = ?", employee.ID).First(&employee).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload employee", err)
		return
	}

	if employee.ResignProofPath != "" {
		employee.ResignProofURL = fmt.Sprintf("/employees/%d/resign-proof", employee.ID)
	}

	respondJSON(w, http.StatusOK, employee)
}

type deleteEmployeesRequest struct {
	IDs []uint `json:"ids"`
}

type deleteEmployeesResponse struct {
	Deleted int    `json:"deleted"`
	IDs     []uint `json:"ids"`
}

func (h *Handler) deleteEmployees(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req deleteEmployeesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if len(req.IDs) == 0 {
		respondError(w, http.StatusBadRequest, "no employees selected for deletion", nil)
		return
	}

	uniqueIDs := make([]uint, 0, len(req.IDs))
	seen := make(map[uint]struct{})
	for _, id := range req.IDs {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	if len(uniqueIDs) == 0 {
		respondError(w, http.StatusBadRequest, "no valid employee ids provided", nil)
		return
	}

	var employees []models.Employee
	if err := h.db.Where("user_id = ? AND id IN ?", userID, uniqueIDs).Find(&employees).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load employees for deletion", err)
		return
	}

	if len(employees) == 0 {
		respondJSON(w, http.StatusOK, deleteEmployeesResponse{Deleted: 0, IDs: []uint{}})
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND id IN ?", userID, uniqueIDs).Delete(&models.Employee{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete employees", err)
		return
	}

	for _, employee := range employees {
		if employee.ResignProofPath == "" {
			continue
		}
		if err := os.Remove(employee.ResignProofPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[deleteEmployees] remove proof failed for employee %d: %v", employee.ID, err)
		}
	}

	ids := make([]uint, 0, len(employees))
	for _, employee := range employees {
		ids = append(ids, employee.ID)
	}

	respondJSON(w, http.StatusOK, deleteEmployeesResponse{Deleted: len(employees), IDs: ids})
}

func (h *Handler) downloadResignProof(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	employeeIDStr := strings.TrimSpace(chi.URLParam(r, "employeeID"))
	if employeeIDStr == "" {
		respondError(w, http.StatusBadRequest, "employee id is required", nil)
		return
	}

	employeeID, err := strconv.ParseUint(employeeIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid employee id", err)
		return
	}

	var employee models.Employee
	if err := h.db.Where("id = ? AND user_id = ?", employeeID, userID).First(&employee).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load employee", err)
		return
	}

	if employee.ResignProofPath == "" {
		respondError(w, http.StatusNotFound, "未找到对应的离职证明文件", nil)
		return
	}

	file, err := os.Open(employee.ResignProofPath)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}
		respondError(w, status, "离职证明文件不存在或已被移除", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])
	_, _ = file.Seek(0, io.SeekStart)

	if stat, err := file.Stat(); err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	}

	filename := employee.ResignProofName
	if filename == "" {
		filename = filepath.Base(employee.ResignProofPath)
	}

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		fmt.Printf("[resign] stream proof failed: %v\n", err)
	}
}

func (h *Handler) restoreEmployees(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req employeeRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid restore payload", err)
		return
	}

	ids := make([]uint, 0, len(req.IDs))
	for _, raw := range req.IDs {
		if raw == 0 {
			continue
		}
		ids = append(ids, raw)
	}

	idNumberSet := make(map[string]struct{})
	for _, raw := range req.IDNumbers {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		idNumberSet[trimmed] = struct{}{}
	}
	refinedIDNumbers := make([]string, 0, len(idNumberSet))
	for value := range idNumberSet {
		refinedIDNumbers = append(refinedIDNumbers, value)
	}

	if len(ids) == 0 && len(refinedIDNumbers) == 0 {
		respondError(w, http.StatusBadRequest, "restore request requires ids or id_numbers", nil)
		return
	}

	buildFilter := func(db *gorm.DB) *gorm.DB {
		query := db.Where("user_id = ?", userID)
		switch {
		case len(ids) > 0 && len(refinedIDNumbers) > 0:
			return query.Where("(id IN ? OR id_number IN ?)", ids, refinedIDNumbers)
		case len(ids) > 0:
			return query.Where("id IN ?", ids)
		default:
			return query.Where("id_number IN ?", refinedIDNumbers)
		}
	}

	var employees []models.Employee
	if err := buildFilter(h.db).Find(&employees).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load employees for restore", err)
		return
	}

	if len(employees) == 0 {
		respondError(w, http.StatusNotFound, "未找到匹配的离职员工", nil)
		return
	}

	for _, employee := range employees {
		if employee.ResignProofPath == "" {
			continue
		}
		if err := os.Remove(employee.ResignProofPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("[restore] remove resign proof failed: %v\n", err)
		}
	}

	updates := map[string]any{
		"status":            "active",
		"resign_date":       "",
		"resign_proof_path": "",
		"resign_proof_name": "",
		"resign_reasons":    "",
		"updated_at":        time.Now(),
	}

	result := buildFilter(h.db.Model(&models.Employee{})).Updates(updates)
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to restore employees", result.Error)
		return
	}

	for idx := range employees {
		employees[idx].Status = "active"
		employees[idx].ResignDate = ""
		employees[idx].ResignProofPath = ""
		employees[idx].ResignProofName = ""
		employees[idx].ResignProofURL = ""
		employees[idx].ResignReasons = ""
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"restored":  len(employees),
		"employees": employees,
	})
}

func validateSocialInsuranceChangeType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case socialInsuranceChangeTypeIncrease, socialInsuranceChangeTypeDecrease:
		return true
	default:
		return false
	}
}

func (h *Handler) listSocialInsuranceChanges(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	query := h.db.Where("user_id = ?", userID).Order("created_at DESC")
	if changeType := strings.TrimSpace(r.URL.Query().Get("change_type")); changeType != "" {
		if !validateSocialInsuranceChangeType(changeType) {
			respondError(w, http.StatusBadRequest, "invalid change_type", nil)
			return
		}
		query = query.Where("change_type = ?", strings.ToLower(changeType))
	}
	if batchIDRaw := strings.TrimSpace(r.URL.Query().Get("batch_id")); batchIDRaw != "" {
		if batchID, errConv := strconv.ParseUint(batchIDRaw, 10, 64); errConv == nil {
			query = query.Where("batch_id = ?", uint(batchID))
		}
	}

	var records []models.SocialInsuranceRecord
	if err := query.Preload("Batch").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list social insurance changes", err)
		return
	}

	responses := make([]socialInsuranceRecordResponse, 0, len(records))
	for _, record := range records {
		responses = append(responses, buildSocialInsuranceRecordResponse(record))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"records": responses,
	})
}

func (h *Handler) importSocialInsuranceChanges(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart payload", err)
		return
	}

	changeType := strings.ToLower(strings.TrimSpace(r.FormValue("change_type")))
	if !validateSocialInsuranceChangeType(changeType) {
		respondError(w, http.StatusBadRequest, "change_type must be increase or decrease", nil)
		return
	}

	payloadRaw := strings.TrimSpace(r.FormValue("payload"))
	if payloadRaw == "" {
		respondError(w, http.StatusBadRequest, "payload is required", nil)
		return
	}

	var payload socialInsuranceImportPayload
	if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if len(payload.Records) == 0 {
		respondError(w, http.StatusBadRequest, "payload.records cannot be empty", nil)
		return
	}

	for _, record := range payload.Records {
		if strings.TrimSpace(record.EmployeeName) == "" {
			respondError(w, http.StatusBadRequest, "record employee_name is required", nil)
			return
		}
		if strings.TrimSpace(record.IdentityNumber) == "" {
			respondError(w, http.StatusBadRequest, "record identity_number is required", nil)
			return
		}
	}

	fileHeader, err := getMultipartFileHeader(r, "file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file upload is required", err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to open uploaded file", err)
		return
	}
	defer file.Close()

	buffer, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read uploaded file", err)
		return
	}

	uuidName := uuid.New().String()
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = ".xls"
	}
	storedFileName := fmt.Sprintf("%s%s", uuidName, ext)

	// Resolve storage path using StorageRouter
	resolvedRoute, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{
		ModuleCode:   "provident",
		ResourceType: "social_insurance_change",
		Filename:     storedFileName,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to resolve storage path", err)
		return
	}

	// Upload file using GlobalManager
	_, err = storage.GlobalManager.UploadFile(r.Context(), resolvedRoute.StorageID, userID, storedFileName, bytes.NewReader(buffer), int64(len(buffer)))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to upload file", err)
		return
	}

	// For backward compatibility, also store the file path locally
	relativeDir := filepath.Join("social-insurance", changeType, strconv.FormatUint(uint64(userID), 10))
	storedPath := filepath.Join(h.uploadBaseDir, relativeDir, storedFileName)
	if err := os.MkdirAll(filepath.Dir(storedPath), 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare social insurance directory", err)
		return
	}
	if err := os.WriteFile(storedPath, buffer, 0o644); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to persist uploaded file", err)
		return
	}

	relativePath := filepath.Join(relativeDir, storedFileName)

	batch := &models.SocialInsuranceBatch{
		UserID:           userID,
		ChangeType:       changeType,
		OriginalFileName: fileHeader.Filename,
		StoredFileName:   storedFileName,
		StoredFilePath:   relativePath,
	}

	records := make([]models.SocialInsuranceRecord, 0, len(payload.Records))
	for _, item := range payload.Records {
		payloadMap := datatypes.JSONMap{}
		for key, value := range item.TemplateValues {
			payloadMap[key] = strings.TrimSpace(value)
		}

		record := models.SocialInsuranceRecord{
			UserID:         userID,
			ChangeType:     changeType,
			EmployeeName:   strings.TrimSpace(item.EmployeeName),
			Department:     strings.TrimSpace(item.Department),
			IdentityNumber: strings.TrimSpace(item.IdentityNumber),
			PersonalNumber: strings.TrimSpace(item.PersonalNumber),
			EffectiveDate:  strings.TrimSpace(item.EffectiveDate),
			Reason:         strings.TrimSpace(item.Reason),
			TemplateValues: payloadMap,
		}
		records = append(records, record)
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(batch).Error; err != nil {
			return err
		}
		for idx := range records {
			batchID := batch.ID
			records[idx].BatchID = &batchID
		}
		if err := tx.Create(&records).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to store social insurance changes", err)
		return
	}

	responseRecords := make([]socialInsuranceRecordResponse, 0, len(records))
	for _, record := range records {
		record.Batch = batch
		responseRecords = append(responseRecords, buildSocialInsuranceRecordResponse(record))
	}

	resp := socialInsuranceImportResponse{
		Batch: socialInsuranceBatchResponse{
			ID:               batch.ID,
			ChangeType:       batch.ChangeType,
			OriginalFileName: batch.OriginalFileName,
			StoredFileName:   batch.StoredFileName,
			StoredFilePath:   batch.StoredFilePath,
			CreatedAt:        batch.CreatedAt,
		},
		Records: responseRecords,
	}

	respondJSON(w, http.StatusCreated, resp)
}

type socialInsuranceCreatePayload struct {
	ChangeType     string            `json:"change_type"`
	EmployeeID     uint              `json:"employee_id"`
	EmployeeName   string            `json:"employee_name"`
	Department     string            `json:"department"`
	IdentityNumber string            `json:"identity_number"`
	PersonalNumber string            `json:"personal_number"`
	EffectiveDate  string            `json:"effective_date"`
	Reason         string            `json:"reason"`
	TemplateValues map[string]string `json:"template_values"`
	SourceFileName string            `json:"source_file_name"`
	SourceFilePath string            `json:"source_file_path"`
	Remark         string            `json:"remark"`
}

func (h *Handler) createSocialInsuranceChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var payload socialInsuranceCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	normalized, err := normalizeSocialInsurancePayload(payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	record := models.SocialInsuranceRecord{
		UserID:         userID,
		ChangeType:     normalized.changeType,
		EmployeeName:   normalized.employeeName,
		Department:     normalized.department,
		IdentityNumber: normalized.identityNumber,
		PersonalNumber: normalized.personalNumber,
		EffectiveDate:  normalized.effectiveDate,
		Reason:         normalized.reason,
		TemplateValues: normalized.templateValues,
	}

	if err := h.db.Create(&record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create social insurance change", err)
		return
	}

	respondJSON(w, http.StatusCreated, buildSocialInsuranceRecordResponse(record))
}

func (h *Handler) updateSocialInsuranceChange(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	changeIDRaw := chi.URLParam(r, "changeID")
	changeID, err := strconv.ParseUint(strings.TrimSpace(changeIDRaw), 10, 64)
	if err != nil || changeID == 0 {
		respondError(w, http.StatusBadRequest, "invalid change id", err)
		return
	}

	var payload socialInsuranceCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	normalized, err := normalizeSocialInsurancePayload(payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var record models.SocialInsuranceRecord
	if err := h.db.Where("id = ? AND user_id = ?", changeID, userID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "record not found", err)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to locate record", err)
		return
	}

	record.ChangeType = normalized.changeType
	record.EmployeeName = normalized.employeeName
	record.Department = normalized.department
	record.IdentityNumber = normalized.identityNumber
	record.PersonalNumber = normalized.personalNumber
	record.EffectiveDate = normalized.effectiveDate
	record.Reason = normalized.reason
	record.TemplateValues = normalized.templateValues

	if err := h.db.Save(&record).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update social insurance change", err)
		return
	}

	respondJSON(w, http.StatusOK, buildSocialInsuranceRecordResponse(record))
}

type socialInsuranceDeleteRequest struct {
	IDs []uint `json:"ids"`
}

type socialInsurancePayloadResult struct {
	changeType     string
	employeeName   string
	department     string
	identityNumber string
	personalNumber string
	effectiveDate  string
	reason         string
	templateValues datatypes.JSONMap
}

func normalizeSocialInsurancePayload(payload socialInsuranceCreatePayload) (socialInsurancePayloadResult, error) {
	result := socialInsurancePayloadResult{}
	changeType := strings.ToLower(strings.TrimSpace(payload.ChangeType))
	if !validateSocialInsuranceChangeType(changeType) {
		return result, fmt.Errorf("change_type must be increase or decrease")
	}

	employeeName := strings.TrimSpace(payload.EmployeeName)
	if employeeName == "" {
		return result, fmt.Errorf("employee_name is required")
	}

	identityNumber := strings.TrimSpace(payload.IdentityNumber)
	if identityNumber == "" {
		return result, fmt.Errorf("identity_number is required")
	}

	templateValuesRaw := payload.TemplateValues
	if templateValuesRaw == nil {
		templateValuesRaw = map[string]string{}
	}

	normalizedTemplateValues := make(map[string]string, len(templateValuesRaw))
	templateValues := datatypes.JSONMap{}
	for key, value := range templateValuesRaw {
		trimmed := strings.TrimSpace(value)
		normalizedTemplateValues[key] = trimmed
		templateValues[key] = trimmed
	}

	requiredTemplateFields := []string{}
	switch changeType {
	case socialInsuranceChangeTypeIncrease:
		requiredTemplateFields = []string{"personalIdentity", "householdType", "education", "pensionStartDate", "baseSalary"}
	case socialInsuranceChangeTypeDecrease:
		requiredTemplateFields = []string{"personalNumber", "decreaseDate", "decreaseReason", "pensionFlag", "unemploymentFlag", "medicalFlag", "injuryFlag", "maternityFlag"}
	}

	missingFields := make([]string, 0)
	for _, field := range requiredTemplateFields {
		if strings.TrimSpace(normalizedTemplateValues[field]) == "" {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingFields) > 0 {
		return result, fmt.Errorf("missing template fields: %s", strings.Join(missingFields, ", "))
	}

	effectiveDate := strings.TrimSpace(payload.EffectiveDate)
	if effectiveDate == "" {
		return result, fmt.Errorf("effective_date is required")
	}
	reason := strings.TrimSpace(payload.Reason)
	if changeType == socialInsuranceChangeTypeDecrease && reason == "" {
		return result, fmt.Errorf("reason is required for decrease records")
	}
	personalNumber := strings.TrimSpace(payload.PersonalNumber)
	if changeType == socialInsuranceChangeTypeDecrease && personalNumber == "" {
		return result, fmt.Errorf("personal_number is required for decrease records")
	}

	result = socialInsurancePayloadResult{
		changeType:     changeType,
		employeeName:   employeeName,
		department:     strings.TrimSpace(payload.Department),
		identityNumber: identityNumber,
		personalNumber: personalNumber,
		effectiveDate:  effectiveDate,
		reason:         reason,
		templateValues: templateValues,
	}
	return result, nil
}

func (h *Handler) deleteSocialInsuranceChanges(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req socialInsuranceDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid delete payload", err)
		return
	}

	filtered := make([]uint, 0, len(req.IDs))
	for _, id := range req.IDs {
		if id == 0 {
			continue
		}
		filtered = append(filtered, id)
	}

	if len(filtered) == 0 {
		respondError(w, http.StatusBadRequest, "ids cannot be empty", nil)
		return
	}

	result := h.db.Where("user_id = ? AND id IN ?", userID, filtered).Delete(&models.SocialInsuranceRecord{})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete records", result.Error)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"deleted": result.RowsAffected,
	})
}

func buildSocialInsuranceRecordResponse(record models.SocialInsuranceRecord) socialInsuranceRecordResponse {
	var batchID uint
	if record.BatchID != nil {
		batchID = *record.BatchID
	}
	values := make(map[string]string, len(record.TemplateValues))
	for key, value := range record.TemplateValues {
		switch typed := value.(type) {
		case string:
			values[key] = typed
		case fmt.Stringer:
			values[key] = typed.String()
		case nil:
			values[key] = ""
		default:
			values[key] = fmt.Sprintf("%v", typed)
		}
	}

	resp := socialInsuranceRecordResponse{
		ID:             record.ID,
		BatchID:        batchID,
		ChangeType:     record.ChangeType,
		EmployeeName:   record.EmployeeName,
		Department:     record.Department,
		IdentityNumber: record.IdentityNumber,
		PersonalNumber: record.PersonalNumber,
		EffectiveDate:  record.EffectiveDate,
		Reason:         record.Reason,
		TemplateValues: values,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}

	if record.Batch != nil {
		resp.OriginalFileName = record.Batch.OriginalFileName
	}

	return resp
}

type createPeriodRequest struct {
	YearMonth        string `json:"year_month"`
	AllowAdjustments *bool  `json:"allow_adjustments,omitempty"`
}

type employeeExportRequest struct {
	Scope     string   `json:"scope"`
	IDs       []uint   `json:"ids"`
	IDNumbers []string `json:"id_numbers"`
}

type employeeRestoreRequest struct {
	IDs       []uint   `json:"ids"`
	IDNumbers []string `json:"id_numbers"`
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

	allowAdjustments := false
	if req.AllowAdjustments != nil {
		allowAdjustments = *req.AllowAdjustments
	}

	var period models.Period
	if err := h.db.Where("user_id = ? AND year_month = ?", userID, req.YearMonth).First(&period).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			period = models.Period{
				UserID:           &userID,
				YearMonth:        req.YearMonth,
				Status:           "draft",
				AllowAdjustments: allowAdjustments,
			}
			if err := h.db.Create(&period).Error; err != nil {
				respondError(w, http.StatusInternalServerError, "failed to create period", err)
				return
			}
		} else {
			respondError(w, http.StatusInternalServerError, "failed to query period", err)
			return
		}
	} else {
		updates := map[string]any{}
		if period.AllowAdjustments != allowAdjustments {
			updates["allow_adjustments"] = allowAdjustments
		}
		if len(updates) > 0 {
			if err := h.db.Model(&period).Updates(updates).Error; err != nil {
				respondError(w, http.StatusInternalServerError, "failed to update period", err)
				return
			}
			period.AllowAdjustments = allowAdjustments
		}
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

	targetDir, err := h.ensureUploadDir(fmt.Sprintf("%d", period.ID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
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

	result, err := h.process.ParseSourceFile(period.ID, period.UserID, storedPath, header.Filename, scheme, part)
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

	targetDir, err := h.ensureUploadDir(fmt.Sprintf("%d", period.ID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
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

		result, err := h.process.ParseSourceFile(period.ID, period.UserID, storedPath, header.Filename, scheme, part)
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

	targetDir, err := h.ensureUploadDir(fmt.Sprintf("%d", period.ID), "roster")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare roster directory", err)
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

	result, err := h.process.ParseRosterFile(period.ID, period.UserID, storedPath, header.Filename)
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

func normalizeResignDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("resign date is required")
	}

	replacer := strings.NewReplacer("年", "-", "月", "-", "日", "", ".", "-", "/", "-")
	cleaned := replacer.Replace(trimmed)

	layouts := []string{
		"2006-01-02",
		"2006-1-2",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, cleaned); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unsupported date format: %s", value)
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

func (h *Handler) downloadEmployeeTemplate(w http.ResponseWriter, r *http.Request) {
	templatePath := service.GetEmployeeTemplatePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := service.GenerateEmployeeTemplate(templatePath); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate employee template", err)
			return
		}
	}

	file, err := os.Open(templatePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to open template", err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"员工导入模板.xlsx\"")
	if _, err := io.Copy(w, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send template", err)
		return
	}
}

func (h *Handler) downloadResignedEmployeeTemplate(w http.ResponseWriter, r *http.Request) {
	templatePath := service.GetResignedEmployeeTemplatePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := service.GenerateResignedEmployeeTemplate(templatePath); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate resigned employee template", err)
			return
		}
	}

	file, err := os.Open(templatePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to open template", err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"离职员工导入模板.xlsx\"")
	if _, err := io.Copy(w, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send template", err)
		return
	}
}

func (h *Handler) exportEmployees(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req employeeExportRequest
	// allow empty body for exporting all data
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid export payload", err)
			return
		}
	}

	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "filtered"
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	department := strings.TrimSpace(r.URL.Query().Get("department"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	ids := make([]uint, 0, len(req.IDs))
	for _, id := range req.IDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}

	idNumbers := make([]string, 0, len(req.IDNumbers))
	for _, raw := range req.IDNumbers {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			idNumbers = append(idNumbers, trimmed)
		}
	}

	if scope == "selected" && len(ids) == 0 && len(idNumbers) == 0 {
		respondError(w, http.StatusBadRequest, "selected scope requires ids or id_numbers", nil)
		return
	}

	query := h.db.Where("user_id = ?", userID)

	if scope == "selected" {
		if len(ids) > 0 {
			query = query.Where("id IN ?", ids)
		} else {
			query = query.Where("id_number IN ?", idNumbers)
		}
	} else {
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}
		if department != "" && department != "all" {
			query = query.Where("department = ?", department)
		}
		if search != "" {
			like := fmt.Sprintf("%%%s%%", search)
			query = query.Where("name LIKE ? OR id_number LIKE ? OR department LIKE ?", like, like, like)
		}
	}

	var employees []models.Employee
	if err := query.Order("name ASC, id_number ASC").Find(&employees).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load employees", err)
		return
	}

	if len(employees) == 0 {
		respondError(w, http.StatusBadRequest, "没有符合条件的员工数据", nil)
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheetName := "员工导出"
	f.SetSheetName("Sheet1", sheetName)

	headers := service.EmployeeTemplateHeaders()
	for idx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to write header", err)
			return
		}
	}

	for rowIdx, employee := range employees {
		status := employee.Status
		switch status {
		case "resigned":
			status = "离职"
		case "active":
			status = "在职"
		}
		resignDate := employee.ResignDate
		if status != "离职" {
			resignDate = ""
		}

		values := []any{
			employee.EmployeeID,
			employee.Name,
			employee.Department,
			employee.Position,
			employee.Gender,
			employee.HireDate,
			employee.Age,
			employee.WorkYears,
			employee.BirthMonth,
			employee.Education,
			employee.PoliticalStatus,
			employee.WorkClothingSize,
			employee.SafetyShoeSize,
			employee.HouseholdType,
			employee.Ethnicity,
			employee.NativePlace,
			employee.IDAddress,
			employee.IDNumber,
			employee.MaritalStatus,
			employee.SocialInsurance,
			employee.HasBirth,
			employee.Phone,
			employee.EmergencyContact,
			employee.EmergencyPhone,
			employee.CurrentAddress,
			employee.GraduateSchool,
			employee.Major,
			employee.GraduationTime,
			employee.SocialInsuranceNumber,
			employee.ProvidentFundNumber,
			employee.Email,
			employee.Remarks,
			status,
			resignDate,
		}

		for colIdx, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				respondError(w, http.StatusInternalServerError, "failed to write rows", err)
				return
			}
		}
	}

	widths := service.EmployeeTemplateColumnWidths()
	for idx, width := range widths {
		col, _ := excelize.ColumnNumberToName(idx + 1)
		if err := f.SetColWidth(sheetName, col, col, width); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to set column width", err)
			return
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to build export file", err)
		return
	}

	filename := fmt.Sprintf("员工导出-%s.xlsx", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send export file", err)
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
	uploadDir := h.uploadPath(fmt.Sprintf("%d", period.ID))
	if _, err := os.Stat(uploadDir); err == nil {
		if err := os.RemoveAll(uploadDir); err != nil {
			// 日志记录错误，但不影响API响应
			fmt.Printf("Warning: failed to remove upload directory %s: %v\n", uploadDir, err)
		}
	}

	result := map[string]interface{}{
		"message":   fmt.Sprintf("账期 %s 已重置，所有数据已清除", period.YearMonth),
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
	uploadDir := h.uploadPath(fmt.Sprintf("%d", period.ID))
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

	targetDir, err := h.ensureUploadDir(fmt.Sprintf("%d", period.ID), "adjustments")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare adjustment upload dir", err)
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

		result, err := h.process.ParseAdjustmentFile(period.ID, period.UserID, storedPath, header.Filename, scheme, part)
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
	adjustmentDir := h.uploadPath(fmt.Sprintf("%d", period.ID), "adjustments")
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
