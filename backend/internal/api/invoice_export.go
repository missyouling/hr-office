package api

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// invoiceExportMaxRows 单次导出最大行数，防止内存与响应过大。
const invoiceExportMaxRows = 10000

// buildInvoiceListQuery 构建发票列表公共筛选查询（list 与 CSV 导出完全共享）。
// scope 由调用方解析：admin 全量、manager 本部门、普通用户本人；软删默认不可见。
func (h *Handler) buildInvoiceListQuery(r *http.Request, scope invoiceAccessScope, userID uint) *gorm.DB {
	q := r.URL.Query()
	query := h.applyInvoiceScope(h.db.Model(&models.Invoice{}), scope, userID)
	if st := q.Get("status"); st != "" {
		query = query.Where("status = ?", st)
	}
	if st := q.Get("archive_status"); st != "" {
		query = query.Where("archive_status = ?", st)
	}
	if seller := q.Get("seller"); seller != "" {
		query = query.Where("seller LIKE ?", "%"+seller+"%")
	}
	if st := q.Get("source_type"); st != "" {
		query = query.Where("source_type = ?", st)
	}
	if applicantID := q.Get("applicant_id"); applicantID != "" {
		if aid, err := strconv.ParseUint(applicantID, 10, 64); err == nil {
			query = query.Where("applicant_id = ?", aid)
		}
	}
	if kw := q.Get("keyword"); kw != "" {
		query = query.Where("(invoice_no LIKE ? OR purpose LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}
	if df := q.Get("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query = query.Where("invoice_date >= ?", t)
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query = query.Where("invoice_date <= ?", t)
		}
	}
	return query
}

// exportInvoicesCSV 导出发票 CSV（与 list 共享筛选参数，需 manager+ 中间件）。
// 安全措施：资源范围过滤、公式注入防护、导出量上限、字段最小化（不含路径/原文）、业务导出审计。
func (h *Handler) exportInvoicesCSV(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	scope := h.resolveInvoiceAccessScope(userID)
	query := h.buildInvoiceListQuery(r, scope, userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "导出发票失败", nil)
		return
	}
	if total > invoiceExportMaxRows {
		respondError(w, http.StatusBadRequest, "导出数据量过大，请缩小筛选范围", nil)
		return
	}

	var invoices []models.Invoice
	if err := query.Order("created_at DESC").Find(&invoices).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "导出发票失败", nil)
		return
	}

	// 内存构建 CSV，审计成功后才写响应（审计失败视为导出失败，保持一致性）
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"ID", "发票号码", "发票代码", "电子票号", "开票日期", "发票类型", "金额", "税额", "价税合计", "销售方", "销售方税号", "购方", "购方税号", "状态", "归档状态", "来源类型", "申请人ID", "备注"})
	for _, invoice := range invoices {
		_ = writer.Write([]string{
			strconv.FormatUint(uint64(invoice.ID), 10),
			sanitizeCSVCell(invoice.InvoiceNo), sanitizeCSVCell(invoice.InvoiceCode), sanitizeCSVCell(invoice.ElectronicInvoiceNo),
			invoice.InvoiceDate.Format("2006-01-02"), sanitizeCSVCell(invoice.InvoiceType),
			strconv.FormatFloat(invoice.Amount, 'f', 2, 64),
			strconv.FormatFloat(invoice.TaxAmount, 'f', 2, 64),
			strconv.FormatFloat(invoice.TotalAmount, 'f', 2, 64),
			sanitizeCSVCell(invoice.Seller), sanitizeCSVCell(invoice.SellerTaxNo),
			sanitizeCSVCell(invoice.Buyer), sanitizeCSVCell(invoice.BuyerTaxNo),
			invoice.Status, string(invoice.ArchiveStatus), invoice.SourceType,
			strconv.FormatUint(uint64(derefUint(invoice.ApplicantID)), 10), sanitizeCSVCell(invoice.Remark),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		respondError(w, http.StatusInternalServerError, "导出发票失败", nil)
		return
	}

	// 业务导出审计：记录筛选摘要与导出条数
	rid := strconv.FormatUint(uint64(total), 10)
	if err := models.CreateAuditLogWithDB(h.db, models.CreateAuditLogParams{
		UserID: &userID, Action: models.ActionExportInvoices, Resource: "invoices", ResourceID: &rid,
		Method: r.Method, Path: r.URL.Path, Status: models.StatusSuccess, StatusCode: http.StatusOK,
		Details: &models.LogDetails{Custom: map[string]any{"filters": r.URL.Query(), "count": len(invoices)}},
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "导出审计写入失败", nil)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=invoices.csv")
	// UTF-8 BOM，便于 Excel 正确识别中文编码
	w.Write([]byte("\xEF\xBB\xBF"))
	_, _ = w.Write(buffer.Bytes())
}

// sanitizeCSVCell 公式注入防护：值以 =、+、-、@ 开头时前置单引号，防止被电子表格当作公式执行。
func sanitizeCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	}
	return strings.TrimSpace(value)
}

// derefUint 解引用 uint 指针，空指针返回 0。
func derefUint(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}
