package api

import (
	"encoding/json"
	"fmt"
	"io"
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

// newMemoTestRouter 注册个人备忘录路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newMemoTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/memos", handler.listMemos)
		ur.Post("/memos", handler.createMemo)
		ur.Put("/memos/{id}", handler.updateMemo)
		ur.Delete("/memos/{id}", handler.deleteMemo)
	})
	return r
}

// migrateMemoTables 迁移备忘录测试所需表
func migrateMemoTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(&models.User{}, &models.Memo{}); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildMemoRequest 构造备忘录请求；auth=true 时写入登录用户上下文
func buildMemoRequest(t *testing.T, method, path, body string, auth bool, userID uint) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req = setAuthContext(req, userID)
	}
	return req
}

// decodeMemoResponse 解析单个备忘录响应体
func decodeMemoResponse(t *testing.T, resp *httptest.ResponseRecorder) models.Memo {
	t.Helper()
	var memo models.Memo
	if err := json.Unmarshal(resp.Body.Bytes(), &memo); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return memo
}

// decodeMemoListResponse 解析备忘录列表响应体
func decodeMemoListResponse(t *testing.T, resp *httptest.ResponseRecorder) []models.Memo {
	t.Helper()
	var out struct {
		Memos []models.Memo `json:"memos"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return out.Memos
}

// seedMemo 直接向数据库插入一条备忘录并返回
func seedMemo(t *testing.T, tx *gorm.DB, userID uint, title string, pinned, completed bool) models.Memo {
	t.Helper()
	memo := models.Memo{
		UserID:    userID,
		Title:     title,
		Pinned:    pinned,
		Completed: completed,
	}
	if err := tx.Create(&memo).Error; err != nil {
		t.Fatalf("创建备忘录失败: %v", err)
	}
	return memo
}

// ---------------------------------------------------------------------------
// POST：正常创建 / 字段与长度 / 白名单
// ---------------------------------------------------------------------------

func TestMemo_Create(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	user := createTestUser(t, tx, "memo_create", "备忘录创建用户")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	validBody := `{"title":"待办清单","content":"买牛奶、交房租","pinned":true,"completed":false}`

	t.Run("正常创建返回201", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", validBody, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("期望 201，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		memo := decodeMemoResponse(t, resp)
		if memo.ID == 0 || memo.Title != "待办清单" || memo.Content != "买牛奶、交房租" || !memo.Pinned || memo.Completed {
			t.Fatalf("备忘录字段不正确: %+v", memo)
		}
		if strings.Contains(resp.Body.String(), `"user_id"`) {
			t.Fatalf("响应不应包含 user_id 字段: %s", resp.Body.String())
		}
	})

	t.Run("title缺失返回400", func(t *testing.T) {
		body := `{"content":"只有正文"}`
		req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("title超长返回400", func(t *testing.T) {
		longTitle := strings.Repeat("题", memoTitleMaxLen+1)
		body := fmt.Sprintf(`{"title":%q}`, longTitle)
		req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("content超长返回400", func(t *testing.T) {
		longContent := strings.Repeat("正", memoContentMaxLen+1)
		body := fmt.Sprintf(`{"title":"t","content":%q}`, longContent)
		req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("未知字段(含user_id)返回400", func(t *testing.T) {
		bodies := []string{
			`{"title":"t","user_id":999}`,
			`{"title":"t","hack":1}`,
		}
		for _, body := range bodies {
			req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("body=%s 期望 400，实际 %d", body, resp.Code)
			}
		}
	})

	t.Run("可选字段缺省", func(t *testing.T) {
		body := `{"title":"纯标题"}`
		req := buildMemoRequest(t, http.MethodPost, "/api/user/memos", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("期望 201，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		memo := decodeMemoResponse(t, resp)
		if memo.Content != "" || memo.Pinned || memo.Completed {
			t.Fatalf("可选字段缺省应为空/零值: %+v", memo)
		}
	})
}

// ---------------------------------------------------------------------------
// GET：排序 / limit / 空数组 / 非法参数
// ---------------------------------------------------------------------------

func TestMemo_List(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	user := createTestUser(t, tx, "memo_list", "备忘录列表用户")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	// 构造排序场景：置顶优先、updated_at 降序、id 降序
	oldPinned := seedMemo(t, tx, user.ID, "旧置顶", true, false)
	time.Sleep(10 * time.Millisecond)
	newPinned := seedMemo(t, tx, user.ID, "新置顶", true, false)
	time.Sleep(10 * time.Millisecond)
	normal := seedMemo(t, tx, user.ID, "普通", false, false)

	t.Run("按pinned/updated_at/id排序", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		memos := decodeMemoListResponse(t, resp)
		if len(memos) != 3 {
			t.Fatalf("期望 3 条备忘录，实际 %d: %+v", len(memos), memos)
		}
		if memos[0].ID != newPinned.ID || memos[1].ID != oldPinned.ID || memos[2].ID != normal.ID {
			t.Fatalf("排序不正确（置顶优先、updated_at 降序）: %+v", memos)
		}
	})

	t.Run("limit截断返回条数", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos?limit=2", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		memos := decodeMemoListResponse(t, resp)
		if len(memos) != 2 {
			t.Fatalf("limit=2 期望 2 条，实际 %d: %+v", len(memos), memos)
		}
		if memos[0].ID != newPinned.ID || memos[1].ID != oldPinned.ID {
			t.Fatalf("limit 截断应保留最新置顶: %+v", memos)
		}
	})

	t.Run("limit超上限按上限返回", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos?limit=99999", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		memos := decodeMemoListResponse(t, resp)
		if len(memos) != 3 {
			t.Fatalf("limit 超上限应返回全部 3 条，实际 %d", len(memos))
		}
	})

	t.Run("非法limit返回400", func(t *testing.T) {
		for _, raw := range []string{"abc", "0", "-1"} {
			req := buildMemoRequest(t, http.MethodGet, "/api/user/memos?limit="+raw, "", true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("limit=%s 期望 400，实际 %d", raw, resp.Code)
			}
		}
	})

	t.Run("空列表返回空数组", func(t *testing.T) {
		emptyUser := createTestUser(t, tx, "memo_list_empty", "空列表用户")
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos", "", true, emptyUser.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", resp.Code)
		}
		if strings.Contains(resp.Body.String(), `"memos":null`) {
			t.Fatalf("memos 不应为 null: %s", resp.Body.String())
		}
		memos := decodeMemoListResponse(t, resp)
		if memos == nil || len(memos) != 0 {
			t.Fatalf("期望空数组，实际 %+v", memos)
		}
	})
}

// ---------------------------------------------------------------------------
// PUT：正常更新 / 不存在 404 / 他人 404 / 非法 payload
// ---------------------------------------------------------------------------

func TestMemo_Update(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	userA := createTestUser(t, tx, "memo_up_a", "更新用户A")
	userB := createTestUser(t, tx, "memo_up_b", "更新用户B")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	memo := seedMemo(t, tx, userA.ID, "原标题", false, false)

	t.Run("正常更新返回200", func(t *testing.T) {
		body := `{"title":"新标题","content":"新正文","pinned":true,"completed":true}`
		path := fmt.Sprintf("/api/user/memos/%d", memo.ID)
		req := buildMemoRequest(t, http.MethodPut, path, body, true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		updated := decodeMemoResponse(t, resp)
		if updated.Title != "新标题" || updated.Content != "新正文" || !updated.Pinned || !updated.Completed {
			t.Fatalf("更新字段不正确: %+v", updated)
		}
		var stored models.Memo
		if err := tx.First(&stored, memo.ID).Error; err != nil {
			t.Fatalf("查询存储记录失败: %v", err)
		}
		if stored.Title != "新标题" || stored.UserID != userA.ID {
			t.Fatalf("存储记录不正确: %+v", stored)
		}
	})

	t.Run("不存在备忘录返回404", func(t *testing.T) {
		body := `{"title":"t"}`
		req := buildMemoRequest(t, http.MethodPut, "/api/user/memos/99999", body, true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
	})

	t.Run("他人备忘录返回404且不生效", func(t *testing.T) {
		body := `{"title":"越权修改"}`
		path := fmt.Sprintf("/api/user/memos/%d", memo.ID)
		req := buildMemoRequest(t, http.MethodPut, path, body, true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
		var stored models.Memo
		if err := tx.First(&stored, memo.ID).Error; err != nil {
			t.Fatalf("查询存储记录失败: %v", err)
		}
		if stored.Title != "新标题" {
			t.Fatalf("他人更新不应生效: %+v", stored)
		}
	})

	t.Run("非法payload返回400", func(t *testing.T) {
		body := `{"title":""}`
		path := fmt.Sprintf("/api/user/memos/%d", memo.ID)
		req := buildMemoRequest(t, http.MethodPut, path, body, true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// DELETE：正常删除 / 不存在 404 / 他人 404 / 非法 id
// ---------------------------------------------------------------------------

func TestMemo_Delete(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	userA := createTestUser(t, tx, "memo_del_a", "删除用户A")
	userB := createTestUser(t, tx, "memo_del_b", "删除用户B")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	memo := seedMemo(t, tx, userA.ID, "待删除", false, false)

	t.Run("正常删除返回204", func(t *testing.T) {
		path := fmt.Sprintf("/api/user/memos/%d", memo.ID)
		req := buildMemoRequest(t, http.MethodDelete, path, "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNoContent {
			t.Fatalf("期望 204，实际 %d", resp.Code)
		}
		var count int64
		if err := tx.Model(&models.Memo{}).Where("id = ?", memo.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 0 {
			t.Fatalf("删除后记录应不存在，实际 %d 条", count)
		}
	})

	t.Run("不存在备忘录返回404", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodDelete, "/api/user/memos/99999", "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
	})

	t.Run("他人备忘录返回404且不生效", func(t *testing.T) {
		other := seedMemo(t, tx, userA.ID, "他人视角备忘录", false, false)
		path := fmt.Sprintf("/api/user/memos/%d", other.ID)
		req := buildMemoRequest(t, http.MethodDelete, path, "", true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
		var count int64
		if err := tx.Model(&models.Memo{}).Where("id = ?", other.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("他人删除不应生效，实际 %d 条", count)
		}
	})

	t.Run("非法id返回400", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodDelete, "/api/user/memos/abc", "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// 鉴权与用户隔离
// ---------------------------------------------------------------------------

func TestMemo_Auth(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	user := createTestUser(t, tx, "memo_auth", "鉴权用户")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	body := `{"title":"t"}`
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/user/memos", ""},
		{http.MethodPost, "/api/user/memos", body},
		{http.MethodPut, "/api/user/memos/1", body},
		{http.MethodDelete, "/api/user/memos/1", ""},
	}
	for _, c := range cases {
		req := buildMemoRequest(t, c.method, c.path, c.body, false, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 未登录期望 401，实际 %d", c.method, c.path, resp.Code)
		}
	}
}

func TestMemo_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateMemoTables(t, tx)

	userA := createTestUser(t, tx, "memo_iso_a", "隔离用户A")
	userB := createTestUser(t, tx, "memo_iso_b", "隔离用户B")
	handler := NewHandler(tx)
	router := newMemoTestRouter(t, handler)

	memoA := seedMemo(t, tx, userA.ID, "A的备忘录", false, false)
	memoB := seedMemo(t, tx, userB.ID, "B的备忘录", false, false)

	t.Run("A列表仅见A备忘录", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos", "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		memos := decodeMemoListResponse(t, resp)
		if len(memos) != 1 || memos[0].ID != memoA.ID {
			t.Fatalf("A 应仅看到自己的备忘录: %+v", memos)
		}
	})

	t.Run("B列表仅见B备忘录", func(t *testing.T) {
		req := buildMemoRequest(t, http.MethodGet, "/api/user/memos", "", true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		memos := decodeMemoListResponse(t, resp)
		if len(memos) != 1 || memos[0].ID != memoB.ID {
			t.Fatalf("B 应仅看到自己的备忘录: %+v", memos)
		}
	})

	t.Run("B操作A备忘录返回404", func(t *testing.T) {
		path := fmt.Sprintf("/api/user/memos/%d", memoA.ID)
		req := buildMemoRequest(t, http.MethodPut, path, `{"title":"x"}`, true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("PUT 他人备忘录期望 404，实际 %d", resp.Code)
		}
		req = buildMemoRequest(t, http.MethodDelete, path, "", true, userB.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("DELETE 他人备忘录期望 404，实际 %d", resp.Code)
		}
		// 确认 A 的备忘录未被删除
		var count int64
		if err := tx.Model(&models.Memo{}).Where("id = ?", memoA.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("A 的备忘录应保留，实际 %d 条", count)
		}
	})
}
