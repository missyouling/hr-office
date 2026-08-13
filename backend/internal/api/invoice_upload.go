package api

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/google/uuid"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

const (
	maxInvoiceUploadFiles      = 50
	maxInvoicePDFSize          = 20 << 20
	invoiceMultipartMemorySize = 1 << 20
	invoiceMultipartOverhead   = 2 << 20
)

type invoiceUploadResult struct {
	OriginalName string `json:"original_name"`
	InvoiceID    uint   `json:"invoice_id,omitempty"`
	TaskID       uint   `json:"task_id,omitempty"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	Error        string `json:"error,omitempty"`
	Duplicate    bool   `json:"duplicate_warning,omitempty"`
}
type invoiceTemporaryPDF struct {
	path string
	size int64
	hash string
}

func (h *Handler) uploadInvoicePDFs(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxInvoiceUploadFiles)*maxInvoicePDFSize+invoiceMultipartOverhead)
	if err := r.ParseMultipartForm(invoiceMultipartMemorySize); err != nil {
		respondError(w, http.StatusBadRequest, "上传表单格式错误", nil)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 || len(files) > maxInvoiceUploadFiles {
		respondError(w, http.StatusBadRequest, "每批必须包含 1 至 50 份文件", nil)
		return
	}
	sourceType, sourceID := invoiceUploadSource(r)
	// 采购关联校验：类型合法、记录存在、当前用户有权访问
	if err := h.validateInvoiceSource(h.db, userID, sourceType, sourceID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	items := make([]invoiceUploadResult, 0, len(files))
	for _, file := range files {
		items = append(items, h.uploadOneInvoicePDF(r, userID, file, sourceType, sourceID))
	}
	respondJSON(w, http.StatusCreated, map[string]any{"items": items})
}

func invoiceUploadSource(r *http.Request) (string, *uint) {
	var sourceID *uint
	if parsed, err := strconv.ParseUint(r.FormValue("source_id"), 10, 64); err == nil && parsed > 0 {
		id := uint(parsed)
		sourceID = &id
	}
	return strings.TrimSpace(r.FormValue("source_type")), sourceID
}

func (h *Handler) uploadOneInvoicePDF(r *http.Request, userID uint, header *multipart.FileHeader, sourceType string, sourceID *uint) invoiceUploadResult {
	result := invoiceUploadResult{OriginalName: header.Filename, Status: "failed"}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		return invoiceUploadFailure(result, "invalid_extension", "仅支持 PDF 文件")
	}
	temporary, err := readInvoicePDF(header)
	if err != nil {
		return invoiceUploadFailure(result, invoiceUploadErrorCode(err), invoiceUploadErrorMessage(err))
	}
	defer os.Remove(temporary.path)
	sysFile, err := h.storeInvoicePDF(r, userID, header.Filename, temporary)
	if err != nil {
		return invoiceUploadFailure(result, "storage_failed", "保存附件失败")
	}
	invoice, task, err := h.createUploadedInvoice(userID, sourceType, sourceID, sysFile.ID, temporary.hash)
	if err != nil {
		h.queueInvoiceSysFileCleanup(r.Context(), sysFile)
		return invoiceUploadFailure(result, "create_failed", "创建发票草稿失败")
	}
	var duplicates int64
	h.db.Model(&models.Invoice{}).Where("file_sha256 = ? AND id <> ?", temporary.hash, invoice.ID).Count(&duplicates)
	result.InvoiceID, result.TaskID, result.Status, result.Duplicate = invoice.ID, task.ID, "pending", duplicates > 0
	return result
}

func (h *Handler) storeInvoicePDF(r *http.Request, userID uint, filename string, temporary *invoiceTemporaryPDF) (*models.SysFile, error) {
	if storage.GlobalManager == nil {
		return nil, errors.New("storage unavailable")
	}
	storedName := uuid.NewString() + ".pdf"
	route, err := h.storageRouter.Resolve(r.Context(), storage.ResolveRequest{ModuleCode: "invoice", ResourceType: "pdf", Filename: storedName, FileSize: temporary.size})
	if err != nil {
		return nil, err
	}
	file, err := os.Open(temporary.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	driver, err := storage.GlobalManager.GetDriver(route.StorageID)
	if err != nil {
		return nil, err
	}
	if err := driver.Upload(r.Context(), route.FullPath, file, temporary.size); err != nil {
		h.handleInvoiceUploadFailure(route.StorageID, route.FullPath, nil, driver, err)
		return nil, err
	}
	fileHash, err := os.Open(temporary.path)
	if err != nil {
		h.handleInvoiceUploadFailure(route.StorageID, route.FullPath, nil, driver, err)
		return nil, err
	}
	defer fileHash.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, fileHash); err != nil {
		h.handleInvoiceUploadFailure(route.StorageID, route.FullPath, nil, driver, err)
		return nil, err
	}
	sysFile := &models.SysFile{StorageType: driver.Type(), Path: route.FullPath, OriginalName: filepath.Base(filename), Size: temporary.size, ContentType: "application/pdf", ETag: hex.EncodeToString(hash.Sum(nil)), StorageConfigID: &route.StorageID, CreatedBy: &userID, MigrationStatus: "none"}
	if err := h.db.Create(sysFile).Error; err != nil {
		h.handleInvoiceUploadFailure(route.StorageID, route.FullPath, nil, driver, err)
		return nil, err
	}
	return sysFile, nil
}

func readInvoicePDF(header *multipart.FileHeader) (*invoiceTemporaryPDF, error) {
	source, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer source.Close()
	target, err := os.CreateTemp("", "invoice-upload-*.pdf")
	if err != nil {
		return nil, err
	}
	path, success := target.Name(), false
	defer func() {
		target.Close()
		if !success {
			_ = os.Remove(path)
		}
	}()
	limited := io.LimitReader(source, maxInvoicePDFSize+1)
	signature := make([]byte, 5)
	n, err := io.ReadFull(limited, signature)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	if n == 0 {
		return nil, errInvoiceEmpty
	}
	if n < len(signature) || string(signature) != "%PDF-" {
		return nil, errInvoiceNotPDF
	}
	hash := sha256.New()
	if _, err := io.MultiWriter(target, hash).Write(signature); err != nil {
		return nil, err
	}
	rest, err := io.Copy(io.MultiWriter(target, hash), limited)
	if err != nil {
		return nil, err
	}
	size := int64(n) + rest
	if size > maxInvoicePDFSize {
		return nil, errInvoiceTooLarge
	}
	success = true
	return &invoiceTemporaryPDF{path: path, size: size, hash: hex.EncodeToString(hash.Sum(nil))}, nil
}

var (
	errInvoiceEmpty    = fmt.Errorf("empty invoice file")
	errInvoiceNotPDF   = fmt.Errorf("invalid pdf signature")
	errInvoiceTooLarge = fmt.Errorf("invoice file too large")
)

func invoiceUploadErrorCode(err error) string {
	if errors.Is(err, errInvoiceEmpty) {
		return "empty_file"
	}
	if errors.Is(err, errInvoiceNotPDF) {
		return "invalid_pdf"
	}
	if errors.Is(err, errInvoiceTooLarge) {
		return "file_too_large"
	}
	return "read_failed"
}
func invoiceUploadErrorMessage(err error) string {
	if errors.Is(err, errInvoiceEmpty) {
		return "文件不能为空"
	}
	if errors.Is(err, errInvoiceNotPDF) {
		return "文件不是有效的 PDF"
	}
	if errors.Is(err, errInvoiceTooLarge) {
		return "单个文件不能超过 20MB"
	}
	return "读取上传文件失败"
}
func invoiceUploadFailure(result invoiceUploadResult, code, message string) invoiceUploadResult {
	result.ErrorCode, result.Error = code, message
	return result
}

func (h *Handler) createUploadedInvoice(userID uint, sourceType string, sourceID *uint, fileID uint, hash string) (*models.Invoice, *models.InvoiceParsingTask, error) {
	invoice := &models.Invoice{UserID: &userID, ApplicantID: &userID, AttachmentFileID: &fileID, FileSHA256: hash, InvoiceDate: time.Now(), Amount: 0.01, Seller: "待识别", Status: models.InvoiceStatusDraft, ArchiveStatus: models.InvoiceArchiveStatusPending, VoucherType: models.InvoiceVoucherTypeOther, SourceType: sourceType, SourceID: sourceID}
	task := &models.InvoiceParsingTask{Status: models.InvoiceParsingTaskPending}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}
		task.InvoiceID = invoice.ID
		return tx.Create(task).Error
	})
	return invoice, task, err
}
