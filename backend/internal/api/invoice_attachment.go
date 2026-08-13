package api

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

func (h *Handler) previewInvoiceAttachment(w http.ResponseWriter, r *http.Request) {
	h.serveInvoiceAttachment(w, r, false)
}

func (h *Handler) downloadInvoiceAttachment(w http.ResponseWriter, r *http.Request) {
	h.serveInvoiceAttachment(w, r, true)
}

func (h *Handler) serveInvoiceAttachment(w http.ResponseWriter, r *http.Request, download bool) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", nil)
		return
	}
	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil || invoice.AttachmentFileID == nil || !h.canAccessInvoiceAttachment(userID, &invoice) {
		respondError(w, http.StatusNotFound, "附件不存在", nil)
		return
	}
	if storage.GlobalManager == nil {
		respondError(w, http.StatusNotFound, "附件不存在", nil)
		return
	}
	reader, file, err := storage.GlobalManager.DownloadFile(r.Context(), *invoice.AttachmentFileID)
	if err != nil {
		respondError(w, http.StatusNotFound, "附件不存在", nil)
		return
	}
	defer reader.Close()
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": safeInvoiceFilename(file.OriginalName)}))
	_, _ = io.Copy(w, reader)
}

func safeInvoiceFilename(filename string) string {
	name := filepath.Base(filename)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		return "invoice.pdf"
	}
	return name
}

// canAccessInvoiceAttachment 附件访问：资源范围（admin 全量、manager 仅本部门、普通用户本人）
// + 归档锁定（仅待确认 pending 可访问，已确认/作废后附件锁定）；软删不可见。
func (h *Handler) canAccessInvoiceAttachment(userID uint, invoice *models.Invoice) bool {
	if invoice.ArchiveStatus != models.InvoiceArchiveStatusPending {
		return false
	}
	return h.canAccessInvoice(userID, invoice)
}
