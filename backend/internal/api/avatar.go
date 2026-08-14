package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service/avatar"
)

// 头像上传限制
const (
	// avatarMaxFileBytes 头像文件内容最大 2 MiB
	avatarMaxFileBytes = 2 << 20
	// avatarMaxRequestBytes MaxBytesReader 限制整个 multipart 请求体（文件 + 边界开销），防 DoS
	avatarMaxRequestBytes = avatarMaxFileBytes + (1 << 20)
)

// avatarExtByContentType 真实 MIME → 存储扩展名映射（仅允许 png/jpeg/webp）
var avatarExtByContentType = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

// avatarMetadataResponse 头像元数据响应（不泄露宿主机绝对路径）
type avatarMetadataResponse struct {
	Type              string `json:"type"`
	Seed              string `json:"seed"`
	CustomFileID      *uint  `json:"custom_file_id,omitempty"`
	CustomContentType string `json:"custom_content_type,omitempty"`
	ContentType       string `json:"content_type"` // 当前头像实际 MIME
}

func (h *Handler) registerAvatarRoutes(r chi.Router) {
	r.Get("/avatar", h.getAvatar)
	r.Post("/avatar", h.uploadAvatar)
	r.Delete("/avatar", h.resetAvatar)
}

// getAvatar GET /api/user/avatar
// 返回当前登录用户的头像二进制：自定义头像读文件，否则生成默认 SVG。
// 始终使用当前身份，不接受任何 user_id 参数，禁止访问他人文件。
func (h *Handler) getAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if h.avatarStore == nil {
		respondError(w, http.StatusInternalServerError, "avatar storage unavailable", nil)
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		respondError(w, http.StatusNotFound, "user not found", err)
		return
	}

	pref, err := h.getAvatarPreference(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load avatar preference", err)
		return
	}

	// 自定义头像：读取文件返回
	if pref.Type == models.AvatarTypeCustom && pref.CustomFileID != nil {
		var sf models.SysFile
		// 校验归属：仅允许读取属于当前用户的头像元数据
		if err := h.db.Where("id = ? AND created_by = ?", *pref.CustomFileID, userID).First(&sf).Error; err == nil {
			data, readErr := h.avatarStore.Read(sf.Path)
			if readErr == nil {
				writeAvatarBinary(w, sf.ContentType, data)
				return
			}
			// 文件缺失：记录错误、重置偏好为默认并清理失效引用，不返回 500（避免头像功能整体不可用）
			log.Printf("[avatar] custom file %d missing for user %d: %v", sf.ID, userID, readErr)
			h.resetAvatarPreferenceToDefault(userID, pref, &sf)
		} else {
			log.Printf("[avatar] custom file %d not owned by user %d: %v", *pref.CustomFileID, userID, err)
			// 元数据失效（不存在或非本人）：仅重置偏好，无法安全定位物理文件故不清理
			h.resetAvatarPreferenceToDefault(userID, pref, nil)
		}
	}

	// 默认 SVG：种子缺失时生成并持久化（保持同一用户稳定）
	seed := pref.Seed
	if seed == "" {
		seed = avatar.GenerateSeed(userID, user.Username)
		pref.Seed = seed
		pref.Type = models.AvatarTypeDefault
		pref.CustomFileID = nil
		pref.CustomContentType = ""
		if err := h.saveAvatarPreference(userID, pref); err != nil {
			log.Printf("[avatar] failed to persist seed for user %d: %v", userID, err)
		}
	}

	displayName := user.FullName
	if displayName == "" {
		displayName = user.Username
	}
	svg := avatar.DefaultSVG(seed, avatar.Initial(displayName))
	writeAvatarBinary(w, "image/svg+xml", []byte(svg))
}

// uploadAvatar POST /api/user/avatar
// multipart 上传自定义头像（字段名 file）。服务端校验真实 MIME 与大小，
// 文件名由服务端生成，只写入专用头像根目录。
func (h *Handler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if h.avatarStore == nil {
		respondError(w, http.StatusInternalServerError, "avatar storage unavailable", nil)
		return
	}

	// MaxBytesReader 限制请求体大小，防止超大上传拖垮服务
	r.Body = http.MaxBytesReader(w, r.Body, avatarMaxRequestBytes)
	if err := r.ParseMultipartForm(avatarMaxRequestBytes); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file field 'file' is required", err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read file", err)
		return
	}
	if len(data) == 0 {
		respondError(w, http.StatusBadRequest, "empty file", nil)
		return
	}
	if int64(len(data)) > avatarMaxFileBytes {
		respondError(w, http.StatusRequestEntityTooLarge, "file too large (max 2 MiB)", nil)
		return
	}

	// 校验真实 MIME（基于文件内容魔数，不信任文件名/Content-Type 头）
	contentType := http.DetectContentType(data)
	ext, ok := avatarExtByContentType[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported image type (png/jpeg/webp only)", nil)
		return
	}

	// 写入专用头像根目录，返回服务端生成的相对路径
	relPath, err := h.avatarStore.Save(userID, ext, data)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save avatar", err)
		return
	}

	// 记录 SysFile 元数据（CreatedBy 标记归属）
	hash := md5.Sum(data)
	sysFile := &models.SysFile{
		StorageType:  "local",
		Path:         relPath,
		OriginalName: header.Filename,
		Size:         int64(len(data)),
		ContentType:  contentType,
		ETag:         hex.EncodeToString(hash[:]),
		CreatedBy:    &userID,
	}
	if err := h.db.Create(sysFile).Error; err != nil {
		_ = h.avatarStore.Delete(relPath) // 回滚新文件，避免孤儿文件
		respondError(w, http.StatusInternalServerError, "failed to record avatar metadata", err)
		return
	}

	// 读取旧偏好（用于替换后清理）
	oldPref, err := h.getAvatarPreference(userID)
	if err != nil {
		_ = h.avatarStore.Delete(relPath)
		_ = h.db.Delete(sysFile)
		respondError(w, http.StatusInternalServerError, "failed to load avatar preference", err)
		return
	}

	// 先写新状态（偏好指向新文件），成功后再清理旧文件
	newPref := &models.AvatarPreference{
		Seed:              oldPref.Seed,
		Type:              models.AvatarTypeCustom,
		CustomFileID:      &sysFile.ID,
		CustomContentType: contentType,
	}
	if err := h.saveAvatarPreference(userID, newPref); err != nil {
		// 新状态写入失败：回滚新文件与元数据，保持旧状态不变
		_ = h.avatarStore.Delete(relPath)
		_ = h.db.Delete(sysFile)
		respondError(w, http.StatusInternalServerError, "failed to update avatar preference", err)
		return
	}

	// 替换成功：清理旧自定义头像（失败仅记录，不影响新状态）
	if oldPref.Type == models.AvatarTypeCustom && oldPref.CustomFileID != nil {
		h.cleanupAvatarFile(userID, *oldPref.CustomFileID)
	}

	respondJSON(w, http.StatusOK, avatarMetadataResponse{
		Type:              newPref.Type,
		Seed:              newPref.Seed,
		CustomFileID:      newPref.CustomFileID,
		CustomContentType: newPref.CustomContentType,
		ContentType:       contentType,
	})
}

// resetAvatar DELETE /api/user/avatar
// 恢复默认头像：先写入新状态（default），成功后再清理旧自定义文件。
func (h *Handler) resetAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	pref, err := h.getAvatarPreference(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load avatar preference", err)
		return
	}

	var oldFileID *uint
	if pref.Type == models.AvatarTypeCustom && pref.CustomFileID != nil {
		oldFileID = pref.CustomFileID
	}

	// 先写新状态
	pref.Type = models.AvatarTypeDefault
	pref.CustomFileID = nil
	pref.CustomContentType = ""
	if err := h.saveAvatarPreference(userID, pref); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update avatar preference", err)
		return
	}

	// 新状态写入成功后再清理旧文件
	if oldFileID != nil {
		h.cleanupAvatarFile(userID, *oldFileID)
	}

	respondJSON(w, http.StatusOK, avatarMetadataResponse{
		Type:        pref.Type,
		Seed:        pref.Seed,
		ContentType: "image/svg+xml",
	})
}

// getAvatarPreference 读取当前用户的头像偏好；无记录时返回默认偏好。
func (h *Handler) getAvatarPreference(userID uint) (*models.AvatarPreference, error) {
	var pref models.UserPreference
	err := h.db.Where("user_id = ? AND pref_key = ?", userID, models.AvatarPrefKey).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.AvatarPreference{Type: models.AvatarTypeDefault}, nil
	}
	if err != nil {
		return nil, err
	}
	var ap models.AvatarPreference
	if len(pref.Value) > 0 {
		if err := json.Unmarshal(pref.Value, &ap); err != nil {
			return nil, err
		}
	}
	if ap.Type == "" {
		ap.Type = models.AvatarTypeDefault
	}
	return &ap, nil
}

// saveAvatarPreference 写入当前用户的头像偏好（UPSERT，兼容现有 UserPreference 表）。
// 使用 ON CONFLICT (user_id, pref_key) DO UPDATE 实现真正的原子 upsert：
// 并发首次 GET 时两个请求同时写入，也只会保留一条记录，不会因唯一约束报错或产生重复偏好。
func (h *Handler) saveAvatarPreference(userID uint, ap *models.AvatarPreference) error {
	bytes, err := json.Marshal(ap)
	if err != nil {
		return err
	}
	pref := models.UserPreference{
		UserID:  &userID,
		PrefKey: models.AvatarPrefKey,
		Value:   datatypes.JSON(bytes),
	}
	return h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "pref_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&pref).Error
}

// resetAvatarPreferenceToDefault 将头像偏好重置为默认并清理失效的自定义文件引用。
// 仅在自定义文件缺失/元数据失效时调用；偏好重置失败则保持原状，下次 GET 再尝试。
func (h *Handler) resetAvatarPreferenceToDefault(userID uint, pref *models.AvatarPreference, sf *models.SysFile) {
	pref.Type = models.AvatarTypeDefault
	pref.CustomFileID = nil
	pref.CustomContentType = ""
	if err := h.saveAvatarPreference(userID, pref); err != nil {
		log.Printf("[avatar] failed to reset preference to default for user %d: %v", userID, err)
		return
	}
	if sf != nil {
		h.cleanupAvatarFile(userID, sf.ID)
	}
}

// cleanupAvatarFile 清理旧自定义头像：软删除 SysFile 元数据 + 删除物理文件。
// 调用方必须已确保新状态写入成功，因此清理失败不会导致数据指向不存在的文件。
func (h *Handler) cleanupAvatarFile(userID, fileID uint) {
	var sf models.SysFile
	if err := h.db.Where("id = ? AND created_by = ?", fileID, userID).First(&sf).Error; err != nil {
		log.Printf("[avatar] cleanup: metadata %d not found for user %d: %v", fileID, userID, err)
		return
	}
	if err := h.db.Delete(&sf).Error; err != nil {
		log.Printf("[avatar] cleanup: failed to soft-delete metadata %d: %v", fileID, err)
		return
	}
	if h.avatarStore != nil {
		if err := h.avatarStore.Delete(sf.Path); err != nil {
			log.Printf("[avatar] cleanup: failed to delete file %q: %v", sf.Path, err)
		}
	}
}

// writeAvatarBinary 输出头像二进制响应（安全 Content-Type + 禁止缓存）。
func writeAvatarBinary(w http.ResponseWriter, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
