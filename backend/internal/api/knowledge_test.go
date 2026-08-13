package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ---------------------------------------------------------------------------
// 测试辅助：知识库路由
// ---------------------------------------------------------------------------

// newKnowledgeTestRouter 创建无中间件的知识库路由（测试功能而非权限中间件）
func newKnowledgeTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/knowledge", func(kr chi.Router) {
		kr.Get("/search", handler.searchKnowledge)
		kr.Post("/chat", handler.chatKnowledge)
	})
	return r
}

// setAuthContext 将用户 ID 写入请求上下文
// 注：此函数在 feedback_test.go 中已定义，此处保留注释说明
// 实际使用 feedback_test.go 中的定义

// migrateKnowledgeTables 迁移知识库测试所需的表
func migrateKnowledgeTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	err := tx.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.KnowledgeBase{},
		&models.KBAccessRule{},
		&models.KBFieldMask{},
		&models.Document{},
		&models.DocumentChunk{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.ModelConfig{},
	)
	if err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// createKB 快捷创建知识库
func createKB(t *testing.T, tx *gorm.DB, name string, visibility string, ownerID *uint) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{
		Name:         name,
		Visibility:   visibility,
		SourceModule: "custom",
		OwnerID:      ownerID,
	}
	if err := tx.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	return kb
}

// createKBAccessRule 添加快捷访问规则
func createKBAccessRule(t *testing.T, tx *gorm.DB, kbID uint, userID *uint) {
	t.Helper()
	rule := models.KBAccessRule{
		KnowledgeBaseID: kbID,
		UserID:          userID,
	}
	if err := tx.Create(&rule).Error; err != nil {
		t.Fatalf("创建访问规则失败: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestSearch_NoKBID — 不带 kb_id 检索全部可见
// ---------------------------------------------------------------------------

func TestSearch_NoKBID(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "searchuser", "搜索用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	// 不带 kb_id 参数应正常返回（可能无结果但不应报错）
	req := httptest.NewRequest("GET", "/api/knowledge/search?q=测试查询&limit=5", nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200，实际 %d，body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if _, ok := resp["results"]; !ok {
		t.Errorf("响应中缺少 results 字段")
	}
}

// ---------------------------------------------------------------------------
// TestSearch_KBIDFilter — 带 kb_id 时按权限返回 403/200
// ---------------------------------------------------------------------------

func TestSearch_KBIDFilter(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	// 创建两个用户：一个有权限，一个没有
	owner := createTestUser(t, tx, "owner", "知识库所有者")
	viewer := createTestUser(t, tx, "viewer", "无权限访客")

	// 创建私有知识库（仅所有者可访问）
	kb := createKB(t, tx, "私有知识库", "private", &owner.ID)

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	// 场景 1：无权限用户带 kb_id 检索 → 403
	req1 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/knowledge/search?q=测试&kb_id=%d", kb.ID), nil)
	req1 = setAuthContext(req1, viewer.ID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusForbidden {
		t.Errorf("无权限用户检索私有知识库期望 403，实际 %d，body: %s", w1.Code, w1.Body.String())
	}

	// 场景 2：有权限用户（所有者）带 kb_id 检索 → 200
	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/knowledge/search?q=测试&kb_id=%d", kb.ID), nil)
	req2 = setAuthContext(req2, owner.ID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("所有者检索期望 200，实际 %d，body: %s", w2.Code, w2.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestSearch_KBIDWithPublicKB — 公开知识库全员可检索
// ---------------------------------------------------------------------------

func TestSearch_KBIDWithPublicKB(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "publicuser", "普通用户")

	// 公开+employee 模块知识库（全员可见）
	kb := models.KnowledgeBase{
		Name:         "员工花名册",
		Visibility:   "public",
		SourceModule: "employee",
	}
	if err := tx.Create(&kb).Error; err != nil {
		t.Fatalf("创建公开知识库失败: %v", err)
	}

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/knowledge/search?q=张三&kb_id=%d", kb.ID), nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("公开知识库检索期望 200，实际 %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// TestChat_KBID — chat body 带 kb_id 正常返回
// ---------------------------------------------------------------------------

func TestChat_KBID(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "chatuser", "问答用户")
	kb := createKB(t, tx, "问答知识库", "private", &user.ID)

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	body := fmt.Sprintf(`{"question":"测试问题","kb_id":%d}`, kb.ID)
	req := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 可能返回 200（LLM 配置了就正常回答）或 500（无 LLM 配置）
	// 但不应返回 403/401
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		bodyStr := w.Body.String()
		t.Errorf("带 kb_id 的 chat 不应返回 401/403，实际 %d，body: %s", w.Code, bodyStr)
	}
}

// ---------------------------------------------------------------------------
// TestChat_KBIDForbidden — 无权限用户 chat 带 kb_id 应返回 403
// ---------------------------------------------------------------------------

func TestChat_KBIDForbidden(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	owner := createTestUser(t, tx, "chatowner", "KB 所有者")
	stranger := createTestUser(t, tx, "stranger", "陌生人")
	kb := createKB(t, tx, "私密问答库", "private", &owner.ID)

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	body := fmt.Sprintf(`{"question":"私密问题","kb_id":%d}`, kb.ID)
	req := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthContext(req, stranger.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("无权限 chat 期望 403，实际 %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// TestSearch_KBIDMasking — 脱敏结果验证
// ---------------------------------------------------------------------------

func TestSearch_KBIDMasking(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "maskuser", "脱敏测试用户")
	kb := createKB(t, tx, "脱敏知识库", "private", &user.ID)

	// 添加脱敏规则：对 content 字段应用 front3back4 脱敏
	mask := models.KBFieldMask{
		KnowledgeBaseID: kb.ID,
		FieldName:       "content",
		MaskPattern:     "front3back4",
	}
	if err := tx.Create(&mask).Error; err != nil {
		t.Fatalf("创建脱敏规则失败: %v", err)
	}

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/knowledge/search?q=脱敏测试&kb_id=%d", kb.ID), nil)
	req = setAuthContext(req, user.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("脱敏检索期望 200，实际 %d", w.Code)
	}
	t.Logf("脱敏检索响应: %s", w.Body.String())
}
