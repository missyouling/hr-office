package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"siapp/internal/models"
)

// ======== identity key 算法三种规则 ========

func TestComputeInvoiceIdentityKey_Rules(t *testing.T) {
	vat := models.InvoiceVoucherTypeVATInput
	receipt := models.InvoiceVoucherTypeReceipt

	// 规则一：增值税发票优先规范化 invoice_code + invoice_no，前缀 vat:
	if key := computeInvoiceIdentityKey(vat, "No-001", "Code-01", "E-999"); key == nil || *key != "vat:CODE-01|NO-001" {
		t.Errorf("增值税发票应生成 vat:code|no: %v", key)
	}
	// 规范化：去除首尾空白 + 统一大写
	if key := computeInvoiceIdentityKey(vat, "  No-002  ", "  Code-02 ", ""); key == nil || *key != "vat:CODE-02|NO-002" {
		t.Errorf("身份键应规范化空白与大小写: %v", key)
	}
	// 规范化：全角数字/字母转半角（OCR 常见输出），不做字符纠错
	if key := computeInvoiceIdentityKey(vat, "Ｎｏ-００３", "Ｃｏｄｅ-０３", ""); key == nil || *key != "vat:CODE-03|NO-003" {
		t.Errorf("身份键应转换全角字符: %v", key)
	}
	// 规则二：缺代码时回退电子票号，前缀 electronic:
	if key := computeInvoiceIdentityKey(vat, "No-004", "", "E-004"); key == nil || *key != "electronic:E-004" {
		t.Errorf("缺代码时应回退电子票号: %v", key)
	}
	// 规则三：其他凭证类型为 NULL，不参与全局去重
	if key := computeInvoiceIdentityKey(receipt, "No-005", "Code-05", "E-005"); key != nil {
		t.Errorf("收据等非增值税凭证身份键应为 NULL: %v", key)
	}
	// 空值不存空字符串
	if key := computeInvoiceIdentityKey(vat, "", "", ""); key != nil {
		t.Errorf("全空字段身份键应为 NULL: %v", key)
	}
	if key := computeInvoiceIdentityKey(vat, "No-006", "", ""); key != nil {
		t.Errorf("缺代码且无电子票号应为 NULL: %v", key)
	}
}

// ======== 字段与 identity_key 同步 ========

func TestInvoiceIdentityKey_FieldSync(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "sync-user", "同步用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	// 创建：vat_input + code + no → identity_key = vat:code|no
	payload := map[string]interface{}{
		"invoice_no": "SYNC-001", "invoice_code": "SYNC-CODE", "electronic_invoice_no": "SYNC-E",
		"invoice_date": time.Now().Format(time.RFC3339), "seller": "测试", "amount": 100.0,
		"total_amount": 113.0, "tax_amount": 13.0, "voucher_type": "vat_input",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("创建发票失败: %d %s", w.Code, w.Body.String())
	}
	var createResp struct {
		Item models.Invoice `json:"item"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	if createResp.Item.IdentityKey == nil || *createResp.Item.IdentityKey != "vat:SYNC-CODE|SYNC-001" {
		t.Fatalf("创建后身份键与字段不同步: %v", createResp.Item.IdentityKey)
	}

	// 更新：仅改 seller，identity_key 应保持（基于现有字段重新计算）
	updateBody, _ := json.Marshal(map[string]interface{}{"seller": "新销售方"})
	req = httptest.NewRequest("PUT", fmt.Sprintf("/invoices/%d", createResp.Item.ID), jsonReader(updateBody))
	req = setAuthContext(req, user.ID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("更新发票失败: %d %s", w.Code, w.Body.String())
	}
	var updated models.Invoice
	if err := tx.First(&updated, createResp.Item.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.IdentityKey == nil || *updated.IdentityKey != "vat:SYNC-CODE|SYNC-001" {
		t.Errorf("更新后身份键未同步: %v", updated.IdentityKey)
	}
}

func TestInvoiceIdentityKey_ParseWriteBackSync(t *testing.T) {
	// 解析写回：vat_input 发票生成 vat:code|no，收据为 NULL
	parsed := parsedInvoice{invoiceNo: "PARSE-001", code: "PARSE-CODE", electronicNo: "PARSE-E", total: 100}
	updates := parsed.invoiceUpdates("测试文本", "ocr", models.InvoiceVoucherTypeVATInput)
	key, ok := updates["identity_key"].(*string)
	if !ok || key == nil || *key != "vat:PARSE-CODE|PARSE-001" {
		t.Errorf("解析写回身份键错误: %v", updates["identity_key"])
	}
	updates = parsed.invoiceUpdates("测试文本", "ocr", models.InvoiceVoucherTypeReceipt)
	if key, ok := updates["identity_key"].(*string); ok && key != nil {
		t.Errorf("收据解析写回身份键应为 NULL: %v", updates["identity_key"])
	}
}

func TestInvoiceIdentityKey_CorrectionSync(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "correct-user", "更正用户")
	uid := user.ID
	key := "vat:CORRECT-CODE|CORRECT-001"
	invoice := models.Invoice{
		UserID: &uid, InvoiceNo: "CORRECT-001", InvoiceCode: "CORRECT-CODE",
		InvoiceDate: time.Now(), Amount: 100, TotalAmount: 113, Seller: "测试",
		Status: models.InvoiceStatusApproved, ArchiveStatus: models.InvoiceArchiveStatusConfirmed,
		VoucherType: models.InvoiceVoucherTypeVATInput, IdentityKey: &key,
	}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatalf("创建发票失败: %v", err)
	}
	handler := NewHandler(tx)
	changes := map[string]any{"invoice_no": map[string]any{"old": "CORRECT-001", "new": "CORRECT-002"}}
	updates := map[string]any{"invoice_no": "CORRECT-002"}
	if err := handler.applyInvoiceCorrection(&invoice, user.ID, "更正票号", changes, updates); err != nil {
		t.Fatalf("更正失败: %v", err)
	}
	var saved models.Invoice
	if err := tx.First(&saved, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.IdentityKey == nil || *saved.IdentityKey != "vat:CORRECT-CODE|CORRECT-002" {
		t.Errorf("更正后身份键未同步: %v", saved.IdentityKey)
	}
}

// ======== 活动重复与软删复用 ========

func TestInvoiceIdentityKey_ConflictAndSoftDeleteReuse(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "conflict-user", "冲突用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	create := func(no string) int {
		payload := map[string]interface{}{
			"invoice_no": no, "invoice_code": "DUP-CODE", "invoice_date": time.Now().Format(time.RFC3339),
			"seller": "测试", "amount": 100.0, "total_amount": 113.0, "tax_amount": 13.0,
			"voucher_type": "vat_input",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	if code := create("DUP-001"); code != http.StatusCreated {
		t.Fatalf("首次创建应成功: %d", code)
	}
	// 相同 identity_key 的活动记录冲突 → 409
	if code := create("DUP-001"); code != http.StatusConflict {
		t.Fatalf("活动重复身份键应返回 409: %d", code)
	}
	// 软删除首张发票后允许复用
	var first models.Invoice
	if err := tx.Where("invoice_no = ?", "DUP-001").First(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Delete(&first).Error; err != nil {
		t.Fatal(err)
	}
	if code := create("DUP-001"); code != http.StatusCreated {
		t.Fatalf("软删除后应允许复用身份键: %d", code)
	}
}

// ======== 并发判重 ========

func TestInvoiceIdentityKey_ConcurrentCreateDedup(t *testing.T) {
	// handler 使用独立 db（非外层事务），并发请求各自独立执行，单连接池串行，等价真实并发
	db := setupTestDB(t)
	migrateInvoiceTables(t, db)

	user := createInvoiceTestUser(t, db, "conc-user", "并发用户")
	handler := NewHandler(db)
	router := newInvoiceTestRouterNoAuth(t, handler)

	const workers = 5
	results := make(chan int, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := map[string]interface{}{
				"invoice_no": "CONC-001", "invoice_code": "CONC-CODE",
				"invoice_date": time.Now().Format(time.RFC3339),
				"seller":       "测试", "amount": 100.0, "total_amount": 113.0, "tax_amount": 13.0,
				"voucher_type": "vat_input",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
			req = setAuthContext(req, user.ID)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}
	wg.Wait()
	close(results)
	created, conflicted := 0, 0
	for code := range results {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicted++
		}
	}
	if created != 1 || conflicted != workers-1 {
		t.Errorf("并发创建应恰好 1 成功 %d 冲突，实际 created=%d conflicted=%d", workers-1, created, conflicted)
	}
}

func TestInvoiceIdentityKey_ConcurrentCorrectionDedup(t *testing.T) {
	// 注意：handler 使用独立 db（非外层事务），并发更正各自开启独立事务，
	// 单连接池下串行执行，等价真实并发场景；避免嵌套事务并发数据竞争。
	db := setupTestDB(t)
	migrateInvoiceTables(t, db)

	user := createInvoiceTestUser(t, db, "conc-corr", "并发更正")
	uid := user.ID
	key1 := "vat:CC-CODE|CC-001"
	key2 := "vat:CC-CODE|CC-002"
	inv1 := models.Invoice{UserID: &uid, InvoiceNo: "CC-001", InvoiceCode: "CC-CODE", IdentityKey: &key1,
		InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusApproved,
		ArchiveStatus: models.InvoiceArchiveStatusConfirmed, VoucherType: models.InvoiceVoucherTypeVATInput}
	inv2 := models.Invoice{UserID: &uid, InvoiceNo: "CC-002", InvoiceCode: "CC-CODE", IdentityKey: &key2,
		InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusApproved,
		ArchiveStatus: models.InvoiceArchiveStatusConfirmed, VoucherType: models.InvoiceVoucherTypeVATInput}
	if err := db.Create(&inv1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&inv2).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(db)

	// 并发把两张发票的 invoice_no 都改为 CC-003（相同 identity_key），只有一个成功
	changes1 := map[string]any{"invoice_no": map[string]any{"old": "CC-001", "new": "CC-003"}}
	updates1 := map[string]any{"invoice_no": "CC-003"}
	changes2 := map[string]any{"invoice_no": map[string]any{"old": "CC-002", "new": "CC-003"}}
	updates2 := map[string]any{"invoice_no": "CC-003"}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs <- handler.applyInvoiceCorrection(&inv1, user.ID, "并发更正", changes1, updates1)
	}()
	go func() {
		defer wg.Done()
		errs <- handler.applyInvoiceCorrection(&inv2, user.ID, "并发更正", changes2, updates2)
	}()
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, errIdentityConflict) {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Errorf("并发更正应恰好 1 成功 1 冲突，实际 success=%d conflict=%d", success, conflict)
	}
}
