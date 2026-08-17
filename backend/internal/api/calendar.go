package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// 个人日历事件：字段长度与查询范围限制
const (
	calendarMaxBytes         = 64 << 10 // 请求体上限（64 KiB），防 DoS
	calendarTitleMaxLen      = 200      // title 最大长度（按字符计）
	calendarLocationMaxLen   = 200      // location 最大长度（按字符计）
	calendarNotesMaxLen      = 1000     // notes 最大长度（按字符计）
	calendarDefaultRangeDays = 365      // 列表查询默认时间范围（以当前时刻为基准前后天数）
)

// calendarEventPayload 创建/更新日历事件的请求体（严格字段白名单）。
// 时间字段以字符串接收，统一按 RFC3339 解析，错误信息可控。
type calendarEventPayload struct {
	Title    string `json:"title"`
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	Location string `json:"location"`
	Notes    string `json:"notes"`
	AllDay   bool   `json:"all_day"`
}

// calendarEventInput 解码并校验后的日历事件字段（时间已解析为 time.Time）。
type calendarEventInput struct {
	Title    string
	StartAt  time.Time
	EndAt    time.Time
	Location string
	Notes    string
	AllDay   bool
}

// registerCalendarRoutes 注册个人日历路由到 /user 路由组下。
func (h *Handler) registerCalendarRoutes(r chi.Router) {
	r.Get("/calendar", h.listCalendarEvents)
	r.Post("/calendar", h.createCalendarEvent)
	r.Put("/calendar/{id}", h.updateCalendarEvent)
	r.Delete("/calendar/{id}", h.deleteCalendarEvent)
}

// listCalendarEvents GET /api/user/calendar?from=&to=
// 返回当前用户指定时间范围内（缺省为当前时刻前后一年）的日历事件。
// from/to 均为可选 RFC3339 时间，传入即按 start_at 区间过滤；
// 结果按 start_at 升序、id 升序稳定排序，保证顺序可复现。
func (h *Handler) listCalendarEvents(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	from, to, err := parseCalendarRange(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var events []models.CalendarEvent
	if err := h.db.Where("user_id = ? AND start_at >= ? AND start_at <= ?", userID, from, to).
		Order("start_at ASC, id ASC").
		Find(&events).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list calendar events", err)
		return
	}
	if events == nil {
		events = make([]models.CalendarEvent, 0) // 保证空数组而非 null
	}
	respondJSON(w, http.StatusOK, map[string]any{"events": events})
}

// createCalendarEvent POST /api/user/calendar
// 创建当前用户的私有日历事件；user_id 一律取自登录上下文，不接受请求体传入。
func (h *Handler) createCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	input, err := decodeCalendarPayload(w, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	event := models.CalendarEvent{
		UserID:   userID,
		Title:    input.Title,
		StartAt:  input.StartAt,
		EndAt:    input.EndAt,
		Location: input.Location,
		Notes:    input.Notes,
		AllDay:   input.AllDay,
	}
	if err := h.db.Create(&event).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create calendar event", err)
		return
	}
	respondJSON(w, http.StatusCreated, event)
}

// updateCalendarEvent PUT /api/user/calendar/{id}
// 全量更新当前用户的日历事件；不存在或属于其他用户的事件统一返回 404。
func (h *Handler) updateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, err := parseCalendarEventID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid event id", err)
		return
	}
	var event models.CalendarEvent
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "calendar event not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load calendar event", err)
		return
	}
	input, err := decodeCalendarPayload(w, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	event.Title = input.Title
	event.StartAt = input.StartAt
	event.EndAt = input.EndAt
	event.Location = input.Location
	event.Notes = input.Notes
	event.AllDay = input.AllDay
	if err := h.db.Save(&event).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update calendar event", err)
		return
	}
	respondJSON(w, http.StatusOK, event)
}

// deleteCalendarEvent DELETE /api/user/calendar/{id}
// 删除当前用户的日历事件；不存在或属于其他用户的事件统一返回 404。
func (h *Handler) deleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, err := parseCalendarEventID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid event id", err)
		return
	}
	var event models.CalendarEvent
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "calendar event not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load calendar event", err)
		return
	}
	if err := h.db.Delete(&event).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete calendar event", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseCalendarEventID 解析路径参数 {id} 为正整数。
func parseCalendarEventID(r *http.Request) (uint, error) {
	raw := strings.TrimSpace(chi.URLParam(r, "id"))
	if raw == "" {
		return 0, errors.New("event id is required")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid event id")
	}
	return uint(id), nil
}

// parseCalendarRange 解析列表查询的时间范围参数 from/to（RFC3339，均可选）。
// 缺省范围为当前时刻前后各一年；from 晚于 to 视为非法。
func parseCalendarRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now()
	from, err := parseOptionalRFC3339(r.URL.Query().Get("from"), now.AddDate(0, 0, -calendarDefaultRangeDays))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from: %v", err)
	}
	to, err := parseOptionalRFC3339(r.URL.Query().Get("to"), now.AddDate(0, 0, calendarDefaultRangeDays))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to: %v", err)
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from must not be later than to")
	}
	return from, to, nil
}

// parseOptionalRFC3339 按 RFC3339 解析时间；raw 为空时返回默认值。
func parseOptionalRFC3339(raw string, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// decodeCalendarPayload 解码并校验日历事件请求体：
// 严格白名单（拒绝未知字段，包括 user_id）、长度受限、时间合法且 end 不早于 start。
func decodeCalendarPayload(w http.ResponseWriter, r *http.Request) (*calendarEventInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, calendarMaxBytes)

	var payload calendarEventPayload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // 拒绝未知字段，防止任意 key 写入（如 user_id）
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}
	return validateCalendarPayload(&payload)
}

// validateCalendarPayload 校验字段值：title 必填且长度受限；start_at/end_at 必填且为 RFC3339；
// end_at 不得早于 start_at；location/notes 可选且长度受限。通过后返回规范化输入。
func validateCalendarPayload(p *calendarEventPayload) (*calendarEventInput, error) {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len([]rune(title)) > calendarTitleMaxLen {
		return nil, fmt.Errorf("title exceeds max length %d", calendarTitleMaxLen)
	}
	startAt, err := parseRequiredRFC3339(p.StartAt, "start_at")
	if err != nil {
		return nil, err
	}
	endAt, err := parseRequiredRFC3339(p.EndAt, "end_at")
	if err != nil {
		return nil, err
	}
	if endAt.Before(startAt) {
		return nil, errors.New("end_at must not be earlier than start_at")
	}
	location := strings.TrimSpace(p.Location)
	if len([]rune(location)) > calendarLocationMaxLen {
		return nil, fmt.Errorf("location exceeds max length %d", calendarLocationMaxLen)
	}
	notes := strings.TrimSpace(p.Notes)
	if len([]rune(notes)) > calendarNotesMaxLen {
		return nil, fmt.Errorf("notes exceeds max length %d", calendarNotesMaxLen)
	}
	return &calendarEventInput{
		Title:    title,
		StartAt:  startAt,
		EndAt:    endAt,
		Location: location,
		Notes:    notes,
		AllDay:   p.AllDay,
	}, nil
}

// parseRequiredRFC3339 按 RFC3339 解析必填时间字段，空串或格式非法均报错。
func parseRequiredRFC3339(raw, field string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: must be RFC3339 format", field)
	}
	return t, nil
}
