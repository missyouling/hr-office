package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

// newDockPrefTestRouter 注册 Dock 偏好路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newDockPrefTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/dock-preferences", handler.getDockPreferences)
		ur.Put("/dock-preferences", handler.updateDockPreferences)
	})
	return r
}

// migrateDockPrefTables 迁移 Dock 偏好测试所需表
func migrateDockPrefTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(&models.User{}, &models.UserPreference{}); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildDockPrefRequest 构造 Dock 偏好请求；auth=true 时写入登录用户上下文
func buildDockPrefRequest(t *testing.T, method, path, body string, auth bool, userID uint) *http.Request {
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

// decodeDockPrefResponse 解析响应体为 DockPreference
func decodeDockPrefResponse(t *testing.T, resp *httptest.ResponseRecorder) models.DockPreference {
	t.Helper()
	var pref models.DockPreference
	if err := json.Unmarshal(resp.Body.Bytes(), &pref); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return pref
}

// countDockPrefRows 统计 user_preferences 表中 Dock 偏好记录数
func countDockPrefRows(t *testing.T, tx *gorm.DB, userID uint) int64 {
	t.Helper()
	var count int64
	err := tx.Model(&models.UserPreference{}).
		Where("user_id = ? AND pref_key = ?", userID, models.DockPreferenceKey).
		Count(&count).Error
	if err != nil {
		t.Fatalf("统计 Dock 偏好记录失败: %v", err)
	}
	return count
}

// ---------------------------------------------------------------------------
// GET：未鉴权 / 默认值 / 保存回读
// ---------------------------------------------------------------------------

func TestDockPreference_Get(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateDockPrefTables(t, tx)

	user := createTestUser(t, tx, "dockget", "Dock偏好用户")
	handler := NewHandler(tx)
	router := newDockPrefTestRouter(t, handler)

	t.Run("未鉴权返回401", func(t *testing.T) {
		req := buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", false, 0)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", resp.Code)
		}
	})

	t.Run("无记录返回安全默认", func(t *testing.T) {
		req := buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		pref := decodeDockPrefResponse(t, resp)
		if pref.DesktopPosition != nil || pref.MobileExpanded {
			t.Fatalf("无记录应返回安全默认，实际 %+v", pref)
		}
		if strings.Contains(resp.Body.String(), `"user_id"`) {
			t.Fatalf("响应不应包含 user_id 字段: %s", resp.Body.String())
		}
	})

	t.Run("保存后GET回读一致", func(t *testing.T) {
		putBody := `{"desktop_position":{"left":120.5,"top":24},"mobile_expanded":true}`
		req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", putBody, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}

		req = buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET 期望 200，实际 %d", resp.Code)
		}
		pref := decodeDockPrefResponse(t, resp)
		if pref.DesktopPosition == nil || pref.DesktopPosition.Left == nil || pref.DesktopPosition.Top == nil {
			t.Fatalf("desktop_position 回读缺失: %+v", pref.DesktopPosition)
		}
		if *pref.DesktopPosition.Left != 120.5 || *pref.DesktopPosition.Top != 24 || !pref.MobileExpanded {
			t.Fatalf("Dock 偏好回读不一致: %+v", pref)
		}
	})
}

// ---------------------------------------------------------------------------
// PUT：非法输入（空 payload/未知字段/非法数值/冗余内容）
// ---------------------------------------------------------------------------

func TestDockPreference_PutInvalidInput(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateDockPrefTables(t, tx)

	user := createTestUser(t, tx, "dockinvalid", "非法输入用户")
	handler := NewHandler(tx)
	router := newDockPrefTestRouter(t, handler)

	invalidCases := []struct {
		name string
		body string
	}{
		{"空请求体", ""},
		{"空对象", `{}`},
		{"非法JSON", `{"desktop_position":`},
		{"未知顶层字段", `{"foo":1}`},
		{"含user_id字段", `{"desktop_position":null,"mobile_expanded":false,"user_id":999}`},
		{"position内未知字段", `{"desktop_position":{"left":10,"top":10,"width":300}}`},
		{"position缺left", `{"desktop_position":{"top":10},"mobile_expanded":false}`},
		{"position缺top", `{"desktop_position":{"left":10},"mobile_expanded":false}`},
		{"position为负数", `{"desktop_position":{"left":-5,"top":10},"mobile_expanded":false}`},
		{"position超上限", `{"desktop_position":{"left":100001,"top":10},"mobile_expanded":false}`},
		{"position为NaN", `{"desktop_position":{"left":NaN,"top":10},"mobile_expanded":false}`},
		{"position为Infinity", `{"desktop_position":{"left":Infinity,"top":10},"mobile_expanded":false}`},
		{"mobile_expanded为字符串", `{"desktop_position":null,"mobile_expanded":"yes"}`},
		{"mobile_expanded为数字", `{"desktop_position":null,"mobile_expanded":1}`},
		{"JSON后冗余内容", `{"desktop_position":null,"mobile_expanded":false} garbage`},
		{"请求体过大", `{"desktop_position":{"left":10,"top":10},"mobile_expanded":true,"pad":"` + strings.Repeat("x", 8<<10) + `"}`},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", tc.body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("期望 400，实际 %d，body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PUT：合法保存 / null 位置恢复默认 / 幂等
// ---------------------------------------------------------------------------

func TestDockPreference_PutValid(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateDockPrefTables(t, tx)

	user := createTestUser(t, tx, "dockvalid", "合法输入用户")
	handler := NewHandler(tx)
	router := newDockPrefTestRouter(t, handler)

	t.Run("保存后落库仅一行且值正确", func(t *testing.T) {
		body := `{"desktop_position":{"left":12.5,"top":88},"mobile_expanded":true}`
		req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		if countDockPrefRows(t, tx, user.ID) != 1 {
			t.Fatalf("应恰好一条 Dock 偏好记录")
		}
	})

	t.Run("desktop_position为null恢复默认", func(t *testing.T) {
		body := `{"desktop_position":null,"mobile_expanded":true}`
		req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		pref := decodeDockPrefResponse(t, resp)
		if pref.DesktopPosition != nil || !pref.MobileExpanded {
			t.Fatalf("null 位置应恢复默认且保留 mobile_expanded，实际 %+v", pref)
		}
		// 回读确认
		req = buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		pref = decodeDockPrefResponse(t, resp)
		if pref.DesktopPosition != nil || !pref.MobileExpanded {
			t.Fatalf("回读 null 位置不一致: %+v", pref)
		}
	})

	t.Run("重复PUT幂等且不产生重复行", func(t *testing.T) {
		body := `{"desktop_position":{"left":10,"top":10},"mobile_expanded":true}`
		for i := 0; i < 2; i++ {
			req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusOK {
				t.Fatalf("第 %d 次 PUT 期望 200，实际 %d", i+1, resp.Code)
			}
		}
		if countDockPrefRows(t, tx, user.ID) != 1 {
			t.Fatalf("重复 PUT 后应仍为一条记录")
		}
		req := buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		pref := decodeDockPrefResponse(t, resp)
		if pref.DesktopPosition == nil || *pref.DesktopPosition.Left != 10 || *pref.DesktopPosition.Top != 10 {
			t.Fatalf("幂等 PUT 后回读不一致: %+v", pref)
		}
	})

	t.Run("未鉴权PUT返回401", func(t *testing.T) {
		req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", `{"desktop_position":null,"mobile_expanded":false}`, false, 0)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// 当前用户隔离：A 写入不泄漏给 B，B 写入不影响 A
// ---------------------------------------------------------------------------

func TestDockPreference_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateDockPrefTables(t, tx)

	userA := createTestUser(t, tx, "dock_a", "用户A")
	userB := createTestUser(t, tx, "dock_b", "用户B")
	handler := NewHandler(tx)
	router := newDockPrefTestRouter(t, handler)

	// A 保存自己的位置
	bodyA := `{"desktop_position":{"left":1,"top":2},"mobile_expanded":true}`
	req := buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", bodyA, true, userA.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("A PUT 期望 200，实际 %d", resp.Code)
	}

	// B GET 应看到安全默认（数据隔离）
	req = buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("B GET 期望 200，实际 %d", resp.Code)
	}
	prefB := decodeDockPrefResponse(t, resp)
	if prefB.DesktopPosition != nil || prefB.MobileExpanded {
		t.Fatalf("B 不应看到 A 的偏好，实际 %+v", prefB)
	}

	// B 保存自己的偏好
	bodyB := `{"desktop_position":null,"mobile_expanded":true}`
	req = buildDockPrefRequest(t, http.MethodPut, "/api/user/dock-preferences", bodyB, true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("B PUT 期望 200，实际 %d", resp.Code)
	}

	// A 的偏好应保持原样（B 写入不影响 A）
	req = buildDockPrefRequest(t, http.MethodGet, "/api/user/dock-preferences", "", true, userA.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	prefA := decodeDockPrefResponse(t, resp)
	if prefA.DesktopPosition == nil || *prefA.DesktopPosition.Left != 1 || *prefA.DesktopPosition.Top != 2 || !prefA.MobileExpanded {
		t.Fatalf("A 的偏好应保持不变，实际 %+v", prefA)
	}

	// 各自恰好一条记录
	if countDockPrefRows(t, tx, userA.ID) != 1 || countDockPrefRows(t, tx, userB.ID) != 1 {
		t.Fatalf("A/B 应各有一条自己的偏好记录: A=%d B=%d",
			countDockPrefRows(t, tx, userA.ID), countDockPrefRows(t, tx, userB.ID))
	}
}
