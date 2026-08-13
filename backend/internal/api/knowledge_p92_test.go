package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ---------------------------------------------------------------------------
// P9.2 强化：管理员三链路全量放行 / 非法 kb_id / kb_id 组合 / 跨用户隔离
// ---------------------------------------------------------------------------

// grantSystemRole 创建系统角色并绑定到用户
func grantSystemRole(t *testing.T, tx *gorm.DB, user models.User, roleName string) {
	t.Helper()
	role := createTestRole(t, tx, roleName, roleName)
	assignRoleToUser(t, tx, user.ID, role.ID)
}

// seedAdminKBData 创建管理员用户 + 他人私有/受限知识库（含文档与脱敏规则）
func seedAdminKBData(t *testing.T, tx *gorm.DB, owner models.User) (models.KnowledgeBase, models.KnowledgeBase) {
	t.Helper()
	privateKB := createKB(t, tx, "他人私有库", "private", &owner.ID)
	restrictedKB := createKB(t, tx, "他人受限库", "restricted", &owner.ID)

	// 私有库文档（含手机号）
	doc := models.Document{
		UserID: owner.ID, DocumentCode: "ADMIN-P-1", FileName: "员工档案",
		ContentText: "管理员测试 联系电话 13800138000",
		SourceType:  "custom", SourceID: 1, SourceKBID: &privateKB.ID,
		Status: "active", OCRStatus: "completed",
	}
	if err := tx.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
	// 受限库文档（含身份证号）
	doc2 := models.Document{
		UserID: owner.ID, DocumentCode: "ADMIN-R-1", FileName: "员工档案",
		ContentText: "管理员测试 身份证号 110101199001011234",
		SourceType:  "custom", SourceID: 1, SourceKBID: &restrictedKB.ID,
		Status: "active", OCRStatus: "completed",
	}
	if err := tx.Create(&doc2).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
	// 字段脱敏规则
	for _, kbID := range []uint{privateKB.ID, restrictedKB.ID} {
		if err := tx.Create(&models.KBFieldMask{
			KnowledgeBaseID: kbID, FieldName: "phone", MaskPattern: "front3back4",
		}).Error; err != nil {
			t.Fatalf("创建脱敏规则失败: %v", err)
		}
	}
	return privateKB, restrictedKB
}

// TestAdminKBID_AllChains admin 指定他人 private/restricted KB：search/chat/SSE 三链路 200
func TestAdminKBID_AllChains(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	owner := createTestUser(t, tx, "kbowner", "KB 所有者")
	admin := createTestUser(t, tx, "sysadmin", "系统管理员")
	grantSystemRole(t, tx, admin, "admin")

	privateKB, restrictedKB := seedAdminKBData(t, tx, owner)

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, kb := range []models.KnowledgeBase{privateKB, restrictedKB} {
		// 1. 搜索（指定 kb_id）→ 200
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/knowledge/search?q=管理员测试&kb_id=%d", kb.ID), nil)
		req = setAuthContext(req, admin.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("admin 搜索他人 %s 期望 200，实际 %d，body: %s", kb.Visibility, w.Code, w.Body.String())
		}
		// 搜索结果不得泄露敏感原文
		var resp struct {
			Results []struct {
				Snippet string `json:"snippet"`
			} `json:"results"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Results) > 0 && strings.Contains(resp.Results[0].Snippet, "110101199001011234") {
			t.Errorf("admin 搜索结果泄露身份证原文: %s", resp.Results[0].Snippet)
		}

		// 2. 问答（body 带 kb_id）→ 200
		chatBody := fmt.Sprintf(`{"question":"管理员测试","kb_id":%d}`, kb.ID)
		req2 := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(chatBody))
		req2.Header.Set("Content-Type", "application/json")
		req2 = setAuthContext(req2, admin.ID)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Errorf("admin 问答他人 %s 期望 200，实际 %d，body: %s", kb.Visibility, w2.Code, w2.Body.String())
		}

		// 3. SSE 流式问答 → 200
		streamBody := fmt.Sprintf(`{"question":"管理员测试","kb_id":%d}`, kb.ID)
		req3 := httptest.NewRequest("POST", "/api/knowledge/chat/stream", strings.NewReader(streamBody))
		req3.Header.Set("Content-Type", "application/json")
		req3 = setAuthContext(req3, admin.ID)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		if w3.Code != http.StatusOK {
			t.Errorf("admin SSE 问答他人 %s 期望 200，实际 %d，body: %s", kb.Visibility, w3.Code, w3.Body.String())
		}
	}
}

// TestSuperAdminKBID_Search super_admin 同样全量放行（搜索他人私有 KB → 200）
func TestSuperAdminKBID_Search(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	owner := createTestUser(t, tx, "superowner", "KB 所有者")
	supAdmin := createTestUser(t, tx, "superadmin", "超级管理员")
	grantSystemRole(t, tx, supAdmin, "super_admin")

	privateKB, _ := seedAdminKBData(t, tx, owner)

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/knowledge/search?q=管理员测试&kb_id=%d", privateKB.ID), nil)
	req = setAuthContext(req, supAdmin.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("super_admin 搜索他人私有 KB 期望 200，实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestKBIDInvalid 非法 kb_id：search 400 / chat 400 / SSE 400
func TestKBIDInvalid(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "invaliduser", "非法参数用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	// 1. search：kb_id 非数字 → 400
	for _, bad := range []string{"abc", "-1", "1.5", "18446744073709551616"} {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/knowledge/search?q=测试&kb_id=%s", bad), nil)
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("search kb_id=%q 期望 400，实际 %d，body: %s", bad, w.Code, w.Body.String())
		}
	}

	// 2. chat：body kb_id 非数字 → 400
	chatBad := `{"question":"测试","kb_id":"abc"}`
	req2 := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(chatBad))
	req2.Header.Set("Content-Type", "application/json")
	req2 = setAuthContext(req2, user.ID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("chat kb_id 非法期望 400，实际 %d，body: %s", w2.Code, w2.Body.String())
	}

	// 3. SSE：body kb_id 非数字 → 400
	req3 := httptest.NewRequest("POST", "/api/knowledge/chat/stream", strings.NewReader(chatBad))
	req3.Header.Set("Content-Type", "application/json")
	req3 = setAuthContext(req3, user.ID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("SSE kb_id 非法期望 400，实际 %d，body: %s", w3.Code, w3.Body.String())
	}
}

// TestChatKBID_DefaultNullZero chat body kb_id 缺省/null/0 → 200（等价全部可见，不校验权限）
func TestChatKBID_DefaultNullZero(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "chatkbzero", "问答零值用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, body := range []string{
		`{"question":"测试问题"}`,              // 缺省 kb_id
		`{"question":"测试问题","kb_id":null}`, // null
		`{"question":"测试问题","kb_id":0}`,    // 显式 0
	} {
		req := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("chat kb_id 缺省/null/0（body=%s）期望 200，实际 %d，body: %s", body, w.Code, w.Body.String())
		}
	}
}

// TestChatKBID_Invalid400 chat body kb_id 非法负数/小数/超大整数 → 400
func TestChatKBID_Invalid400(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "chatkbinvalid", "问答非法参数用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, body := range []string{
		`{"question":"测试问题","kb_id":-1}`,                   // 负数
		`{"question":"测试问题","kb_id":1.5}`,                  // 小数
		`{"question":"测试问题","kb_id":18446744073709551616}`, // 超大整数（溢出 uint）
	} {
		req := httptest.NewRequest("POST", "/api/knowledge/chat", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("chat kb_id 非法（body=%s）期望 400，实际 %d，body: %s", body, w.Code, w.Body.String())
		}
	}
}

// TestChatStreamKBID_DefaultNullZero SSE body kb_id 缺省/null/0 → 200
func TestChatStreamKBID_DefaultNullZero(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "streamkbzero", "流式零值用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, body := range []string{
		`{"question":"测试问题"}`,
		`{"question":"测试问题","kb_id":null}`,
		`{"question":"测试问题","kb_id":0}`,
	} {
		req := httptest.NewRequest("POST", "/api/knowledge/chat/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("SSE kb_id 缺省/null/0（body=%s）期望 200，实际 %d，body: %s", body, w.Code, w.Body.String())
		}
	}
}

// TestChatStreamKBID_Invalid400 SSE body kb_id 非法负数/小数/超大整数 → 400
func TestChatStreamKBID_Invalid400(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "streamkbinvalid", "流式非法参数用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, body := range []string{
		`{"question":"测试问题","kb_id":-1}`,
		`{"question":"测试问题","kb_id":1.5}`,
		`{"question":"测试问题","kb_id":18446744073709551616}`,
	} {
		req := httptest.NewRequest("POST", "/api/knowledge/chat/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("SSE kb_id 非法（body=%s）期望 400，实际 %d，body: %s", body, w.Code, w.Body.String())
		}
	}
}

// TestKBIDZeroNullMissing kb_id=0 / null / 缺省 组合：search 均 200
func TestKBIDZeroNullMissing(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	user := createTestUser(t, tx, "combouser", "组合参数用户")

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	for _, suffix := range []string{"", "&kb_id=0", "&kb_id=null", "&kb_id=NULL"} {
		req := httptest.NewRequest("GET",
			"/api/knowledge/search?q=测试"+suffix, nil)
		req = setAuthContext(req, user.ID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("search 缺省/kb_id=0/null 期望 200（suffix=%q），实际 %d，body: %s", suffix, w.Code, w.Body.String())
		}
	}
}

// TestSearch_KBIDZeroExcludesPrivate kb_id=0 检索全部可见：
// 普通用户不得命中他人私有 KB 的文档
func TestSearch_KBIDZeroExcludesPrivate(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateKnowledgeTables(t, tx)

	owner := createTestUser(t, tx, "privowner", "私有库所有者")
	stranger := createTestUser(t, tx, "privstranger", "无关用户")
	kb := createKB(t, tx, "他人私密库", "private", &owner.ID)
	doc := models.Document{
		UserID: owner.ID, DocumentCode: "PRIV-SCOPE-1", FileName: "员工-杨过",
		ContentText: "杨过 黯然销魂掌",
		SourceType:  "custom", SourceID: 1, SourceKBID: &kb.ID,
		Status: "active", OCRStatus: "completed",
	}
	if err := tx.Create(&doc).Error; err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	handler := NewHandler(tx)
	router := newKnowledgeTestRouter(t, handler)

	// 陌生人 kb_id=0 检索，不得命中他人私有 KB 文档
	req := httptest.NewRequest("GET", "/api/knowledge/search?q=杨过", nil)
	req = setAuthContext(req, stranger.ID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("kb_id=0 检索期望 200，实际 %d", w.Code)
	}
	var resp struct {
		Results []struct {
			Title string `json:"title"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, r := range resp.Results {
		if strings.Contains(r.Title, "杨过") {
			t.Errorf("kb_id=0 检索泄露他人私有 KB 文档: %+v", r)
		}
	}
}
