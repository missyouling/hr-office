package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// ======== 发票管理路由注册（P7.3） ========

// registerInvoiceRoutes 注册发票管理所有路由（前缀 /api/invoices）
func (h *Handler) registerInvoiceRoutes(r chi.Router) {
	r.Route("/invoices", func(ir chi.Router) {
		// 创建草稿（任意登录用户）
		ir.Post("/", h.createInvoice)

		// 列表和统计（经理及以上）
		ir.Group(func(mgr chi.Router) {
			mgr.Use(middleware.RequireManagerOrAbove(h.db))
			mgr.Get("/", h.listInvoices)
			mgr.Get("/stats", h.invoiceStats)
		})

		// 单条发票操作
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/", h.getInvoice)
			sr.Put("/", h.updateInvoice)
			sr.Delete("/", h.deleteInvoice)
			sr.Post("/submit", h.submitInvoice)

			// 审批操作（仅 admin）
			sr.Group(func(adm chi.Router) {
				adm.Use(middleware.RequireAdmin(h.db))
				adm.Post("/approve", h.approveInvoice)
				adm.Post("/reject", h.rejectInvoice)
			})

			// 报销操作（admin/manager）
			sr.With(middleware.RequireManagerOrAbove(h.db)).
				Post("/reimburse", h.reimburseInvoice)
		})
	})
}

// ======== 辅助函数 ========

// getInvoiceUserID 从请求上下文提取当前用户 ID
func getInvoiceUserID(r *http.Request) (uint, error) {
	return auth.GetUserIDFromContext(r.Context())
}

// parseInvoiceID 从 URL 路径参数中解析发票 ID
func parseInvoiceID(r *http.Request) (uint, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// ======== 发票 CRUD ========

// listInvoices 列表查询（分页 + 多条件筛选，需 manager+）
func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&models.Invoice{})

	// 多条件筛选
	if st := q.Get("status"); st != "" {
		query = query.Where("status = ?", st)
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

	var total int64
	query.Count(&total)

	var invoices []models.Invoice
	if err := query.Preload("Applicant").Preload("Approver").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&invoices).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询发票列表失败", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":     invoices,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// createInvoice 创建发票（默认草稿状态）
func (h *Handler) createInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	var payload models.Invoice
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", err)
		return
	}

	// 必填字段校验
	if payload.InvoiceNo == "" {
		respondError(w, http.StatusBadRequest, "发票号不能为空", nil)
		return
	}
	if payload.Seller == "" {
		respondError(w, http.StatusBadRequest, "销售方不能为空", nil)
		return
	}
	if payload.Amount <= 0 {
		respondError(w, http.StatusBadRequest, "金额必须大于0", nil)
		return
	}

	payload.ID = 0
	payload.UserID = &userID
	payload.Status = models.InvoiceStatusDraft
	if payload.ApplicantID == nil {
		payload.ApplicantID = &userID
	}
	// 若未指定日期，默认今天
	if payload.InvoiceDate.IsZero() {
		payload.InvoiceDate = time.Now()
	}

	if err := h.db.Create(&payload).Error; err != nil {
		respondError(w, http.StatusConflict, "创建发票失败，发票号可能重复", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"item": payload,
	})
}

// getInvoice 获取单个发票详情
func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.Preload("Applicant").Preload("Approver").First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// updateInvoice 更新发票（仅草稿状态可修改）
func (h *Handler) updateInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	// 仅草稿可改
	if invoice.Status != models.InvoiceStatusDraft {
		respondError(w, http.StatusForbidden, "仅草稿状态的发票可修改", nil)
		return
	}

	// 仅创建者可改（除非自己是 admin）
	if *invoice.UserID != userID {
		respondError(w, http.StatusForbidden, "无权修改他人发票", nil)
		return
	}

	var payload models.Invoice
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", err)
		return
	}

	// 仅允许更新特定字段，防止篡改状态和发票号
	updates := map[string]interface{}{
		"invoice_type":   payload.InvoiceType,
		"amount":         payload.Amount,
		"tax_amount":     payload.TaxAmount,
		"total_amount":   payload.TotalAmount,
		"seller":         payload.Seller,
		"seller_tax_no":  payload.SellerTaxNo,
		"buyer":          payload.Buyer,
		"purpose":        payload.Purpose,
		"remark":         payload.Remark,
		"attachment_url": payload.AttachmentURL,
		"source_type":    payload.SourceType,
		"source_id":      payload.SourceID,
		"invoice_date":   payload.InvoiceDate,
	}
	if err := h.db.Model(&invoice).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新发票失败", err)
		return
	}

	h.db.Preload("Applicant").Preload("Approver").First(&invoice, id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// deleteInvoice 删除发票（仅草稿状态可删除）
func (h *Handler) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	if invoice.Status != models.InvoiceStatusDraft {
		respondError(w, http.StatusForbidden, "仅草稿状态的发票可删除", nil)
		return
	}

	if *invoice.UserID != userID {
		respondError(w, http.StatusForbidden, "无权删除他人发票", nil)
		return
	}

	if err := h.db.Delete(&invoice).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除发票失败", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// ======== 发票业务操作 ========

// submitInvoice 提交发票（草稿 → 已提交）
func (h *Handler) submitInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	if invoice.Status != models.InvoiceStatusDraft {
		respondError(w, http.StatusForbidden, "仅草稿状态可提交", nil)
		return
	}

	// 仅创建者可提交
	if *invoice.UserID != userID {
		respondError(w, http.StatusForbidden, "无权提交他人发票", nil)
		return
	}

	if err := h.db.Model(&invoice).Update("status", models.InvoiceStatusSubmitted).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "提交失败", err)
		return
	}

	invoice.Status = models.InvoiceStatusSubmitted
	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// approveInvoice 审批通过（已提交 → 已审批，需 admin 中间件）
func (h *Handler) approveInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	if invoice.Status != models.InvoiceStatusSubmitted {
		respondError(w, http.StatusForbidden, "仅已提交状态可审批", nil)
		return
	}

	var body struct {
		Remark string `json:"approval_remark"`
	}
	// 可选备注，忽略解析错误
	json.NewDecoder(r.Body).Decode(&body)

	now := time.Now()
	updates := map[string]interface{}{
		"status":          models.InvoiceStatusApproved,
		"approver_id":     userID,
		"approved_at":     now,
		"approval_remark": body.Remark,
	}
	if err := h.db.Model(&invoice).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "审批失败", err)
		return
	}

	invoice.Status = models.InvoiceStatusApproved
	invoice.ApproverID = &userID
	invoice.ApprovedAt = &now
	invoice.ApprovalRemark = body.Remark
	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// rejectInvoice 驳回发票（已提交 → 已驳回，需 admin 中间件）
func (h *Handler) rejectInvoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", err)
		return
	}

	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	if invoice.Status != models.InvoiceStatusSubmitted {
		respondError(w, http.StatusForbidden, "仅已提交状态可驳回", nil)
		return
	}

	var body struct {
		Remark string `json:"approval_remark"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	now := time.Now()
	updates := map[string]interface{}{
		"status":          models.InvoiceStatusRejected,
		"approver_id":     userID,
		"approved_at":     now,
		"approval_remark": body.Remark,
	}
	if err := h.db.Model(&invoice).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "驳回失败", err)
		return
	}

	invoice.Status = models.InvoiceStatusRejected
	invoice.ApproverID = &userID
	invoice.ApprovedAt = &now
	invoice.ApprovalRemark = body.Remark
	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// reimburseInvoice 报销发票（已审批 → 已报销，需 admin/manager 中间件）
func (h *Handler) reimburseInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效的发票ID", err)
		return
	}

	var invoice models.Invoice
	if err := h.db.First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	if invoice.Status != models.InvoiceStatusApproved {
		respondError(w, http.StatusForbidden, "仅已审批状态可报销", nil)
		return
	}

	var body struct {
		ReimburseAmount float64 `json:"reimburse_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", err)
		return
	}

	// 实报销金额默认等于发票总金额
	reimburseAmount := body.ReimburseAmount
	if reimburseAmount <= 0 {
		reimburseAmount = invoice.TotalAmount
	}

	updates := map[string]interface{}{
		"status":           models.InvoiceStatusReimbursed,
		"reimburse_amount": reimburseAmount,
	}
	if err := h.db.Model(&invoice).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "报销操作失败", err)
		return
	}

	invoice.Status = models.InvoiceStatusReimbursed
	invoice.ReimburseAmount = reimburseAmount
	respondJSON(w, http.StatusOK, map[string]interface{}{"item": invoice})
}

// invoiceStats 发票统计（总数/各状态分布/来源分布，需 manager+ 中间件）
func (h *Handler) invoiceStats(w http.ResponseWriter, r *http.Request) {
	type statusCount struct {
		Status string  `json:"status"`
		Count  int64   `json:"count"`
		Amount float64 `json:"amount"`
	}

	var totalCount int64
	var totalAmount float64
	h.db.Model(&models.Invoice{}).Count(&totalCount)
	h.db.Model(&models.Invoice{}).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalAmount)

	// 各状态统计
	var statusCounts []statusCount
	h.db.Model(&models.Invoice{}).
		Select("status, COUNT(*) as count, COALESCE(SUM(total_amount), 0) as amount").
		Group("status").
		Find(&statusCounts)

	// 按来源类型统计
	type sourceCount struct {
		SourceType string `json:"source_type"`
		Count      int64  `json:"count"`
	}
	var sourceCounts []sourceCount
	h.db.Model(&models.Invoice{}).
		Select("source_type, COUNT(*) as count").
		Where("source_type != ''").
		Group("source_type").
		Find(&sourceCounts)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_count":  totalCount,
		"total_amount": totalAmount,
		"by_status":    statusCounts,
		"by_source":    sourceCounts,
	})
}
