package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ======== 资源级授权：单条详情 ========

func TestInvoiceAccess_DetailAuthorization(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	managerRole := createInvoiceTestRole(t, tx, "manager")

	admin := createInvoiceTestUser(t, tx, "admin-access", "管理员")
	manager := createInvoiceTestUser(t, tx, "manager-access", "无部门经理")
	owner := createInvoiceTestUser(t, tx, "owner-access", "发票本人")
	other := createInvoiceTestUser(t, tx, "other-access", "他人")
	managerDept := createInvoiceTestUser(t, tx, "mgrdpt-access", "同部门经理")

	assignRole(t, tx, admin.ID, adminRole.ID)
	assignRole(t, tx, manager.ID, managerRole.ID)
	assignRole(t, tx, managerDept.ID, managerRole.ID)

	// 部门划分：owner 与 managerDept 同部门；manager/other 无部门
	tx.Model(&owner).Update("department", "财务部")
	tx.Model(&managerDept).Update("department", "财务部")

	uid := owner.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "ACCESS-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)
	path := fmt.Sprintf("/invoices/%d", invoice.ID)

	tests := []struct {
		name string
		user uint
		want int
	}{
		{"本人", owner.ID, http.StatusOK},
		{"他人普通用户", other.ID, http.StatusNotFound},
		{"无部门经理", manager.ID, http.StatusNotFound},
		{"同部门经理", managerDept.ID, http.StatusOK},
		{"管理员", admin.ID, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req = setAuthContext(req, tt.user)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("期望 %d，实际 %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

// ======== UserID=nil 安全拒绝普通用户 ========

func TestInvoiceAccess_UserIDNilRejectedForNonAdmin(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "admin-nil", "管理员")
	normal := createInvoiceTestUser(t, tx, "normal-nil", "普通用户")
	assignRole(t, tx, admin.ID, adminRole.ID)

	// UserID=nil 的发票（系统遗留/孤儿记录）
	invoice := models.Invoice{InvoiceNo: "NIL-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)
	path := fmt.Sprintf("/invoices/%d", invoice.ID)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = setAuthContext(req, normal.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("普通用户访问 UserID=nil 发票应拒绝: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, path, nil)
	req = setAuthContext(req, admin.ID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("admin 应可访问 UserID=nil 发票: %d", w.Code)
	}
}

// ======== 软删除默认不可见 ========

func TestInvoiceAccess_SoftDeletedInvisible(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "admin-del", "管理员")
	owner := createInvoiceTestUser(t, tx, "owner-del", "本人")
	assignRole(t, tx, admin.ID, adminRole.ID)

	uid := owner.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "DEL-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Delete(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(tx)
	router := newInvoiceTestRouterNoAuth(t, handler)
	path := fmt.Sprintf("/invoices/%d", invoice.ID)

	// 本人与 admin 均不可见（GORM 默认过滤软删 + 授权兜底）
	for _, user := range []uint{owner.ID, admin.ID} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = setAuthContext(req, user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("软删除发票对用户 %d 应不可见: %d", user, w.Code)
		}
	}
}

// ======== 采购关联校验 ========

func TestInvoiceSource_Validation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	adminRole := createInvoiceTestRole(t, tx, "admin")
	admin := createInvoiceTestUser(t, tx, "admin-src", "管理员")
	owner := createInvoiceTestUser(t, tx, "owner-src", "采购本人")
	other := createInvoiceTestUser(t, tx, "other-src", "他人")
	assignRole(t, tx, admin.ID, adminRole.ID)

	oid := owner.ID
	purchase := models.OfficePurchase{UserID: &oid, OrderNo: "PO-001", PurchaseDate: time.Now(), TotalAmount: 100}
	if err := tx.Create(&purchase).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(tx)

	// 合法：本人采购单
	if err := handler.validateInvoiceSource(tx, owner.ID, models.InvoiceSourceOffice, &purchase.ID); err != nil {
		t.Errorf("本人采购单应通过: %v", err)
	}
	// 越权：他人采购单（普通用户）
	if err := handler.validateInvoiceSource(tx, other.ID, models.InvoiceSourceOffice, &purchase.ID); !errors.Is(err, errInvoiceSourceForbidden) {
		t.Errorf("他人采购单应拒绝: %v", err)
	}
	// admin 全量
	if err := handler.validateInvoiceSource(tx, admin.ID, models.InvoiceSourceOffice, &purchase.ID); err != nil {
		t.Errorf("admin 应可通过: %v", err)
	}
	// 不存在
	missing := uint(99999)
	if err := handler.validateInvoiceSource(tx, owner.ID, models.InvoiceSourceOffice, &missing); !errors.Is(err, errInvoiceSourceMissing) {
		t.Errorf("不存在的采购单应报缺失: %v", err)
	}
	// 非法类型
	if err := handler.validateInvoiceSource(tx, owner.ID, "unknown_type", &purchase.ID); !errors.Is(err, errInvoiceSourceInvalid) {
		t.Errorf("非法类型应拒绝: %v", err)
	}
	// 独立发票允许无关联
	if err := handler.validateInvoiceSource(tx, owner.ID, models.InvoiceSourceIndependent, nil); err != nil {
		t.Errorf("独立发票应通过: %v", err)
	}
	// 类型非空但 ID 为空
	if err := handler.validateInvoiceSource(tx, owner.ID, models.InvoiceSourceOffice, nil); !errors.Is(err, errInvoiceSourceInvalid) {
		t.Errorf("类型非空但无 ID 应拒绝: %v", err)
	}
}

// ======== 事务内审计：失败整体回滚 ========

func TestAuditLog_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateInvoiceTables(t, tx)

	user := createInvoiceTestUser(t, tx, "audit-user", "审计用户")
	uid := user.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "AUDIT-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}

	// 事务内：业务更新 + 审计写入，随后返回错误 → 整体回滚
	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Model(&invoice).Update("status", models.InvoiceStatusSubmitted).Error; err != nil {
			return err
		}
		rid := strconv.FormatUint(uint64(invoice.ID), 10)
		if err := models.CreateAuditLogWithDB(t, models.CreateAuditLogParams{
			UserID: &uid, Action: "SUBMIT_INVOICE", Resource: "invoices", ResourceID: &rid,
			Status: models.StatusSuccess, StatusCode: http.StatusOK,
		}); err != nil {
			return err
		}
		return errors.New("模拟后续失败")
	})
	if err == nil {
		t.Fatal("事务应因模拟失败而回滚")
	}

	var saved models.Invoice
	if err := tx.First(&saved, invoice.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != models.InvoiceStatusDraft {
		t.Errorf("业务更新未随事务回滚: %s", saved.Status)
	}
	var auditCount int64
	tx.Model(&models.AuditLog{}).Count(&auditCount)
	if auditCount != 0 {
		t.Errorf("审计写入未随事务回滚: %d", auditCount)
	}

	// 独立事务成功路径：审计可正常写入
	rid := strconv.FormatUint(uint64(invoice.ID), 10)
	if err := models.CreateAuditLogWithDB(tx, models.CreateAuditLogParams{
		UserID: &uid, Action: "SUBMIT_INVOICE", Resource: "invoices", ResourceID: &rid,
		Status: models.StatusSuccess, StatusCode: http.StatusOK,
	}); err != nil {
		t.Fatalf("事务外审计写入失败: %v", err)
	}
}
