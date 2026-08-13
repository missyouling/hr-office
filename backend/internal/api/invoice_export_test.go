package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== CSV 公式注入防护 ========

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"等号开头", "=1+1", "'=1+1"},
		{"加号开头", "+SUM(A1)", "'+SUM(A1)"},
		{"减号开头", "-2+3", "'-2+3"},
		{"@开头", "@cmd", "'@cmd"},
		{"普通值", "普通文本", "普通文本"},
		{"数字", "12345", "12345"},
		{"空值", "", ""},
		{"含空白", "  文本  ", "文本"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeCSVCell(tt.value); got != tt.want {
				t.Errorf("sanitizeCSVCell(%q) = %q，期望 %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestExportInvoicesCSV_FormulaInjectionPrevented(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "csv-admin", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	uid := admin.ID
	malicious := "=HYPERLINK(\"http://evil.example\")"
	if err := tx.Create(&models.Invoice{UserID: &uid, InvoiceNo: "CSV-001", InvoiceDate: time.Now(),
		Amount: 100, Seller: malicious, Status: models.InvoiceStatusDraft}).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)
	req := httptest.NewRequest("GET", "/invoices/export", nil)
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("导出失败: %d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "'=HYPERLINK") {
		t.Errorf("公式注入字段应以单引号转义，实际 CSV: %s", body)
	}
	if strings.Contains(body, "malicious") {
		t.Errorf("CSV 不应包含未转义的原始公式: %s", body)
	}
}

func TestExportInvoicesCSV_ExportAuditLogged(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "csv-audit", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	uid := admin.ID
	tx.Create(&models.Invoice{UserID: &uid, InvoiceNo: "CSV-AUDIT", InvoiceDate: time.Now(),
		Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft})

	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)
	req := httptest.NewRequest("GET", "/invoices/export?status=draft", nil)
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("导出失败: %d", w.Code)
	}

	var audit models.AuditLog
	if err := tx.Where("action = ?", string(models.ActionExportInvoices)).First(&audit).Error; err != nil {
		t.Fatalf("导出审计未记录: %v", err)
	}
	details := audit.GetParsedDetails()
	if details == nil || details.Custom == nil {
		t.Fatal("导出审计缺少筛选摘要")
	}
	if details.Custom["count"] == nil {
		t.Errorf("导出审计应包含导出条数: %+v", details.Custom)
	}
}

func TestExportInvoicesCSV_MaxRowsLimit(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "csv-limit", "管理员")
	assignRole(t, tx, admin.ID, adminRole.ID)

	// 批量插入超过导出上限的记录
	uid := admin.ID
	rows := make([]models.Invoice, 0, invoiceExportMaxRows+1)
	for i := 0; i < invoiceExportMaxRows+1; i++ {
		rows = append(rows, models.Invoice{UserID: &uid, InvoiceNo: "LIMIT", InvoiceDate: time.Now(),
			Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft})
	}
	if err := tx.CreateInBatches(&rows, 500).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)
	req := httptest.NewRequest("GET", "/invoices/export", nil)
	req = setAuthContext(req, admin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("超过导出上限应返回 400，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "导出数据量过大") {
		t.Errorf("应提示缩小筛选范围: %s", w.Body.String())
	}
}

// 保持 encoding/json 引用（响应解析辅助）
var _ = json.Unmarshal
