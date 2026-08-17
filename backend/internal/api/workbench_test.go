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

// newWorkbenchTestRouter 注册工作台配置路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newWorkbenchTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/workbench-config", handler.getWorkbenchConfig)
		ur.Put("/workbench-config", handler.updateWorkbenchConfig)
	})
	return r
}

// migrateWorkbenchTables 迁移工作台配置测试所需表
func migrateWorkbenchTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(&models.User{}, &models.UserPreference{}); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildWorkbenchRequest 构造工作台配置请求；auth=true 时写入登录用户上下文
func buildWorkbenchRequest(t *testing.T, method, path, body string, auth bool, userID uint) *http.Request {
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

// decodeWorkbenchResponse 解析响应体为 WorkbenchConfig
func decodeWorkbenchResponse(t *testing.T, resp *httptest.ResponseRecorder) models.WorkbenchConfig {
	t.Helper()
	var cfg models.WorkbenchConfig
	if err := json.Unmarshal(resp.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return cfg
}

// countWorkbenchPrefRows 统计 user_preferences 表中工作台配置记录数
func countWorkbenchPrefRows(t *testing.T, tx *gorm.DB, userID uint) int64 {
	t.Helper()
	var count int64
	err := tx.Model(&models.UserPreference{}).
		Where("user_id = ? AND pref_key = ?", userID, models.WorkbenchConfigPrefKey).
		Count(&count).Error
	if err != nil {
		t.Fatalf("统计工作台配置记录失败: %v", err)
	}
	return count
}

// ---------------------------------------------------------------------------
// GET：空状态 / 已保存回读 / 未鉴权
// ---------------------------------------------------------------------------

func TestWorkbenchConfig_Get(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchTables(t, tx)

	user := createTestUser(t, tx, "wbgetuser", "工作台用户")
	handler := NewHandler(tx)
	router := newWorkbenchTestRouter(t, handler)

	t.Run("未鉴权返回401", func(t *testing.T) {
		req := buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", false, 0)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", resp.Code)
		}
	})

	t.Run("无配置返回空状态", func(t *testing.T) {
		req := buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		cfg := decodeWorkbenchResponse(t, resp)
		if cfg.Weather != nil || cfg.News != nil {
			t.Fatalf("无配置时应返回空状态，实际 %+v", cfg)
		}
	})

	t.Run("保存后GET回读一致", func(t *testing.T) {
		putBody := `{"weather":{"enabled":true,"city":"北京"},"news":{"enabled":false,"categories":["国内","科技"]}}`
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", putBody, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}

		req = buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET 期望 200，实际 %d", resp.Code)
		}
		cfg := decodeWorkbenchResponse(t, resp)
		if cfg.Weather == nil || !cfg.Weather.Enabled || cfg.Weather.City != "北京" {
			t.Fatalf("天气配置回读不一致: %+v", cfg.Weather)
		}
		if cfg.News == nil || cfg.News.Enabled || len(cfg.News.Categories) != 2 {
			t.Fatalf("新闻配置回读不一致: %+v", cfg.News)
		}
	})
}

// ---------------------------------------------------------------------------
// PUT：非法输入（非法JSON/未知字段/字段值不合法/冗余内容）
// ---------------------------------------------------------------------------

func TestWorkbenchConfig_PutInvalidInput(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchTables(t, tx)

	user := createTestUser(t, tx, "wbinvalid", "非法输入用户")
	handler := NewHandler(tx)
	router := newWorkbenchTestRouter(t, handler)

	invalidCases := []struct {
		name string
		body string
	}{
		{"非法JSON", `{"weather":`},
		{"未知顶层字段", `{"foo":1}`},
		{"weather内未知字段", `{"weather":{"enabled":true,"city":"北京","temperature":30}}`},
		{"news内未知字段", `{"news":{"enabled":true,"categories":["科技"],"refresh":1}}`},
		{"weather.city为空", `{"weather":{"enabled":true,"city":"  "}}`},
		{"weather.city超长", `{"weather":{"enabled":true,"city":"` + strings.Repeat("城", 51) + `"}}`},
		{"news.categories超数量", `{"news":{"enabled":true,"categories":[` + strings.Repeat(`"a",`, 10) + `"b"]}}`},
		{"news.categories含空项", `{"news":{"enabled":true,"categories":["科技","  "]}}`},
		{"news.categories项超长", `{"news":{"enabled":true,"categories":["` + strings.Repeat("长", 21) + `"]}}`},
		{"JSON后冗余内容", `{"weather":null} garbage`},
		{"请求体过大", `{"news":{"enabled":true,"categories":["` + strings.Repeat("x", 70<<10) + `"]}}`},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", tc.body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("期望 400，实际 %d，body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PUT：正常保存 / 覆盖更新 / 清空 / 规范化
// ---------------------------------------------------------------------------

func TestWorkbenchConfig_PutValid(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchTables(t, tx)

	user := createTestUser(t, tx, "wbvalid", "合法输入用户")
	handler := NewHandler(tx)
	router := newWorkbenchTestRouter(t, handler)

	t.Run("保存后落库仅一行且值正确", func(t *testing.T) {
		body := `{"weather":{"enabled":true,"city":" 北京 "}}`
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		cfg := decodeWorkbenchResponse(t, resp)
		// city 首尾空白应被规范化去除
		if cfg.Weather == nil || cfg.Weather.City != "北京" {
			t.Fatalf("city 应被规范化，实际 %+v", cfg.Weather)
		}
		if countWorkbenchPrefRows(t, tx, user.ID) != 1 {
			t.Fatalf("应恰好一条工作台配置记录")
		}
	})

	t.Run("覆盖更新不产生重复行", func(t *testing.T) {
		body1 := `{"weather":{"enabled":true,"city":"北京"}}`
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body1, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("首次 PUT 期望 200，实际 %d", resp.Code)
		}
		body2 := `{"weather":{"enabled":false,"city":"上海"},"news":{"enabled":true,"categories":["财经"]}}`
		req = buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body2, true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("二次 PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		if countWorkbenchPrefRows(t, tx, user.ID) != 1 {
			t.Fatalf("覆盖更新后应仍为一条记录")
		}
		cfg := decodeWorkbenchResponse(t, resp)
		if cfg.Weather == nil || cfg.Weather.Enabled || cfg.Weather.City != "上海" {
			t.Fatalf("二次 PUT 应覆盖首次配置: %+v", cfg.Weather)
		}
	})

	t.Run("空对象清空配置", func(t *testing.T) {
		body := `{}`
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d", resp.Code)
		}
		req = buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET 期望 200，实际 %d", resp.Code)
		}
		cfg := decodeWorkbenchResponse(t, resp)
		if cfg.Weather != nil || cfg.News != nil {
			t.Fatalf("清空后应返回空状态，实际 %+v", cfg)
		}
	})

	t.Run("部分模块为空允许保存", func(t *testing.T) {
		body := `{"weather":null,"news":{"enabled":true,"categories":["科技"]}}`
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT 期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		cfg := decodeWorkbenchResponse(t, resp)
		if cfg.Weather != nil || cfg.News == nil || len(cfg.News.Categories) != 1 {
			t.Fatalf("部分模块配置回读不一致: %+v", cfg)
		}
	})

	t.Run("未鉴权PUT返回401", func(t *testing.T) {
		req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", `{"weather":null}`, false, 0)
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

func TestWorkbenchConfig_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateWorkbenchTables(t, tx)

	userA := createTestUser(t, tx, "wbuser_a", "用户A")
	userB := createTestUser(t, tx, "wbuser_b", "用户B")
	handler := NewHandler(tx)
	router := newWorkbenchTestRouter(t, handler)

	// A 保存天气配置
	body := `{"weather":{"enabled":true,"city":"北京"}}`
	req := buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", body, true, userA.ID)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("A PUT 期望 200，实际 %d", resp.Code)
	}

	// B GET 应看到空状态（数据隔离）
	req = buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("B GET 期望 200，实际 %d", resp.Code)
	}
	cfgB := decodeWorkbenchResponse(t, resp)
	if cfgB.Weather != nil || cfgB.News != nil {
		t.Fatalf("B 不应看到 A 的配置，实际 %+v", cfgB)
	}

	// B 保存自己的新闻配置
	bodyB := `{"news":{"enabled":true,"categories":["财经"]}}`
	req = buildWorkbenchRequest(t, http.MethodPut, "/api/user/workbench-config", bodyB, true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("B PUT 期望 200，实际 %d", resp.Code)
	}

	// A 的配置应保持原样（B 写入不影响 A）
	req = buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, userA.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("A GET 期望 200，实际 %d", resp.Code)
	}
	cfgA := decodeWorkbenchResponse(t, resp)
	if cfgA.Weather == nil || cfgA.Weather.City != "北京" {
		t.Fatalf("A 的配置应保持不变，实际 %+v", cfgA.Weather)
	}
	if cfgA.News != nil {
		t.Fatalf("A 不应被写入 B 的新闻配置，实际 %+v", cfgA.News)
	}

	// B 自己的配置正确
	req = buildWorkbenchRequest(t, http.MethodGet, "/api/user/workbench-config", "", true, userB.ID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	cfgB = decodeWorkbenchResponse(t, resp)
	if cfgB.News == nil || len(cfgB.News.Categories) != 1 {
		t.Fatalf("B 的配置应正确保存，实际 %+v", cfgB)
	}
	if countWorkbenchPrefRows(t, tx, userA.ID) != 1 || countWorkbenchPrefRows(t, tx, userB.ID) != 1 {
		t.Fatalf("A/B 应各有一条自己的配置记录: A=%d B=%d",
			countWorkbenchPrefRows(t, tx, userA.ID), countWorkbenchPrefRows(t, tx, userB.ID))
	}
}
