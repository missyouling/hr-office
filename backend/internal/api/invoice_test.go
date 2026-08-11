package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/middleware"
	"siapp/internal/models"
)

// ======== 测试辅助函数 ========

// newInvoiceTestRouterNoAuth 创建无权限中间件的发票路由（用于 CRUD 功能测试）
func newInvoiceTestRouterNoAuth(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/invoices", func(ir chi.Router) {
		ir.Get("/", handler.listInvoices)
		ir.Post("/", handler.createInvoice)
		ir.Get("/stats", handler.invoiceStats)
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/", handler.getInvoice)
			sr.Put("/", handler.updateInvoice)
			sr.Delete("/", handler.deleteInvoice)
			sr.Post("/submit", handler.submitInvoice)
			sr.Post("/approve", handler.approveInvoice)
			sr.Post("/reject", handler.rejectInvoice)
			sr.Post("/reimburse", handler.reimburseInvoice)
		})
	})
	return r
}

// newInvoiceTestRouter 创建带权限中间件的发票路由（用于权限测试）
func newInvoiceTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/invoices", func(ir chi.Router) {
		ir.Post("/", handler.createInvoice)
		ir.Group(func(mgr chi.Router) {
			mgr.Use(middleware.RequireManagerOrAbove(handler.db))
			mgr.Get("/", handler.listInvoices)
			mgr.Get("/stats", handler.invoiceStats)
		})
		ir.Route("/{id}", func(sr chi.Router) {
			sr.Get("/", handler.getInvoice)
			sr.Put("/", handler.updateInvoice)
			sr.Delete("/", handler.deleteInvoice)
			sr.Post("/submit", handler.submitInvoice)
			sr.Group(func(adm chi.Router) {
				adm.Use(middleware.RequireAdmin(handler.db))
				adm.Post("/approve", handler.approveInvoice)
				adm.Post("/reject", handler.rejectInvoice)
			})
			sr.With(middleware.RequireManagerOrAbove(handler.db)).
				Post("/reimburse", handler.reimburseInvoice)
		})
	})
	return r
}

// newInvoiceTestRouter 创建带权限中间件的发票路由（用于权限测试）
func createInvoiceTestUser(t *testing.T, tx *gorm.DB, username string, fullName string) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Email:    username + "@invoice-test.local",
		Password: "placeholder",
		FullName: fullName,
		Active:   true,
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

// createInvoiceTestRole 创建测试角色
func createInvoiceTestRole(t *testing.T, tx *gorm.DB, name string) models.Role {
	t.Helper()
	role := models.Role{Name: name, IsSystem: true}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}
	return role
}

// assignRole 给用户分配角色
func assignRole(t *testing.T, tx *gorm.DB, userID uint, roleID uint) {
	t.Helper()
	ur := models.UserRole{UserID: userID, RoleID: roleID}
	if err := tx.Create(&ur).Error; err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
}

// migrateInvoiceTables 迁移发票测试所需的表
func migrateInvoiceTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.Invoice{},
	); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

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

func TestCreateInvoice_UniqueInvoiceNo(t *testing.T) {
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

	// 尝试创建相同发票号
	payload := map[string]interface{}{
		"invoice_no": "INV-DUP", "seller": "测试公司", "amount": 100.0,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/invoices", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("重复发票号应返回 409，实际: %d, 响应: %s", w.Code, w.Body.String())
	}
}

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

// jsonReader 将字节转为 io.Reader（辅助函数）
func jsonReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
