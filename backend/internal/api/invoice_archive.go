package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// voidInvoice 作废归档（仅 admin；pending/confirmed -> voided；已报销禁止）。
// 事务内 CAS：仅当归档状态仍为 pending/confirmed 且未报销时更新，并发作废只有一个成功。
func (h *Handler) voidInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", nil)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", nil)
		return
	}
	// TrimSpace 后原因必填
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
		return
	}
	if invoice.Status == models.InvoiceStatusReimbursed {
		respondError(w, http.StatusForbidden, "已报销发票不可作废", nil)
		return
	}
	if invoice.ArchiveStatus != models.InvoiceArchiveStatusPending && invoice.ArchiveStatus != models.InvoiceArchiveStatusConfirmed {
		respondError(w, http.StatusForbidden, "仅待确认或已确认发票可作废", nil)
		return
	}

	now := time.Now()
	err = h.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Invoice{}).
			Where("id = ? AND archive_status IN (?, ?) AND status != ? AND deleted_at IS NULL",
				id, models.InvoiceArchiveStatusPending, models.InvoiceArchiveStatusConfirmed, models.InvoiceStatusReimbursed).
			Updates(map[string]any{
				"archive_status": models.InvoiceArchiveStatusVoided,
				"voided_by":      userID,
				"voided_at":      now,
				"voided_reason":  reason,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errInvoiceStateConflict
		}
		// 同事务审计：对象、旧新归档状态、作废原因
		rid := strconv.FormatUint(uint64(id), 10)
		return models.CreateAuditLogWithDB(tx, models.CreateAuditLogParams{
			UserID: &userID, Action: models.ActionVoidInvoice, Resource: "invoices", ResourceID: &rid,
			Status: models.StatusSuccess, StatusCode: http.StatusOK,
			Details: &models.LogDetails{Custom: map[string]any{
				"old_archive_status": invoice.ArchiveStatus,
				"new_archive_status": models.InvoiceArchiveStatusVoided,
				"reason":             reason,
			}},
		})
	})
	if err != nil {
		if errors.Is(err, errInvoiceStateConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "作废失败", nil)
		return
	}
	invoice.ArchiveStatus = models.InvoiceArchiveStatusVoided
	respondJSON(w, http.StatusOK, map[string]any{"item": invoice})
}

// correctInvoice 更正已确认发票（仅 admin、仅 confirmed；记录 old/new 审计）。
func (h *Handler) correctInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", nil)
		return
	}
	var payload invoiceCorrectRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", nil)
		return
	}
	if strings.TrimSpace(payload.Reason) == "" {
		respondError(w, http.StatusBadRequest, "更正原因必填", nil)
		return
	}
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
		return
	}
	if invoice.ArchiveStatus != models.InvoiceArchiveStatusConfirmed {
		respondError(w, http.StatusForbidden, "仅已确认发票可更正", nil)
		return
	}
	changes, updates := payload.changes(invoice)
	if len(changes) == 0 {
		respondError(w, http.StatusBadRequest, "没有可更正的字段", nil)
		return
	}
	if err := h.applyInvoiceCorrection(&invoice, userID, payload.Reason, changes, updates); err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"item": invoice})
}

// changes 计算 old/new 差异并生成白名单更新映射。
func (request invoiceCorrectRequest) changes(invoice models.Invoice) (map[string]any, map[string]any) {
	changes := map[string]any{}
	updates := map[string]any{}
	record := func(field string, oldValue, newValue any) {
		changes[field] = map[string]any{"old": oldValue, "new": newValue}
		updates[field] = newValue
	}
	if request.InvoiceNo != nil && *request.InvoiceNo != invoice.InvoiceNo {
		record("invoice_no", invoice.InvoiceNo, *request.InvoiceNo)
	}
	if request.InvoiceCode != nil && *request.InvoiceCode != invoice.InvoiceCode {
		record("invoice_code", invoice.InvoiceCode, *request.InvoiceCode)
	}
	if request.ElectronicInvoiceNo != nil && *request.ElectronicInvoiceNo != invoice.ElectronicInvoiceNo {
		record("electronic_invoice_no", invoice.ElectronicInvoiceNo, *request.ElectronicInvoiceNo)
	}
	if request.InvoiceDate != nil && !request.InvoiceDate.Equal(invoice.InvoiceDate) {
		record("invoice_date", invoice.InvoiceDate, *request.InvoiceDate)
	}
	if request.InvoiceType != nil && *request.InvoiceType != invoice.InvoiceType {
		record("invoice_type", invoice.InvoiceType, *request.InvoiceType)
	}
	if request.Amount != nil && *request.Amount != invoice.Amount {
		record("amount", invoice.Amount, *request.Amount)
	}
	if request.TaxAmount != nil && *request.TaxAmount != invoice.TaxAmount {
		record("tax_amount", invoice.TaxAmount, *request.TaxAmount)
	}
	if request.TotalAmount != nil && *request.TotalAmount != invoice.TotalAmount {
		record("total_amount", invoice.TotalAmount, *request.TotalAmount)
	}
	if request.Seller != nil && *request.Seller != invoice.Seller {
		record("seller", invoice.Seller, *request.Seller)
	}
	if request.SellerTaxNo != nil && *request.SellerTaxNo != invoice.SellerTaxNo {
		record("seller_tax_no", invoice.SellerTaxNo, *request.SellerTaxNo)
	}
	if request.Buyer != nil && *request.Buyer != invoice.Buyer {
		record("buyer", invoice.Buyer, *request.Buyer)
	}
	if request.BuyerTaxNo != nil && *request.BuyerTaxNo != invoice.BuyerTaxNo {
		record("buyer_tax_no", invoice.BuyerTaxNo, *request.BuyerTaxNo)
	}
	if request.Purpose != nil && *request.Purpose != invoice.Purpose {
		record("purpose", invoice.Purpose, *request.Purpose)
	}
	if request.Remark != nil && *request.Remark != invoice.Remark {
		record("remark", invoice.Remark, *request.Remark)
	}
	if request.VoucherType != nil && *request.VoucherType != invoice.VoucherType {
		record("voucher_type", invoice.VoucherType, *request.VoucherType)
	}
	return changes, updates
}

// applyInvoiceCorrection 事务执行更正：判重、写字段、重匹配、双审计（InvoiceCorrectionAudit + AuditLog）。
func (h *Handler) applyInvoiceCorrection(invoice *models.Invoice, userID uint, reason string, changes, updates map[string]any) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		var current models.Invoice
		if err := tx.First(&current, invoice.ID).Error; err != nil {
			return errInvoiceStateConflict
		}
		if current.ArchiveStatus != models.InvoiceArchiveStatusConfirmed {
			return errInvoiceStateConflict
		}
		// 身份键必须使用更正后的 voucher_type（更正可改凭证类型）
		newVoucher := current.VoucherType
		if value, ok := updates["voucher_type"].(models.InvoiceVoucherType); ok {
			newVoucher = value
		}
		newNo, newCode, newElectronic := current.InvoiceNo, current.InvoiceCode, current.ElectronicInvoiceNo
		if value, ok := updates["invoice_no"].(string); ok {
			newNo = value
		}
		if value, ok := updates["invoice_code"].(string); ok {
			newCode = value
		}
		if value, ok := updates["electronic_invoice_no"].(string); ok {
			newElectronic = value
		}
		identityKey := computeInvoiceIdentityKey(newVoucher, newNo, newCode, newElectronic)
		if identityKey != nil && invoiceIdentityConflict(tx, *identityKey, current.ID) {
			return errIdentityConflict
		}
		updates["identity_key"] = identityKey

		// 更正 buyer/buyer_tax_no 时重新计算匹配状态（不自动强匹配，仅评估）
		if buyerValue, ok := updates["buyer"].(string); ok {
			current.Buyer = buyerValue
		}
		if taxNoValue, ok := updates["buyer_tax_no"].(string); ok {
			current.BuyerTaxNo = taxNoValue
		}
		_, buyerChanged := updates["buyer"]
		_, taxNoChanged := updates["buyer_tax_no"]
		if buyerChanged || taxNoChanged {
			h.buyerMatchAtConfirm(tx, &current)
			updates["buyer_matched"] = current.BuyerMatched
			updates["buyer_match_note"] = current.BuyerMatchNote
		}

		if err := tx.Model(&current).Updates(updates).Error; err != nil {
			return err
		}
		// 双审计同事务：更正差异审计 + 全局审计日志
		audit := models.InvoiceCorrectionAudit{
			InvoiceID:   current.ID,
			Changes:     mustJSON(changes),
			Reason:      reason,
			CorrectedBy: userID,
		}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}
		rid := strconv.FormatUint(uint64(current.ID), 10)
		if err := models.CreateAuditLogWithDB(tx, models.CreateAuditLogParams{
			UserID: &userID, Action: models.ActionCorrectInvoice, Resource: "invoices", ResourceID: &rid,
			Status: models.StatusSuccess, StatusCode: http.StatusOK,
			Details: &models.LogDetails{Custom: map[string]any{"changes": changes, "reason": reason}},
		}); err != nil {
			return err
		}
		*invoice = current
		return nil
	})
}
