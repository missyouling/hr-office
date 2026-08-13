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

// ======== 提交发票测试 ========

func TestSubmitInvoice_StatusTransition(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	draftInv := models.Invoice{
		UserID: &uid, InvoiceNo: "INV-TRANS", InvoiceDate: time.Now(),
		Amount: 100, Seller: "公司", Status: models.InvoiceStatusDraft,
	}
	tx.Create(&draftInv)

	// 提交草稿
	req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/submit", draftInv.ID), nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("提交发票失败，状态码 %d，响应: %s", w.Code, w.Body.String())
	}

	// 验证状态已变更
	var updated models.Invoice
	tx.First(&updated, draftInv.ID)
	if updated.Status != models.InvoiceStatusSubmitted {
		t.Errorf("期望状态 submitted，实际 %s", updated.Status)
	}

	// 重复提交应被拒绝
	req = httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/submit", draftInv.ID), nil)
	req = setAuthContext(req, user.ID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("重复提交应返回 403，实际 %d", w.Code)
	}
}

// ======== 审批权限测试 ========

func TestApproveInvoice_PermissionDenied(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	// 创建 admin 和 viewer 用户
	adminRole := createInvoiceTestRole(t, tx, "admin")
	viewerRole := createInvoiceTestRole(t, tx, "viewer")

	adminUser := createInvoiceTestUser(t, tx, "adminuser", "管理员")
	viewerUser := createInvoiceTestUser(t, tx, "vieweruser", "观察者")

	assignRole(t, tx, adminUser.ID, adminRole.ID)
	assignRole(t, tx, viewerUser.ID, viewerRole.ID)

	// 创建已提交的发票
	uid := viewerUser.ID
	submittedInv := models.Invoice{
		UserID: &uid, InvoiceNo: "INV-PERM", InvoiceDate: time.Now(),
		Amount: 100, Seller: "公司", Status: models.InvoiceStatusSubmitted,
	}
	tx.Create(&submittedInv)

	handler := NewHandler(tx)

	// viewer 尝试审批（使用带中间件的路由器，应被拒绝）
	routerWithAuth := newInvoiceTestRouter(t, handler)
	req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/approve", submittedInv.ID), nil)
	req = setAuthContext(req, viewerUser.ID)
	w := httptest.NewRecorder()
	routerWithAuth.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("viewer 审批应返回 403，实际 %d", w.Code)
	}

	// admin 审批应成功
	body, _ := json.Marshal(map[string]string{"approval_remark": "同意报销"})
	req = httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/approve", submittedInv.ID), jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, adminUser.ID)
	w = httptest.NewRecorder()
	routerWithAuth.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin 审批应返回 200，实际 %d，响应: %s", w.Code, w.Body.String())
	}
}

// ======== 统计测试 ========

func TestInvoiceStats_CountCorrect(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	uid := user.ID
	now := time.Now()
	statuses := []string{"draft", "draft", "submitted", "approved", "reimbursed"}
	for i, s := range statuses {
		tx.Create(&models.Invoice{
			UserID: &uid, InvoiceNo: fmt.Sprintf("INV-STAT-%d", i),
			InvoiceDate: now, Amount: 100.0, TotalAmount: 113.0,
			Seller: "公司", Status: s,
		})
	}

	req := httptest.NewRequest("GET", "/invoices/stats", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("统计查询失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	var resp struct {
		TotalCount  int64   `json:"total_count"`
		TotalAmount float64 `json:"total_amount"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.TotalCount != 5 {
		t.Errorf("期望总数 5，实际 %d", resp.TotalCount)
	}
}

// ======== 状态机与权限综合测试 ========

func TestInvoiceStateMachine(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	// 创建用户与角色
	adminRole := createInvoiceTestRole(t, tx, "admin")
	managerRole := createInvoiceTestRole(t, tx, "manager")

	adminUser := createInvoiceTestUser(t, tx, "adminuser", "管理员")
	managerUser := createInvoiceTestUser(t, tx, "mgruser", "经理")
	normalUser := createInvoiceTestUser(t, tx, "normaluser", "普通用户")

	assignRole(t, tx, adminUser.ID, adminRole.ID)
	assignRole(t, tx, managerUser.ID, managerRole.ID)

	// manager 仅可管理本部门发票，普通用户与经理同部门以便报销
	tx.Model(&normalUser).Update("department", "财务部")
	tx.Model(&managerUser).Update("department", "财务部")

	handler := NewHandler(tx)
	noAuthRouter := newInvoiceTestRouterNoAuth(t, handler)
	authRouter := newInvoiceTestRouter(t, handler)

	// 1. 普通用户创建草稿
	payload := map[string]interface{}{
		"invoice_no": "INV-SM-001", "invoice_date": time.Now().Format(time.RFC3339),
		"seller": "测试公司", "amount": 100.0, "total_amount": 113.0,
		"tax_amount": 13.0,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, normalUser.ID)
	w := httptest.NewRecorder()
	noAuthRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("创建草稿失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Item models.Invoice `json:"item"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	invID := createResp.Item.ID
	if invID == 0 {
		t.Fatal("创建发票后未返回有效 ID")
	}

	// 2. 提交草稿
	req = httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/submit", invID), nil)
	req = setAuthContext(req, normalUser.ID)
	w = httptest.NewRecorder()
	noAuthRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("提交发票失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	// 3. admin 审批通过
	approveBody, _ := json.Marshal(map[string]string{"approval_remark": "同意"})
	req = httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/approve", invID), jsonReader(approveBody))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, adminUser.ID)
	w = httptest.NewRecorder()
	authRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("审批通过失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	// 4. manager 执行报销
	reimburseBody, _ := json.Marshal(map[string]float64{"reimburse_amount": 113.0})
	req = httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/reimburse", invID), jsonReader(reimburseBody))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, managerUser.ID)
	w = httptest.NewRecorder()
	authRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("报销操作失败: %d, 响应: %s", w.Code, w.Body.String())
	}

	// 5. 验证最终状态
	var finalInv models.Invoice
	tx.First(&finalInv, invID)
	if finalInv.Status != models.InvoiceStatusReimbursed {
		t.Errorf("期望最终状态 reimbursed，实际 %s", finalInv.Status)
	}
	if finalInv.ReimburseAmount != 113.0 {
		t.Errorf("期望报销金额 113.0，实际 %.2f", finalInv.ReimburseAmount)
	}
}
