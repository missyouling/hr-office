package api

import (
	"strings"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 更正：voucher_type 身份键 ========

func TestCorrectInvoice_VoucherTypeIdentityKey(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "corr-vt", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	key := "vat:VT-CODE|VT-001"
	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "VT-001", InvoiceCode: "VT-CODE", IdentityKey: &key,
		InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusApproved,
		ArchiveStatus: models.InvoiceArchiveStatusConfirmed, VoucherType: models.InvoiceVoucherTypeVATInput}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)

	// 更正 voucher_type → receipt：identity_key 必须按更正后的类型计算（非 vat_input → NULL）
	receipt := models.InvoiceVoucherTypeReceipt
	changes := map[string]any{"voucher_type": map[string]any{"old": models.InvoiceVoucherTypeVATInput, "new": receipt}}
	updates := map[string]any{"voucher_type": receipt}
	if err := handler.applyInvoiceCorrection(&invoice, admin.ID, "更正凭证类型", changes, updates); err != nil {
		t.Fatalf("更正凭证类型失败: %v", err)
	}
	var saved models.Invoice
	tx.First(&saved, invoice.ID)
	if saved.IdentityKey != nil {
		t.Errorf("更正为收据后身份键应为 NULL（使用更正后的 voucher_type）: %v", saved.IdentityKey)
	}
}

// ======== 更正：buyer 重新匹配 ========

func TestCorrectInvoice_BuyerRematch(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "corr-buyer", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)
	setting := models.BuyerEntitySetting{ID: 1, Name: "新购方", TaxNo: "TAX-123"}
	if err := tx.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "BUYER-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed,
		Buyer: "旧购方", BuyerTaxNo: ""}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)

	// 更正 buyer → 新购方（缺税号）：重算匹配 → 名称匹配缺税号，仅 warning 不自动强匹配
	changes := map[string]any{"buyer": map[string]any{"old": "旧购方", "new": "新购方"}}
	updates := map[string]any{"buyer": "新购方"}
	if err := handler.applyInvoiceCorrection(&invoice, admin.ID, "更正购方", changes, updates); err != nil {
		t.Fatalf("更正购方失败: %v", err)
	}
	var saved models.Invoice
	tx.First(&saved, invoice.ID)
	if saved.BuyerMatched {
		t.Error("名称匹配缺税号不应强匹配")
	}
	if !strings.Contains(saved.BuyerMatchNote, "缺少税号") {
		t.Errorf("应记录缺税号提示: %q", saved.BuyerMatchNote)
	}

	// 更正 buyer_tax_no → TAX-123：税号完全匹配 → 强匹配
	changes2 := map[string]any{"buyer_tax_no": map[string]any{"old": "", "new": "TAX-123"}}
	updates2 := map[string]any{"buyer_tax_no": "TAX-123"}
	if err := handler.applyInvoiceCorrection(&invoice, admin.ID, "更正购方税号", changes2, updates2); err != nil {
		t.Fatalf("更正购方税号失败: %v", err)
	}
	tx.First(&saved, invoice.ID)
	if !saved.BuyerMatched {
		t.Errorf("税号完全匹配应强匹配: %+v", saved)
	}
}

// ======== 更正：双审计同事务 ========

func TestCorrectInvoice_DoubleAudit(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "corr-audit", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "CORR-AUDIT", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)
	changes := map[string]any{"seller": map[string]any{"old": "测试", "new": "新销售方"}}
	updates := map[string]any{"seller": "新销售方"}
	if err := handler.applyInvoiceCorrection(&invoice, admin.ID, "更正销售方", changes, updates); err != nil {
		t.Fatalf("更正失败: %v", err)
	}

	var corrAudit models.InvoiceCorrectionAudit
	if err := tx.Where("invoice_id = ?", invoice.ID).First(&corrAudit).Error; err != nil {
		t.Fatalf("更正差异审计未记录: %v", err)
	}
	if corrAudit.Reason != "更正销售方" {
		t.Errorf("更正差异审计应记录原因: %q", corrAudit.Reason)
	}
	var logAudit models.AuditLog
	if err := tx.Where("action = ?", string(models.ActionCorrectInvoice)).First(&logAudit).Error; err != nil {
		t.Fatalf("更正全局审计未记录: %v", err)
	}
	details := logAudit.GetParsedDetails()
	if details == nil || details.Custom == nil || details.Custom["reason"] != "更正销售方" {
		t.Errorf("更正全局审计应包含原因: %+v", details)
	}
}
