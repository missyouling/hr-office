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

// newCalendarTestRouter 注册个人日历路由（无 JWT 中间件，测试用 setAuthContext 模拟登录）
func newCalendarTestRouter(t *testing.T, handler *Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/user", func(ur chi.Router) {
		ur.Get("/calendar", handler.listCalendarEvents)
		ur.Post("/calendar", handler.createCalendarEvent)
		ur.Put("/calendar/{id}", handler.updateCalendarEvent)
		ur.Delete("/calendar/{id}", handler.deleteCalendarEvent)
	})
	return r
}

// migrateCalendarTables 迁移日历测试所需表
func migrateCalendarTables(t *testing.T, tx *gorm.DB) {
	t.Helper()
	if err := tx.AutoMigrate(&models.User{}, &models.CalendarEvent{}); err != nil {
		t.Fatalf("自动迁移表结构失败: %v", err)
	}
}

// buildCalendarRequest 构造日历请求；auth=true 时写入登录用户上下文
func buildCalendarRequest(t *testing.T, method, path, body string, auth bool, userID uint) *http.Request {
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

// decodeCalendarEventResponse 解析单个事件响应体
func decodeCalendarEventResponse(t *testing.T, resp *httptest.ResponseRecorder) models.CalendarEvent {
	t.Helper()
	var ev models.CalendarEvent
	if err := json.Unmarshal(resp.Body.Bytes(), &ev); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return ev
}

// decodeCalendarListResponse 解析事件列表响应体
func decodeCalendarListResponse(t *testing.T, resp *httptest.ResponseRecorder) []models.CalendarEvent {
	t.Helper()
	var out struct {
		Events []models.CalendarEvent `json:"events"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, resp.Body.String())
	}
	return out.Events
}

// seedCalendarEvent 直接向数据库插入一条日历事件并返回
func seedCalendarEvent(t *testing.T, tx *gorm.DB, userID uint, title string, startAt, endAt time.Time) models.CalendarEvent {
	t.Helper()
	ev := models.CalendarEvent{
		UserID:  userID,
		Title:   title,
		StartAt: startAt,
		EndAt:   endAt,
	}
	if err := tx.Create(&ev).Error; err != nil {
		t.Fatalf("创建日历事件失败: %v", err)
	}
	return ev
}

// ---------------------------------------------------------------------------
// POST：正常创建 / 非法字段与时间 / 白名单
// ---------------------------------------------------------------------------

func TestCalendar_Create(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	user := createTestUser(t, tx, "cal_create", "日历创建用户")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	validBody := `{"title":"项目评审会","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T11:00:00Z","location":"3F 会议室","notes":"季度评审","all_day":false}`

	t.Run("正常创建返回201", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", validBody, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("期望 201，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		ev := decodeCalendarEventResponse(t, resp)
		if ev.ID == 0 || ev.Title != "项目评审会" || ev.Location != "3F 会议室" || ev.Notes != "季度评审" {
			t.Fatalf("事件字段不正确: %+v", ev)
		}
		wantStart, _ := time.Parse(time.RFC3339, "2026-09-01T09:00:00Z")
		if !ev.StartAt.Equal(wantStart) {
			t.Fatalf("start_at 不正确: %v", ev.StartAt)
		}
		if strings.Contains(resp.Body.String(), `"user_id"`) {
			t.Fatalf("响应不应包含 user_id 字段: %s", resp.Body.String())
		}
	})

	t.Run("title缺失返回400", func(t *testing.T) {
		body := `{"start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z"}`
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("title超长返回400", func(t *testing.T) {
		longTitle := strings.Repeat("题", calendarTitleMaxLen+1)
		body := fmt.Sprintf(`{"title":%q,"start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z"}`, longTitle)
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("时间缺失或格式非法返回400", func(t *testing.T) {
		bodies := []string{
			`{"title":"t","end_at":"2026-09-01T10:00:00Z"}`,                         // start_at 缺失
			`{"title":"t","start_at":"2026-09-01T10:00:00Z"}`,                       // end_at 缺失
			`{"title":"t","start_at":"2026/09/01","end_at":"2026-09-01T10:00:00Z"}`, // start_at 非 RFC3339
		}
		for _, body := range bodies {
			req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("body=%s 期望 400，实际 %d", body, resp.Code)
			}
		}
	})

	t.Run("end早于start返回400", func(t *testing.T) {
		body := `{"title":"t","start_at":"2026-09-01T10:00:00Z","end_at":"2026-09-01T09:00:00Z"}`
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("location或notes超长返回400", func(t *testing.T) {
		longLocation := strings.Repeat("地", calendarLocationMaxLen+1)
		body := fmt.Sprintf(`{"title":"t","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z","location":%q}`, longLocation)
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("location 超长期望 400，实际 %d", resp.Code)
		}
		longNotes := strings.Repeat("备", calendarNotesMaxLen+1)
		body = fmt.Sprintf(`{"title":"t","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z","notes":%q}`, longNotes)
		req = buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("notes 超长期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("未知字段(含user_id)返回400", func(t *testing.T) {
		bodies := []string{
			`{"title":"t","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z","user_id":999}`,
			`{"title":"t","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z","hack":1}`,
		}
		for _, body := range bodies {
			req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("body=%s 期望 400，实际 %d", body, resp.Code)
			}
		}
	})

	t.Run("all_day与可选字段缺省", func(t *testing.T) {
		body := `{"title":"全天事件","start_at":"2026-09-01T00:00:00Z","end_at":"2026-09-01T23:59:59Z","all_day":true}`
		req := buildCalendarRequest(t, http.MethodPost, "/api/user/calendar", body, true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("期望 201，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		ev := decodeCalendarEventResponse(t, resp)
		if !ev.AllDay {
			t.Fatalf("all_day 应为 true: %+v", ev)
		}
		if ev.Location != "" || ev.Notes != "" {
			t.Fatalf("可选字段缺省应为空: %+v", ev)
		}
	})
}

// ---------------------------------------------------------------------------
// GET：默认范围 / from-to 过滤 / 稳定排序 / 空数组 / 非法参数
// ---------------------------------------------------------------------------

func TestCalendar_List(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	user := createTestUser(t, tx, "cal_list", "日历列表用户")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	now := time.Now()
	near := seedCalendarEvent(t, tx, user.ID, "近月事件", now.AddDate(0, 0, -30), now.AddDate(0, 0, -30).Add(time.Hour))
	seedCalendarEvent(t, tx, user.ID, "过远事件", now.AddDate(0, 0, -400), now.AddDate(0, 0, -400).Add(time.Hour))
	sameStart := now.AddDate(0, 0, 5)
	first := seedCalendarEvent(t, tx, user.ID, "排序甲", sameStart, sameStart.Add(time.Hour))
	second := seedCalendarEvent(t, tx, user.ID, "排序乙", sameStart, sameStart.Add(time.Hour))

	t.Run("默认范围仅返回一年内事件", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		events := decodeCalendarListResponse(t, resp)
		if len(events) != 3 {
			t.Fatalf("期望 3 条事件，实际 %d: %+v", len(events), events)
		}
	})

	t.Run("按start_at/id稳定排序", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		events := decodeCalendarListResponse(t, resp)
		if events[0].ID != near.ID {
			t.Fatalf("第 1 条应为近月事件(%d)，实际 %d", near.ID, events[0].ID)
		}
		if events[1].ID != first.ID || events[2].ID != second.ID {
			t.Fatalf("相同 start_at 应按 id 升序: %+v", events)
		}
	})

	t.Run("from/to范围过滤", func(t *testing.T) {
		from := now.AddDate(0, 0, -100).UTC().Format(time.RFC3339)
		to := now.AddDate(0, 0, 100).UTC().Format(time.RFC3339)
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar?from="+from+"&to="+to, "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", resp.Code)
		}
		events := decodeCalendarListResponse(t, resp)
		if len(events) != 3 {
			t.Fatalf("范围过滤后期望 3 条，实际 %d: %+v", len(events), events)
		}
	})

	t.Run("仅指定from返回该时刻之后事件", func(t *testing.T) {
		from := now.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar?from="+from, "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		events := decodeCalendarListResponse(t, resp)
		if len(events) != 2 {
			t.Fatalf("仅 from 过滤后期望 2 条，实际 %d: %+v", len(events), events)
		}
	})

	t.Run("非法from或from晚于to返回400", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar?from=2026/09/01", "", true, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("非法 from 期望 400，实际 %d", resp.Code)
		}
		req = buildCalendarRequest(t, http.MethodGet, "/api/user/calendar?from=2026-09-02T00:00:00Z&to=2026-09-01T00:00:00Z", "", true, user.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("from>to 期望 400，实际 %d", resp.Code)
		}
	})

	t.Run("空列表返回空数组", func(t *testing.T) {
		emptyUser := createTestUser(t, tx, "cal_list_empty", "空列表用户")
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar", "", true, emptyUser.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", resp.Code)
		}
		if strings.Contains(resp.Body.String(), `"events":null`) {
			t.Fatalf("events 不应为 null: %s", resp.Body.String())
		}
		events := decodeCalendarListResponse(t, resp)
		if events == nil || len(events) != 0 {
			t.Fatalf("期望空数组，实际 %+v", events)
		}
	})
}

// ---------------------------------------------------------------------------
// PUT：正常更新 / 不存在 404 / 他人 404 / 非法 payload
// ---------------------------------------------------------------------------

func TestCalendar_Update(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	userA := createTestUser(t, tx, "cal_up_a", "更新用户A")
	userB := createTestUser(t, tx, "cal_up_b", "更新用户B")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	start := time.Now().AddDate(0, 0, 1)
	ev := seedCalendarEvent(t, tx, userA.ID, "原标题", start, start.Add(time.Hour))

	t.Run("正常更新返回200", func(t *testing.T) {
		body := `{"title":"新标题","start_at":"2026-10-01T09:00:00Z","end_at":"2026-10-01T12:00:00Z","location":"新地点","notes":"新备注","all_day":true}`
		path := fmt.Sprintf("/api/user/calendar/%d", ev.ID)
		req := buildCalendarRequest(t, http.MethodPut, path, body, true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d，body=%s", resp.Code, resp.Body.String())
		}
		updated := decodeCalendarEventResponse(t, resp)
		if updated.Title != "新标题" || updated.Location != "新地点" || updated.Notes != "新备注" || !updated.AllDay {
			t.Fatalf("更新字段不正确: %+v", updated)
		}
		var stored models.CalendarEvent
		if err := tx.First(&stored, ev.ID).Error; err != nil {
			t.Fatalf("查询存储记录失败: %v", err)
		}
		if stored.Title != "新标题" || stored.UserID != userA.ID {
			t.Fatalf("存储记录不正确: %+v", stored)
		}
	})

	t.Run("不存在事件返回404", func(t *testing.T) {
		body := `{"title":"t","start_at":"2026-10-01T09:00:00Z","end_at":"2026-10-01T10:00:00Z"}`
		req := buildCalendarRequest(t, http.MethodPut, "/api/user/calendar/99999", body, true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
	})

	t.Run("他人事件返回404且不生效", func(t *testing.T) {
		body := `{"title":"越权修改","start_at":"2026-10-01T09:00:00Z","end_at":"2026-10-01T10:00:00Z"}`
		path := fmt.Sprintf("/api/user/calendar/%d", ev.ID)
		req := buildCalendarRequest(t, http.MethodPut, path, body, true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
		var stored models.CalendarEvent
		if err := tx.First(&stored, ev.ID).Error; err != nil {
			t.Fatalf("查询存储记录失败: %v", err)
		}
		if stored.Title != "新标题" {
			t.Fatalf("他人更新不应生效: %+v", stored)
		}
	})

	t.Run("非法payload返回400", func(t *testing.T) {
		body := `{"title":"","start_at":"2026-10-01T09:00:00Z","end_at":"2026-10-01T10:00:00Z"}`
		path := fmt.Sprintf("/api/user/calendar/%d", ev.ID)
		req := buildCalendarRequest(t, http.MethodPut, path, body, true, userA.ID)
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

func TestCalendar_Delete(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	userA := createTestUser(t, tx, "cal_del_a", "删除用户A")
	userB := createTestUser(t, tx, "cal_del_b", "删除用户B")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	start := time.Now().AddDate(0, 0, 2)
	ev := seedCalendarEvent(t, tx, userA.ID, "待删除", start, start.Add(time.Hour))

	t.Run("正常删除返回204", func(t *testing.T) {
		path := fmt.Sprintf("/api/user/calendar/%d", ev.ID)
		req := buildCalendarRequest(t, http.MethodDelete, path, "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNoContent {
			t.Fatalf("期望 204，实际 %d", resp.Code)
		}
		var count int64
		if err := tx.Model(&models.CalendarEvent{}).Where("id = ?", ev.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 0 {
			t.Fatalf("删除后记录应不存在，实际 %d 条", count)
		}
	})

	t.Run("不存在事件返回404", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodDelete, "/api/user/calendar/99999", "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
	})

	t.Run("他人事件返回404且不生效", func(t *testing.T) {
		other := seedCalendarEvent(t, tx, userA.ID, "他人视角事件", start, start.Add(time.Hour))
		path := fmt.Sprintf("/api/user/calendar/%d", other.ID)
		req := buildCalendarRequest(t, http.MethodDelete, path, "", true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", resp.Code)
		}
		var count int64
		if err := tx.Model(&models.CalendarEvent{}).Where("id = ?", other.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("他人删除不应生效，实际 %d 条", count)
		}
	})

	t.Run("非法id返回400", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodDelete, "/api/user/calendar/abc", "", true, userA.ID)
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

func TestCalendar_Auth(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	user := createTestUser(t, tx, "cal_auth", "鉴权用户")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	body := `{"title":"t","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z"}`
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/user/calendar", ""},
		{http.MethodPost, "/api/user/calendar", body},
		{http.MethodPut, "/api/user/calendar/1", body},
		{http.MethodDelete, "/api/user/calendar/1", ""},
	}
	for _, c := range cases {
		req := buildCalendarRequest(t, c.method, c.path, c.body, false, user.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 未登录期望 401，实际 %d", c.method, c.path, resp.Code)
		}
	}
}

func TestCalendar_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	tx := newTestTransaction(t, db)
	migrateCalendarTables(t, tx)

	userA := createTestUser(t, tx, "cal_iso_a", "隔离用户A")
	userB := createTestUser(t, tx, "cal_iso_b", "隔离用户B")
	handler := NewHandler(tx)
	router := newCalendarTestRouter(t, handler)

	start := time.Now().AddDate(0, 0, 3)
	evA := seedCalendarEvent(t, tx, userA.ID, "A的事件", start, start.Add(time.Hour))
	evB := seedCalendarEvent(t, tx, userB.ID, "B的事件", start, start.Add(time.Hour))

	t.Run("A列表仅见A事件", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar", "", true, userA.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		events := decodeCalendarListResponse(t, resp)
		if len(events) != 1 || events[0].ID != evA.ID {
			t.Fatalf("A 应仅看到自己的事件: %+v", events)
		}
	})

	t.Run("B列表仅见B事件", func(t *testing.T) {
		req := buildCalendarRequest(t, http.MethodGet, "/api/user/calendar", "", true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		events := decodeCalendarListResponse(t, resp)
		if len(events) != 1 || events[0].ID != evB.ID {
			t.Fatalf("B 应仅看到自己的事件: %+v", events)
		}
	})

	t.Run("B操作A事件返回404", func(t *testing.T) {
		path := fmt.Sprintf("/api/user/calendar/%d", evA.ID)
		req := buildCalendarRequest(t, http.MethodPut, path, `{"title":"x","start_at":"2026-09-01T09:00:00Z","end_at":"2026-09-01T10:00:00Z"}`, true, userB.ID)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("PUT 他人事件期望 404，实际 %d", resp.Code)
		}
		req = buildCalendarRequest(t, http.MethodDelete, path, "", true, userB.ID)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("DELETE 他人事件期望 404，实际 %d", resp.Code)
		}
		// 确认 A 的事件未被删除
		var count int64
		if err := tx.Model(&models.CalendarEvent{}).Where("id = ?", evA.ID).Count(&count).Error; err != nil {
			t.Fatalf("统计失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("A 的事件应保留，实际 %d 条", count)
		}
	})
}
