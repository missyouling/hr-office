package api

import (
	"encoding/json"
	"net/http"

	"siapp/internal/models"
)

// ======== 发票详情 / 更新 / 删除 ========

// getInvoice 获取单个发票详情（资源级授权：普通用户仅本人，manager 本部门，admin 全量）
func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
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
	if err := h.db.Preload("Applicant").Preload("Approver").First(&invoice, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", err)
		return
	}

	// 越权访问与不存在统一返回 404，避免泄露资源存在性
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
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

	// 确认后锁定：归档状态非待确认时禁止编辑
	if invoice.ArchiveStatus != models.InvoiceArchiveStatusPending {
		respondError(w, http.StatusForbidden, "已确认或已作废的发票不可修改", nil)
		return
	}

	// 仅创建者可改（除非自己是 admin/manager 本部门）
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusForbidden, "无权修改他人发票", nil)
		return
	}

	var payload invoiceWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", err)
		return
	}

	// 采购关联校验：类型合法、记录存在、当前用户有权访问
	if err := h.validateInvoiceSource(h.db, userID, payload.SourceType, payload.SourceID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// 仅允许更新特定字段，防止篡改状态和发票号
	updates := payload.updates()
	// 身份键基于更新后的完整字段计算（发票号/代码/电子票号/凭证类型不可通过更新修改，取现值）
	identityKey := computeInvoiceIdentityKey(invoice.VoucherType, invoice.InvoiceNo, invoice.InvoiceCode, invoice.ElectronicInvoiceNo)
	if identityKey != nil && invoiceIdentityConflict(h.db, *identityKey, invoice.ID) {
		respondError(w, http.StatusConflict, "发票身份键与活动记录重复", nil)
		return
	}
	updates["identity_key"] = identityKey
	if err := h.db.Model(&invoice).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "更新发票失败", nil)
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

	// 确认后锁定：归档状态非待确认时禁止删除
	if invoice.ArchiveStatus != models.InvoiceArchiveStatusPending {
		respondError(w, http.StatusForbidden, "已确认或已作废的发票不可删除", nil)
		return
	}

	// 仅创建者可删（除非自己是 admin/manager 本部门）
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusForbidden, "无权删除他人发票", nil)
		return
	}

	if err := h.db.Delete(&invoice).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "删除发票失败", nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
