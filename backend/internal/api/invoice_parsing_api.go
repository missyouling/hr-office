package api

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// invoiceParsingTaskResponse 解析任务对外契约：隐藏 worker 内部调度字段（锁 token、锁租约、可用时间）。
type invoiceParsingTaskResponse struct {
	ID           uint       `json:"id"`
	InvoiceID    uint       `json:"invoice_id"`
	Status       string     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	MaxAttempts  int        `json:"max_attempts"`
	ErrorCode    string     `json:"error_code"`
	LastError    string     `json:"last_error"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// newInvoiceParsingTaskResponse 将任务模型转换为对外响应结构。
func newInvoiceParsingTaskResponse(task *models.InvoiceParsingTask) invoiceParsingTaskResponse {
	return invoiceParsingTaskResponse{
		ID:           task.ID,
		InvoiceID:    task.InvoiceID,
		Status:       string(task.Status),
		AttemptCount: task.AttemptCount,
		MaxAttempts:  task.MaxAttempts,
		ErrorCode:    task.ErrorCode,
		LastError:    task.LastError,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

// loadInvoiceForParsingTask 加载发票并做资源级授权；授权失败时写入响应并返回 false。
// 越权访问与不存在统一返回 404，避免泄露资源存在性（与 getInvoice 一致）。
func (h *Handler) loadInvoiceForParsingTask(w http.ResponseWriter, userID, invoiceID uint) (*models.Invoice, bool) {
	var invoice models.Invoice
	if err := h.db.First(&invoice, invoiceID).Error; err != nil {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
		return nil, false
	}
	if !h.canAccessInvoice(userID, &invoice) {
		respondError(w, http.StatusNotFound, "发票不存在", nil)
		return nil, false
	}
	return &invoice, true
}

// getInvoiceParsingTask 查询当前用户可访问发票的解析任务详情。
// GET /api/invoices/{id}/parsing-task
func (h *Handler) getInvoiceParsingTask(w http.ResponseWriter, r *http.Request) {
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
	if _, ok := h.loadInvoiceForParsingTask(w, userID, id); !ok {
		return
	}
	var task models.InvoiceParsingTask
	if err := h.db.Where("invoice_id = ?", id).First(&task).Error; err != nil {
		respondError(w, http.StatusNotFound, "该发票暂无解析任务", nil)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"item": newInvoiceParsingTaskResponse(&task)})
}

// retryInvoiceParsingTask 手动重试失败解析任务：仅 failed 可重置为 pending。
// POST /api/invoices/{id}/parsing-task/retry
func (h *Handler) retryInvoiceParsingTask(w http.ResponseWriter, r *http.Request) {
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
	if _, ok := h.loadInvoiceForParsingTask(w, userID, id); !ok {
		return
	}
	var task models.InvoiceParsingTask
	if err := h.db.Where("invoice_id = ?", id).First(&task).Error; err != nil {
		respondError(w, http.StatusNotFound, "该发票暂无解析任务", nil)
		return
	}
	if task.Status != models.InvoiceParsingTaskFailed {
		respondError(w, http.StatusConflict, "仅失败状态的解析任务可重试", nil)
		return
	}
	// 条件更新：仅 failed 状态可重置，避免与 worker 或并发请求竞争
	now := time.Now()
	result := h.db.Model(&models.InvoiceParsingTask{}).
		Where("id = ? AND invoice_id = ? AND status = ?", task.ID, id, models.InvoiceParsingTaskFailed).
		Updates(map[string]any{
			"status":        models.InvoiceParsingTaskPending,
			"attempt_count": 0,                 // 重置尝试次数：达到 max_attempts 的任务若不重置将无法被 worker 领取
			"available_at":  gorm.Expr("NULL"), // 立即可领取
			"locked_by":     "",
			"locked_until":  gorm.Expr("NULL"),
			"error_code":    "",
			"last_error":    "",
			"started_at":    gorm.Expr("NULL"),
			"completed_at":  gorm.Expr("NULL"),
			"updated_at":    now,
		})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "重试解析任务失败", nil)
		return
	}
	if result.RowsAffected != 1 {
		// 并发下状态已变化（如 worker 已领取），按当前状态拒绝
		respondError(w, http.StatusConflict, "仅失败状态的解析任务可重试", nil)
		return
	}
	// 注意：重新加载必须使用新的零值变量。GORM 在 SQLite 下将字段更新为
	// NULL 后，若复用同一结构体，*time.Time 指针字段会保留旧值不重置。
	var refreshed models.InvoiceParsingTask
	if err := h.db.First(&refreshed, task.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "重试解析任务失败", nil)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"item": newInvoiceParsingTaskResponse(&refreshed)})
}
