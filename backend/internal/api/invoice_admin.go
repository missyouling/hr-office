package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// 发票管理业务错误（不向客户端泄露 SQL/路径等内部信息）。
var (
	errInvoiceStateConflict = errors.New("发票状态已变化，请刷新后重试")
	errPurchaseMissing      = errors.New("关联采购单不存在")
	errIdentityConflict     = errors.New("发票身份键与活动记录冲突")
)

// mustJSON 将结构序列化为 JSON 字段值（序列化失败时返回空对象）。
func mustJSON(value any) datatypes.JSON {
	encoded, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(encoded)
}

// invoiceCorrectableFields 允许管理员更正的业务字段白名单。
// 受控身份/附件/责任/审批字段一律禁止更正。
var invoiceCorrectableFields = map[string]bool{
	"invoice_no": true, "invoice_code": true, "electronic_invoice_no": true,
	"invoice_date": true, "invoice_type": true, "amount": true, "tax_amount": true,
	"total_amount": true, "seller": true, "seller_tax_no": true, "buyer": true,
	"buyer_tax_no": true, "purpose": true, "remark": true, "voucher_type": true,
}

// invoiceCorrectRequest 更正请求白名单 DTO，仅接受明确列出的字段。
type invoiceCorrectRequest struct {
	Reason              string                     `json:"reason"`
	InvoiceNo           *string                    `json:"invoice_no"`
	InvoiceCode         *string                    `json:"invoice_code"`
	ElectronicInvoiceNo *string                    `json:"electronic_invoice_no"`
	InvoiceDate         *time.Time                 `json:"invoice_date"`
	InvoiceType         *string                    `json:"invoice_type"`
	Amount              *float64                   `json:"amount"`
	TaxAmount           *float64                   `json:"tax_amount"`
	TotalAmount         *float64                   `json:"total_amount"`
	Seller              *string                    `json:"seller"`
	SellerTaxNo         *string                    `json:"seller_tax_no"`
	Buyer               *string                    `json:"buyer"`
	BuyerTaxNo          *string                    `json:"buyer_tax_no"`
	Purpose             *string                    `json:"purpose"`
	Remark              *string                    `json:"remark"`
	VoucherType         *models.InvoiceVoucherType `json:"voucher_type"`
}

// invoiceIdentityConflict 判断身份键是否与活动记录冲突（软删除记录允许复用）。
func invoiceIdentityConflict(db *gorm.DB, identityKey string, excludeID uint) bool {
	var count int64
	db.Model(&models.Invoice{}).
		Where("identity_key = ? AND deleted_at IS NULL AND id != ?", identityKey, excludeID).
		Count(&count)
	return count > 0
}

// confirmInvoice 确认归档（仅 admin、仅 approved -> confirmed）。
func (h *Handler) confirmInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", nil)
		return
	}
	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
		return
	}
	if invoice.Status != models.InvoiceStatusApproved || invoice.ArchiveStatus != models.InvoiceArchiveStatusPending {
		respondError(w, http.StatusForbidden, "仅已审批且待确认的发票可确认", nil)
		return
	}
	if invoice.Status == models.InvoiceStatusReimbursed {
		respondError(w, http.StatusForbidden, "已报销发票不可确认", nil)
		return
	}
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	warnings, err := h.confirmInvoiceWithChecks(&invoice, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"item": invoice, "warnings": warnings})
}

// confirmInvoiceWithChecks 执行确认事务：来源校验、购方匹配、状态写入、同事务审计。
func (h *Handler) confirmInvoiceWithChecks(invoice *models.Invoice, userID uint) ([]string, error) {
	warnings := []string{}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var current models.Invoice
		if err := tx.First(&current, invoice.ID).Error; err != nil {
			return err
		}
		if current.Status != models.InvoiceStatusApproved || current.ArchiveStatus != models.InvoiceArchiveStatusPending {
			return errInvoiceStateConflict
		}
		// 完整来源校验：所有合法来源、记录存在未删除、当前用户有权访问
		if err := h.validateInvoiceSource(tx, userID, current.SourceType, current.SourceID); err != nil {
			return err
		}
		if current.SourceType == models.InvoiceSourceOffice && current.SourceID != nil {
			var purchase models.OfficePurchase
			if err := tx.First(&purchase, *current.SourceID).Error; err != nil {
				return errPurchaseMissing
			}
			warnings = append(warnings, invoicePurchaseWarnings(current, purchase)...)
		}
		// 购方主体匹配：税号完全匹配强匹配；名称匹配缺税号仅 warning，不自动强匹配
		warnings = append(warnings, h.buyerMatchAtConfirm(tx, &current)...)
		now := time.Now()
		// 显式写入当前快照，确保响应对象与数据库一致
		// （buyer_matched/buyer_match_note 已由 buyerMatchAtConfirm 通过指针更新）
		current.ArchiveStatus = models.InvoiceArchiveStatusConfirmed
		current.ConfirmedBy = &userID
		current.ConfirmedAt = &now
		updates := map[string]any{
			"archive_status":   current.ArchiveStatus,
			"confirmed_by":     current.ConfirmedBy,
			"confirmed_at":     current.ConfirmedAt,
			"buyer_matched":    current.BuyerMatched,
			"buyer_match_note": current.BuyerMatchNote,
		}
		if err := tx.Model(&current).Updates(updates).Error; err != nil {
			return err
		}
		// 同事务审计：对象、旧新归档状态、警告摘要
		rid := strconv.FormatUint(uint64(current.ID), 10)
		if err := models.CreateAuditLogWithDB(tx, models.CreateAuditLogParams{
			UserID: &userID, Action: models.ActionConfirmInvoice, Resource: "invoices", ResourceID: &rid,
			Status: models.StatusSuccess, StatusCode: http.StatusOK,
			Details: &models.LogDetails{Custom: map[string]any{
				"old_archive_status": models.InvoiceArchiveStatusPending,
				"new_archive_status": models.InvoiceArchiveStatusConfirmed,
				"warnings":           warnings,
			}},
		}); err != nil {
			return err
		}
		*invoice = current
		return nil
	})
	return warnings, err
}

// invoicePurchaseWarnings 生成采购关联软预警（不阻塞确认）。
func invoicePurchaseWarnings(invoice models.Invoice, purchase models.OfficePurchase) []string {
	warnings := []string{}
	if purchase.SupplierName != "" && invoice.Seller != "" && purchase.SupplierName != invoice.Seller {
		warnings = append(warnings, "发票销售方与采购单供应商不一致")
	}
	if purchase.TotalAmount > 0 && invoice.TotalAmount > 0 && math.Abs(purchase.TotalAmount-invoice.TotalAmount) > 0.01 {
		warnings = append(warnings, "发票金额与采购金额不一致")
	}
	if !purchase.PurchaseDate.IsZero() && !invoice.InvoiceDate.IsZero() && invoice.InvoiceDate.Before(purchase.PurchaseDate) {
		warnings = append(warnings, "发票日期早于采购日期")
	}
	return warnings
}
