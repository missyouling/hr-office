package api

import (
	"time"

	"siapp/internal/models"
)

// invoiceWriteRequest 仅包含用户可写的发票基础业务字段。
// 未列出的字段（包括遗留 attachment_url）会由 JSON 解码器忽略。
type invoiceWriteRequest struct {
	InvoiceNo           string                    `json:"invoice_no"`
	InvoiceCode         string                    `json:"invoice_code"`
	ElectronicInvoiceNo string                    `json:"electronic_invoice_no"`
	InvoiceDate         time.Time                 `json:"invoice_date"`
	InvoiceType         string                    `json:"invoice_type"`
	Amount              float64                   `json:"amount"`
	TaxAmount           float64                   `json:"tax_amount"`
	TotalAmount         float64                   `json:"total_amount"`
	Seller              string                    `json:"seller"`
	SellerTaxNo         string                    `json:"seller_tax_no"`
	Buyer               string                    `json:"buyer"`
	BuyerTaxNo          string                    `json:"buyer_tax_no"`
	Purpose             string                    `json:"purpose"`
	Remark              string                    `json:"remark"`
	SourceType          string                    `json:"source_type"`
	SourceID            *uint                     `json:"source_id"`
	ApplicantID         *uint                     `json:"applicant_id"`
	VoucherType         models.InvoiceVoucherType `json:"voucher_type"`
}

func (request invoiceWriteRequest) newInvoice(userID uint) models.Invoice {
	identityKey := computeInvoiceIdentityKey(request.VoucherType, request.InvoiceNo, request.InvoiceCode, request.ElectronicInvoiceNo)
	invoice := models.Invoice{
		UserID:              &userID,
		InvoiceNo:           request.InvoiceNo,
		InvoiceCode:         request.InvoiceCode,
		ElectronicInvoiceNo: request.ElectronicInvoiceNo,
		IdentityKey:         identityKey,
		InvoiceDate:         request.InvoiceDate,
		InvoiceType:         request.InvoiceType,
		Amount:              request.Amount,
		TaxAmount:           request.TaxAmount,
		TotalAmount:         request.TotalAmount,
		Seller:              request.Seller,
		SellerTaxNo:         request.SellerTaxNo,
		Buyer:               request.Buyer,
		BuyerTaxNo:          request.BuyerTaxNo,
		Purpose:             request.Purpose,
		Remark:              request.Remark,
		SourceType:          request.SourceType,
		SourceID:            request.SourceID,
		ApplicantID:         request.ApplicantID,
		VoucherType:         request.VoucherType,
		Status:              models.InvoiceStatusDraft,
		ArchiveStatus:       models.InvoiceArchiveStatusPending,
	}
	if invoice.ApplicantID == nil {
		invoice.ApplicantID = &userID
	}
	if invoice.InvoiceDate.IsZero() {
		invoice.InvoiceDate = time.Now()
	}
	return invoice
}

func (request invoiceWriteRequest) updates() map[string]interface{} {
	// 注意：发票号/代码/电子票号/凭证类型不可通过更新修改，identity_key 由
	// updateInvoice 基于现有字段统一计算并写入，此处不包含。
	return map[string]interface{}{
		"invoice_type":  request.InvoiceType,
		"amount":        request.Amount,
		"tax_amount":    request.TaxAmount,
		"total_amount":  request.TotalAmount,
		"seller":        request.Seller,
		"seller_tax_no": request.SellerTaxNo,
		"buyer":         request.Buyer,
		"buyer_tax_no":  request.BuyerTaxNo,
		"purpose":       request.Purpose,
		"remark":        request.Remark,
		"source_type":   request.SourceType,
		"source_id":     request.SourceID,
		"invoice_date":  request.InvoiceDate,
	}
}
