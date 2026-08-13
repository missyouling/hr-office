package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// errInvoiceIdentityConflict 身份键与活动记录冲突的哨兵错误（事务内判重用）
var errInvoiceIdentityConflict = errors.New("发票身份键与活动记录重复")

// ======== 发票管理路由注册（P7.3） ========

// registerInvoiceRoutes 注册发票管理所有路由（前缀 /api/invoices）
func (h *Handler) registerInvoiceRoutes(r chi.Router) {
	ensureBuyerEntitySettingTable(h.db)
	r.Route("/invoices", func(ir chi.Router) {
		// 创建草稿（任意登录用户）
		ir.Post("/", h.createInvoice)
		ir.Post("/upload", h.uploadInvoicePDFs)

		// 列表、统计和 CSV 导出（经理及以上；manager 仅本部门，admin 全量）
		ir.Group(func(mgr chi.Router) {
			mgr.Use(middleware.RequireManagerOrAbove(h.db))
			mgr.Get("/", h.listInvoices)
			mgr.Get("/stats", h.invoiceStats)
			mgr.Get("/export", h.exportInvoicesCSV)
		})

		// 购方主体设置（仅 admin）
		ir.Group(func(adm chi.Router) {
			adm.Use(middleware.RequireAdmin(h.db))
			adm.Get("/buyer-entity", h.getBuyerEntitySetting)
			adm.Put("/buyer-entity", h.updateBuyerEntitySetting)
		})

		// 单条发票操作
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/attachment", h.previewInvoiceAttachment)
			sr.Get("/attachment/download", h.downloadInvoiceAttachment)
			sr.Get("/", h.getInvoice)
			sr.Put("/", h.updateInvoice)
			sr.Delete("/", h.deleteInvoice)
			sr.Post("/submit", h.submitInvoice)

			// 审批与归档操作（仅 admin）
			sr.Group(func(adm chi.Router) {
				adm.Use(middleware.RequireAdmin(h.db))
				adm.Post("/approve", h.approveInvoice)
				adm.Post("/reject", h.rejectInvoice)
				adm.Post("/confirm", h.confirmInvoice)
				adm.Post("/void", h.voidInvoice)
				adm.Post("/correct", h.correctInvoice)
			})

			// 报销操作（admin/manager）
			sr.With(middleware.RequireManagerOrAbove(h.db)).
				Post("/reimburse", h.reimburseInvoice)
		})
	})
}

// ensureBuyerEntitySettingTable 确保购方主体设置表存在（幂等）。
func ensureBuyerEntitySettingTable(db *gorm.DB) {
	if err := db.AutoMigrate(&models.BuyerEntitySetting{}); err != nil {
		return
	}
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

// ======== 发票列表与创建 ========

// listInvoices 列表查询（分页 + 多条件筛选，需 manager+；资源范围过滤）。
func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	scope := h.resolveInvoiceAccessScope(userID)
	query := h.buildInvoiceListQuery(r, scope, userID)

	var total int64
	query.Count(&total)

	var invoices []models.Invoice
	if err := query.Preload("Applicant").Preload("Approver").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&invoices).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "查询发票列表失败", nil)
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

	var payload invoiceWriteRequest
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

	// 采购关联校验：类型合法、记录存在、当前用户有权访问
	if err := h.validateInvoiceSource(h.db, userID, payload.SourceType, payload.SourceID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	invoice := payload.newInvoice(userID)

	// 活动记录身份键硬判重 + 创建放入同一事务，避免并发下检查与写入之间的竞态窗口
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if invoice.IdentityKey != nil && invoiceIdentityConflict(tx, *invoice.IdentityKey, 0) {
			return errInvoiceIdentityConflict
		}
		return tx.Create(&invoice).Error
	})
	if err != nil {
		if errors.Is(err, errInvoiceIdentityConflict) {
			respondError(w, http.StatusConflict, "发票身份键与活动记录重复", nil)
			return
		}
		respondError(w, http.StatusConflict, "创建发票失败，发票号可能重复", nil)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"item": invoice,
	})
}

// 保持 time 导入引用（InvoiceDate 默认值使用）
var _ = time.Now
