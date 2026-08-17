package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// ---------------------------------------------------------------------------
// 测试辅助函数
// ---------------------------------------------------------------------------

// newFeedbackTestRouter 创建带权限中间件的反馈路由（用于权限测试）
func newFeedbackTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/feedback", func(fr chi.Router) {
		fr.Post("/", handler.submitFeedback)
		fr.Group(func(mgr chi.Router) {
			mgr.Use(middleware.RequireManagerOrAbove(handler.db))
			mgr.Get("/", handler.listFeedback)
			mgr.Get("/stats", handler.feedbackStats)
		})
		fr.With(middleware.RequireAdmin(handler.db)).Put("/{id}/reply", handler.replyFeedback)
	})
	return r
}

// newFeedbackTestRouterNoAuth 创建无权限中间件的反馈路由（用于功能测试）
func newFeedbackTestRouterNoAuth(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/feedback", func(fr chi.Router) {
		fr.Post("/", handler.submitFeedback)
		fr.Get("/", handler.listFeedback)
		fr.Get("/stats", handler.feedbackStats)
		fr.Put("/{id}/reply", handler.replyFeedback)
	})
	return r
}

// setAuthContext 将用户 ID 写入请求上下文（模拟 JWT 中间件已鉴权）
func setAuthContext(r *http.Request, userID uint) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
	return r.WithContext(ctx)
}

// createTestUser 创建测试用户并返回
func createTestUser(t *testing.T, tx *gorm.DB, username string, fullName string) models.User {
	t.Helper()
	user := models.User{
		Username: username,
		Email:    username + "@test.local",
		Password: "hashed-password-placeholder",
		FullName: fullName,
		Active:   true,
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

// createTestRole 创建测试角色
func createTestRole(t *testing.T, tx *gorm.DB, name string, label string) models.Role {
	t.Helper()
	role := models.Role{Name: name, Label: label, IsSystem: true}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}
	return role
}

// assignRoleToUser 给用户分配角色
func assignRoleToUser(t *testing.T, tx *gorm.DB, userID uint, roleID uint) {
	t.Helper()
	ur := models.UserRole{UserID: userID, RoleID: roleID}
	if err := tx.Create(&ur).Error; err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
}

// migrateFeedbackTables 迁移反馈测试所需的表
func migrateFeedbackTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	err := tx.AutoMigrate(
		&models.User{},
		&models.ChatMessage{},
		&models.ChatFeedback{},
		&models.Role{},
		&models.UserRole{},
	)
	if err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// ---------------------------------------------------------------------------
// submitFeedback 测试
// ---------------------------------------------------------------------------

func TestSubmitFeedback(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	user := createTestUser(t, tx, "testuser", "测试用户")
	handler := NewHandler(tx)
	router := newFeedbackTestRouterNoAuth(t, handler)

	t.Run("创建成功", func(t *testing.T) {
		body := `{"message_id":"msg-001","rating":"positive","comment":"很好"}`
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
		req = setAuthContext(req, user.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("期望 201, 得到 %d: %s", w.Code, w.Body.String())
		}
		var fb models.ChatFeedback
		if err := json.Unmarshal(w.Body.Bytes(), &fb); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}
		if fb.Rating != "positive" || fb.UserID != user.ID {
			t.Fatalf("反馈数据不正确: rating=%s user_id=%d", fb.Rating, fb.UserID)
		}
	})

	t.Run("缺少 message_id", func(t *testing.T) {
		body := `{"rating":"negative","comment":"不好"}`
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
		req = setAuthContext(req, user.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 400, 得到 %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("缺少认证用户", func(t *testing.T) {
		body := `{"message_id":"msg-002","rating":"positive"}`
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401, 得到 %d", w.Code)
		}
	})

	t.Run("已存在则更新", func(t *testing.T) {
		body1 := `{"message_id":"msg-dup","rating":"negative","comment":"不好"}`
		req1 := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body1))
		req1 = setAuthContext(req1, user.ID)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		if w1.Code != http.StatusCreated {
			t.Fatalf("首次创建失败: %s", w1.Body.String())
		}

		body2 := `{"message_id":"msg-dup","rating":"positive","comment":"改好了"}`
		req2 := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body2))
		req2 = setAuthContext(req2, user.ID)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("更新失败: %s", w2.Body.String())
		}
		var fb models.ChatFeedback
		json.Unmarshal(w2.Body.Bytes(), &fb)
		if fb.Rating != "positive" {
			t.Fatalf("评分未更新: 期望 positive, 得到 %s", fb.Rating)
		}
	})
}

// ---------------------------------------------------------------------------
// listFeedback 测试
// ---------------------------------------------------------------------------

func TestListFeedback(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	userA := createTestUser(t, tx, "userA", "用户A")
	userB := createTestUser(t, tx, "userB", "用户B")
	handler := NewHandler(tx)
	router := newFeedbackTestRouterNoAuth(t, handler)

	// 创建 25 条反馈记录（20 positive + 5 negative），跨越两个用户
	for i := 0; i < 20; i++ {
		tx.Create(&models.ChatFeedback{
			UserID:    userA.ID,
			MessageID: fmt.Sprintf("msg-a-%d", i),
			Rating:    "positive",
			Comment:   "好评",
		})
	}
	for i := 0; i < 5; i++ {
		tx.Create(&models.ChatFeedback{
			UserID:    userB.ID,
			MessageID: fmt.Sprintf("msg-b-%d", i),
			Rating:    "negative",
			Comment:   "差评",
		})
	}

	t.Run("分页首页20条", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback?page=1", nil)
		req = setAuthContext(req, userA.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("请求失败: %d", w.Code)
		}
		var resp feedbackListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Items) != 20 || resp.Total != 25 || resp.Page != 1 {
			t.Fatalf("分页不正确: items=%d total=%d page=%d", len(resp.Items), resp.Total, resp.Page)
		}
	})

	t.Run("分页第二页5条", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback?page=2", nil)
		req = setAuthContext(req, userA.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("请求失败: %d", w.Code)
		}
		var resp feedbackListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Items) != 5 || resp.Page != 2 {
			t.Fatalf("第二页不正确: items=%d page=%d", len(resp.Items), resp.Page)
		}
	})

	t.Run("按rating筛选negative", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback?rating=negative", nil)
		req = setAuthContext(req, userA.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var resp feedbackListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Total != 5 {
			t.Fatalf("差评数量应为 5, 得到 %d", resp.Total)
		}
	})

	t.Run("按user_id筛选", func(t *testing.T) {
		url := fmt.Sprintf("/api/feedback?user_id=%d", userB.ID)
		req := httptest.NewRequest("GET", url, nil)
		req = setAuthContext(req, userA.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var resp feedbackListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Total != 5 {
			t.Fatalf("userB 的反馈应为 5, 得到 %d", resp.Total)
		}
		for _, item := range resp.Items {
			if item.UserID != userB.ID {
				t.Fatalf("筛选结果包含了非 userB 的数据: user_id=%d", item.UserID)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// replyFeedback 测试
// ---------------------------------------------------------------------------

func TestReplyFeedback(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	user := createTestUser(t, tx, "admin1", "管理员")
	handler := NewHandler(tx)
	router := newFeedbackTestRouterNoAuth(t, handler)

	fb := models.ChatFeedback{
		UserID:    user.ID,
		MessageID: "msg-reply-1",
		Rating:    "negative",
		Comment:   "需要改进",
	}
	if err := tx.Create(&fb).Error; err != nil {
		t.Fatalf("创建反馈失败: %v", err)
	}

	t.Run("回复成功更新Reply与RepliedAt", func(t *testing.T) {
		body := `{"reply":"已收到，会尽快修复"}`
		url := fmt.Sprintf("/api/feedback/%d/reply", fb.ID)
		req := httptest.NewRequest("PUT", url, strings.NewReader(body))
		req = setAuthContext(req, user.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("回复失败(期望200): %d %s", w.Code, w.Body.String())
		}
		var result feedbackItem
		json.Unmarshal(w.Body.Bytes(), &result)
		if result.Reply != "已收到，会尽快修复" || result.RepliedAt == nil {
			t.Fatalf("回复字段未更新: Reply=%s RepliedAt=%v", result.Reply, result.RepliedAt)
		}
	})

	t.Run("不存在的反馈返回404", func(t *testing.T) {
		body := `{"reply":"没用的回复"}`
		req := httptest.NewRequest("PUT", "/api/feedback/99999/reply", strings.NewReader(body))
		req = setAuthContext(req, user.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("应返回 404, 得到 %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// feedbackStats 测试
// ---------------------------------------------------------------------------

func TestFeedbackStats(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	user := createTestUser(t, tx, "statsuser", "统计用户")
	handler := NewHandler(tx)
	router := newFeedbackTestRouterNoAuth(t, handler)

	// 创建 7 positive + 3 negative
	for i := 0; i < 7; i++ {
		tx.Create(&models.ChatFeedback{
			UserID:    user.ID,
			MessageID: fmt.Sprintf("stats-p-%d", i),
			Rating:    "positive",
			Comment:   "赞",
		})
	}
	for i := 0; i < 3; i++ {
		tx.Create(&models.ChatFeedback{
			UserID:    user.ID,
			MessageID: fmt.Sprintf("stats-n-%d", i),
			Rating:    "negative",
			Comment:   "差",
		})
	}

	t.Run("统计正确计算", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback/stats", nil)
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("统计请求失败: %d", w.Code)
		}
		var stats feedbackStatsResponse
		json.Unmarshal(w.Body.Bytes(), &stats)
		if stats.Total != 10 {
			t.Fatalf("总数应为 10, 得到 %d", stats.Total)
		}
		if stats.Positive != 7 {
			t.Fatalf("好评应为 7, 得到 %d", stats.Positive)
		}
		if stats.Negative != 3 {
			t.Fatalf("差评应为 3, 得到 %d", stats.Negative)
		}
		expectedRate := 7.0 / 10.0
		if stats.PositiveRate != expectedRate {
			t.Fatalf("好评率应为 %.2f, 得到 %.2f", expectedRate, stats.PositiveRate)
		}
	})

	t.Run("无数据时统计", func(t *testing.T) {
		// 使用新事务确保空表
		db2 := setupTestDB(t)
		tx2 := newTestTransaction(t, db2)
		migrateFeedbackTables(t, tx2)
		handler2 := NewHandler(tx2)
		router2 := newFeedbackTestRouterNoAuth(t, handler2)
		user2 := createTestUser(t, tx2, "emptyuser", "空用户")

		req := httptest.NewRequest("GET", "/api/feedback/stats", nil)
		req = setAuthContext(req, user2.ID)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("空统计请求失败: %d", w.Code)
		}
		var stats feedbackStatsResponse
		json.Unmarshal(w.Body.Bytes(), &stats)
		if stats.Total != 0 || stats.PositiveRate != 0 {
			t.Fatalf("空数据统计不正确: total=%d rate=%.2f", stats.Total, stats.PositiveRate)
		}
	})
}

// ---------------------------------------------------------------------------
// RBAC 权限测试
// ---------------------------------------------------------------------------

func TestFeedbackPermissions(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	// 创建角色
	adminRole := createTestRole(t, tx, models.RoleAdmin, "管理员")
	mgrRole := createTestRole(t, tx, models.RoleManager, "经理")

	// 创建用户并分配角色
	adminUser := createTestUser(t, tx, "admintest", "管理员用户")
	assignRoleToUser(t, tx, adminUser.ID, adminRole.ID)

	mgrUser := createTestUser(t, tx, "mgrtest", "经理用户")
	assignRoleToUser(t, tx, mgrUser.ID, mgrRole.ID)

	viewerUser := createTestUser(t, tx, "viewertest", "普通查看者")
	// viewer 不分配任何角色

	handler := NewHandler(tx)
	router := newFeedbackTestRouter(t, handler)

	// 创建一条反馈用于回复测试
	fb := models.ChatFeedback{
		UserID:    viewerUser.ID,
		MessageID: "perm-msg-1",
		Rating:    "negative",
		Comment:   "测试权限",
	}
	if err := tx.Create(&fb).Error; err != nil {
		t.Fatalf("创建权限测试反馈失败: %v", err)
	}

	// --- 用例：admin 可以访问 listFeedback ---
	t.Run("admin可查看列表", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback", nil)
		req = setAuthContext(req, adminUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("admin 应能查看列表(200), 得到 %d: %s", w.Code, w.Body.String())
		}
	})

	// --- 用例：manager 可以访问 listFeedback ---
	t.Run("manager可查看列表", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback", nil)
		req = setAuthContext(req, mgrUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("manager 应能查看列表(200), 得到 %d: %s", w.Code, w.Body.String())
		}
	})

	// --- 用例：viewer 被拦截 listFeedback ---
	t.Run("viewer被拦截查看列表", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback", nil)
		req = setAuthContext(req, viewerUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("viewer 应被拦截(403), 得到 %d", w.Code)
		}
	})

	// --- 用例：admin 可以访问 stats ---
	t.Run("admin可查看统计", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback/stats", nil)
		req = setAuthContext(req, adminUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("admin 应能查看统计(200), 得到 %d", w.Code)
		}
	})

	// --- 用例：manager 可以访问 stats ---
	t.Run("manager可查看统计", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback/stats", nil)
		req = setAuthContext(req, mgrUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("manager 应能查看统计(200), 得到 %d", w.Code)
		}
	})

	// --- 用例：viewer 被拦截 stats ---
	t.Run("viewer被拦截查看统计", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feedback/stats", nil)
		req = setAuthContext(req, viewerUser.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("viewer 应被拦截(403), 得到 %d", w.Code)
		}
	})

	// --- 用例：admin 可以回复 ---
	t.Run("admin可回复", func(t *testing.T) {
		body := `{"reply":"管理员回复"}`
		url := fmt.Sprintf("/api/feedback/%d/reply", fb.ID)
		req := httptest.NewRequest("PUT", url, strings.NewReader(body))
		req = setAuthContext(req, adminUser.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("admin 应能回复(200), 得到 %d: %s", w.Code, w.Body.String())
		}
	})

	// --- 用例：manager 被拦截回复 ---
	t.Run("manager被拦截回复", func(t *testing.T) {
		body := `{"reply":"经理想回复"}`
		url := fmt.Sprintf("/api/feedback/%d/reply", fb.ID)
		req := httptest.NewRequest("PUT", url, strings.NewReader(body))
		req = setAuthContext(req, mgrUser.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("manager 应被拦截回复(403), 得到 %d", w.Code)
		}
	})

	// --- 用例：viewer 可以提交反馈 ---
	t.Run("viewer可提交反馈", func(t *testing.T) {
		body := `{"message_id":"perm-msg-2","rating":"positive","comment":"好"}`
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
		req = setAuthContext(req, viewerUser.ID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("viewer 应能提交反馈(201), 得到 %d: %s", w.Code, w.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// buildFeedbackItem 提问回填测试
// ---------------------------------------------------------------------------

func TestBuildFeedbackItemQuestion(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateFeedbackTables(t, tx)

	userA := createTestUser(t, tx, "quserA", "提问用户A")
	userB := createTestUser(t, tx, "quserB", "提问用户B")
	handler := NewHandler(tx)
	router := newFeedbackTestRouterNoAuth(t, handler)

	// 场景1：反馈 session_id 为空，message_id 指向携带真实会话 ID 的助手消息
	userMsgA := models.ChatMessage{UserID: userA.ID, SessionID: "sess-1", Role: "user", Content: "原始提问内容"}
	if err := tx.Create(&userMsgA).Error; err != nil {
		t.Fatalf("创建用户消息失败: %v", err)
	}
	assistantMsgA := models.ChatMessage{UserID: userA.ID, SessionID: "sess-1", Role: "assistant", Content: "助手回答", Sources: datatypes.JSON(`[1,2]`)}
	if err := tx.Create(&assistantMsgA).Error; err != nil {
		t.Fatalf("创建助手消息失败: %v", err)
	}
	fbA := models.ChatFeedback{
		UserID:    userA.ID,
		MessageID: strconv.FormatUint(uint64(assistantMsgA.ID), 10),
		SessionID: "",
		Rating:    "positive",
		Comment:   "好",
	}
	if err := tx.Create(&fbA).Error; err != nil {
		t.Fatalf("创建反馈失败: %v", err)
	}

	// 场景2：助手消息不存在（历史非数值 ID），回退反馈记录的 session_id
	userMsgB := models.ChatMessage{UserID: userB.ID, SessionID: "sess-2", Role: "user", Content: "历史提问内容"}
	if err := tx.Create(&userMsgB).Error; err != nil {
		t.Fatalf("创建用户消息失败: %v", err)
	}
	fbB := models.ChatFeedback{
		UserID:    userB.ID,
		MessageID: "legacy-msg-1",
		SessionID: "sess-2",
		Rating:    "negative",
		Comment:   "差",
	}
	if err := tx.Create(&fbB).Error; err != nil {
		t.Fatalf("创建反馈失败: %v", err)
	}

	t.Run("session_id为空时按助手消息会话回填提问", func(t *testing.T) {
		url := fmt.Sprintf("/api/feedback?user_id=%d", userA.ID)
		req := httptest.NewRequest("GET", url, nil)
		req = setAuthContext(req, userA.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("列表请求失败: %d %s", w.Code, w.Body.String())
		}
		var resp feedbackListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("应返回 1 条反馈, 得到 %d", len(resp.Items))
		}
		item := resp.Items[0]
		if item.Question != "原始提问内容" {
			t.Fatalf("提问未按助手消息会话回填: 期望 %q, 得到 %q", "原始提问内容", item.Question)
		}
		// 验证 Answer/Sources/AnswerUnavailable 行为保持不变
		if item.Answer != "助手回答" || item.AnswerUnavailable {
			t.Fatalf("Answer 行为被破坏: answer=%q unavailable=%v", item.Answer, item.AnswerUnavailable)
		}
		if string(item.Sources) != "[1,2]" {
			t.Fatalf("Sources 行为被破坏: %s", string(item.Sources))
		}
	})

	t.Run("助手消息不存在时回退反馈记录的session_id", func(t *testing.T) {
		url := fmt.Sprintf("/api/feedback?user_id=%d", userB.ID)
		req := httptest.NewRequest("GET", url, nil)
		req = setAuthContext(req, userB.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("列表请求失败: %d %s", w.Code, w.Body.String())
		}
		var resp feedbackListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("应返回 1 条反馈, 得到 %d", len(resp.Items))
		}
		if resp.Items[0].Question != "历史提问内容" {
			t.Fatalf("回退查询失败: 期望 %q, 得到 %q", "历史提问内容", resp.Items[0].Question)
		}
	})
}
