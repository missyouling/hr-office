package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

// OfficeSupplyHandler 办公用品模块 HTTP handler
type OfficeSupplyHandler struct {
	db           *gorm.DB
	analyticsSvc *service.OfficeAnalyticsService
}

// NewOfficeSupplyHandler 构造函数
func NewOfficeSupplyHandler(db *gorm.DB) *OfficeSupplyHandler {
	return &OfficeSupplyHandler{
		db:           db,
		analyticsSvc: service.NewOfficeAnalyticsService(db),
	}
}

// RegisterOfficeSupplyRoutes 注册办公用品模块所有路由（前缀 /api/office）
func (h *OfficeSupplyHandler) RegisterOfficeSupplyRoutes(r chi.Router) {
	r.Route("/office", func(or chi.Router) {
		// 分类
		or.Get("/categories", h.listOfficeCategories)
		or.Post("/categories", h.createOfficeCategory)
		or.Put("/categories/{id}", h.updateOfficeCategory)
		or.Delete("/categories/{id}", h.deleteOfficeCategory)

		// 供应商
		or.Get("/suppliers", h.listOfficeSuppliers)
		or.Post("/suppliers", h.createOfficeSupplier)
		or.Put("/suppliers/{id}", h.updateOfficeSupplier)
		or.Delete("/suppliers/{id}", h.deleteOfficeSupplier)

		// 用品（静态路由必须在 :id 前）
		or.Get("/supplies/units", h.listSupplyUnits)
		or.Get("/supplies/export", h.exportSuppliesCSV)
		or.Post("/supplies/import", h.importSuppliesCSV)
		or.Get("/supplies", h.listSupplies)
		or.Post("/supplies", h.createSupply)
		or.Put("/supplies/{id}", h.updateSupply)
		or.Delete("/supplies/{id}", h.deleteSupply)
		or.Get("/supplies/{id}", h.getSupply)

		// 采购单（见 office_purchase.go）
		or.Get("/purchases/unpaid", h.listUnpaidPurchases)
		or.Get("/purchases/export", h.exportPurchasesExcel)
		or.Get("/purchases", h.listPurchases)
		or.Post("/purchases", h.createPurchase)
		or.Post("/purchases/{id}/copy", h.copyPurchase)
		or.Get("/purchases/{id}/pdf", h.getPurchasePDF)
		or.Get("/purchases/{id}/excel", h.getPurchaseExcel)
		or.Get("/purchases/{id}", h.getPurchaseDetail)
		or.Put("/purchases/{id}", h.updatePurchase)
		or.Delete("/purchases/{id}", h.deletePurchase)

		// 请款单（见 office_payment.go）
		or.Get("/payment-requests", h.listPaymentRequests)
		or.Post("/payment-requests", h.createPaymentRequest)
		or.Get("/payment-requests/{id}", h.getPaymentRequest)
		or.Put("/payment-requests/{id}", h.updatePaymentRequest)
		or.Delete("/payment-requests/{id}", h.deletePaymentRequest)

		// 分析（见 office_analytics_api.go）
		or.Route("/analytics", func(ar chi.Router) {
			ar.Get("/summary", h.getAnalyticsSummary)
			ar.Get("/category-trend", h.getCategoryTrend)
			ar.Get("/frequency", h.getFrequency)
			ar.Get("/top-items", h.getTopItems)
			ar.Get("/price-anomaly", h.getPriceAnomaly)
			ar.Get("/suggestions", h.getSuggestions)
			ar.Get("/trend", h.getTrend)
			ar.Post("/report-pdf", h.getReportPDF)
		})

		// 系统（见 office_system.go）
		or.Post("/system/reset", h.resetSystem)
		or.Get("/system/backups", h.listBackups)
		or.Post("/system/backups", h.createBackup)
		or.Post("/system/backups/{id}/restore", h.restoreBackup)
		or.Delete("/system/backups/{id}", h.deleteBackup)
	})
}

// ======== 通用响应辅助 ========

func respondOfficeOK(w http.ResponseWriter, data map[string]interface{}) {
	data["ok"] = true
	respondJSON(w, http.StatusOK, data)
}

func respondOfficeError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"ok":    false,
		"error": message,
	})
}

func parseOfficeID(r *http.Request, key string) (uint, error) {
	value := strings.TrimSpace(chi.URLParam(r, key))
	if value == "" {
		return 0, fmt.Errorf("缺少参数 %s", key)
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func fetchOfficeUser(r *http.Request) (uint, error) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func mathRound(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}

// ======== 分类 ========

func (h *OfficeSupplyHandler) listOfficeCategories(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var categories []models.OfficeCategory
	if err := h.db.Where("user_id = ?", userID).Order("sort_order, id").Find(&categories).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询分类失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"items": categories})
}

func (h *OfficeSupplyHandler) createOfficeCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var payload struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondOfficeError(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	cat := models.OfficeCategory{
		UserID:    uintPointer(userID),
		Name:      payload.Name,
		SortOrder: payload.SortOrder,
	}
	if err := h.db.Create(&cat).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建分类失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"id": cat.ID})
}

func (h *OfficeSupplyHandler) updateOfficeCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的分类ID")
		return
	}
	var payload struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	result := h.db.Model(&models.OfficeCategory{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"name":       payload.Name,
			"sort_order": payload.SortOrder,
			"updated_at": time.Now(),
		})
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "未找到或未修改")
		return
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) deleteOfficeCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的分类ID")
		return
	}
	var count int64
	h.db.Model(&models.OfficeSupply{}).
		Where("category_id = ? AND user_id = ?", id, userID).Count(&count)
	if count > 0 {
		respondOfficeError(w, http.StatusBadRequest,
			fmt.Sprintf("该分类被 %d 个用品引用，无法删除", count))
		return
	}
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.OfficeCategory{})
	respondOfficeOK(w, map[string]interface{}{})
}

// ======== 供应商 ========

func (h *OfficeSupplyHandler) listOfficeSuppliers(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var suppliers []models.OfficeSupplier
	if err := h.db.Where("user_id = ?", userID).Order("is_default DESC, id DESC").Find(&suppliers).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询供应商失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"items": suppliers})
}

func (h *OfficeSupplyHandler) createOfficeSupplier(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var payload models.OfficeSupplier
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondOfficeError(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	payload.UserID = uintPointer(userID)
	if err := h.db.Create(&payload).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建供应商失败")
		return
	}
	if payload.IsDefault == 1 {
		h.db.Model(&models.OfficeSupplier{}).
			Where("id != ? AND user_id = ?", payload.ID, userID).
			Update("is_default", 0)
	}
	respondOfficeOK(w, map[string]interface{}{"id": payload.ID})
}

func (h *OfficeSupplyHandler) updateOfficeSupplier(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的供应商ID")
		return
	}
	var payload models.OfficeSupplier
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	result := h.db.Model(&models.OfficeSupplier{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"name": payload.Name, "contact": payload.Contact,
			"phone": payload.Phone, "bank_name": payload.BankName,
			"bank_account": payload.BankAccount, "is_default": payload.IsDefault,
			"remark": payload.Remark, "updated_at": time.Now(),
		})
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "未找到")
		return
	}
	if payload.IsDefault == 1 {
		h.db.Model(&models.OfficeSupplier{}).
			Where("id != ? AND user_id = ?", id, userID).Update("is_default", 0)
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) deleteOfficeSupplier(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的供应商ID")
		return
	}
	var count int64
	h.db.Model(&models.OfficeSupply{}).
		Where("supplier_id = ? AND user_id = ?", id, userID).Count(&count)
	if count > 0 {
		respondOfficeError(w, http.StatusBadRequest,
			fmt.Sprintf("该供应商被 %d 个用品引用，无法删除", count))
		return
	}
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.OfficeSupplier{})
	respondOfficeOK(w, map[string]interface{}{})
}

// ======== 用品 ========

type supplyListQuery struct {
	Keyword    string
	CategoryID string
	Status     string
	Page       int
	Limit      int
}

func parseSupplyListQuery(r *http.Request) supplyListQuery {
	return supplyListQuery{
		Keyword:    r.URL.Query().Get("keyword"),
		CategoryID: r.URL.Query().Get("category_id"),
		Status:     r.URL.Query().Get("status"),
		Page:       parseQueryInt(r, "page", 1),
		Limit:      parseQueryInt(r, "limit", 20),
	}
}

func (h *OfficeSupplyHandler) listSupplies(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	q := parseSupplyListQuery(r)
	query := h.db.Table("office_supplies s").
		Select("s.*, oc.name as category_name, osp.name as supplier_name").
		Joins("LEFT JOIN office_categories oc ON s.category_id = oc.id").
		Joins("LEFT JOIN office_suppliers osp ON s.supplier_id = osp.id").
		Where("s.user_id = ?", userID)
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("(s.name LIKE ? OR s.spec LIKE ?)", kw, kw)
	}
	if q.CategoryID != "" && q.CategoryID != "all" && q.CategoryID != "0" {
		query = query.Where("s.category_id = ?", q.CategoryID)
	}
	if q.Status != "" && q.Status != "all" {
		query = query.Where("s.status = ?", q.Status)
	}

	var total int64
	query.Count(&total)

	offset := (q.Page - 1) * q.Limit
	type supplyRow struct {
		models.OfficeSupply
		CategoryName string `json:"category_name"`
		SupplierName string `json:"supplier_name"`
	}
	var items []supplyRow
	query.Order("s.created_at DESC").Limit(q.Limit).Offset(offset).Scan(&items)

	respondOfficeOK(w, map[string]interface{}{
		"items": items, "total": total, "page": q.Page, "limit": q.Limit,
	})
}

func (h *OfficeSupplyHandler) getSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的用品ID")
		return
	}
	type supplyRow struct {
		models.OfficeSupply
		CategoryName string `json:"category_name"`
		SupplierName string `json:"supplier_name"`
	}
	var s supplyRow
	result := h.db.Table("office_supplies s").
		Select("s.*, oc.name as category_name, osp.name as supplier_name").
		Joins("LEFT JOIN office_categories oc ON s.category_id = oc.id").
		Joins("LEFT JOIN office_suppliers osp ON s.supplier_id = osp.id").
		Where("s.id = ? AND s.user_id = ?", id, userID).Scan(&s)
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	respondOfficeOK(w, map[string]interface{}{
		"id": s.ID, "name": s.Name, "spec": s.Spec, "unit": s.Unit,
		"reference_price": s.ReferencePrice, "safety_stock": s.SafetyStock,
		"category_id": s.CategoryID, "category_name": s.CategoryName,
		"supplier_id": s.SupplierID, "supplier_name": s.SupplierName,
		"status": s.Status, "remark": s.Remark,
		"created_at": s.CreatedAt, "updated_at": s.UpdatedAt,
	})
}

func (h *OfficeSupplyHandler) createSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var payload models.OfficeSupply
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondOfficeError(w, http.StatusBadRequest, "品名不能为空")
		return
	}
	payload.UserID = uintPointer(userID)
	if payload.Unit == "" {
		payload.Unit = "个"
	}
	if payload.Status == "" {
		payload.Status = "active"
	}
	if err := h.db.Create(&payload).Error; err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建用品失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"id": payload.ID})
}

func (h *OfficeSupplyHandler) updateSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的用品ID")
		return
	}
	var payload models.OfficeSupply
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	updates := map[string]interface{}{
		"name": payload.Name, "spec": payload.Spec, "unit": payload.Unit,
		"reference_price": payload.ReferencePrice, "safety_stock": payload.SafetyStock,
		"category_id": payload.CategoryID, "supplier_id": payload.SupplierID,
		"status": payload.Status, "remark": payload.Remark,
		"updated_at": time.Now(),
	}
	result := h.db.Model(&models.OfficeSupply{}).
		Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "未找到")
		return
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) deleteSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的用品ID")
		return
	}
	var count int64
	h.db.Model(&models.OfficePurchaseItem{}).
		Where("supply_id = ? AND user_id = ?", id, userID).Count(&count)
	if count > 0 {
		respondOfficeError(w, http.StatusBadRequest,
			fmt.Sprintf("该用品已被 %d 条采购记录引用，请先停用", count))
		return
	}
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.OfficeSupply{})
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) listSupplyUnits(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var units []string
	h.db.Model(&models.OfficeSupply{}).
		Where("user_id = ? AND unit IS NOT NULL AND unit != ''", userID).
		Distinct("unit").Order("unit").Pluck("unit", &units)
	defaults := []string{"个", "包", "箱", "瓶", "支", "双", "卷", "盒", "条", "袋"}
	unitSet := make(map[string]bool, len(units))
	for _, u := range units {
		unitSet[u] = true
	}
	for _, d := range defaults {
		if !unitSet[d] {
			units = append(units, d)
		}
	}
	respondOfficeOK(w, map[string]interface{}{"units": units})
}

func (h *OfficeSupplyHandler) exportSuppliesCSV(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	q := parseSupplyListQuery(r)
	query := h.db.Where("user_id = ?", userID)
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("name LIKE ? OR spec LIKE ?", kw, kw)
	}
	if q.CategoryID != "" && q.CategoryID != "all" && q.CategoryID != "0" {
		query = query.Where("category_id = ?", q.CategoryID)
	}
	var supplies []models.OfficeSupply
	query.Find(&supplies)
	catMap, supMap := h.buildNameMaps(userID)
	csv := service.ExportSuppliesCSV(supplies, catMap, supMap)
	w.Header().Set("Content-Type", "text/csv;charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=supplies.csv")
	w.Write([]byte(csv))
}

func (h *OfficeSupplyHandler) buildNameMaps(userID uint) (catMap map[uint]string, supMap map[uint]string) {
	catMap = make(map[uint]string)
	supMap = make(map[uint]string)
	var cats []models.OfficeCategory
	var sups []models.OfficeSupplier
	h.db.Where("user_id = ?", userID).Find(&cats)
	h.db.Where("user_id = ?", userID).Find(&sups)
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}
	for _, s := range sups {
		supMap[s.ID] = s.Name
	}
	return
}

func (h *OfficeSupplyHandler) importSuppliesCSV(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "读取数据失败")
		return
	}
	content := service.DecodeCSVContent(body)
	items := service.ParseCSVSupplies(content)
	if len(items) == 0 {
		respondOfficeError(w, http.StatusBadRequest, "数据不足（至少含表头和一行数据）")
		return
	}
	ok, errCount := 0, 0
	for _, item := range items {
		var catID *uint
		if item.CategoryName != "" {
			var cat models.OfficeCategory
			if err := h.db.Where("name = ? AND user_id = ?", item.CategoryName, userID).First(&cat).Error; err == nil {
				catID = &cat.ID
			}
		}
		supply := models.OfficeSupply{
			UserID: uintPointer(userID), Name: item.Name, Spec: item.Spec,
			Unit: item.Unit, ReferencePrice: item.ReferencePrice,
			SafetyStock: item.SafetyStock, CategoryID: catID,
			Status: "active", Remark: item.Remark,
		}
		if supply.Unit == "" {
			supply.Unit = "个"
		}
		if err := h.db.Create(&supply).Error; err != nil {
			errCount++
		} else {
			ok++
		}
	}
	respondOfficeOK(w, map[string]interface{}{"ok_count": ok, "err_count": errCount})
}
