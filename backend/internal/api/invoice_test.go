package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== 创建发票测试 ========

func TestCreateInvoice_RequiredFields(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	tests := []struct {
		name       string
		payload    map[string]interface{}
		wantStatus int
	}{
		{
			name:       "发票号为空",
			payload:    map[string]interface{}{"seller": "测试公司", "amount": 100.0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "销售方为空",
			payload:    map[string]interface{}{"invoice_no": "INV-001", "amount": 100.0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "金额为0",
			payload:    map[string]interface{}{"invoice_no": "INV-001", "seller": "测试公司", "amount": 0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "正常创建",
			payload: map[string]interface{}{
				"invoice_no":   "INV-001",
				"invoice_date": "2026-08-01T00:00:00Z",
				"seller":       "测试销售方",
				"amount":       100.0,
				"tax_amount":   13.0,
				"total_amount": 113.0,
				"purpose":      "办公用品采购",
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = setAuthContext(req, user.ID)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("期望状态码 %d，实际 %d，响应: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateInvoice_AllowsDuplicateInvoiceNo(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	// 直接插入一条已有发票
	uid := user.ID
	tx.Create(&models.Invoice{
		UserID: &uid, InvoiceNo: "INV-DUP", InvoiceDate: time.Now(),
		Amount: 100, Seller: "测试公司", Status: models.InvoiceStatusDraft,
	})

	// 收据等凭证可能没有票号，票号本身不承担全局去重职责。
	payload := map[string]interface{}{
		"invoice_no": "INV-DUP", "seller": "测试公司", "amount": 100.0,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("重复发票号应允许创建，实际: %d, 响应: %s", w.Code, w.Body.String())
	}
}

func TestCreateInvoice_IgnoresControlledFields(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "creator", "创建者")
	otherUser := createInvoiceTestUser(t, tx, "other", "其他用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	identityKey := "被抢占的身份键"
	payload := map[string]interface{}{
		"invoice_no": "INV-CONTROLLED", "seller": "测试公司", "amount": 100.0,
		"identity_key": identityKey, "attachment_file_id": uint(999),
		"file_sha256": "malicious-hash", "archive_status": "confirmed",
		"user_id": otherUser.ID, "status": "approved", "approver_id": otherUser.ID,
		"approved_at": time.Now().Format(time.RFC3339), "approval_remark": "伪造审批",
		"reimburse_amount": 100.0, "parsing_tasks": []map[string]interface{}{{"status": "succeeded"}},
		"correction_audits": []map[string]interface{}{{"reason": "伪造审计"}},
		"attachment_url":    "https://untrusted.example/invoice.pdf",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建发票失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	var invoice models.Invoice
	if err := tx.Where("invoice_no = ?", "INV-CONTROLLED").First(&invoice).Error; err != nil {
		t.Fatalf("查询创建的发票失败: %v", err)
	}
	if invoice.UserID == nil || *invoice.UserID != user.ID || invoice.IdentityKey != nil || invoice.AttachmentFileID != nil || invoice.FileSHA256 != "" {
		t.Errorf("受控归属或附件字段被采用: %+v", invoice)
	}
	if invoice.Status != models.InvoiceStatusDraft || invoice.ArchiveStatus != models.InvoiceArchiveStatusPending || invoice.ApproverID != nil || invoice.ApprovedAt != nil || invoice.ApprovalRemark != "" || invoice.ReimburseAmount != 0 {
		t.Errorf("受控状态或审批字段被采用: %+v", invoice)
	}
	if invoice.AttachmentURL != "" {
		t.Errorf("遗留 attachment_url 新写入应被忽略，实际为 %q", invoice.AttachmentURL)
	}
	var parsingTaskCount, auditCount int64
	tx.Model(&models.InvoiceParsingTask{}).Where("invoice_id = ?", invoice.ID).Count(&parsingTaskCount)
	tx.Model(&models.InvoiceCorrectionAudit{}).Where("invoice_id = ?", invoice.ID).Count(&auditCount)
	if parsingTaskCount != 0 || auditCount != 0 {
		t.Errorf("受控关联被创建: tasks=%d audits=%d", parsingTaskCount, auditCount)
	}
}
