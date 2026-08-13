package api

import (
	"encoding/json"
	"net/http"
	"time"

	"siapp/internal/models"
)

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

	// 仅创建者可提交（除非自己是 admin/manager 本部门）
	if !h.canAccessInvoice(userID, &invoice) {
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

	// 资源级授权（admin 中间件已保证角色，此处防御性兜底）
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusForbidden, "无权审批该发票", nil)
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

	// 资源级授权（admin 中间件已保证角色，此处防御性兜底）
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusForbidden, "无权驳回该发票", nil)
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

	// 资源级授权：manager 仅本部门，admin 全量
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusForbidden, "无权报销该发票", nil)
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
