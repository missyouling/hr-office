package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service"
)

// ======== 采购单 ========

// purchaseListItem 采购单列表项（含子查询 summary 字段）
type purchaseListItem struct {
	ID            uint    `json:"id"`
	OrderNo       string  `json:"order_no"`
	PurchaseDate  string  `json:"purchase_date"`
	TotalAmount   float64 `json:"total_amount"`
	Status        string  `json:"status"`
	Remark        string  `json:"remark"`
	SupplierID    *uint   `json:"supplier_id"`
	SupplierName  string  `json:"supplier_name"`
	PaymentStatus string  `json:"payment_status"`
	PaymentDate   *string `json:"payment_date"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	ItemCount     int64   `json:"item_count"`
	ItemNames     string  `json:"item_names"`
}

func (h *OfficeSupplyHandler) listPurchases(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	keyword := r.URL.Query().Get("keyword")

	// 计数
	var total int64
	countDB := h.db.Table("office_purchases").Where("user_id = ?", userID)
	countDB = applyStringFilter(countDB, "purchase_date >=", dateFrom)
	countDB = applyStringFilter(countDB, "purchase_date <=", dateTo)
	countDB = applyLikeFilter(countDB, "order_no", keyword)
	countDB.Count(&total)

	// 列表查询
	offset := (page - 1) * limit
	var items []purchaseListItem
	selectSQL := `p.id, p.order_no, p.purchase_date, p.total_amount, p.status, p.remark,
		p.supplier_id, osp.name as supplier_name, p.payment_status, p.payment_date,
		p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM office_purchase_items WHERE purchase_id=p.id) as item_count,
		COALESCE((SELECT GROUP_CONCAT(DISTINCT osi.name, '、') FROM office_purchase_items pi2
			JOIN office_supplies osi ON pi2.supply_id=osi.id WHERE pi2.purchase_id=p.id), '') as item_names`

	listDB := h.db.Table("office_purchases p").
		Select(selectSQL).
		Joins("LEFT JOIN office_suppliers osp ON p.supplier_id = osp.id").
		Where("p.user_id = ?", userID)
	listDB = applyStringFilter(listDB, "p.purchase_date >=", dateFrom)
	listDB = applyStringFilter(listDB, "p.purchase_date <=", dateTo)
	listDB = applyLikeFilter(listDB, "p.order_no", keyword)
	listDB.Order("p.created_at DESC").Limit(limit).Offset(offset).Scan(&items)

	// 汇总金额
	var totalSum float64
	sumDB := h.db.Table("office_purchases").Where("user_id = ?", userID)
	sumDB = applyStringFilter(sumDB, "purchase_date >=", dateFrom)
	sumDB = applyStringFilter(sumDB, "purchase_date <=", dateTo)
	sumDB = applyLikeFilter(sumDB, "order_no", keyword)
	sumDB.Select("COALESCE(SUM(total_amount), 0)").Scan(&totalSum)

	respondOfficeOK(w, map[string]interface{}{
		"items": items, "total": total, "page": page, "limit": limit,
		"total_sum": mathRound(totalSum, 2),
	})
}

func applyStringFilter(db *gorm.DB, clause string, value string) *gorm.DB {
	if value != "" {
		return db.Where(clause+" ?", value)
	}
	return db
}

func applyLikeFilter(db *gorm.DB, column string, value string) *gorm.DB {
	if value != "" {
		return db.Where(column+" LIKE ?", "%"+value+"%")
	}
	return db
}

func (h *OfficeSupplyHandler) getPurchaseDetail(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	type detailRow struct {
		models.OfficePurchase
		SupplierName string `json:"supplier_name"`
	}
	var p detailRow
	result := h.db.Table("office_purchases p").
		Select("p.*, osp.name as supplier_name").
		Joins("LEFT JOIN office_suppliers osp ON p.supplier_id = osp.id").
		Where("p.id = ? AND p.user_id = ?", id, userID).Scan(&p)
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}

	type itemRow struct {
		models.OfficePurchaseItem
		SupplyName     string  `json:"supply_name"`
		SupplySpec     string  `json:"supply_spec"`
		Unit           string  `json:"unit"`
		ReferencePrice float64 `json:"reference_price"`
	}
	var items []itemRow
	h.db.Table("office_purchase_items pi").
		Select("pi.*, os.name as supply_name, os.spec as supply_spec, os.unit, os.reference_price").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Where("pi.purchase_id = ?", id).Order("pi.id").Scan(&items)

	respondOfficeOK(w, map[string]interface{}{"id": p.ID, "order_no": p.OrderNo,
		"purchase_date": p.PurchaseDate, "total_amount": p.TotalAmount,
		"status": p.Status, "remark": p.Remark, "supplier_id": p.SupplierID,
		"supplier_name": p.SupplierName, "payment_status": p.PaymentStatus,
		"payment_date": p.PaymentDate, "created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt, "items": items,
	})
}

// purchaseCreatePayload 创建采购单的请求体
type purchaseCreatePayload struct {
	PurchaseDate string                `json:"purchase_date"`
	Items        []purchaseItemPayload `json:"items"`
	Status       string                `json:"status"`
	Remark       string                `json:"remark"`
	SupplierID   *uint                 `json:"supplier_id"`
}

type purchaseItemPayload struct {
	SupplyID  uint    `json:"supply_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Date      string  `json:"date"`
}

func (h *OfficeSupplyHandler) createPurchase(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var payload purchaseCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	if len(payload.Items) == 0 {
		respondOfficeError(w, http.StatusBadRequest, "明细不能为空")
		return
	}
	if payload.PurchaseDate == "" {
		payload.PurchaseDate = time.Now().Format("2006-01-02")
	}
	if payload.Status == "" {
		payload.Status = "confirmed"
	}

	// 在事务中创建（含单号生成）
	var orderNo string
	var purchaseID uint
	err = h.db.Transaction(func(tx *gorm.DB) error {
		on, genErr := service.GenerateOrderNo(tx)
		if genErr != nil {
			return genErr
		}
		orderNo = on

		// 计算总额
		var total float64
		for _, item := range payload.Items {
			total += float64(item.Quantity) * item.UnitPrice
		}
		total = mathRound(total, 2)

		// 查找供应商名称
		supplierName := ""
		if payload.SupplierID != nil {
			var sup models.OfficeSupplier
			if tx.Where("id = ?", *payload.SupplierID).First(&sup).Error == nil {
				supplierName = sup.Name
			}
		}

		p := models.OfficePurchase{
			UserID:        uintPointer(userID),
			OrderNo:       orderNo,
			PurchaseDate:  parseDate(payload.PurchaseDate),
			TotalAmount:   total,
			Status:        payload.Status,
			Remark:        payload.Remark,
			SupplierID:    payload.SupplierID,
			SupplierName:  supplierName,
			PaymentStatus: "未付款",
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		purchaseID = p.ID

		for _, item := range payload.Items {
			subtotal := mathRound(float64(item.Quantity)*item.UnitPrice, 2)
			itemDate := item.Date
			if itemDate == "" {
				itemDate = payload.PurchaseDate
			}
			pi := models.OfficePurchaseItem{
				UserID:     uintPointer(userID),
				PurchaseID: purchaseID,
				SupplyID:   item.SupplyID,
				Quantity:   item.Quantity,
				UnitPrice:  item.UnitPrice,
				Subtotal:   subtotal,
				Date:       parseDatePtr(itemDate),
			}
			if err := tx.Create(&pi).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建采购单失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"id": purchaseID, "order_no": orderNo})
}

func (h *OfficeSupplyHandler) updatePurchase(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	var payload purchaseCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}

	var existing models.OfficePurchase
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}

	// 计算总额
	var total float64
	for _, item := range payload.Items {
		total += float64(item.Quantity) * item.UnitPrice
	}
	total = mathRound(total, 2)

	supplierName := ""
	if payload.SupplierID != nil {
		var sup models.OfficeSupplier
		if h.db.Where("id = ?", *payload.SupplierID).First(&sup).Error == nil {
			supplierName = sup.Name
		}
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"purchase_date": parseDate(payload.PurchaseDate),
			"total_amount":  total,
			"status":        payload.Status,
			"remark":        payload.Remark,
			"supplier_id":   payload.SupplierID,
			"supplier_name": supplierName,
			"updated_at":    time.Now(),
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		// 删除旧明细
		if err := tx.Where("purchase_id = ?", id).Delete(&models.OfficePurchaseItem{}).Error; err != nil {
			return err
		}
		// 插入新明细
		for _, item := range payload.Items {
			subtotal := mathRound(float64(item.Quantity)*item.UnitPrice, 2)
			itemDate := item.Date
			if itemDate == "" {
				itemDate = payload.PurchaseDate
			}
			pi := models.OfficePurchaseItem{
				UserID:     uintPointer(userID),
				PurchaseID: id,
				SupplyID:   item.SupplyID,
				Quantity:   item.Quantity,
				UnitPrice:  item.UnitPrice,
				Subtotal:   subtotal,
				Date:       parseDatePtr(itemDate),
			}
			if err := tx.Create(&pi).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "更新采购单失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"id": id, "order_no": existing.OrderNo})
}

func (h *OfficeSupplyHandler) deletePurchase(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	h.db.Where("purchase_id = ?", id).Delete(&models.OfficePurchaseItem{})
	h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.OfficePurchase{})
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) copyPurchase(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	// 获取原采购单明细
	var items []models.OfficePurchaseItem
	h.db.Where("purchase_id = ? AND user_id = ?", id, userID).Find(&items)
	if len(items) == 0 {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	// 构造新采购单
	piPayloads := make([]purchaseItemPayload, len(items))
	for i, item := range items {
		piPayloads[i] = purchaseItemPayload{
			SupplyID: item.SupplyID, Quantity: item.Quantity, UnitPrice: item.UnitPrice,
		}
	}
	payload := purchaseCreatePayload{
		PurchaseDate: time.Now().Format("2006-01-02"),
		Items:        piPayloads,
		Status:       "draft",
	}
	// 复用创建逻辑（需要事务生成单号）
	var orderNo string
	var newID uint
	err = h.db.Transaction(func(tx *gorm.DB) error {
		on, genErr := service.GenerateOrderNo(tx)
		if genErr != nil {
			return genErr
		}
		orderNo = on
		var total float64
		for _, item := range payload.Items {
			total += float64(item.Quantity) * item.UnitPrice
		}
		total = mathRound(total, 2)
		p := models.OfficePurchase{
			UserID: uintPointer(userID), OrderNo: orderNo,
			PurchaseDate: parseDate(payload.PurchaseDate),
			TotalAmount:  total, Status: "draft", PaymentStatus: "未付款",
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		newID = p.ID
		for _, item := range payload.Items {
			subtotal := mathRound(float64(item.Quantity)*item.UnitPrice, 2)
			pi := models.OfficePurchaseItem{
				UserID: uintPointer(userID), PurchaseID: newID,
				SupplyID: item.SupplyID, Quantity: item.Quantity,
				UnitPrice: item.UnitPrice, Subtotal: subtotal,
			}
			if err := tx.Create(&pi).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "复制采购单失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"id": newID, "order_no": orderNo})
}

func (h *OfficeSupplyHandler) listUnpaidPurchases(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	type unpaidRow struct {
		ID           uint    `json:"id"`
		OrderNo      string  `json:"order_no"`
		PurchaseDate string  `json:"purchase_date"`
		TotalAmount  float64 `json:"total_amount"`
		SupplierName string  `json:"supplier_name"`
	}
	var items []unpaidRow
	h.db.Table("office_purchases p").
		Select("p.id, p.order_no, p.purchase_date, p.total_amount, osp.name as supplier_name").
		Joins("LEFT JOIN office_suppliers osp ON p.supplier_id = osp.id").
		Where("p.user_id = ? AND (p.payment_status != ? OR p.payment_status IS NULL)", userID, "已付款").
		Order("p.created_at DESC").Scan(&items)
	respondOfficeOK(w, map[string]interface{}{"items": items})
}

func (h *OfficeSupplyHandler) getPurchasePDF(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	// 获取采购单详情
	type detailRow struct {
		models.OfficePurchase
		SupplierName string `json:"supplier_name"`
	}
	var p detailRow
	result := h.db.Table("office_purchases p").
		Select("p.*, osp.name as supplier_name").
		Joins("LEFT JOIN office_suppliers osp ON p.supplier_id = osp.id").
		Where("p.id = ? AND p.user_id = ?", id, userID).Scan(&p)
	if result.RowsAffected == 0 {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	type itemRow struct {
		SupplyName string  `json:"supply_name"`
		SupplySpec string  `json:"supply_spec"`
		Unit       string  `json:"unit"`
		UnitPrice  float64 `json:"unit_price"`
		Quantity   int     `json:"quantity"`
		Subtotal   float64 `json:"subtotal"`
	}
	var items []itemRow
	h.db.Table("office_purchase_items pi").
		Select("os.name as supply_name, os.spec as supply_spec, os.unit, pi.unit_price, pi.quantity, pi.subtotal").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Where("pi.purchase_id = ?", id).Order("pi.id").Scan(&items)

	// 构建打印友好 HTML（与源格式一致）
	itemHTML := ""
	for i, item := range items {
		even := ""
		if i%2 == 0 {
			even = " class=\"even\""
		}
		itemHTML += fmt.Sprintf(`<tr%s><td>%d</td><td>%s</td><td>%s</td><td class="num">%s</td><td class="num">¥%.2f</td><td class="num">%d</td><td class="num">¥%.2f</td></tr>`,
			even, i+1, item.SupplyName, item.SupplySpec, item.Unit, item.UnitPrice, item.Quantity, item.Subtotal)
	}

	html := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>%s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:"Microsoft YaHei","PingFang SC","Noto Sans SC",sans-serif;padding:40px 50px;color:#333;font-size:14px}
h1{font-size:24px;margin-bottom:6px}
.meta{color:#666;font-size:13px;margin-bottom:20px;display:flex;justify-content:space-between}
table{width:100%%;border-collapse:collapse;margin-bottom:24px}
th{background:#1e40af;color:#fff;padding:8px 6px;text-align:left;font-size:13px}
td{padding:7px 6px;border-bottom:1px solid #e5e7eb;font-size:13px}
tr.even td{background:#f8fafc}
.num{text-align:right;font-family:"Courier New",monospace}
.total{font-size:18px;font-weight:bold;color:#dc2626;text-align:right;margin-bottom:30px}
@media print{body{padding:20px 30px}th{background:#1e40af!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}
</style></head><body>
<h1>采购单</h1>
<div class="meta"><span><strong>单号：</strong>%s</span><span><strong>日期：</strong>%s</span></div>
<table><thead><tr><th style="width:40px">序号</th><th>品名</th><th>规格</th><th style="width:50px">单位</th><th style="width:80px">单价</th><th style="width:60px">数量</th><th style="width:90px">小计</th></tr></thead><tbody>%s</tbody></table>
<div class="total">合计：¥%.2f</div>
</body></html>`, p.OrderNo, p.OrderNo, p.PurchaseDate.Format("2006-01-02"), itemHTML, p.TotalAmount)

	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.Write([]byte(html))
}

func (h *OfficeSupplyHandler) getPurchaseExcel(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的采购单ID")
		return
	}
	var p models.OfficePurchase
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&p).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	var items []models.OfficePurchaseItem
	h.db.Where("purchase_id = ?", id).Find(&items)

	data, filename, err := service.ExportPurchaseExcel(&p, items)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "生成Excel失败")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(data)
}

func (h *OfficeSupplyHandler) exportPurchasesExcel(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	keyword := r.URL.Query().Get("keyword")

	db := h.db.Table("office_purchases p").
		Select("p.id, p.order_no, p.purchase_date, p.total_amount, p.status, p.remark, osp.name as supplier_name, p.payment_status, p.created_at, (SELECT COUNT(*) FROM office_purchase_items WHERE purchase_id=p.id) as item_count").
		Joins("LEFT JOIN office_suppliers osp ON p.supplier_id = osp.id").
		Where("p.user_id = ?", userID)
	db = applyStringFilter(db, "p.purchase_date >=", dateFrom)
	db = applyStringFilter(db, "p.purchase_date <=", dateTo)
	db = applyLikeFilter(db, "p.order_no", keyword)
	db = db.Order("p.created_at DESC")

	var items []service.PurchaseExportItem
	db.Scan(&items)

	data, filename, err := service.ExportPurchasesExcel(items, dateFrom, dateTo, keyword)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "生成Excel失败")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(data)
}

// 解析日期辅助函数
func parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}

func parseDatePtr(dateStr string) *time.Time {
	t := parseDate(dateStr)
	return &t
}
