package api

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/service/docreader"
	"siapp/internal/service/storage"
)

const (
	invoiceParseLease   = 2 * time.Minute
	invoiceParseLimit   = 10
	invoiceParseTimeout = 90 * time.Second
)

type invoiceTextExtractor interface {
	Extract(context.Context, string) (string, error)
}
type docreaderInvoiceExtractor struct{ client *docreader.Client }

func (e docreaderInvoiceExtractor) Extract(ctx context.Context, path string) (string, error) {
	result, err := e.client.Parse(ctx, docreader.ParseRequest{FilePath: path, FileType: "pdf"})
	if err != nil {
		return "", err
	}
	return result.FullText, nil
}

type unavailableInvoiceExtractor struct{}

func (unavailableInvoiceExtractor) Extract(context.Context, string) (string, error) {
	return "", errors.New("extractor unavailable")
}

type ocrInvoiceExtractor struct{ service *service.OCRService }

func (e ocrInvoiceExtractor) Extract(ctx context.Context, path string) (string, error) {
	result, err := e.service.ExtractGlobalDefaultWithContext(ctx, path, 0)
	if err != nil || result == nil || !result.Success || result.Text == "" {
		return "", errors.New("ocr unavailable")
	}
	return result.Text, nil
}

type invoiceParseError struct {
	Code      string
	Retryable bool
}
type invoiceSnapshot struct {
	id           uint
	updatedAt    time.Time
	attachmentID uint
	archive      models.InvoiceArchiveStatus
}

func (e invoiceParseError) Error() string { return e.Code }

func (h *Handler) StartInvoiceParsingWorker(ctx context.Context) {
	h.runInvoiceParsingBatch(ctx, invoiceParseLimit)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runInvoiceParsingBatch(ctx, invoiceParseLimit)
			}
		}
	}()
}

func (h *Handler) runInvoiceParsingBatch(ctx context.Context, limit int) {
	if limit < 1 {
		limit = invoiceParseLimit
	}
	for i := 0; i < limit; i++ {
		task, ok := h.claimInvoiceParsingTask(ctx)
		if !ok {
			return
		}
		h.processInvoiceParsingTask(ctx, task)
	}
}

func (h *Handler) claimInvoiceParsingTask(ctx context.Context) (*models.InvoiceParsingTask, bool) {
	now := time.Now()
	h.failExpiredInvoiceTasks(ctx, now)
	var tasks []models.InvoiceParsingTask
	query := "(status = ? AND (available_at IS NULL OR available_at <= ?)) OR (status = ? AND locked_until < ?)"
	if h.db.WithContext(ctx).Where(query, models.InvoiceParsingTaskPending, now, models.InvoiceParsingTaskRunning, now).Where("attempt_count < max_attempts").Order("id").Limit(10).Find(&tasks).Error != nil {
		return nil, false
	}
	for i := range tasks {
		token, until := uuid.NewString(), now.Add(invoiceParseLease)
		result := h.db.WithContext(ctx).Model(&models.InvoiceParsingTask{}).Where("id = ? AND ((status = ? AND (available_at IS NULL OR available_at <= ?)) OR (status = ? AND locked_until < ?)) AND attempt_count < max_attempts", tasks[i].ID, models.InvoiceParsingTaskPending, now, models.InvoiceParsingTaskRunning, now).Updates(map[string]any{"status": models.InvoiceParsingTaskRunning, "locked_by": token, "locked_until": until, "attempt_count": gorm.Expr("attempt_count + 1")})
		if result.Error == nil && result.RowsAffected == 1 {
			tasks[i].LockedBy, tasks[i].LockedUntil, tasks[i].AttemptCount = token, &until, tasks[i].AttemptCount+1
			return &tasks[i], true
		}
	}
	return nil, false
}

func (h *Handler) failExpiredInvoiceTasks(ctx context.Context, now time.Time) {
	h.db.WithContext(ctx).Model(&models.InvoiceParsingTask{}).Where("status = ? AND locked_until < ? AND attempt_count >= max_attempts", models.InvoiceParsingTaskRunning, now).Updates(map[string]any{"status": models.InvoiceParsingTaskFailed, "error_code": "worker_lease_expired", "last_error": "worker lease expired", "completed_at": now, "locked_by": "", "locked_until": gorm.Expr("NULL"), "available_at": gorm.Expr("NULL")})
}

func (h *Handler) processInvoiceParsingTask(parent context.Context, task *models.InvoiceParsingTask) {
	ctx, cancel := context.WithTimeout(parent, invoiceParseTimeout)
	defer cancel()
	snapshot, err := h.loadInvoiceSnapshot(ctx, task.InvoiceID)
	if err != nil {
		h.finishInvoiceParseFailure(ctx, task, err)
		return
	}
	text, source, err := h.extractInvoiceText(ctx, snapshot)
	if err != nil {
		h.finishInvoiceParseFailure(ctx, task, err)
		return
	}
	if isInvoiceParseContextError(ctx, nil) {
		h.finishInvoiceParseFailure(ctx, task, invoiceParseError{"execution_cancelled", true})
		return
	}
	parsed := parseInvoiceFields(text)
	if err := h.saveParsedInvoice(ctx, task, snapshot, parsed, text, source); err != nil {
		h.finishInvoiceParseFailure(ctx, task, err)
	}
}

func (h *Handler) loadInvoiceSnapshot(ctx context.Context, invoiceID uint) (invoiceSnapshot, error) {
	var invoice models.Invoice
	if err := h.db.WithContext(ctx).First(&invoice, invoiceID).Error; err != nil {
		if isInvoiceParseContextError(ctx, err) {
			return invoiceSnapshot{}, invoiceParseError{"execution_cancelled", true}
		}
		return invoiceSnapshot{}, invoiceParseError{"invoice_changed", false}
	}
	if invoice.AttachmentFileID == nil || invoice.ArchiveStatus != models.InvoiceArchiveStatusPending {
		return invoiceSnapshot{}, invoiceParseError{"invoice_changed", false}
	}
	return invoiceSnapshot{invoice.ID, invoice.UpdatedAt, *invoice.AttachmentFileID, invoice.ArchiveStatus}, nil
}

func (h *Handler) extractInvoiceText(ctx context.Context, snapshot invoiceSnapshot) (string, string, error) {
	path, err := h.downloadInvoiceParseFile(ctx, snapshot.attachmentID)
	if err != nil {
		if isInvoiceParseContextError(ctx, err) {
			return "", "", invoiceParseError{"execution_cancelled", true}
		}
		return "", "", invoiceParseError{"attachment_unavailable", true}
	}
	defer os.Remove(path)
	text, extractErr := h.invoicePDFText.Extract(ctx, path)
	if isInvoiceParseContextError(ctx, extractErr) {
		return "", "", invoiceParseError{"execution_cancelled", true}
	}
	if invoiceTextQuality(text) {
		return limitInvoiceText(text), "docreader", nil
	}
	ocrText, ocrErr := h.invoiceOCRText.Extract(ctx, path)
	if isInvoiceParseContextError(ctx, ocrErr) {
		return "", "", invoiceParseError{"execution_cancelled", true}
	}
	if ocrErr != nil {
		return "", "", invoiceParseError{"extractor_unavailable", true}
	}
	if invoiceTextQuality(ocrText) {
		return limitInvoiceText(ocrText), "ocr", nil
	}
	return "", "", invoiceParseError{"no_recognizable_content", false}
}

func isInvoiceParseContextError(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (h *Handler) downloadInvoiceParseFile(ctx context.Context, fileID uint) (string, error) {
	if storage.GlobalManager == nil {
		return "", errors.New("attachment unavailable")
	}
	reader, file, err := storage.GlobalManager.DownloadFile(ctx, fileID)
	if err != nil || file.Size > maxInvoicePDFSize {
		return "", errors.New("download failed")
	}
	defer reader.Close()
	target, err := os.CreateTemp("", "invoice-parse-*.pdf")
	if err != nil {
		return "", err
	}
	path := target.Name()
	defer target.Close()
	if err := target.Chmod(0600); err != nil {
		os.Remove(path)
		return "", err
	}
	n, err := io.Copy(target, io.LimitReader(reader, maxInvoicePDFSize+1))
	if err != nil || n > maxInvoicePDFSize {
		os.Remove(path)
		return "", errors.New("invalid download")
	}
	return path, nil
}

func (h *Handler) finishInvoiceParseFailure(_ context.Context, task *models.InvoiceParsingTask, err error) {
	parseErr, ok := err.(invoiceParseError)
	if !ok {
		parseErr = invoiceParseError{"parse_failed", true}
	}
	now := time.Now()
	updates := map[string]any{"error_code": parseErr.Code, "last_error": parseErr.Code, "locked_by": "", "locked_until": nil}
	if !parseErr.Retryable || task.AttemptCount >= task.MaxAttempts {
		updates["status"], updates["completed_at"] = models.InvoiceParsingTaskFailed, now
	} else {
		delay := time.Minute * time.Duration(1<<(task.AttemptCount-1))
		updates["status"], updates["available_at"] = models.InvoiceParsingTaskPending, now.Add(delay)
	}
	h.updateInvoiceParseTask(task, updates)
}

func (h *Handler) updateInvoiceParseTask(task *models.InvoiceParsingTask, updates map[string]any) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := h.db.WithContext(ctx).Model(&models.InvoiceParsingTask{}).Where("id = ? AND status = ? AND locked_by = ? AND locked_until > ?", task.ID, models.InvoiceParsingTaskRunning, task.LockedBy, time.Now()).Updates(updates)
	return result.Error == nil && result.RowsAffected == 1
}

func (h *Handler) saveParsedInvoice(ctx context.Context, task *models.InvoiceParsingTask, snapshot invoiceSnapshot, parsed parsedInvoice, text, source string) error {
	if h.invoiceParseBeforeSave != nil {
		h.invoiceParseBeforeSave()
	}
	if isInvoiceParseContextError(ctx, nil) {
		return invoiceParseError{"execution_cancelled", true}
	}
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := h.db.WithContext(writeCtx).Transaction(func(tx *gorm.DB) error {
		if isInvoiceParseContextError(writeCtx, nil) {
			return invoiceParseError{"execution_cancelled", true}
		}
		var invoice models.Invoice
		if err := tx.First(&invoice, task.InvoiceID).Error; err != nil {
			if isInvoiceParseContextError(writeCtx, err) {
				return invoiceParseError{"execution_cancelled", true}
			}
			return invoiceParseError{"invoice_changed", false}
		}
		if invoice.DeletedAt.Valid || invoice.ArchiveStatus != snapshot.archive || invoice.AttachmentFileID == nil {
			return invoiceParseError{"invoice_changed", false}
		}
		if isInvoiceParseContextError(writeCtx, nil) {
			return invoiceParseError{"execution_cancelled", true}
		}
		// 解析后自动匹配购方主体；仅匹配成功时以主体设置覆盖解析值，
		// 避免无设置时错误清空解析出的购方字段
		matched := h.matchBuyerEntity(tx, &invoice)
		updates := parsed.invoiceUpdates(text, source, invoice.VoucherType)
		if matched {
			updates["buyer"] = invoice.Buyer
			updates["buyer_tax_no"] = invoice.BuyerTaxNo
			updates["buyer_matched"] = invoice.BuyerMatched
			updates["buyer_match_note"] = invoice.BuyerMatchNote
		}
		result := tx.Model(&models.Invoice{}).Where("id = ? AND updated_at = ? AND archive_status = ? AND attachment_file_id = ? AND deleted_at IS NULL", snapshot.id, snapshot.updatedAt, snapshot.archive, snapshot.attachmentID).Updates(updates)
		if result.Error != nil || result.RowsAffected != 1 {
			if isInvoiceParseContextError(writeCtx, result.Error) {
				return invoiceParseError{"execution_cancelled", true}
			}
			return invoiceParseError{"invoice_changed", false}
		}
		if err := tx.Where("invoice_id = ?", invoice.ID).Delete(&models.InvoiceItem{}).Error; err != nil {
			return err
		}
		for i := range parsed.items {
			parsed.items[i].InvoiceID, parsed.items[i].LineNo = invoice.ID, i+1
		}
		if len(parsed.items) > 0 {
			if err := tx.Create(&parsed.items).Error; err != nil {
				return err
			}
		}
		if isInvoiceParseContextError(writeCtx, nil) {
			return invoiceParseError{"execution_cancelled", true}
		}
		result = tx.Model(&models.InvoiceParsingTask{}).Where("id = ? AND status = ? AND locked_by = ? AND locked_until > ?", task.ID, models.InvoiceParsingTaskRunning, task.LockedBy, time.Now()).Updates(map[string]any{"status": models.InvoiceParsingTaskSucceeded, "completed_at": time.Now(), "locked_by": "", "locked_until": nil})
		if result.Error != nil || result.RowsAffected != 1 {
			return invoiceParseError{"worker_lease_expired", false}
		}
		return nil
	})
	if isInvoiceParseContextError(writeCtx, err) {
		return invoiceParseError{"execution_cancelled", true}
	}
	return err
}
