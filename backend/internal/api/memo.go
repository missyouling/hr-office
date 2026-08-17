package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// 个人备忘录：字段长度与列表查询限制
const (
	memoMaxBytes      = 64 << 10 // 请求体上限（64 KiB），防 DoS
	memoTitleMaxLen   = 200      // title 最大长度（按字符计）
	memoContentMaxLen = 5000     // content 最大长度（按字符计）
	memoDefaultLimit  = 50       // 列表默认返回条数
	memoMaxLimit      = 100      // 列表最大返回条数
)

// memoPayload 创建/更新备忘录的请求体（严格字段白名单）。
type memoPayload struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Pinned    bool   `json:"pinned"`
	Completed bool   `json:"completed"`
}

// memoInput 解码并校验后的备忘录字段。
type memoInput struct {
	Title     string
	Content   string
	Pinned    bool
	Completed bool
}

// registerMemoRoutes 注册个人备忘录路由到 /user 路由组下。
func (h *Handler) registerMemoRoutes(r chi.Router) {
	r.Get("/memos", h.listMemos)
	r.Post("/memos", h.createMemo)
	r.Put("/memos/{id}", h.updateMemo)
	r.Delete("/memos/{id}", h.deleteMemo)
}

// listMemos GET /api/user/memos?limit=
// 返回当前用户的最近备忘录，按 pinned 置顶优先、updated_at 降序、id 降序稳定排序；
// limit 可选，默认 50、上限 100，非法值返回 400。
func (h *Handler) listMemos(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	limit, err := parseMemoLimit(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var memos []models.Memo
	if err := h.db.Where("user_id = ?", userID).
		Order("pinned DESC, updated_at DESC, id DESC").
		Limit(limit).
		Find(&memos).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list memos", err)
		return
	}
	if memos == nil {
		memos = make([]models.Memo, 0) // 保证空数组而非 null
	}
	respondJSON(w, http.StatusOK, map[string]any{"memos": memos})
}

// createMemo POST /api/user/memos
// 创建当前用户的私有备忘录；user_id 一律取自登录上下文，不接受请求体传入。
func (h *Handler) createMemo(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	input, err := decodeMemoPayload(w, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	memo := models.Memo{
		UserID:    userID,
		Title:     input.Title,
		Content:   input.Content,
		Pinned:    input.Pinned,
		Completed: input.Completed,
	}
	if err := h.db.Create(&memo).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create memo", err)
		return
	}
	respondJSON(w, http.StatusCreated, memo)
}

// updateMemo PUT /api/user/memos/{id}
// 全量更新当前用户的备忘录；不存在或属于其他用户的备忘录统一返回 404。
func (h *Handler) updateMemo(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, err := parseMemoID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memo id", err)
		return
	}
	var memo models.Memo
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&memo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "memo not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load memo", err)
		return
	}
	input, err := decodeMemoPayload(w, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	memo.Title = input.Title
	memo.Content = input.Content
	memo.Pinned = input.Pinned
	memo.Completed = input.Completed
	if err := h.db.Save(&memo).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update memo", err)
		return
	}
	respondJSON(w, http.StatusOK, memo)
}

// deleteMemo DELETE /api/user/memos/{id}
// 删除当前用户的备忘录；不存在或属于其他用户的备忘录统一返回 404。
func (h *Handler) deleteMemo(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, err := parseMemoID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memo id", err)
		return
	}
	var memo models.Memo
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&memo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "memo not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load memo", err)
		return
	}
	if err := h.db.Delete(&memo).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete memo", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseMemoID 解析路径参数 {id} 为正整数。
func parseMemoID(r *http.Request) (uint, error) {
	raw := strings.TrimSpace(chi.URLParam(r, "id"))
	if raw == "" {
		return 0, errors.New("memo id is required")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid memo id")
	}
	return uint(id), nil
}

// parseMemoLimit 解析列表查询参数 limit：可选，默认 memoDefaultLimit，上限 memoMaxLimit。
func parseMemoLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return memoDefaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("invalid limit: must be a positive integer")
	}
	if limit > memoMaxLimit {
		return memoMaxLimit, nil
	}
	return limit, nil
}

// decodeMemoPayload 解码并校验备忘录请求体：
// 严格白名单（拒绝未知字段，包括 user_id）、长度受限。
func decodeMemoPayload(w http.ResponseWriter, r *http.Request) (*memoInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, memoMaxBytes)

	var payload memoPayload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // 拒绝未知字段，防止任意 key 写入（如 user_id）
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}
	return validateMemoPayload(&payload)
}

// validateMemoPayload 校验字段值：title 必填且长度受限；content 可选且长度受限。
// 通过后返回规范化输入（去除首尾空白）。
func validateMemoPayload(p *memoPayload) (*memoInput, error) {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len([]rune(title)) > memoTitleMaxLen {
		return nil, fmt.Errorf("title exceeds max length %d", memoTitleMaxLen)
	}
	content := strings.TrimSpace(p.Content)
	if len([]rune(content)) > memoContentMaxLen {
		return nil, fmt.Errorf("content exceeds max length %d", memoContentMaxLen)
	}
	return &memoInput{
		Title:     title,
		Content:   content,
		Pinned:    p.Pinned,
		Completed: p.Completed,
	}, nil
}
