package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// workbenchConfigMaxBytes 工作台配置请求体上限（64 KiB），防 DoS
const workbenchConfigMaxBytes = 64 << 10

// 工作台配置字段校验限制
const (
	workbenchCityMaxLen    = 50 // 城市名最大长度（按字符计）
	workbenchMaxCategories = 10 // 新闻分类最大数量
	workbenchCatMaxLen     = 20 // 单个新闻分类最大长度（按字符计）
)

func (h *Handler) registerWorkbenchRoutes(r chi.Router) {
	r.Get("/workbench-config", h.getWorkbenchConfig)
	r.Put("/workbench-config", h.updateWorkbenchConfig)
	r.Get("/workbench-reminders", h.getWorkbenchReminders)
}

// getWorkbenchConfig GET /api/user/workbench-config
// 返回当前用户的工作台配置；未配置时返回空状态（weather/news 均为 null）。
// 始终按当前登录用户隔离，不接受任何 user_id 参数。
func (h *Handler) getWorkbenchConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	cfg, err := h.loadWorkbenchConfig(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load workbench config", err)
		return
	}
	respondJSON(w, http.StatusOK, cfg)
}

// updateWorkbenchConfig PUT /api/user/workbench-config
// 全量保存当前用户的工作台配置：仅允许 weather/news 两个白名单字段，
// 未知字段或字段值不合法返回 400；配置属于当前用户私有，不影响其他用户。
// 空对象 {} 表示清空配置（保存后 GET 返回空状态）。
func (h *Handler) updateWorkbenchConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, workbenchConfigMaxBytes)

	var cfg models.WorkbenchConfig
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // 拒绝未知字段，防止任意 key 写入造成范围失控
	if err := dec.Decode(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := ensureJSONEOF(dec); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateWorkbenchConfig(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.saveWorkbenchConfig(userID, &cfg); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save workbench config", err)
		return
	}
	respondJSON(w, http.StatusOK, cfg)
}

// ensureJSONEOF 校验解码器已到达输入末尾，拒绝第一个 JSON 之后的冗余内容。
func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("unexpected trailing data")
	}
	return nil
}

// validateWorkbenchConfig 校验工作台配置字段值：白名单字段已由 DisallowUnknownFields
// 保证，此处校验值合法性（非空/长度/数量边界），并规范化去除首尾空白。
func validateWorkbenchConfig(cfg *models.WorkbenchConfig) error {
	if cfg.Weather != nil {
		city := strings.TrimSpace(cfg.Weather.City)
		if city == "" {
			return errors.New("weather.city is required")
		}
		if len([]rune(city)) > workbenchCityMaxLen {
			return fmt.Errorf("weather.city exceeds max length %d", workbenchCityMaxLen)
		}
		cfg.Weather.City = city
	}
	if cfg.News != nil {
		if len(cfg.News.Categories) > workbenchMaxCategories {
			return fmt.Errorf("news.categories exceeds max count %d", workbenchMaxCategories)
		}
		for i, cat := range cfg.News.Categories {
			cat = strings.TrimSpace(cat)
			if cat == "" {
				return errors.New("news.categories contains empty item")
			}
			if len([]rune(cat)) > workbenchCatMaxLen {
				return fmt.Errorf("news.categories item exceeds max length %d", workbenchCatMaxLen)
			}
			cfg.News.Categories[i] = cat
		}
	}
	return nil
}

// loadWorkbenchConfig 读取当前用户的工作台配置；无记录时返回空状态。
func (h *Handler) loadWorkbenchConfig(userID uint) (*models.WorkbenchConfig, error) {
	var pref models.UserPreference
	err := h.db.Where("user_id = ? AND pref_key = ?", userID, models.WorkbenchConfigPrefKey).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.WorkbenchConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := &models.WorkbenchConfig{}
	if len(pref.Value) > 0 {
		if err := json.Unmarshal(pref.Value, cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// saveWorkbenchConfig 全量写入当前用户的工作台配置（UPSERT，兼容现有 UserPreference 表）。
// 使用 ON CONFLICT (user_id, pref_key) DO UPDATE 实现原子 upsert，避免重复记录。
func (h *Handler) saveWorkbenchConfig(userID uint, cfg *models.WorkbenchConfig) error {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	pref := models.UserPreference{
		UserID:  &userID,
		PrefKey: models.WorkbenchConfigPrefKey,
		Value:   datatypes.JSON(bytes),
	}
	return h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "pref_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&pref).Error
}
