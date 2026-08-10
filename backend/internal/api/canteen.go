package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

// CanteenHandler 食堂模块 HTTP 处理器
type CanteenHandler struct {
	db         *gorm.DB
	analytics  *service.CanteenAnalyticsService
	csvService *service.CanteenCSVService
}

// NewCanteenHandler 构造函数
func NewCanteenHandler(db *gorm.DB) *CanteenHandler {
	return &CanteenHandler{
		db:         db,
		analytics:  service.NewCanteenAnalyticsService(db),
		csvService: service.NewCanteenCSVService(db),
	}
}

// RegisterCanteenRoutes 注册食堂模块路由（/api/canteen）
func RegisterCanteenRoutes(r chi.Router, db *gorm.DB) {
	h := NewCanteenHandler(db)
	r.Route("/api/canteen", func(cr chi.Router) {
		// 食材分类
		cr.Get("/categories", h.listCategories)
		cr.Post("/categories", h.createCategory)
		cr.Put("/categories/{id}", h.updateCategory)
		cr.Delete("/categories/{id}", h.deleteCategory)

		// 食材字典
		cr.Get("/supplies", h.listSupplies)
		cr.Get("/supplies/all", h.listSuppliesAll)
		cr.Post("/supplies", h.createSupply)
		cr.Put("/supplies/{id}", h.updateSupply)
		cr.Delete("/supplies/{id}", h.deleteSupply)

		// 费用科目
		cr.Get("/expense-categories", h.listExpenseCategories)
		cr.Post("/expense-categories", h.createExpenseCategory)
		cr.Put("/expense-categories/{id}", h.updateExpenseCategory)
		cr.Delete("/expense-categories/{id}", h.deleteExpenseCategory)

		// 采购单
		cr.Get("/purchases", h.listPurchases)
		cr.Post("/purchases", h.createPurchase)
		cr.Put("/purchases/{id}", h.updatePurchase)
		cr.Delete("/purchases/{id}", h.deletePurchase)
		cr.Get("/purchases/export/csv", h.exportPurchasesCSV)

		// 其他费用
		cr.Get("/expenses", h.listOtherExpenses)
		cr.Post("/expenses", h.createOtherExpense)
		cr.Post("/expenses/upsert", h.upsertOtherExpenses)
		cr.Put("/expenses/{id}", h.updateOtherExpense)
		cr.Delete("/expenses/{id}", h.deleteOtherExpense)

		// 每日收入
		cr.Get("/income", h.listDailyIncome)
		cr.Post("/income", h.saveDailyIncome)
		cr.Put("/income/{id}", h.updateDailyIncome)
		cr.Delete("/income/{id}", h.deleteDailyIncome)

		// 资源占用费
		cr.Get("/resource-fees", h.listResourceFees)
		cr.Post("/resource-fees", h.createResourceFee)
		cr.Put("/resource-fees/{id}", h.updateResourceFee)
		cr.Delete("/resource-fees/{id}", h.deleteResourceFee)
		cr.Get("/resource-fees/summary/{month}", h.summaryResourceFees)

		// 每周菜单
		cr.Get("/menus", h.getWeeklyMenu)
		cr.Post("/menus", h.saveWeeklyMenu)
		cr.Post("/menus/copy", h.copyWeeklyMenu)

		// 菜单模板
		cr.Get("/menu-templates", h.listMenuTemplates)
		cr.Post("/menu-templates", h.createMenuTemplate)
		cr.Delete("/menu-templates/{id}", h.deleteMenuTemplate)

		// 饭卡充值
		cr.Get("/recharges", h.listRecharges)
		cr.Get("/recharges/summary", h.summaryRecharges)
		cr.Delete("/recharges/{id}", h.deleteRecharge)
		cr.Post("/recharges/import", h.importRecharges)

		// 饭卡退费
		cr.Get("/refunds", h.listRefunds)
		cr.Get("/refunds/summary", h.summaryRefunds)
		cr.Delete("/refunds/{id}", h.deleteRefund)
		cr.Post("/refunds/import", h.importRefunds)

		// 数据分析
		cr.Route("/analytics", func(ar chi.Router) {
			ar.Get("/summary", h.analyticsSummary)
			ar.Get("/daily-trend", h.analyticsDailyTrend)
			ar.Get("/expense-breakdown", h.analyticsExpenseBreakdown)
			ar.Get("/food-share", h.analyticsFoodShare)
			ar.Get("/top-supplies", h.analyticsTopSupplies)
			ar.Get("/monthly-compare", h.analyticsMonthlyCompare)
			ar.Get("/suggestions", h.analyticsSuggestions)
			ar.Get("/cost-summary", h.analyticsCostSummary)
			ar.Get("/cost-summary-range", h.analyticsCostSummaryRange)
		})
	})
}

// ========== 食材分类 CRUD ==========

func (h *CanteenHandler) listCategories(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	var items []models.CanteenCategory
	if err := h.db.Where("user_id = ?", userID).Order("sort_order, id").Find(&items).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "加载分类失败", err)
		return
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *CanteenHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	var payload struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondError(w, http.StatusBadRequest, "名称不能为空", nil)
		return
	}
	cat := models.CanteenCategory{UserID: uintPointer(userID), Name: payload.Name, SortOrder: payload.SortOrder}
	if err := h.db.Create(&cat).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建分类失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, cat)
}

func (h *CanteenHandler) updateCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效ID", err)
		return
	}
	var cat models.CanteenCategory
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		respondError(w, http.StatusNotFound, "分类未找到", err)
		return
	}
	var payload struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	h.db.Model(&cat).Updates(map[string]any{"name": payload.Name, "sort_order": payload.SortOrder})
	respondJSON(w, http.StatusOK, cat)
}

func (h *CanteenHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效ID", err)
		return
	}
	var count int64
	h.db.Model(&models.CanteenSupply{}).Where("category_id = ? AND user_id = ?", id, userID).Count(&count)
	if count > 0 {
		respondError(w, http.StatusConflict, "该分类被食材引用中，无法删除", nil)
		return
	}
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenCategory{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 食材字典 CRUD ==========

func (h *CanteenHandler) listSupplies(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	query := h.db.Where("s.user_id = ?", userID).Table("canteen_supplies s").Select("s.*, c.name as category_name").Joins("LEFT JOIN canteen_categories c ON s.category_id = c.id")

	if kw := r.URL.Query().Get("keyword"); kw != "" {
		query = query.Where("(s.name LIKE ? OR s.spec LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}
	if cat := r.URL.Query().Get("category_id"); cat != "" {
		query = query.Where("s.category_id = ?", cat)
	}
	if st := r.URL.Query().Get("status"); st != "" {
		query = query.Where("s.status = ?", st)
	}
	var total int64
	query.Count(&total)

	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 50)
	var items []models.CanteenSupply
	query.Order("s.category_id, s.id").Offset((page - 1) * limit).Limit(limit).Find(&items)

	// 手动补充 CategoryName（GORM Join 可能不映射到 struct 外部字段）
	type supplyWithCat struct {
		models.CanteenSupply
		CategoryName string `json:"category_name"`
	}
	result := make([]supplyWithCat, len(items))
	for i := range items {
		result[i].CanteenSupply = items[i]
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": result, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) listSuppliesAll(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	var items []models.CanteenSupply
	h.db.Where("user_id = ? AND status = ?", userID, "active").Order("category_id, id").Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CanteenHandler) createSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	var payload struct {
		Name           string  `json:"name"`
		Spec           string  `json:"spec"`
		Unit           string  `json:"unit"`
		ReferencePrice float64 `json:"reference_price"`
		CategoryID     *uint   `json:"category_id"`
		Status         string  `json:"status"`
		Remark         string  `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondError(w, http.StatusBadRequest, "品名不能为空", nil)
		return
	}
	unit := payload.Unit
	if unit == "" {
		unit = "斤"
	}
	status := payload.Status
	if status == "" {
		status = "active"
	}
	s := models.CanteenSupply{
		UserID: uintPointer(userID), Name: payload.Name, Spec: payload.Spec,
		Unit: unit, ReferencePrice: payload.ReferencePrice, CategoryID: payload.CategoryID,
		Status: status, Remark: payload.Remark,
	}
	if err := h.db.Create(&s).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "创建食材失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, s)
}

func (h *CanteenHandler) updateSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效ID", err)
		return
	}
	var s models.CanteenSupply
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&s).Error; err != nil {
		respondError(w, http.StatusNotFound, "食材未找到", err)
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	h.db.Model(&s).Updates(payload)
	respondJSON(w, http.StatusOK, s)
}

func (h *CanteenHandler) deleteSupply(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未授权", err)
		return
	}
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "无效ID", err)
		return
	}
	var count int64
	h.db.Model(&models.CanteenPurchaseItem{}).Where("supply_id = ? AND user_id = ?", id, userID).Count(&count)
	if count > 0 {
		respondError(w, http.StatusConflict, "该食材被采购记录引用中，无法删除", nil)
		return
	}
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenSupply{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 费用科目 CRUD ==========

func (h *CanteenHandler) listExpenseCategories(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var items []models.CanteenExpenseCategory
	h.db.Where("user_id = ?", userID).Order("sort_order, id").Find(&items)
	respondJSON(w, http.StatusOK, items)
}

func (h *CanteenHandler) createExpenseCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var p struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	json.NewDecoder(r.Body).Decode(&p)
	if strings.TrimSpace(p.Name) == "" {
		respondError(w, http.StatusBadRequest, "名称不能为空", nil)
		return
	}
	cat := models.CanteenExpenseCategory{UserID: uintPointer(userID), Name: p.Name, SortOrder: p.SortOrder}
	h.db.Create(&cat)
	respondJSON(w, http.StatusCreated, cat)
}

func (h *CanteenHandler) updateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	var cat models.CanteenExpenseCategory
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		respondError(w, http.StatusNotFound, "科目未找到", err)
		return
	}
	var p struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	json.NewDecoder(r.Body).Decode(&p)
	h.db.Model(&cat).Updates(map[string]any{"name": p.Name, "sort_order": p.SortOrder})
	respondJSON(w, http.StatusOK, cat)
}

func (h *CanteenHandler) deleteExpenseCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenExpenseCategory{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 采购单 CRUD ==========

func (h *CanteenHandler) listPurchases(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("p.user_id = ?", userID).Table("canteen_purchases p")

	if kw := q.Get("keyword"); kw != "" {
		query = query.Where("(p.order_no LIKE ? OR p.supplier_name LIKE ?)", "%"+kw+"%", "%"+kw+"%")
	}
	if df := q.Get("date_from"); df != "" {
		query = query.Where("p.purchase_date >= ?", df)
	}
	if dt := q.Get("date_to"); dt != "" {
		query = query.Where("p.purchase_date <= ?", dt)
	}
	var total int64
	query.Count(&total)

	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 20)

	// 子查询：item_count 和 category_count
	type purchaseRow struct {
		models.CanteenPurchase
		ItemCount     int `json:"item_count"`
		CategoryCount int `json:"category_count"`
	}
	var items []purchaseRow
	query.Select(`p.*, (SELECT COUNT(*) FROM canteen_purchase_items pi WHERE pi.purchase_id=p.id) as item_count,
		(SELECT COUNT(DISTINCT s.category_id) FROM canteen_purchase_items pi2
			LEFT JOIN canteen_supplies s ON pi2.supply_id=s.id WHERE pi2.purchase_id=p.id) as category_count`).
		Order("p.purchase_date DESC, p.id DESC").
		Offset((page - 1) * limit).Limit(limit).Scan(&items)

	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) createPurchase(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var payload struct {
		PurchaseDate string  `json:"purchase_date"`
		SupplierID   *uint   `json:"supplier_id"`
		SupplierName string  `json:"supplier_name"`
		Channel      string  `json:"channel"`
		ActualPay    float64 `json:"actual_pay"`
		Remark       string  `json:"remark"`
		Items        []struct {
			SupplyID  uint    `json:"supply_id"`
			Quantity  float64 `json:"quantity"`
			UnitPrice float64 `json:"unit_price"`
			Subtotal  float64 `json:"subtotal"`
			Remark    string  `json:"remark"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if len(payload.Items) == 0 {
		respondError(w, http.StatusBadRequest, "明细不能为空", nil)
		return
	}
	// 事务：生成单号并保存
	tx := h.db.Begin()
	now := time.Now()
	prefix := "CT-" + now.Format("20060102") + "-"
	var seq int64
	tx.Raw("SELECT COUNT(*) FROM canteen_purchases WHERE user_id=? AND order_no LIKE ?", userID, prefix+"%").Scan(&seq)
	orderNo := prefix + padZero(int(seq+1), 2)

	var total float64
	for _, it := range payload.Items {
		sub := it.Subtotal
		if sub == 0 {
			sub = it.Quantity * it.UnitPrice
		}
		total += sub
	}
	purchaseDate, _ := time.Parse("2006-01-02", payload.PurchaseDate)
	p := models.CanteenPurchase{
		UserID:       uintPointer(userID),
		OrderNo:      orderNo,
		PurchaseDate: purchaseDate,
		TotalAmount:  total,
		SupplierID:   payload.SupplierID,
		SupplierName: payload.SupplierName,
		Channel:      payload.Channel,
		ActualPay:    payload.ActualPay,
		Remark:       payload.Remark,
	}
	if err := tx.Create(&p).Error; err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "创建采购单失败", err)
		return
	}
	for _, it := range payload.Items {
		sub := it.Subtotal
		if sub == 0 {
			sub = it.Quantity * it.UnitPrice
		}
		item := models.CanteenPurchaseItem{
			UserID:     uintPointer(userID),
			PurchaseID: p.ID,
			SupplyID:   it.SupplyID,
			Quantity:   it.Quantity,
			UnitPrice:  it.UnitPrice,
			Subtotal:   sub,
			Remark:     it.Remark,
		}
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			respondError(w, http.StatusInternalServerError, "创建明细失败", err)
			return
		}
	}
	tx.Commit()
	respondJSON(w, http.StatusCreated, map[string]any{"id": p.ID, "order_no": orderNo, "total_amount": total})
}

func (h *CanteenHandler) updatePurchase(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	var existing models.CanteenPurchase
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
		respondError(w, http.StatusNotFound, "采购单未找到", err)
		return
	}
	var payload struct {
		PurchaseDate string  `json:"purchase_date"`
		SupplierID   *uint   `json:"supplier_id"`
		SupplierName string  `json:"supplier_name"`
		Channel      string  `json:"channel"`
		ActualPay    float64 `json:"actual_pay"`
		Remark       string  `json:"remark"`
		Items        []struct {
			SupplyID  uint    `json:"supply_id"`
			Quantity  float64 `json:"quantity"`
			UnitPrice float64 `json:"unit_price"`
			Subtotal  float64 `json:"subtotal"`
			Remark    string  `json:"remark"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	tx := h.db.Begin()
	var total float64
	for _, it := range payload.Items {
		sub := it.Subtotal
		if sub == 0 {
			sub = it.Quantity * it.UnitPrice
		}
		total += sub
	}
	purchaseDate, _ := time.Parse("2006-01-02", payload.PurchaseDate)
	updates := map[string]any{
		"purchase_date": purchaseDate, "total_amount": total,
		"supplier_id": payload.SupplierID, "supplier_name": payload.SupplierName,
		"channel": payload.Channel, "actual_pay": payload.ActualPay, "remark": payload.Remark,
	}
	tx.Model(&existing).Updates(updates)
	// 先删后插明细
	tx.Where("purchase_id = ? AND user_id = ?", existing.ID, userID).Delete(&models.CanteenPurchaseItem{})
	for _, it := range payload.Items {
		sub := it.Subtotal
		if sub == 0 {
			sub = it.Quantity * it.UnitPrice
		}
		item := models.CanteenPurchaseItem{
			UserID: uintPointer(userID), PurchaseID: existing.ID,
			SupplyID: it.SupplyID, Quantity: it.Quantity, UnitPrice: it.UnitPrice,
			Subtotal: sub, Remark: it.Remark,
		}
		tx.Create(&item)
	}
	tx.Commit()
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "total_amount": total})
}

func (h *CanteenHandler) deletePurchase(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	tx := h.db.Begin()
	tx.Where("purchase_id = ? AND user_id = ?", id, userID).Delete(&models.CanteenPurchaseItem{})
	tx.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenPurchase{})
	tx.Commit()
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (h *CanteenHandler) exportPurchasesCSV(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	csv, err := h.csvService.ExportPurchasesCSV(userID, q.Get("date_from"), q.Get("date_to"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "导出失败", err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"canteen-purchases.csv\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(csv))
}

// ========== 其他费用 CRUD ==========

func (h *CanteenHandler) listOtherExpenses(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("user_id = ?", userID).Model(&models.CanteenOtherExpense{})
	if month := q.Get("month"); month != "" {
		query = query.Where("substr(expense_date,1,7) = ?", month)
	}
	if year := q.Get("year"); year != "" {
		query = query.Where("substr(expense_date,1,4) = ?", year)
	}
	if cat := q.Get("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	var total int64
	query.Count(&total)
	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 100)
	var items []models.CanteenOtherExpense
	query.Order("expense_date DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) createOtherExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var e models.CanteenOtherExpense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if e.ExpenseDate.IsZero() || strings.TrimSpace(e.Category) == "" {
		respondError(w, http.StatusBadRequest, "日期和科目不能为空", nil)
		return
	}
	e.UserID = uintPointer(userID)
	h.db.Create(&e)
	respondJSON(w, http.StatusCreated, e)
}

func (h *CanteenHandler) upsertOtherExpenses(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var payload struct {
		Month string `json:"month"`
		Items []struct {
			Category     string  `json:"category"`
			Amount       float64 `json:"amount"`
			ActualAmount float64 `json:"actual_amount"`
			Params       string  `json:"params"`
			Remark       string  `json:"remark"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if payload.Month == "" || len(payload.Items) == 0 {
		respondError(w, http.StatusBadRequest, "参数错误", nil)
		return
	}
	date := payload.Month + "-01"
	var updated, inserted int
	for _, it := range payload.Items {
		it.Category = strings.TrimSpace(it.Category)
		if it.Category == "" {
			continue
		}
		var existing models.CanteenOtherExpense
		err := h.db.Where("user_id = ? AND substr(expense_date,1,7) = ? AND category = ?", userID, payload.Month, it.Category).First(&existing).Error
		if err == nil {
			h.db.Model(&existing).Updates(map[string]any{
				"amount": it.Amount, "actual_amount": it.ActualAmount, "params": it.Params, "remark": it.Remark,
			})
			updated++
		} else {
			expDate, _ := time.Parse("2006-01-02", date)
			rec := models.CanteenOtherExpense{
				UserID: uintPointer(userID), ExpenseDate: expDate, Category: it.Category,
				Amount: it.Amount, ActualAmount: it.ActualAmount, Params: it.Params, Remark: it.Remark,
			}
			h.db.Create(&rec)
			inserted++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": updated, "inserted": inserted})
}

func (h *CanteenHandler) updateOtherExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	var e models.CanteenOtherExpense
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&e).Error; err != nil {
		respondError(w, http.StatusNotFound, "费用未找到", err)
		return
	}
	var payload map[string]any
	json.NewDecoder(r.Body).Decode(&payload)
	delete(payload, "id")
	delete(payload, "user_id")
	h.db.Model(&e).Updates(payload)
	respondJSON(w, http.StatusOK, e)
}

func (h *CanteenHandler) deleteOtherExpense(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenOtherExpense{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 每日收入 ==========

func (h *CanteenHandler) listDailyIncome(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("user_id = ?", userID).Model(&models.CanteenDailyIncome{})
	if month := q.Get("month"); month != "" {
		query = query.Where("substr(income_date,1,7) = ?", month)
	}
	if df := q.Get("date_from"); df != "" {
		query = query.Where("income_date >= ?", df)
	}
	if dt := q.Get("date_to"); dt != "" {
		query = query.Where("income_date <= ?", dt)
	}
	var total int64
	query.Count(&total)
	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 100)
	var items []models.CanteenDailyIncome
	query.Order("income_date ASC").Offset((page - 1) * limit).Limit(limit).Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) saveDailyIncomeRaw(userID uint, data models.CanteenDailyIncome) (models.CanteenDailyIncome, bool, error) {
	// 自动计算：总人次=午餐+晚餐，总收入=早+中+晚
	data.TotalCount = data.LunchCount + data.DinnerCount
	data.TotalAmount = math.Round((data.BreakfastAmount+data.LunchAmount+data.DinnerAmount)*100) / 100

	var existing models.CanteenDailyIncome
	err := h.db.Where("user_id = ? AND income_date = ?", userID, data.IncomeDate).First(&existing).Error
	if err == nil {
		// 已存在，更新
		data.ID = existing.ID
		data.UserID = uintPointer(userID)
		h.db.Save(&data)
		return data, true, nil
	}
	// 新增
	data.UserID = uintPointer(userID)
	h.db.Create(&data)
	return data, false, nil
}

// saveDailyIncome POST /api/canteen/income - 创建/更新每日收入（同日期 upsert）
func (h *CanteenHandler) saveDailyIncome(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var d models.CanteenDailyIncome
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if d.IncomeDate.IsZero() {
		respondError(w, http.StatusBadRequest, "日期不能为空", nil)
		return
	}
	result, updated, err := h.saveDailyIncomeRaw(userID, d)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "保存收入失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "id": result.ID, "updated": updated})
}

// updateDailyIncome PUT /api/canteen/income/{id} - 复用 save 实现 upsert
func (h *CanteenHandler) updateDailyIncome(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var d models.CanteenDailyIncome
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if d.IncomeDate.IsZero() {
		respondError(w, http.StatusBadRequest, "日期不能为空", nil)
		return
	}
	result, updated, err := h.saveDailyIncomeRaw(userID, d)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "更新收入失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "id": result.ID, "updated": updated})
}

func (h *CanteenHandler) deleteDailyIncome(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenDailyIncome{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 资源占用费 ==========

func (h *CanteenHandler) listResourceFees(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("user_id = ?", userID).Model(&models.CanteenResourceFee{})
	if month := q.Get("month"); month != "" {
		query = query.Where("substr(fee_date,1,7) = ?", month)
	}
	if year := q.Get("year"); year != "" {
		query = query.Where("substr(fee_date,1,4) = ?", year)
	}
	if payer := q.Get("payer"); payer != "" {
		query = query.Where("payer = ?", payer)
	}
	var total int64
	query.Count(&total)
	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 200)
	var items []models.CanteenResourceFee
	query.Order("fee_date DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) createResourceFee(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var f models.CanteenResourceFee
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if f.FeeDate.IsZero() || strings.TrimSpace(f.Payer) == "" {
		respondError(w, http.StatusBadRequest, "日期和缴费人不能为空", nil)
		return
	}
	f.UserID = uintPointer(userID)
	if f.MealType == "" {
		f.MealType = "午餐"
	}
	h.db.Create(&f)
	respondJSON(w, http.StatusCreated, f)
}

func (h *CanteenHandler) updateResourceFee(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	var f models.CanteenResourceFee
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&f).Error; err != nil {
		respondError(w, http.StatusNotFound, "资源占用费未找到", err)
		return
	}
	var payload map[string]any
	json.NewDecoder(r.Body).Decode(&payload)
	delete(payload, "id")
	delete(payload, "user_id")
	h.db.Model(&f).Updates(payload)
	respondJSON(w, http.StatusOK, f)
}

func (h *CanteenHandler) deleteResourceFee(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenResourceFee{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// summaryResourceFees 资源占用费月度汇总（同人合并）
func (h *CanteenHandler) summaryResourceFees(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	month := chi.URLParam(r, "month")

	type SummaryRow struct {
		Payer       string  `json:"payer"`
		TotalAmount float64 `json:"total_amount"`
		Times       int     `json:"times"`
	}
	type DetailRow struct {
		models.CanteenResourceFee
	}

	var summary []SummaryRow
	h.db.Raw(`SELECT payer, COALESCE(SUM(amount),0) as total_amount, COUNT(*) as times
		FROM canteen_resource_fees WHERE user_id=? AND substr(fee_date,1,7)=?
		GROUP BY payer ORDER BY total_amount DESC`, userID, month).Scan(&summary)

	var detail []models.CanteenResourceFee
	h.db.Where("user_id = ? AND substr(fee_date,1,7) = ?", userID, month).Order("fee_date, id").Find(&detail)

	var total float64
	for _, r := range summary {
		total += r.TotalAmount
	}
	respondJSON(w, http.StatusOK, map[string]any{"summary": summary, "detail": detail, "total": total})
}

// ========== 每周菜单 ==========

func (h *CanteenHandler) getWeeklyMenu(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	week := r.URL.Query().Get("week")
	if week == "" {
		respondError(w, http.StatusBadRequest, "缺少 week 参数", nil)
		return
	}
	var rows []models.CanteenWeeklyMenu
	h.db.Where("user_id = ? AND week_start_date = ?", userID, week).Order("day_of_week").Find(&rows)

	// 组织成 7天×3餐矩阵
	type dayMenu struct {
		DayOfWeek int    `json:"day_of_week"`
		Breakfast string `json:"早餐"`
		Lunch     string `json:"午餐"`
		Dinner    string `json:"晚餐"`
		Remark    string `json:"remark"`
	}
	matrix := make([]dayMenu, 7)
	for d := 0; d < 7; d++ {
		matrix[d] = dayMenu{DayOfWeek: d + 1}
	}
	for _, r := range rows {
		idx := r.DayOfWeek - 1
		if idx < 0 || idx >= 7 {
			continue
		}
		switch r.MealType {
		case "早餐":
			matrix[idx].Breakfast = r.Dishes
		case "午餐":
			matrix[idx].Lunch = r.Dishes
		case "晚餐":
			matrix[idx].Dinner = r.Dishes
		}
		matrix[idx].Remark = r.Remark
	}
	respondJSON(w, http.StatusOK, map[string]any{"week_start_date": week, "days": matrix, "rows": rows})
}

func (h *CanteenHandler) saveWeeklyMenu(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var payload struct {
		WeekStartDate string `json:"week_start_date"`
		Days          []struct {
			DayOfWeek int    `json:"day_of_week"`
			Breakfast string `json:"早餐"`
			Lunch     string `json:"午餐"`
			Dinner    string `json:"晚餐"`
			Remark    string `json:"remark"`
		} `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if payload.WeekStartDate == "" || len(payload.Days) == 0 {
		respondError(w, http.StatusBadRequest, "缺少参数", nil)
		return
	}
	// 先删后插
	tx := h.db.Begin()
	tx.Where("user_id = ? AND week_start_date = ?", userID, payload.WeekStartDate).Delete(&models.CanteenWeeklyMenu{})
	for _, d := range payload.Days {
		for _, meal := range []string{"早餐", "午餐", "晚餐"} {
			var dishes string
			switch meal {
			case "早餐":
				dishes = strings.TrimSpace(d.Breakfast)
			case "午餐":
				dishes = strings.TrimSpace(d.Lunch)
			case "晚餐":
				dishes = strings.TrimSpace(d.Dinner)
			}
			if dishes == "" {
				continue
			}
			tx.Create(&models.CanteenWeeklyMenu{
				UserID: uintPointer(userID), WeekStartDate: payload.WeekStartDate,
				DayOfWeek: d.DayOfWeek, MealType: meal, Dishes: dishes, Remark: d.Remark,
			})
		}
	}
	tx.Commit()
	respondJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CanteenHandler) copyWeeklyMenu(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var payload struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if payload.From == "" || payload.To == "" {
		respondError(w, http.StatusBadRequest, "缺少参数", nil)
		return
	}
	var fromRows []models.CanteenWeeklyMenu
	h.db.Where("user_id = ? AND week_start_date = ?", userID, payload.From).Find(&fromRows)
	if len(fromRows) == 0 {
		respondError(w, http.StatusBadRequest, "源周无菜单可复制", nil)
		return
	}
	tx := h.db.Begin()
	tx.Where("user_id = ? AND week_start_date = ?", userID, payload.To).Delete(&models.CanteenWeeklyMenu{})
	for _, r := range fromRows {
		tx.Create(&models.CanteenWeeklyMenu{
			UserID: uintPointer(userID), WeekStartDate: payload.To,
			DayOfWeek: r.DayOfWeek, MealType: r.MealType, Dishes: r.Dishes, Remark: r.Remark,
		})
	}
	tx.Commit()
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "copied": len(fromRows)})
}

// ========== 菜单模板 ==========

func (h *CanteenHandler) listMenuTemplates(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var items []models.CanteenMenuTemplate
	h.db.Where("user_id = ?", userID).Order("id DESC").Find(&items)
	respondJSON(w, http.StatusOK, items)
}

func (h *CanteenHandler) createMenuTemplate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var payload struct {
		Name string         `json:"name"`
		Data datatypes.JSON `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "无效请求", err)
		return
	}
	if strings.TrimSpace(payload.Name) == "" || payload.Data == nil {
		respondError(w, http.StatusBadRequest, "名称和内容不能为空", nil)
		return
	}
	t := models.CanteenMenuTemplate{UserID: uintPointer(userID), Name: payload.Name, Data: payload.Data}
	h.db.Create(&t)
	respondJSON(w, http.StatusCreated, t)
}

func (h *CanteenHandler) deleteMenuTemplate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenMenuTemplate{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// ========== 饭卡充值 ==========

func (h *CanteenHandler) listRecharges(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("user_id = ?", userID).Model(&models.CanteenCardRecharge{})
	if month := q.Get("month"); month != "" {
		query = query.Where("substr(recharge_date,1,7) = ?", month)
	}
	if year := q.Get("year"); year != "" {
		query = query.Where("substr(recharge_date,1,4) = ?", year)
	}
	if kw := q.Get("keyword"); kw != "" {
		query = query.Where("(user_name LIKE ? OR card_user_id LIKE ? OR card_no LIKE ? OR external_sn LIKE ?)", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	query.Count(&total)
	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 20)
	var items []models.CanteenCardRecharge
	query.Order("recharge_date DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) deleteRecharge(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenCardRecharge{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (h *CanteenHandler) summaryRecharges(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	type Summary struct {
		Total  float64 `json:"total"`
		Count  int     `json:"count"`
		People int     `json:"people"`
	}
	var s Summary
	h.db.Raw(`SELECT COALESCE(SUM(amount),0) as total, COUNT(*) as count, COUNT(DISTINCT user_name) as people
		FROM canteen_card_recharges WHERE user_id=? AND substr(recharge_date,1,7)=?`, userID, month).Scan(&s)
	respondJSON(w, http.StatusOK, map[string]any{"month": month, "total": s.Total, "count": s.Count, "people": s.People})
}

func (h *CanteenHandler) importRecharges(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "文件解析失败", err)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "缺少 CSV 文件", err)
		return
	}
	defer file.Close()
	mode := r.FormValue("mode")
	if mode == "" {
		mode = "upsert"
	}
	mapping := r.FormValue("mapping")
	result, err := h.csvService.ImportRecharges(userID, file, mode, mapping)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// ========== 饭卡退费 ==========

func (h *CanteenHandler) listRefunds(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	query := h.db.Where("user_id = ?", userID).Model(&models.CanteenCardRefund{})
	if month := q.Get("month"); month != "" {
		query = query.Where("substr(refund_date,1,7) = ?", month)
	}
	if year := q.Get("year"); year != "" {
		query = query.Where("substr(refund_date,1,4) = ?", year)
	}
	if kw := q.Get("keyword"); kw != "" {
		query = query.Where("(user_name LIKE ? OR card_user_id LIKE ? OR card_no LIKE ? OR external_sn LIKE ?)", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	query.Count(&total)
	page, limit := parseIntParam(r, "page", 1), parseIntParam(r, "limit", 20)
	var items []models.CanteenCardRefund
	query.Order("refund_date DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items)
	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *CanteenHandler) deleteRefund(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	id, _ := parseUintParam(r, "id")
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CanteenCardRefund{})
	respondJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (h *CanteenHandler) summaryRefunds(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	type Summary struct {
		Total  float64
		Count  int
		People int
	}
	var s Summary
	h.db.Raw(`SELECT COALESCE(SUM(amount),0) as total, COUNT(*) as count, COUNT(DISTINCT user_name) as people
		FROM canteen_card_refunds WHERE user_id=? AND substr(refund_date,1,7)=?`, userID, month).Scan(&s)
	respondJSON(w, http.StatusOK, map[string]any{"month": month, "total": s.Total, "count": s.Count, "people": s.People})
}

func (h *CanteenHandler) importRefunds(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "文件解析失败", err)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "缺少 CSV 文件", err)
		return
	}
	defer file.Close()
	mode := r.FormValue("mode")
	if mode == "" {
		mode = "upsert"
	}
	mapping := r.FormValue("mapping")
	result, err := h.csvService.ImportRefunds(userID, file, mode, mapping)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// ========== 数据分析 ==========

func (h *CanteenHandler) analyticsSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	data, err := h.analytics.MonthlySummary(userID, r.URL.Query().Get("month"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (h *CanteenHandler) analyticsDailyTrend(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	items, err := h.analytics.DailyTrend(userID, r.URL.Query().Get("month"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CanteenHandler) analyticsExpenseBreakdown(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	data, err := h.analytics.ExpenseBreakdown(userID, r.URL.Query().Get("month"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (h *CanteenHandler) analyticsFoodShare(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	items, err := h.analytics.FoodCategoryShare(userID, r.URL.Query().Get("month"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CanteenHandler) analyticsTopSupplies(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	limit := parseIntParam(r, "limit", 10)
	items, err := h.analytics.TopSupplies(userID, r.URL.Query().Get("month"), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CanteenHandler) analyticsMonthlyCompare(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	items, err := h.analytics.MonthlyCompare(userID, q.Get("from"), q.Get("to"), q.Get("year"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CanteenHandler) analyticsSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	items, err := h.analytics.Suggestions(userID, r.URL.Query().Get("month"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

// analyticsCostSummary GET /api/canteen/analytics/cost-summary?month=YYYY-MM or ?from=...&to=...&year=...
func (h *CanteenHandler) analyticsCostSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	if month := q.Get("month"); month != "" {
		item, err := h.analytics.CostSummary(userID, month)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "分析失败", err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"item": item})
		return
	}
	items, err := h.analytics.CostSummaryRange(userID, q.Get("from"), q.Get("to"), q.Get("year"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

// analyticsCostSummaryRange GET /api/canteen/analytics/cost-summary-range?from=...&to=...&year=...
func (h *CanteenHandler) analyticsCostSummaryRange(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	q := r.URL.Query()
	items, err := h.analytics.CostSummaryRange(userID, q.Get("from"), q.Get("to"), q.Get("year"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "分析失败", err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ========== 工具函数 ==========

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return defaultVal
	}
	return n
}

func padZero(n, width int) string {
	s := fmt.Sprintf("%d", n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
