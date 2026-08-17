package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// dockPrefMaxBytes Dock 偏好请求体上限（4 KiB），防 DoS
const dockPrefMaxBytes = 4 << 10

// dockPositionMaxValue Dock 位置像素坐标上限（防不合理数值）
const dockPositionMaxValue = 100000.0

// registerDockPreferenceRoutes 注册桌面 Dock 偏好路由到 /user 路由组下。
func (h *Handler) registerDockPreferenceRoutes(r chi.Router) {
	r.Get("/dock-preferences", h.getDockPreferences)
	r.Put("/dock-preferences", h.updateDockPreferences)
}

// getDockPreferences GET /api/user/dock-preferences
// 返回当前用户的 Dock 偏好；无记录时返回安全默认
// {"desktop_position":null,"mobile_expanded":false}。
// 始终按当前登录用户隔离，不接受任何 user_id 参数。
func (h *Handler) getDockPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	pref, err := h.loadDockPreferences(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load dock preferences", err)
		return
	}
	respondJSON(w, http.StatusOK, pref)
}

// updateDockPreferences PUT /api/user/dock-preferences
// 全量保存当前用户的 Dock 偏好：仅允许 desktop_position/mobile_expanded 两个
// 白名单字段，拒绝未知字段（含 user_id）、空 payload 与非法数值；
// desktop_position 允许为 null（恢复默认）。使用原子 UPSERT 落库 UserPreference。
func (h *Handler) updateDockPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, dockPrefMaxBytes))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(bytes.TrimSpace(body)) == 0 {
		respondError(w, http.StatusBadRequest, "empty payload", nil)
		return
	}
	// 先按 map 解析以识别空对象 {}（结构体解码无法区分缺省与零值）
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(raw) == 0 {
		respondError(w, http.StatusBadRequest, "empty payload", nil)
		return
	}
	// 严格白名单解码：拒绝未知字段（含 user_id）
	var pref models.DockPreference
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pref); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := ensureJSONEOF(dec); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateDockPreference(&pref); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.saveDockPreferences(userID, &pref); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save dock preferences", err)
		return
	}
	respondJSON(w, http.StatusOK, pref)
}

// validateDockPreference 校验 Dock 偏好字段值：
// desktop_position 为 null 合法（恢复默认）；非 null 时 left/top 必须提供、
// 不得为负、不得超过上限。mobile_expanded 由 bool 类型保证必为布尔。
func validateDockPreference(p *models.DockPreference) error {
	if p.DesktopPosition == nil {
		return nil
	}
	if p.DesktopPosition.Left == nil || p.DesktopPosition.Top == nil {
		return errors.New("desktop_position.left and desktop_position.top are required")
	}
	if *p.DesktopPosition.Left < 0 || *p.DesktopPosition.Top < 0 {
		return errors.New("desktop_position values must not be negative")
	}
	if *p.DesktopPosition.Left > dockPositionMaxValue || *p.DesktopPosition.Top > dockPositionMaxValue {
		return fmt.Errorf("desktop_position values exceed max %v", dockPositionMaxValue)
	}
	return nil
}

// loadDockPreferences 读取当前用户的 Dock 偏好；无记录时返回安全默认值。
func (h *Handler) loadDockPreferences(userID uint) (*models.DockPreference, error) {
	var pref models.UserPreference
	err := h.db.Where("user_id = ? AND pref_key = ?", userID, models.DockPreferenceKey).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.DockPreference{}, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := &models.DockPreference{}
	if len(pref.Value) > 0 {
		if err := json.Unmarshal(pref.Value, cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// saveDockPreferences 全量写入当前用户的 Dock 偏好（UPSERT，复用 UserPreference 表）。
// 使用 ON CONFLICT (user_id, pref_key) DO UPDATE 实现原子 upsert，避免重复记录。
func (h *Handler) saveDockPreferences(userID uint, cfg *models.DockPreference) error {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	pref := models.UserPreference{
		UserID:  &userID,
		PrefKey: models.DockPreferenceKey,
		Value:   datatypes.JSON(bytes),
	}
	return h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "pref_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&pref).Error
}
