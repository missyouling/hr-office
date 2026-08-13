package api

import (
	"context"
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

// newInvoiceParsingTaskTestRouter 构建解析任务 API 测试路由（无权限中间件，鉴权用 setAuthContext 模拟）。
func newInvoiceParsingTaskTestRouter(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Route("/invoices/{id}", func(sr chi.Router) {
		sr.Get("/parsing-task", handler.getInvoiceParsingTask)
		sr.Post("/parsing-task/retry", handler.retryInvoiceParsingTask)
	})
	return r
}

// setupInvoiceParsingTaskTest 初始化解析任务 API 测试环境。
func setupInvoiceParsingTaskTest(t *testing.T) (*Handler, chi.Router, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	migrateInvoiceTables(t, db)
	handler := NewHandler(db)
	return handler, newInvoiceParsingTaskTestRouter(handler), db
}

// createParsingTaskInvoice 创建归属指定用户的测试发票及其解析任务。
func createParsingTaskInvoice(t *testing.T, db *gorm.DB, username string) (models.User, models.Invoice, models.InvoiceParsingTask) {
	t.Helper()
	user := createInvoiceTestUser(t, db, username, "解析任务用户")
	uid := user.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "PT-001", InvoiceDate: time.Now(), Amount: 100, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	task := models.InvoiceParsingTask{InvoiceID: invoice.ID}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	return user, invoice, task
}

// performParsingTaskRequest 构造带认证上下文的请求并执行。
func performParsingTaskRequest(router chi.Router, method, path string, userID uint) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if userID != 0 {
		req = setAuthContext(req, userID)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// decodeParsingTaskResponse 解析响应中的任务对象。
func decodeParsingTaskResponse(t *testing.T, body []byte) invoiceParsingTaskResponse {
	t.Helper()
	var payload struct {
		Item invoiceParsingTaskResponse `json:"item"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, body)
	}
	return payload.Item
}

// ======== 查询接口 GET /api/invoices/{id}/parsing-task ========

func TestInvoiceParsingTaskAPI_GetRequiresAuth(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	_, invoice, _ := createParsingTaskInvoice(t, db, "get-unauth")
	w := performParsingTaskRequest(router, http.MethodGet, fmt.Sprintf("/invoices/%d/parsing-task", invoice.ID), 0)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestInvoiceParsingTaskAPI_GetSuccess(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	user, invoice, task := createParsingTaskInvoice(t, db, "get-owner")
	// 预置失败任务数据，验证响应字段完整
	if err := db.Model(&models.InvoiceParsingTask{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":        models.InvoiceParsingTaskFailed,
		"attempt_count": 3,
		"max_attempts":  3,
		"error_code":    "no_recognizable_content",
		"last_error":    "no_recognizable_content",
		"locked_by":     "internal-worker-token",
		"locked_until":  time.Now().Add(time.Minute),
		"completed_at":  time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	w := performParsingTaskRequest(router, http.MethodGet, fmt.Sprintf("/invoices/%d/parsing-task", invoice.ID), user.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("查询失败: %d %s", w.Code, w.Body.String())
	}
	item := decodeParsingTaskResponse(t, w.Body.Bytes())
	if item.ID != task.ID || item.InvoiceID != invoice.ID || item.Status != string(models.InvoiceParsingTaskFailed) ||
		item.AttemptCount != 3 || item.MaxAttempts != 3 || item.ErrorCode != "no_recognizable_content" || item.LastError != "no_recognizable_content" {
		t.Fatalf("响应字段不完整: %+v", item)
	}
	// worker 内部调度字段不得泄露
	raw := string(w.Body.Bytes())
	for _, leaked := range []string{`"locked_by"`, `"locked_until"`, `"available_at"`} {
		if strings.Contains(raw, leaked) {
			t.Fatalf("响应不得泄露内部字段 %s: %s", leaked, raw)
		}
	}
}

func TestInvoiceParsingTaskAPI_GetForbiddenAndNoTask(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	owner, invoice, _ := createParsingTaskInvoice(t, db, "get-forbid")
	other := createInvoiceTestUser(t, db, "get-other", "他人")

	// 他人无权访问 → 404（与不存在统一，避免泄露存在性）
	w := performParsingTaskRequest(router, http.MethodGet, fmt.Sprintf("/invoices/%d/parsing-task", invoice.ID), other.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("他人发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}

	// 发票不存在 → 404
	w = performParsingTaskRequest(router, http.MethodGet, "/invoices/99999/parsing-task", owner.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("不存在的发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}

	// 发票存在但无解析任务 → 404
	uid := owner.ID
	plain := models.Invoice{UserID: &uid, InvoiceNo: "PT-NONE", InvoiceDate: time.Now(), Amount: 50, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := db.Create(&plain).Error; err != nil {
		t.Fatal(err)
	}
	w = performParsingTaskRequest(router, http.MethodGet, fmt.Sprintf("/invoices/%d/parsing-task", plain.ID), owner.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("无任务发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}
}

// ======== 重试接口 POST /api/invoices/{id}/parsing-task/retry ========

func TestInvoiceParsingTaskAPI_RetryRequiresAuthAndForbidden(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	_, invoice, task := createParsingTaskInvoice(t, db, "retry-auth")
	if err := db.Model(&models.InvoiceParsingTask{}).Where("id = ?", task.ID).Update("status", models.InvoiceParsingTaskFailed).Error; err != nil {
		t.Fatal(err)
	}
	other := createInvoiceTestUser(t, db, "retry-other", "他人")
	path := fmt.Sprintf("/invoices/%d/parsing-task/retry", invoice.ID)

	// 未登录 → 401
	if w := performParsingTaskRequest(router, http.MethodPost, path, 0); w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401，实际 %d: %s", w.Code, w.Body.String())
	}
	// 他人发票 → 404
	if w := performParsingTaskRequest(router, http.MethodPost, path, other.ID); w.Code != http.StatusNotFound {
		t.Fatalf("他人发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}
	// 发票不存在 → 404
	if w := performParsingTaskRequest(router, http.MethodPost, "/invoices/99999/parsing-task/retry", other.ID); w.Code != http.StatusNotFound {
		t.Fatalf("不存在的发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestInvoiceParsingTaskAPI_RetryInvalidState(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	for _, status := range []models.InvoiceParsingTaskStatus{
		models.InvoiceParsingTaskPending,
		models.InvoiceParsingTaskRunning,
		models.InvoiceParsingTaskSucceeded,
	} {
		user, invoice, task := createParsingTaskInvoice(t, db, "retry-state-"+string(status))
		if err := db.Model(&models.InvoiceParsingTask{}).Where("id = ?", task.ID).Update("status", status).Error; err != nil {
			t.Fatal(err)
		}
		w := performParsingTaskRequest(router, http.MethodPost, fmt.Sprintf("/invoices/%d/parsing-task/retry", invoice.ID), user.ID)
		if w.Code != http.StatusConflict {
			t.Fatalf("状态 %s 重试应返回 409，实际 %d: %s", status, w.Code, w.Body.String())
		}
		// 状态保持不变
		var stored models.InvoiceParsingTask
		if err := db.First(&stored, task.ID).Error; err != nil || stored.Status != status {
			t.Fatalf("非法状态重试不得改动任务: %+v %v", stored, err)
		}
	}
}

func TestInvoiceParsingTaskAPI_RetryNoTask(t *testing.T) {
	_, router, db := setupInvoiceParsingTaskTest(t)
	user := createInvoiceTestUser(t, db, "retry-notask", "无任务用户")
	uid := user.ID
	invoice := models.Invoice{UserID: &uid, InvoiceNo: "PT-NONE", InvoiceDate: time.Now(), Amount: 50, Seller: "测试", Status: models.InvoiceStatusDraft}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	w := performParsingTaskRequest(router, http.MethodPost, fmt.Sprintf("/invoices/%d/parsing-task/retry", invoice.ID), user.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("无任务发票应返回 404，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestInvoiceParsingTaskAPI_RetrySuccessAndClaimable(t *testing.T) {
	handler, router, db := setupInvoiceParsingTaskTest(t)
	user, invoice, task := createParsingTaskInvoice(t, db, "retry-owner")
	if err := db.Model(&models.InvoiceParsingTask{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":        models.InvoiceParsingTaskFailed,
		"attempt_count": 3,
		"max_attempts":  3,
		"available_at":  time.Now().Add(-time.Hour),
		"locked_by":     "stale-token",
		"locked_until":  time.Now().Add(time.Hour),
		"error_code":    "worker_lease_expired",
		"last_error":    "worker_lease_expired",
		"completed_at":  time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	w := performParsingTaskRequest(router, http.MethodPost, fmt.Sprintf("/invoices/%d/parsing-task/retry", invoice.ID), user.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("重试失败: %d %s", w.Code, w.Body.String())
	}
	item := decodeParsingTaskResponse(t, w.Body.Bytes())
	if item.Status != string(models.InvoiceParsingTaskPending) || item.AttemptCount != 0 {
		t.Fatalf("重试后任务状态错误: %+v", item)
	}
	if item.ErrorCode != "" || item.LastError != "" || item.CompletedAt != nil || item.StartedAt != nil {
		t.Fatalf("重试后错误/完成时间未清理: %+v", item)
	}

	// 数据库侧断言：锁字段、可用时间、错误字段全部清理
	var stored models.InvoiceParsingTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.InvoiceParsingTaskPending || stored.AttemptCount != 0 ||
		stored.AvailableAt != nil || stored.LockedBy != "" || stored.LockedUntil != nil ||
		stored.ErrorCode != "" || stored.LastError != "" || stored.CompletedAt != nil {
		t.Fatalf("重试后数据库字段未完整清理: %+v", stored)
	}

	// 重试后必须可被 worker 立即领取（attempt_count 重置为 0 + available_at 为空）
	claimed, ok := handler.claimInvoiceParsingTask(context.Background())
	if !ok || claimed.InvoiceID != invoice.ID {
		t.Fatalf("重试后任务应可被 worker 立即领取: %+v ok=%v", claimed, ok)
	}
}
