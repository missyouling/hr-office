package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 列表查询测试 ========

func TestListInvoices_Pagination(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	// 创建 5 张发票
	uid := user.ID
	now := time.Now()
	for i := 1; i <= 5; i++ {
		tx.Create(&models.Invoice{
			UserID: &uid, InvoiceNo: fmt.Sprintf("INV-%03d", i),
			InvoiceDate: now.AddDate(0, 0, -i), Amount: float64(i * 100),
			Seller: fmt.Sprintf("公司%d", i), Status: models.InvoiceStatusDraft,
		})
	}

	req := httptest.NewRequest("GET", "/invoices?page=1&page_size=2", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("列表查询失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items    []models.Invoice `json:"items"`
		Total    int64            `json:"total"`
		Page     int              `json:"page"`
		PageSize int              `json:"page_size"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 5 {
		t.Errorf("期望总数 5，实际 %d", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Errorf("期望返回 2 条，实际 %d", len(resp.Items))
	}
}

func TestListInvoices_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	now := time.Now()
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-A", InvoiceDate: now,
		Amount: 100, Seller: "公司A", Status: models.InvoiceStatusDraft,
	})
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-B", InvoiceDate: now,
		Amount: 200, Seller: "公司B", Status: models.InvoiceStatusApproved,
	})

	req := httptest.NewRequest("GET", "/invoices?status=approved", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Items []models.Invoice `json:"items"`
		Total int64            `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 1 {
		t.Errorf("按 status=approved 筛选期望 1 条，实际 %d", resp.Total)
	}
	if len(resp.Items) > 0 && resp.Items[0].InvoiceNo != "INV-B" {
		t.Errorf("期望筛选到 INV-B，实际 %s", resp.Items[0].InvoiceNo)
	}
}

func TestListInvoices_FilterBySourceType(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	now := time.Now()
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-O", InvoiceDate: now,
		Amount: 100, Seller: "公司", Status: models.InvoiceStatusDraft,
		SourceType: models.InvoiceSourceOffice,
	})
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-C", InvoiceDate: now,
		Amount: 200, Seller: "公司", Status: models.InvoiceStatusDraft,
		SourceType: models.InvoiceSourceCanteen,
	})

	req := httptest.NewRequest("GET", "/invoices?source_type=canteen", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Items []models.Invoice `json:"items"`
		Total int64            `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 1 {
		t.Errorf("按 source_type=canteen 筛选期望 1 条，实际 %d", resp.Total)
	}
}

func TestListInvoices_FilterByDateRange(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-JAN",
		InvoiceDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:      100, Seller: "公司A", Status: models.InvoiceStatusDraft,
	})
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-AUG",
		InvoiceDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Amount:      200, Seller: "公司B", Status: models.InvoiceStatusDraft,
	})

	req := httptest.NewRequest("GET", "/invoices?date_from=2026-06-01&date_to=2026-12-31", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Items []models.Invoice `json:"items"`
		Total int64            `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 1 {
		t.Errorf("按日期范围筛选期望 1 条，实际 %d", resp.Total)
	}
}
