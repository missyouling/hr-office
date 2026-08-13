package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 作废：原因必填 ========

func TestVoidInvoice_ReasonRequired(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "void-admin", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "VOID-REQ", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)

	for _, reason := range []string{"", "   "} {
		body, _ := json.Marshal(map[string]string{"reason": reason})
		req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/void", invoice.ID), jsonReader(body))
		req = setAuthContext(req, admin.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("空白作废原因应 400，实际 %d", w.Code)
		}
	}
}

// ======== 作废：事务内 CAS 并发 ========

func TestVoidInvoice_ConcurrentCAS(t *testing.T) {
	// handler 使用独立 db（非外层事务），并发请求各自独立事务，单连接池串行，等价真实并发
	db := setupTestDB(t)
	migrateInvoiceTables(t, db)

	adminRole := createInvoiceTestRole(t, db, "admin")
	admin := createInvoiceTestUser(t, db, "void-cas", "管理员")
	assignRole(t, db, admin.ID, adminRole.ID)

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "VOID-CAS", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(db)
	router := newInvoiceTestRouter(t, handler)

	const workers = 4
	results := make(chan int, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(map[string]string{"reason": "并发作废"})
			req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/void", invoice.ID), jsonReader(body))
			req = setAuthContext(req, admin.ID)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}
	wg.Wait()
	close(results)
	success, failed := 0, 0
	for code := range results {
		switch code {
		case http.StatusOK:
			success++
		default:
			// 并发竞争下：CAS 冲突返回 409，先读后写窗口返回 403，均视为失败
			failed++
		}
	}
	if success != 1 || failed != workers-1 {
		t.Errorf("并发作废应恰好 1 成功 %d 失败，实际 success=%d failed=%d", workers-1, success, failed)
	}
	var saved models.Invoice
	db.First(&saved, invoice.ID)
	if saved.ArchiveStatus != models.InvoiceArchiveStatusVoided {
		t.Errorf("作废后归档状态应为 voided: %s", saved.ArchiveStatus)
	}
}

// ======== 作废：同事务审计 ========

func TestVoidInvoice_AuditLogged(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "void-audit", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	invoice := models.Invoice{UserID: &admin.ID, InvoiceNo: "VOID-AUDIT", InvoiceDate: time.Now(), Amount: 100, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)
	body, _ := json.Marshal(map[string]string{"reason": "作废审计"})
	req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/void", invoice.ID), jsonReader(body))
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("作废失败: %d %s", w.Code, w.Body.String())
	}
	var audit models.AuditLog
	if err := tx.Where("action = ?", string(models.ActionVoidInvoice)).First(&audit).Error; err != nil {
		t.Fatalf("作废审计未记录: %v", err)
	}
	details := audit.GetParsedDetails()
	if details == nil || details.Custom == nil || details.Custom["reason"] != "作废审计" {
		t.Errorf("作废审计应包含原因: %+v", details)
	}
}
