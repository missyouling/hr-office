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

// ======== 资源范围：列表 / 统计 ========

func TestInvoiceScope_ListAndStats(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	managerRole := createInvoiceTestRole(t, tx, "manager")
	admin := createInvoiceTestUser(t, tx, "scope-admin", "管理员")
	mgrFinance := createInvoiceTestUser(t, tx, "scope-mgr-fin", "财务经理")
	mgrTech := createInvoiceTestUser(t, tx, "scope-mgr-tech", "技术经理")
	userFinance := createInvoiceTestUser(t, tx, "scope-user-fin", "财务用户")
	userTech := createInvoiceTestUser(t, tx, "scope-user-tech", "技术用户")
	assignRole(t, tx, admin.ID, adminRole.ID)
	assignRole(t, tx, mgrFinance.ID, managerRole.ID)
	assignRole(t, tx, mgrTech.ID, managerRole.ID)
	tx.Model(&mgrFinance).Update("department", "财务部")
	tx.Model(&userFinance).Update("department", "财务部")
	tx.Model(&mgrTech).Update("department", "技术部")
	tx.Model(&userTech).Update("department", "技术部")

	create := func(user models.User, no string) {
		uid := user.ID
		if err := tx.Create(&models.Invoice{UserID: &uid, InvoiceNo: no, InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}).Error; err != nil {
			t.Fatal(err)
		}
	}
	create(mgrFinance, "FIN-1")
	create(userFinance, "FIN-2")
	create(mgrTech, "TECH-1")
	create(userTech, "TECH-2")
	create(admin, "ADM-1")

	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)

	list := func(user uint) (int64, []string) {
		req := httptest.NewRequest("GET", "/invoices?page_size=100", nil)
		req = setAuthContext(req, user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("列表请求失败: %d", w.Code)
		}
		var resp struct {
			Items []models.Invoice `json:"items"`
			Total int64            `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		codes := make([]string, 0, len(resp.Items))
		for _, inv := range resp.Items {
			codes = append(codes, inv.InvoiceNo)
		}
		return resp.Total, codes
	}
	stats := func(user uint) int64 {
		req := httptest.NewRequest("GET", "/invoices/stats", nil)
		req = setAuthContext(req, user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var resp struct {
			TotalCount int64 `json:"total_count"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.TotalCount
	}

	// admin 全量
	if total, _ := list(admin.ID); total != 5 {
		t.Errorf("admin 列表应全量 5 条，实际 %d", total)
	}
	// 财务经理：仅本部门（FIN-1、FIN-2，顺序为 created_at DESC）
	if total, codes := list(mgrFinance.ID); total != 2 || !sameStringSet(codes, []string{"FIN-1", "FIN-2"}) {
		t.Errorf("财务经理列表应仅本部门 2 条: total=%d codes=%v", total, codes)
	}
	// 技术经理：仅本部门
	if total, _ := list(mgrTech.ID); total != 2 {
		t.Errorf("技术经理列表应仅本部门 2 条，实际 %d", total)
	}
	// 普通用户：仅本人
	if total, codes := list(userFinance.ID); total != 1 || len(codes) != 1 || codes[0] != "FIN-2" {
		t.Errorf("普通用户列表应仅本人 1 条: total=%d codes=%v", total, codes)
	}
	// 统计同范围
	if got := stats(admin.ID); got != 5 {
		t.Errorf("admin 统计应 5，实际 %d", got)
	}
	if got := stats(mgrFinance.ID); got != 2 {
		t.Errorf("财务经理统计应 2，实际 %d", got)
	}
	if got := stats(userFinance.ID); got != 1 {
		t.Errorf("普通用户统计应 1，实际 %d", got)
	}

	// 软删不可见：删除 FIN-2 后财务经理列表/统计均减少
	var fin2 models.Invoice
	if err := tx.Where("invoice_no = ?", "FIN-2").First(&fin2).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Delete(&fin2).Error; err != nil {
		t.Fatal(err)
	}
	if total, _ := list(mgrFinance.ID); total != 1 {
		t.Errorf("软删后财务经理列表应 1，实际 %d", total)
	}
	if got := stats(mgrFinance.ID); got != 1 {
		t.Errorf("软删后财务经理统计应 1，实际 %d", got)
	}
}

// sameStringSet 判断两个字符串集合是否相等（忽略顺序）。
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, value := range a {
		counts[value]++
	}
	for _, value := range b {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

// ======== 资源范围：CSV 导出 ========

func TestInvoiceScope_Export(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	managerRole := createInvoiceTestRole(t, tx, "manager")
	admin := createInvoiceTestUser(t, tx, "exp-admin", "管理员")
	mgrFinance := createInvoiceTestUser(t, tx, "exp-mgr", "财务经理")
	userFinance := createInvoiceTestUser(t, tx, "exp-user-fin", "财务用户")
	userTech := createInvoiceTestUser(t, tx, "exp-user-tech", "技术用户")
	normal := createInvoiceTestUser(t, tx, "exp-normal", "普通用户")
	assignRole(t, tx, admin.ID, adminRole.ID)
	assignRole(t, tx, mgrFinance.ID, managerRole.ID)
	tx.Model(&mgrFinance).Update("department", "财务部")
	tx.Model(&userFinance).Update("department", "财务部")

	create := func(user models.User, no string) {
		uid := user.ID
		tx.Create(&models.Invoice{UserID: &uid, InvoiceNo: no, InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft})
	}
	create(userFinance, "EXP-FIN")
	create(userTech, "EXP-TECH")

	handler := NewHandler(tx)
	router := newInvoiceTestRouter(t, handler)

	export := func(user uint) (int, string) {
		req := httptest.NewRequest("GET", "/invoices/export", nil)
		req = setAuthContext(req, user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}

	// 普通用户：导出路由要求 manager+ → 403
	if code, _ := export(normal.ID); code != http.StatusForbidden {
		t.Errorf("普通用户导出应 403，实际 %d", code)
	}
	// manager 本部门：仅含 EXP-FIN
	if code, body := export(mgrFinance.ID); code != http.StatusOK || !strings.Contains(body, "EXP-FIN") || strings.Contains(body, "EXP-TECH") {
		t.Errorf("财务经理导出应仅含本部门: code=%d body=%s", code, body)
	}
	// admin 全量
	if code, body := export(admin.ID); code != http.StatusOK || !strings.Contains(body, "EXP-TECH") {
		t.Errorf("admin 导出应全量: code=%d", code)
	}
}

// ======== 资源范围：附件访问 ========

func TestInvoiceScope_AttachmentAccess(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	managerRole := createInvoiceTestRole(t, tx, "manager")
	admin := createInvoiceTestUser(t, tx, "att-admin", "管理员")
	mgrFinance := createInvoiceTestUser(t, tx, "att-mgr-fin", "财务经理")
	mgrTech := createInvoiceTestUser(t, tx, "att-mgr-tech", "技术经理")
	owner := createInvoiceTestUser(t, tx, "att-owner", "本人")
	otherFinance := createInvoiceTestUser(t, tx, "att-other", "同部门他人")
	assignRole(t, tx, admin.ID, adminRole.ID)
	assignRole(t, tx, mgrFinance.ID, managerRole.ID)
	assignRole(t, tx, mgrTech.ID, managerRole.ID)
	tx.Model(&mgrFinance).Update("department", "财务部")
	tx.Model(&owner).Update("department", "财务部")
	tx.Model(&otherFinance).Update("department", "财务部")

	uid := owner.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "ATT-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	tests := []struct {
		name string
		user uint
		want bool
	}{
		{"本人", owner.ID, true},
		{"同部门普通用户", otherFinance.ID, false},
		{"同部门经理", mgrFinance.ID, true},
		{"不同部门经理", mgrTech.ID, false},
		{"管理员", admin.ID, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.canAccessInvoiceAttachment(tt.user, &invoice); got != tt.want {
				t.Errorf("附件访问期望 %v，实际 %v", tt.want, got)
			}
		})
	}

	// 软删不可见
	if err := tx.Delete(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	if handler.canAccessInvoiceAttachment(admin.ID, &invoice) {
		t.Error("软删发票附件对 admin 也应不可见")
	}

	// 附件 HTTP 路由越权：无存储时返回 404 不泄露存在性
	var stored models.Invoice
	stored = models.Invoice{UserID: &uid, InvoiceNo: "ATT-002", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&stored).Error; err != nil {
		t.Fatal(err)
	}
	router := newInvoiceTestRouterNoAuth(t, handler)
	req := httptest.NewRequest("GET", fmt.Sprintf("/invoices/%d/attachment", stored.ID), nil)
	req = setAuthContext(req, mgrTech.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("不同部门经理访问附件应 404（不泄露存在性），实际 %d", w.Code)
	}
}
