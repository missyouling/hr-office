package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service"
)

// ======== 请款单 ========

func (h *OfficeSupplyHandler) listPaymentRequests(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)
	keyword := r.URL.Query().Get("keyword")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	query := h.db.Where("user_id = ?", userID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("(request_no LIKE ? OR content LIKE ? OR payee LIKE ?)", kw, kw, kw)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if dateFrom != "" {
		query = query.Where("request_date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("request_date <= ?", dateTo)
	}

	var total int64
	query.Model(&models.OfficePaymentRequest{}).Count(&total)

	offset := (page - 1) * limit
	var items []models.OfficePaymentRequest
	query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items)

	respondOfficeOK(w, map[string]interface{}{
		"items": items, "total": total, "page": page, "limit": limit,
	})
}

func (h *OfficeSupplyHandler) getPaymentRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的请款单ID")
		return
	}
	var pr models.OfficePaymentRequest
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&pr).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	respondOfficeOK(w, map[string]interface{}{
		"id": pr.ID, "request_no": pr.RequestNo, "payment_unit": pr.PaymentUnit,
		"department": pr.Department, "applicant": pr.Applicant,
		"request_date": pr.RequestDate, "content": pr.Content,
		"payee": pr.Payee, "payee_supplier_id": pr.PayeeSupplierID,
		"bank_name": pr.BankName, "bank_account": pr.BankAccount,
		"amount": pr.Amount, "amount_cn": pr.AmountCN,
		"payment_method": pr.PaymentMethod, "remark": pr.Remark,
		"company_head": pr.CompanyHead, "finance_head": pr.FinanceHead,
		"dept_head": pr.DeptHead, "handler": pr.Handler,
		"status": pr.Status, "purchase_ids": pr.PurchaseIDs,
		"created_at": pr.CreatedAt, "updated_at": pr.UpdatedAt,
	})
}

func (h *OfficeSupplyHandler) createPaymentRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var payload models.OfficePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}
	if payload.RequestDate.IsZero() {
		respondOfficeError(w, http.StatusBadRequest, "申请日期不能为空")
		return
	}
	payload.UserID = uintPointer(userID)
	payload.RequestNo = service.GenerateRequestNo()
	if payload.PaymentMethod == "" {
		payload.PaymentMethod = "转支"
	}
	if payload.Status == "" {
		payload.Status = "draft"
	}
	// 生成金额大写
	if payload.AmountCN == "" {
		payload.AmountCN = service.AmountToCN(payload.Amount)
	}

	// 创建请款单并联动更新采购单付款状态
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&payload).Error; err != nil {
			return err
		}
		// 关联采购单置为已付款
		h.syncPurchasePaymentStatus(tx, payload.PurchaseIDs, payload.RequestDate)
		return nil
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "创建请款单失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{
		"id": payload.ID, "request_no": payload.RequestNo,
	})
}

func (h *OfficeSupplyHandler) updatePaymentRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的请款单ID")
		return
	}
	var existing models.OfficePaymentRequest
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}

	var payload models.OfficePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondOfficeError(w, http.StatusBadRequest, "请求数据格式错误")
		return
	}

	// 联动处理采购单付款状态
	err = h.db.Transaction(func(tx *gorm.DB) error {
		oldIDs := parseIDList(existing.PurchaseIDs)
		newIDs := parseIDList(payload.PurchaseIDs)

		// 移除不再关联的 → 恢复为未付款
		for _, pid := range oldIDs {
			if !containsID(newIDs, pid) {
				tx.Model(&models.OfficePurchase{}).Where("id = ?", pid).
					Updates(map[string]interface{}{"payment_status": "未付款", "payment_date": nil})
			}
		}
		// 新增关联的 → 设为已付款
		for _, pid := range newIDs {
			if !containsID(oldIDs, pid) {
				tx.Model(&models.OfficePurchase{}).Where("id = ?", pid).
					Updates(map[string]interface{}{
						"payment_status": "已付款",
						"payment_date":   payload.RequestDate,
					})
			}
		}

		amountCN := payload.AmountCN
		if amountCN == "" {
			amountCN = service.AmountToCN(payload.Amount)
		}
		updates := map[string]interface{}{
			"payment_unit": payload.PaymentUnit, "department": payload.Department,
			"applicant": payload.Applicant, "request_date": payload.RequestDate,
			"content": payload.Content, "payee": payload.Payee,
			"payee_supplier_id": payload.PayeeSupplierID,
			"bank_name":         payload.BankName, "bank_account": payload.BankAccount,
			"amount": payload.Amount, "amount_cn": amountCN,
			"payment_method": payload.PaymentMethod, "remark": payload.Remark,
			"company_head": payload.CompanyHead, "finance_head": payload.FinanceHead,
			"dept_head": payload.DeptHead, "handler": payload.Handler,
			"status": payload.Status, "purchase_ids": payload.PurchaseIDs,
			"updated_at": time.Now(),
		}
		return tx.Model(&existing).Updates(updates).Error
	})
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "更新请款单失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{})
}

func (h *OfficeSupplyHandler) deletePaymentRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	id, err := parseOfficeID(r, "id")
	if err != nil {
		respondOfficeError(w, http.StatusBadRequest, "无效的请款单ID")
		return
	}
	var existing models.OfficePaymentRequest
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
		respondOfficeError(w, http.StatusNotFound, "不存在")
		return
	}
	// 删除前恢复关联采购单为未付款
	h.db.Transaction(func(tx *gorm.DB) error {
		purchaseIDs := parseIDList(existing.PurchaseIDs)
		for _, pid := range purchaseIDs {
			tx.Model(&models.OfficePurchase{}).Where("id = ?", pid).
				Updates(map[string]interface{}{"payment_status": "未付款", "payment_date": nil})
		}
		return tx.Where("id = ?", id).Delete(&models.OfficePaymentRequest{}).Error
	})
	respondOfficeOK(w, map[string]interface{}{})
}

// syncPurchasePaymentStatus 联动更新采购单付款状态
func (h *OfficeSupplyHandler) syncPurchasePaymentStatus(tx *gorm.DB, purchaseIDsStr string, requestDate time.Time) {
	ids := parseIDList(purchaseIDsStr)
	for _, pid := range ids {
		tx.Model(&models.OfficePurchase{}).Where("id = ?", pid).
			Updates(map[string]interface{}{
				"payment_status": "已付款",
				"payment_date":   requestDate,
			})
	}
}

// parseIDList 把 "1,2,3" 格式的字符串转为 []uint
func parseIDList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var ids []uint
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func containsID(ids []uint, target uint) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
