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

// ======== 更新发票测试 ========

func TestUpdateInvoice_OnlyDraftCanUpdate(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	draftInv := models.Invoice{
		UserID: &uid, InvoiceNo: "INV-DRAFT", InvoiceDate: time.Now(),
		Amount: 100, Seller: "公司", Status: models.InvoiceStatusDraft,
	}
	tx.Create(&draftInv)

	submittedInv := models.Invoice{
		UserID: &uid, InvoiceNo: "INV-SUB", InvoiceDate: time.Now(),
		Amount: 200, Seller: "公司", Status: models.InvoiceStatusSubmitted,
	}
	tx.Create(&submittedInv)

	// 草稿可以更新
	body, _ := json.Marshal(map[string]interface{}{"seller": "更新后的公司", "amount": 150.0})
	req := httptest.NewRequest("PUT", fmt.Sprintf("/invoices/%d", draftInv.ID), jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("更新草稿发票应返回 200，实际 %d，响应: %s", w.Code, w.Body.String())
	}

	// 已提交的发票不能更新
	body, _ = json.Marshal(map[string]interface{}{"seller": "不能更新"})
	req = httptest.NewRequest("PUT", fmt.Sprintf("/invoices/%d", submittedInv.ID), jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("更新已提交发票应返回 403，实际 %d", w.Code)
	}
}

func TestUpdateInvoice_IgnoresControlledFields(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "updater", "更新者")
	otherUser := createInvoiceTestUser(t, tx, "otherupdater", "其他用户")
	uid := user.ID
	identityKey := "可信身份键"
	invoice := models.Invoice{
		UserID: &uid, InvoiceNo: "INV-UPDATE-CONTROLLED", InvoiceDate: time.Now(),
		Amount: 100, Seller: "原销售方", Status: models.InvoiceStatusDraft,
		IdentityKey: &identityKey, ArchiveStatus: models.InvoiceArchiveStatusPending,
	}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatalf("创建测试发票失败: %v", err)
	}

	payload := map[string]interface{}{
		"seller": "更新后的销售方", "amount": 200.0, "identity_key": "抢占后的身份键",
		"attachment_file_id": uint(999), "file_sha256": "malicious-hash", "archive_status": "confirmed",
		"user_id": otherUser.ID, "status": "approved", "approver_id": otherUser.ID,
		"approval_remark": "伪造审批", "attachment_url": "https://untrusted.example/invoice.pdf",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/invoices/%d", invoice.ID), jsonReader(body))
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	newInvoiceTestRouterNoAuth(t, NewHandler(tx)).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("更新发票失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	var updated models.Invoice
	if err := tx.First(&updated, invoice.ID).Error; err != nil {
		t.Fatalf("查询更新后的发票失败: %v", err)
	}
	if updated.Seller != "更新后的销售方" || updated.Amount != 200 {
		t.Errorf("允许的业务字段未更新: %+v", updated)
	}
	if updated.UserID == nil || *updated.UserID != user.ID || updated.IdentityKey != nil || updated.AttachmentFileID != nil || updated.FileSHA256 != "" {
		t.Errorf("受控归属、身份或附件字段被篡改: %+v", updated)
	}
	if updated.Status != models.InvoiceStatusDraft || updated.ArchiveStatus != models.InvoiceArchiveStatusPending || updated.ApproverID != nil || updated.ApprovalRemark != "" || updated.AttachmentURL != "" {
		t.Errorf("受控状态、审批或遗留附件字段被篡改: %+v", updated)
	}
}
