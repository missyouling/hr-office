package api

import (
	"net/http"

	"siapp/internal/models"
)

// invoiceStats 发票统计（总数/各状态分布/来源分布，需 manager+ 中间件；资源范围过滤）。
func (h *Handler) invoiceStats(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	scope := h.resolveInvoiceAccessScope(userID)

	type statusCount struct {
		Status string  `json:"status"`
		Count  int64   `json:"count"`
		Amount float64 `json:"amount"`
	}

	var totalCount int64
	var totalAmount float64
	h.applyInvoiceScope(h.db.Model(&models.Invoice{}), scope, userID).Count(&totalCount)
	h.applyInvoiceScope(h.db.Model(&models.Invoice{}), scope, userID).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalAmount)

	// 各状态统计
	var statusCounts []statusCount
	h.applyInvoiceScope(h.db.Model(&models.Invoice{}), scope, userID).
		Select("status, COUNT(*) as count, COALESCE(SUM(total_amount), 0) as amount").
		Group("status").
		Find(&statusCounts)

	// 按来源类型统计
	type sourceCount struct {
		SourceType string `json:"source_type"`
		Count      int64  `json:"count"`
	}
	var sourceCounts []sourceCount
	h.applyInvoiceScope(h.db.Model(&models.Invoice{}), scope, userID).
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
