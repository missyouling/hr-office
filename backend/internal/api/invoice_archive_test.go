package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 确认：来源校验 ========

func TestConfirmInvoice_SourceValidation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "cfm-admin", "管理员")
	owner := createInvoiceTestUser(t, tx, "cfm-owner", "采购本人")
	assignRole(t, tx, admin.ID, adminRole.ID)

	oid := owner.ID
	purchase := models.OfficePurchase{UserID: &oid, OrderNo: "CFM-PO", PurchaseDate: time.Now(), TotalAmount: 100}
	if err := tx.Create(&purchase).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)

	// 合法来源：确认成功
	valid := models.Invoice{UserID: &oid, InvoiceNo: "CFM-OK", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		SourceType: models.InvoiceSourceOffice, SourceID: &purchase.ID}
	if err := tx.Create(&valid).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := handler.confirmInvoiceWithChecks(&valid, admin.ID); err != nil {
		t.Errorf("合法来源确认应成功: %v", err)
	}

	// 来源不存在：确认失败
	missing := models.Invoice{UserID: &oid, InvoiceNo: "CFM-MISS", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		SourceType: models.InvoiceSourceOffice, SourceID: uintPtr(99999)}
	if err := tx.Create(&missing).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := handler.confirmInvoiceWithChecks(&missing, admin.ID); err == nil {
		t.Error("来源不存在时确认应失败")
	}

	// 非法来源类型：确认失败
	invalid := models.Invoice{UserID: &oid, InvoiceNo: "CFM-BAD", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		SourceType: "unknown_type", SourceID: &purchase.ID}
	if err := tx.Create(&invalid).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := handler.confirmInvoiceWithChecks(&invalid, admin.ID); err == nil {
		t.Error("非法来源类型确认应失败")
	}
}

// ======== 确认：主体匹配 warning ========

func TestConfirmInvoice_BuyerMatchWarnings(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "bm-admin", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)
	handler := NewHandler(tx)

	// 无主体设置：不匹配且不报错
	noSetting := models.Invoice{UserID: &admin.ID, InvoiceNo: "BM-NONE", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := tx.Create(&noSetting).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := handler.confirmInvoiceWithChecks(&noSetting, admin.ID); err != nil {
		t.Fatalf("无设置确认应成功: %v", err)
	}
	if noSetting.BuyerMatched {
		t.Error("无主体设置不应匹配")
	}

	// 配置主体：名称 + 税号
	setting := models.BuyerEntitySetting{ID: 1, Name: "测试购方", TaxNo: "91310000TEST12345X"}
	if err := tx.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}

	// 税号完全匹配 → 强匹配，无 warning
	matched := models.Invoice{UserID: &admin.ID, InvoiceNo: "BM-MATCH", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		Buyer: "测试购方", BuyerTaxNo: "91310000TEST12345X"}
	if err := tx.Create(&matched).Error; err != nil {
		t.Fatal(err)
	}
	warnings, err := handler.confirmInvoiceWithChecks(&matched, admin.ID)
	if err != nil {
		t.Fatalf("税号匹配确认失败: %v", err)
	}
	if !matched.BuyerMatched || len(warnings) != 0 {
		t.Errorf("税号完全匹配应强匹配且无 warning: matched=%v warnings=%v", matched.BuyerMatched, warnings)
	}

	// 名称匹配但缺税号 → 仅 warning，不自动强匹配（不写入税号）
	nameOnly := models.Invoice{UserID: &admin.ID, InvoiceNo: "BM-NAME", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		Buyer: "测试购方", BuyerTaxNo: ""}
	if err := tx.Create(&nameOnly).Error; err != nil {
		t.Fatal(err)
	}
	warnings, err = handler.confirmInvoiceWithChecks(&nameOnly, admin.ID)
	if err != nil {
		t.Fatalf("名称匹配确认失败: %v", err)
	}
	if nameOnly.BuyerMatched || len(warnings) != 1 || !strings.Contains(warnings[0], "缺少税号") {
		t.Errorf("名称匹配缺税号应仅 warning: matched=%v warnings=%v", nameOnly.BuyerMatched, warnings)
	}
	var savedNameOnly models.Invoice
	tx.First(&savedNameOnly, nameOnly.ID)
	if savedNameOnly.BuyerTaxNo != "" {
		t.Errorf("名称匹配缺税号不得自动强匹配写入税号: %q", savedNameOnly.BuyerTaxNo)
	}

	// 购方不匹配 → warning
	mismatch := models.Invoice{UserID: &admin.ID, InvoiceNo: "BM-MIS", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		Buyer: "其他公司", BuyerTaxNo: "999"}
	if err := tx.Create(&mismatch).Error; err != nil {
		t.Fatal(err)
	}
	warnings, err = handler.confirmInvoiceWithChecks(&mismatch, admin.ID)
	if err != nil {
		t.Fatalf("购方不匹配确认失败: %v", err)
	}
	if mismatch.BuyerMatched || len(warnings) != 1 {
		t.Errorf("购方不匹配应保留 warning: matched=%v warnings=%v", mismatch.BuyerMatched, warnings)
	}
}

// ======== 确认：同事务审计 ========

func TestConfirmInvoice_AuditLogged(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "cfm-audit", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "CFM-AUDIT", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)
	if _, err := handler.confirmInvoiceWithChecks(&invoice, admin.ID); err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	var audit models.AuditLog
	if err := tx.Where("action = ?", string(models.ActionConfirmInvoice)).First(&audit).Error; err != nil {
		t.Fatalf("确认审计未记录: %v", err)
	}
	details := audit.GetParsedDetails()
	if details == nil || details.Custom == nil || details.Custom["new_archive_status"] != string(models.InvoiceArchiveStatusConfirmed) {
		t.Errorf("确认审计应包含归档状态变化: %+v", details)
	}
}

// ======== 确认：响应对象与数据库一致 ========

func TestConfirmInvoice_ResponseReflectsDB(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "cfm-resp", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	// 购方不匹配，确认后应产生 buyer_match_note
	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "CFM-RESP", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusPending,
		Buyer: "其他公司", BuyerTaxNo: "999"}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)

	req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/confirm", invoice.ID), nil)
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("确认失败: %d %s", w.Code, w.Body.String())
	}

	var resp struct {
		Item models.Invoice `json:"item"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Item.ArchiveStatus != models.InvoiceArchiveStatusConfirmed {
		t.Errorf("响应 archive_status 应为 confirmed: %s", resp.Item.ArchiveStatus)
	}
	if resp.Item.ConfirmedBy == nil || *resp.Item.ConfirmedBy != admin.ID {
		t.Errorf("响应 confirmed_by 应为 %d: %v", admin.ID, resp.Item.ConfirmedBy)
	}
	if resp.Item.ConfirmedAt == nil {
		t.Error("响应 confirmed_at 不应为空")
	}

	var saved models.Invoice
	tx.First(&saved, invoice.ID)
	if saved.ArchiveStatus != resp.Item.ArchiveStatus {
		t.Errorf("响应与数据库 archive_status 不一致: resp=%s db=%s", resp.Item.ArchiveStatus, saved.ArchiveStatus)
	}
	if saved.BuyerMatched != resp.Item.BuyerMatched || saved.BuyerMatchNote != resp.Item.BuyerMatchNote {
		t.Errorf("响应与数据库匹配结果不一致: resp(matched=%v note=%q) db(matched=%v note=%q)",
			resp.Item.BuyerMatched, resp.Item.BuyerMatchNote, saved.BuyerMatched, saved.BuyerMatchNote)
	}
	if saved.ConfirmedAt == nil || resp.Item.ConfirmedAt == nil || !saved.ConfirmedAt.Equal(*resp.Item.ConfirmedAt) {
		t.Errorf("响应与数据库 confirmed_at 不一致: resp=%v db=%v", resp.Item.ConfirmedAt, saved.ConfirmedAt)
	}
}

// uintPtr 返回 uint 指针（测试辅助）
func uintPtr(value uint) *uint {
	return &value
}
