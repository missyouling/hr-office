package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 主体设置：白名单 DTO + 审计 + 不回写历史发票 ========

func TestBuyerEntitySetting_WhitelistAndAudit(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "be-admin", "管理员")
	normal := createInvoiceTestUser(t, tx, "be-normal", "普通用户")
	assignRole(t, tx, admin.ID, adminRole.ID)

	// 历史发票：购方为旧主体
	uid := normal.ID
	oldInvoice := models.Invoice{UserID: &uid, InvoiceNo: "BE-OLD", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusDraft, Buyer: "旧购方", BuyerTaxNo: "OLD-TAX"}
	if err := tx.Create(&oldInvoice).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)

	// 白名单 DTO：多余字段（id/created_at/deleted_at）应被忽略
	payload := map[string]interface{}{
		"name": "新购方", "tax_no": "NEW-TAX", "address": "地址", "phone": "123",
		"bank_name": "银行", "bank_account": "账号",
		"id": 999, "created_at": "2020-01-01T00:00:00Z", "deleted_at": "2020-01-01T00:00:00Z",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/invoices/buyer-entity", jsonReader(body))
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("更新主体设置失败: %d %s", w.Code, w.Body.String())
	}

	var setting models.BuyerEntitySetting
	if err := tx.First(&setting, 1).Error; err != nil {
		t.Fatal(err)
	}
	if setting.Name != "新购方" || setting.TaxNo != "NEW-TAX" || setting.ID != 1 {
		t.Errorf("白名单字段未正确保存或受控字段被采用: %+v", setting)
	}

	// 审计：UPDATE_BUYER_ENTITY 含旧值/新值
	var audit models.AuditLog
	if err := tx.Where("action = ?", string(models.ActionUpdateBuyerEntity)).First(&audit).Error; err != nil {
		t.Fatalf("主体设置审计未记录: %v", err)
	}
	details := audit.GetParsedDetails()
	if details == nil || details.Custom == nil || details.Custom["old"] == nil || details.Custom["new"] == nil {
		t.Errorf("主体设置审计应包含旧值/新值: %+v", details)
	}

	// 不回写历史发票
	var savedInvoice models.Invoice
	tx.First(&savedInvoice, oldInvoice.ID)
	if savedInvoice.Buyer != "旧购方" || savedInvoice.BuyerTaxNo != "OLD-TAX" {
		t.Errorf("主体设置更新不应回写历史发票: %+v", savedInvoice)
	}

	// 普通用户更新主体设置 → 403（仅 admin）
	body, _ = json.Marshal(map[string]string{"name": "越权"})
	req = httptest.NewRequest("PUT", "/invoices/buyer-entity", jsonReader(body))
	req = setAuthContext(req, normal.ID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("普通用户更新主体设置应 403，实际 %d", w.Code)
	}
}
