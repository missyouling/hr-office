package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

// newWorkbenchRemindersTestRouter 注册工作台提醒路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newWorkbenchRemindersTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/workbench-reminders", handler.getWorkbenchReminders)
	})
	return r
}

// migrateWorkbenchReminderTables 迁移工作台提醒测试所需表
func migrateWorkbenchReminderTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(
		&models.User{},
		&models.SysFile{},
		&models.Document{},
		&models.DormBill{},
		&models.Invoice{},
		&models.InvoiceItem{},
		&models.OfficePaymentRequest{},
		&models.AdminContract{}, // 行政合同到期提醒（P12.3.5）
	); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildRemindersRequest 构造工作台提醒请求；auth=true 时写入登录用户上下文
func buildRemindersRequest(t *testing.T, path string, auth bool, userID uint) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if auth {
		req = setAuthContext(req, userID)
	}
	return req
}

// decodeRemindersResponse 解析响应体为 workbenchRemindersResponse
func decodeRemindersResponse(t *testing.T, resp *httptest.ResponseRecorder) workbenchRemindersResponse {
	t.Helper()
	var out workbenchRemindersResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return out
}

// seedWorkbenchReminderData 构造用户 A 的四类提醒测试数据。
// 应返回：档案 active(3天后到期)、账单 pending(5天后到期)、发票 submitted+approved、请款 submitted，共 5 条。
// 不应返回：已过期/归档档案、已付账单、草稿/已报销发票、草稿请款。
func seedWorkbenchReminderData(t *testing.T, tx *gorm.DB, user models.User) {
	t.Helper()
	uid := user.ID

	exp := time.Now().AddDate(0, 0, 3)
	doc := models.Document{UserID: uid, DocumentCode: "WS-01-2026-001", FileName: "劳动合同", ExpirationDate: &exp, Status: "active"}
	if err := tx.Create(&doc).Error; err != nil {
		t.Fatalf("创建档案失败: %v", err)
	}

	expired := time.Now().AddDate(0, 0, -5)
	docExpired := models.Document{UserID: uid, DocumentCode: "WS-01-2026-002", FileName: "已过期合同", ExpirationDate: &expired, Status: "active"}
	if err := tx.Create(&docExpired).Error; err != nil {
		t.Fatalf("创建已过期档案失败: %v", err)
	}

	expFuture := time.Now().AddDate(0, 0, 10)
	docArchived := models.Document{UserID: uid, DocumentCode: "WS-01-2026-003", FileName: "已归档合同", ExpirationDate: &expFuture, Status: "archived"}
	if err := tx.Create(&docArchived).Error; err != nil {
		t.Fatalf("创建归档档案失败: %v", err)
	}

	due := time.Now().AddDate(0, 0, 5)
	bill := models.DormBill{UserID: &uid, BillCode: "BILL-001", PeriodLabel: "2026-08", DueDate: due, Status: "pending"}
	if err := tx.Create(&bill).Error; err != nil {
		t.Fatalf("创建账单失败: %v", err)
	}

	duePaid := time.Now().AddDate(0, 0, 7)
	billPaid := models.DormBill{UserID: &uid, BillCode: "BILL-002", PeriodLabel: "2026-08", DueDate: duePaid, Status: "paid"}
	if err := tx.Create(&billPaid).Error; err != nil {
		t.Fatalf("创建已付账单失败: %v", err)
	}

	invSubmitted := models.Invoice{UserID: &uid, InvoiceNo: "INV-001", InvoiceDate: time.Now(), Seller: "测试公司", Amount: 100, TotalAmount: 113, Status: models.InvoiceStatusSubmitted}
	if err := tx.Create(&invSubmitted).Error; err != nil {
		t.Fatalf("创建发票失败: %v", err)
	}

	invApproved := models.Invoice{UserID: &uid, InvoiceNo: "INV-002", InvoiceDate: time.Now(), Seller: "测试公司B", Amount: 200, TotalAmount: 226, Status: models.InvoiceStatusApproved}
	if err := tx.Create(&invApproved).Error; err != nil {
		t.Fatalf("创建已审批发票失败: %v", err)
	}

	invReimbursed := models.Invoice{UserID: &uid, InvoiceNo: "INV-003", InvoiceDate: time.Now(), Seller: "测试公司C", Amount: 50, TotalAmount: 56, Status: models.InvoiceStatusReimbursed}
	if err := tx.Create(&invReimbursed).Error; err != nil {
		t.Fatalf("创建已报销发票失败: %v", err)
	}

	invDraft := models.Invoice{UserID: &uid, InvoiceNo: "INV-004", InvoiceDate: time.Now(), Seller: "测试公司D", Amount: 10, TotalAmount: 11, Status: models.InvoiceStatusDraft}
	if err := tx.Create(&invDraft).Error; err != nil {
		t.Fatalf("创建草稿发票失败: %v", err)
	}

	prSubmitted := models.OfficePaymentRequest{UserID: &uid, RequestNo: "PR-20260814-0001", PaymentUnit: "测试单位", RequestDate: time.Now(), Status: "submitted"}
	if err := tx.Create(&prSubmitted).Error; err != nil {
		t.Fatalf("创建请款单失败: %v", err)
	}

	prDraft := models.OfficePaymentRequest{UserID: &uid, RequestNo: "PR-20260814-0002", PaymentUnit: "测试单位B", RequestDate: time.Now(), Status: "draft"}
	if err := tx.Create(&prDraft).Error; err != nil {
		t.Fatalf("创建草稿请款单失败: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GET：鉴权 / 非法 days / 正常返回 / 敏感字段 / 空数组 / 用户隔离
// ---------------------------------------------------------------------------

func TestWorkbenchReminders_Get(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchReminderTables(t, tx)

	user := createTestUser(t, tx, "wbreminder_user", "提醒用户")
	handler := NewHandler(tx)
	router := newWorkbenchRemindersTestRouter(t, handler)
	seedWorkbenchReminderData(t, tx, user)

	t.Run("未鉴权返回401", func(t *testing.T) {
		req := buildRemindersRequest(t, "/api/user/workbench-reminders", false, 0)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", resp.Code)
		}
	})

	t.Run("非法days返回400", func(t *testing.T) {
		for _, raw := range []string{"0", "-1", "abc", "366", "30.5"} {
			req := buildRemindersRequest(t, "/api/user/workbench-reminders?days="+raw, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("days=%q 期望 400，实际 %d，body=%s", raw, resp.Code, resp.Body.String())
			}
		}
	})

	t.Run("缺省days返回默认30", func(t *testing.T) {
		req := buildRemindersRequest(t, "/api/user/workbench-reminders", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		out := decodeRemindersResponse(t, resp)
		if out.Days != 30 {
			t.Fatalf("期望 days=30，实际 %d", out.Days)
		}
	})

	t.Run("四类提醒统一返回且排序稳定", func(t *testing.T) {
		req := buildRemindersRequest(t, "/api/user/workbench-reminders?days=30", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		out := decodeRemindersResponse(t, resp)
		if len(out.Items) != 5 {
			t.Fatalf("期望 5 条提醒，实际 %d: %+v", len(out.Items), out.Items)
		}
		// 稳定排序：档案(3天后到期) → 账单(5天后到期) → 发票/请款(null 排最后，按 type 字典序)
		wantTypes := []string{
			reminderTypeDocumentExpiration,
			reminderTypeDormBillDue,
			reminderTypeInvoicePending,
			reminderTypeInvoicePending,
			reminderTypePaymentPending,
		}
		for i, want := range wantTypes {
			if out.Items[i].ReminderType != want {
				t.Fatalf("第 %d 项类型应为 %s，实际 %s（%+v）", i, want, out.Items[i].ReminderType, out.Items)
			}
		}
		// due_at：档案/账单非空，发票/请款为 null
		for i, item := range out.Items {
			switch item.ReminderType {
			case reminderTypeDocumentExpiration, reminderTypeDormBillDue:
				if item.DueAt == nil {
					t.Fatalf("第 %d 项 %s 的 due_at 不应为 null", i, item.ReminderType)
				}
			case reminderTypeInvoicePending, reminderTypePaymentPending:
				if item.DueAt != nil {
					t.Fatalf("第 %d 项 %s 的 due_at 应为 null", i, item.ReminderType)
				}
			}
		}
		// 去重：同一 (reminder_type, id) 不重复
		seen := make(map[string]bool)
		for _, item := range out.Items {
			key := fmt.Sprintf("%s:%d", item.ReminderType, item.ID)
			if seen[key] {
				t.Fatalf("存在重复提醒: %s", key)
			}
			seen[key] = true
		}
		// 标题与状态正确性抽查
		if out.Items[0].Title != "WS-01-2026-001" || out.Items[0].Status != "active" {
			t.Fatalf("档案提醒标题/状态不正确: %+v", out.Items[0])
		}
		if out.Items[1].Title != "BILL-001" || out.Items[1].Status != "pending" {
			t.Fatalf("账单提醒标题/状态不正确: %+v", out.Items[1])
		}
		if out.Items[2].Title != "INV-001" || out.Items[2].Status != models.InvoiceStatusSubmitted {
			t.Fatalf("发票提醒标题/状态不正确: %+v", out.Items[2])
		}
		if out.Items[4].Title != "PR-20260814-0001" || out.Items[4].Status != "submitted" {
			t.Fatalf("请款提醒标题/状态不正确: %+v", out.Items[4])
		}
	})

	t.Run("不暴露敏感字段", func(t *testing.T) {
		req := buildRemindersRequest(t, "/api/user/workbench-reminders?days=30", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		body := resp.Body.String()
		for _, forbidden := range []string{"total_amount", "bank_account", "seller_tax_no", "identity", "id_number"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("响应不应包含敏感字段 %s: %s", forbidden, body)
			}
		}
	})
}

func TestWorkbenchReminders_Empty(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchReminderTables(t, tx)

	user := createTestUser(t, tx, "wbreminder_empty", "空用户")
	handler := NewHandler(tx)
	router := newWorkbenchRemindersTestRouter(t, handler)

	req := buildRemindersRequest(t, "/api/user/workbench-reminders", true, user.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", resp.Code)
	}
	if strings.Contains(resp.Body.String(), `"items":null`) {
		t.Fatalf("items 不应为 null: %s", resp.Body.String())
	}
	out := decodeRemindersResponse(t, resp)
	if out.Items == nil || len(out.Items) != 0 {
		t.Fatalf("期望空数组，实际 %+v", out.Items)
	}
}

func TestWorkbenchReminders_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchReminderTables(t, tx)

	userA := createTestUser(t, tx, "wbreminder_a", "用户A")
	userB := createTestUser(t, tx, "wbreminder_b", "用户B")
	handler := NewHandler(tx)
	router := newWorkbenchRemindersTestRouter(t, handler)
	seedWorkbenchReminderData(t, tx, userA)

	// B 查询应看不到 A 的任何提醒
	req := buildRemindersRequest(t, "/api/user/workbench-reminders", true, userB.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", resp.Code)
	}
	out := decodeRemindersResponse(t, resp)
	if len(out.Items) != 0 {
		t.Fatalf("B 不应看到 A 的提醒，实际 %d 条: %+v", len(out.Items), out.Items)
	}

	// A 查询仍能正常看到自己的提醒
	req = buildRemindersRequest(t, "/api/user/workbench-reminders", true, userA.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	out = decodeRemindersResponse(t, resp)
	if len(out.Items) != 5 {
		t.Fatalf("A 应看到 5 条提醒，实际 %d 条", len(out.Items))
	}
}

// seedAdminContractReminderData 构造行政合同到期提醒测试数据（P12.3.5）：
// 应返回：active 且 5 天后到期 1 条；不应返回：草稿/已作废/已过期。
func seedAdminContractReminderData(t *testing.T, tx *gorm.DB, userID uint) {
	t.Helper()

	// active 且 5 天后到期 → 应出现
	soon := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	active := models.AdminContract{
		UserID: userID, ContractNo: "XZ-WB-001", Name: "保洁服务合同", Counterparty: "某某公司",
		ContractType: "服务合同", StartDate: "2026-01-01", EndDate: soon,
		Status: models.AdminContractStatusActive,
	}
	if err := tx.Create(&active).Error; err != nil {
		t.Fatalf("创建生效行政合同失败: %v", err)
	}

	// draft → 不应出现
	draft := models.AdminContract{
		UserID: userID, ContractNo: "XZ-WB-002", Name: "草稿合同", Counterparty: "某某公司",
		ContractType: "服务合同", StartDate: "2026-01-01", EndDate: soon,
		Status: models.AdminContractStatusDraft,
	}
	if err := tx.Create(&draft).Error; err != nil {
		t.Fatalf("创建草稿行政合同失败: %v", err)
	}

	// cancelled → 不应出现
	cancelled := models.AdminContract{
		UserID: userID, ContractNo: "XZ-WB-003", Name: "已作废合同", Counterparty: "某某公司",
		ContractType: "服务合同", StartDate: "2026-01-01", EndDate: soon,
		Status: models.AdminContractStatusCancelled,
	}
	if err := tx.Create(&cancelled).Error; err != nil {
		t.Fatalf("创建已作废行政合同失败: %v", err)
	}

	// expired（end_date 过去）→ 不应出现
	past := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	expired := models.AdminContract{
		UserID: userID, ContractNo: "XZ-WB-004", Name: "已到期合同", Counterparty: "某某公司",
		ContractType: "服务合同", StartDate: "2025-01-01", EndDate: past,
		Status: models.AdminContractStatusExpired,
	}
	if err := tx.Create(&expired).Error; err != nil {
		t.Fatalf("创建已到期行政合同失败: %v", err)
	}
}

// TestWorkbenchReminders_AdminContractExpiring 工作台统一提醒（P12.3.5）：
// active 且未来 30 天内到期的行政合同以 admin_contract_expiring 类型出现；
// 草稿/已作废/已到期不出现；due_at 为到期日；用户隔离生效。
func TestWorkbenchReminders_AdminContractExpiring(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchReminderTables(t, tx)

	userA := createTestUser(t, tx, "wbreminder_ac_a", "行政合同用户A")
	userB := createTestUser(t, tx, "wbreminder_ac_b", "行政合同用户B")
	handler := NewHandler(tx)
	router := newWorkbenchRemindersTestRouter(t, handler)
	seedAdminContractReminderData(t, tx, userA.ID)

	// A 查询：仅 1 条 active 提醒，due_at 为到期日
	req := buildRemindersRequest(t, "/api/user/workbench-reminders?days=30", true, userA.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
	}
	out := decodeRemindersResponse(t, resp)
	if len(out.Items) != 1 {
		t.Fatalf("期望 1 条行政合同提醒，实际 %d 条: %+v", len(out.Items), out.Items)
	}
	item := out.Items[0]
	if item.ReminderType != reminderTypeAdminContractExpiring {
		t.Fatalf("提醒类型应为 %s，实际 %s", reminderTypeAdminContractExpiring, item.ReminderType)
	}
	if item.Title != "XZ-WB-001" {
		t.Fatalf("提醒标题应为合同编号 XZ-WB-001，实际 %s", item.Title)
	}
	if item.Status != models.AdminContractStatusActive {
		t.Fatalf("提醒状态应为 active，实际 %s", item.Status)
	}
	if item.DueAt == nil {
		t.Fatalf("行政合同提醒的 due_at 不应为 null")
	}
	want := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if item.DueAt.Format("2006-01-02") != want {
		t.Fatalf("due_at 应为 %s，实际 %s", want, item.DueAt.Format("2006-01-02"))
	}

	// B 查询：看不到 A 的提醒
	req = buildRemindersRequest(t, "/api/user/workbench-reminders", true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", resp.Code)
	}
	out = decodeRemindersResponse(t, resp)
	if len(out.Items) != 0 {
		t.Fatalf("B 不应看到 A 的行政合同提醒，实际 %d 条: %+v", len(out.Items), out.Items)
	}
}
