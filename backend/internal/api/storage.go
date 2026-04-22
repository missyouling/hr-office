package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/datatypes"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service/storage"
)

// listStorageConfigs GET /api/admin/storage
func (h *Handler) listStorageConfigs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var configs []models.StorageConfig
	if err := h.db.Where("user_id = ? OR (is_default = ? AND type = ?)", userID, true, "local").Order("is_default DESC, priority DESC, created_at DESC").Find(&configs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load storage configs", err)
		return
	}

	respondJSON(w, http.StatusOK, configs)
}

type createStorageConfigRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Enabled       bool                   `json:"enabled"`
	IsDefault     bool                   `json:"is_default"`
	IsBackup      bool                   `json:"is_backup"`
	Priority      int                    `json:"priority"`
	Config        map[string]interface{} `json:"config"`
	ResourceTypes []string               `json:"resource_types"`
}

// createStorageConfig POST /api/admin/storage
func (h *Handler) createStorageConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req createStorageConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Name == "" || req.Type == "" {
		respondError(w, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault {
		if err := h.db.Model(&models.StorageConfig{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update default config", err)
			return
		}
	}

	// Convert config to JSON
	var configJSON datatypes.JSON
	if req.Config != nil {
		data, err := json.Marshal(req.Config)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid config", err)
			return
		}
		configJSON = data
	}

	// Convert resource types to JSON
	var resourceTypesJSON datatypes.JSON
	if len(req.ResourceTypes) > 0 {
		data, err := json.Marshal(req.ResourceTypes)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid resource_types", err)
			return
		}
		resourceTypesJSON = data
	}

	config := models.StorageConfig{
		UserID:        &userID,
		Name:          req.Name,
		Type:          req.Type,
		Enabled:       req.Enabled,
		IsDefault:     req.IsDefault,
		IsBackup:      req.IsBackup,
		Priority:      req.Priority,
		Config:        configJSON,
		ResourceTypes: resourceTypesJSON,
	}

	if err := h.db.Create(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create storage config", err)
		return
	}

	respondJSON(w, http.StatusCreated, config)
}

type updateStorageConfigRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Enabled       bool                   `json:"enabled"`
	IsDefault     bool                   `json:"is_default"`
	IsBackup      bool                   `json:"is_backup"`
	Priority      int                    `json:"priority"`
	Config        map[string]interface{} `json:"config"`
	ResourceTypes []string               `json:"resource_types"`
}

// updateStorageConfig PUT /api/admin/storage/{id}
func (h *Handler) updateStorageConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req updateStorageConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// Check if config exists and belongs to user
	var config models.StorageConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage config not found", err)
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault && !config.IsDefault {
		if err := h.db.Model(&models.StorageConfig{}).
			Where("user_id = ? AND id != ?", userID, id).
			Update("is_default", false).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update default config", err)
			return
		}
	}

	// Convert config to JSON
	var configJSON datatypes.JSON
	if req.Config != nil {
		data, err := json.Marshal(req.Config)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid config", err)
			return
		}
		configJSON = data
	}

	// Convert resource types to JSON
	var resourceTypesJSON datatypes.JSON
	if len(req.ResourceTypes) > 0 {
		data, err := json.Marshal(req.ResourceTypes)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid resource_types", err)
			return
		}
		resourceTypesJSON = data
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"type":           req.Type,
		"enabled":        req.Enabled,
		"is_default":     req.IsDefault,
		"is_backup":      req.IsBackup,
		"priority":       req.Priority,
		"config":         configJSON,
		"resource_types": resourceTypesJSON,
	}

	if err := h.db.Model(&config).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update storage config", err)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// deleteStorageConfig DELETE /api/admin/storage/{id}
func (h *Handler) deleteStorageConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	// Check if config exists and belongs to user
	var config models.StorageConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage config not found", err)
		return
	}

	// Delete associated rules
	if err := h.db.Where("storage_id = ?", id).Delete(&models.StorageRule{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete storage rules", err)
		return
	}

	// Delete config
	if err := h.db.Delete(&config).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete storage config", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "storage config deleted"})
}

type testStorageConnectionRequest struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type testStorageConnectionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
}

func (h *Handler) listStorageDirectories(w http.ResponseWriter, r *http.Request) {
	log.Printf("[listStorageDirectories] called")

	var req struct {
		Type   string                 `json:"type"`
		Config map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[listStorageDirectories] decode error: %v", err)
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	log.Printf("[listStorageDirectories] type=%s, config=%+v", req.Type, req.Config)

	configBytes, err := json.Marshal(req.Config)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid config", err)
		return
	}

	driver, err := storage.DefaultRegistry.Create(req.Type, configBytes)
	if err != nil {
		log.Printf("[listStorageDirectories] create driver error: %v", err)
		respondError(w, http.StatusBadRequest, "failed to initialize driver", err)
		return
	}

	files, err := driver.List(r.Context(), "")
	if err != nil {
		log.Printf("[listStorageDirectories] driver.List error: %v", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list directories: %v", err), nil)
		return
	}

	log.Printf("[listStorageDirectories] success, files count: %d", len(files))

	dirs := make([]string, 0)
	for _, f := range files {
		if f.IsDir {
			dirs = append(dirs, f.Path)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"directories": dirs,
	})
}

// testStorageConnectionNew POST /api/admin/storage/test
func (h *Handler) testStorageConnectionNew(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req testStorageConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.Type == "" || req.Config == nil {
		respondError(w, http.StatusBadRequest, "missing required fields", nil)
		return
	}

	start := time.Now()
	var success bool
	var message string

	switch req.Type {
	case "local":
		path, ok := req.Config["root_path"].(string)
		if !ok || path == "" {
			path, ok = req.Config["path"].(string)
		}
		if !ok || path == "" {
			respondError(w, http.StatusBadRequest, "missing root_path for local storage", nil)
			return
		}
		_, err := os.Stat(path)
		if err != nil {
			success = false
			message = fmt.Sprintf("path not accessible: %v", err)
		} else {
			success = true
			message = "local path is accessible"
		}

	case "s3":
		endpoint, ok := req.Config["endpoint"].(string)
		if !ok || endpoint == "" {
			respondError(w, http.StatusBadRequest, "missing endpoint for s3 storage", nil)
			return
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Head(endpoint)
		if err != nil {
			success = false
			message = fmt.Sprintf("s3 endpoint not accessible: %v", err)
		} else {
			defer resp.Body.Close()
			success = true
			message = "s3 endpoint is accessible"
		}

	case "webdav":
		url, ok := req.Config["url"].(string)
		if !ok || url == "" {
			url, ok = req.Config["webdav_url"].(string)
		}
		if !ok || url == "" {
			respondError(w, http.StatusBadRequest, "missing url for webdav storage", nil)
			return
		}
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("PROPFIND", url, nil)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid webdav url", err)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			success = false
			message = fmt.Sprintf("webdav url not accessible: %v", err)
		} else {
			defer resp.Body.Close()
			success = true
			message = "webdav url is accessible"
		}

	case "nas":
		host, ok := req.Config["host"].(string)
		if !ok || host == "" {
			respondError(w, http.StatusBadRequest, "missing host for nas storage", nil)
			return
		}
		port, ok := req.Config["port"].(float64)
		if !ok {
			port = 445
		}
		addr := fmt.Sprintf("%s:%d", host, int(port))
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			success = false
			message = fmt.Sprintf("nas host not accessible: %v", err)
		} else {
			defer conn.Close()
			success = true
			message = "nas host is accessible"
		}

	case "google_drive", "onedrive", "aliyun_drive", "cmcc_cloud", "115_drive":
		// OAuth-based cloud drives: test by checking if token/credentials are provided
		token, _ := req.Config["access_token"].(string)
		if token == "" {
			success = false
			message = fmt.Sprintf("%s: missing access_token in config", req.Type)
		} else {
			success = true
			message = fmt.Sprintf("%s: credentials configured (actual connection test requires OAuth flow)", req.Type)
		}

	default:
		respondError(w, http.StatusBadRequest, "unsupported storage type", nil)
		return
	}

	latency := time.Since(start).Milliseconds()
	respondJSON(w, http.StatusOK, testStorageConnectionResponse{
		Success:   success,
		Message:   message,
		LatencyMs: latency,
	})
}

// listStorageRules GET /api/admin/storage/rules
func (h *Handler) listStorageRules(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	query := h.db.Where("user_id = ?", userID)

	if moduleCode := r.URL.Query().Get("module_code"); moduleCode != "" {
		query = query.Where("module_code = ?", moduleCode)
	}

	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var rules []models.StorageRule
	if err := query.Order("priority DESC, created_at DESC").Find(&rules).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load storage rules", err)
		return
	}

	respondJSON(w, http.StatusOK, rules)
}

type updateStorageRulesRequest struct {
	Rules []struct {
		ID           uint   `json:"id"`
		StorageID    uint   `json:"storage_id"`
		CategoryCode string `json:"category_code"`
		Priority     int    `json:"priority"`
		Enabled      bool   `json:"enabled"`
	} `json:"rules"`
}

// updateStorageRules PUT /api/admin/storage/rules
func (h *Handler) updateStorageRules(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req updateStorageRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	// Delete old rules
	if err := h.db.Where("user_id = ?", userID).Delete(&models.StorageRule{}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete old rules", err)
		return
	}

	// Create new rules
	var rules []models.StorageRule
	for _, r := range req.Rules {
		rules = append(rules, models.StorageRule{
			UserID:       &userID,
			StorageID:    r.StorageID,
			CategoryCode: r.CategoryCode,
			Priority:     r.Priority,
			Enabled:      r.Enabled,
		})
	}

	if len(rules) > 0 {
		if err := h.db.Create(&rules).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create storage rules", err)
			return
		}
	}

	respondJSON(w, http.StatusOK, rules)
}

// getStorageStatus GET /api/admin/storage/{id}/status
func (h *Handler) getStorageStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var config models.StorageConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage config not found", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":                config.ID,
		"name":              config.Name,
		"type":              config.Type,
		"status":            config.Status,
		"enabled":           config.Enabled,
		"is_default":        config.IsDefault,
		"last_health_check": config.LastHealthCheck,
		"fail_count":        config.FailCount,
		"max_fail_count":    config.MaxFailCount,
	})
}

// setStoragePrimary POST /api/admin/storage/{id}/set-primary
func (h *Handler) setStoragePrimary(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var config models.StorageConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage config not found", err)
		return
	}

	if err := h.db.Model(&models.StorageConfig{}).
		Where("user_id = ?", userID).
		Update("is_default", false).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update default config", err)
		return
	}

	if err := h.db.Model(&config).Update("is_default", true).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to set primary", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "storage set as primary"})
}

type createStorageRuleRequest struct {
	StorageID         uint   `json:"storage_id"`
	ModuleCode        string `json:"module_code"`
	ResourceType      string `json:"resource_type"`
	CategoryCode      string `json:"category_code"`
	Priority          int    `json:"priority"`
	Enabled           bool   `json:"enabled"`
	Name              string `json:"name"`
	TargetType        string `json:"target_type"`
	TargetPattern     string `json:"target_pattern"`
	SizeMin           *int64 `json:"size_min"`
	SizeMax           *int64 `json:"size_max"`
	FallbackStorageID *uint  `json:"fallback_storage_id"`
}

// createStorageRule POST /api/admin/storage/rules
func (h *Handler) createStorageRule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req createStorageRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	rule := models.StorageRule{
		UserID:            &userID,
		StorageID:         req.StorageID,
		ModuleCode:        req.ModuleCode,
		ResourceType:      req.ResourceType,
		CategoryCode:      req.CategoryCode,
		Priority:          req.Priority,
		Enabled:           req.Enabled,
		Name:              req.Name,
		TargetType:        req.TargetType,
		TargetPattern:     req.TargetPattern,
		SizeMin:           req.SizeMin,
		SizeMax:           req.SizeMax,
		FallbackStorageID: req.FallbackStorageID,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create storage rule", err)
		return
	}

	respondJSON(w, http.StatusCreated, rule)
}

// updateStorageRule PUT /api/admin/storage/rules/{id}
func (h *Handler) updateStorageRule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var rule models.StorageRule
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&rule).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage rule not found", err)
		return
	}

	var req createStorageRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	updates := map[string]interface{}{
		"storage_id":          req.StorageID,
		"module_code":         req.ModuleCode,
		"resource_type":       req.ResourceType,
		"category_code":       req.CategoryCode,
		"priority":            req.Priority,
		"enabled":             req.Enabled,
		"name":                req.Name,
		"target_type":         req.TargetType,
		"target_pattern":      req.TargetPattern,
		"size_min":            req.SizeMin,
		"size_max":            req.SizeMax,
		"fallback_storage_id": req.FallbackStorageID,
	}

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update storage rule", err)
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

// deleteStorageRule DELETE /api/admin/storage/rules/{id}
func (h *Handler) deleteStorageRule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var rule models.StorageRule
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&rule).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage rule not found", err)
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete storage rule", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "storage rule deleted"})
}

// listStorageDirectories GET /api/storage/directories

// getStorageCapacity GET /api/admin/storage/{id}/capacity
func (h *Handler) getStorageCapacity(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var config models.StorageConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage config not found", err)
		return
	}

	driver, err := storage.DefaultRegistry.Create(config.Type, []byte(config.Config))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storage.CapacityInfo{
			Available: false,
			Message:   "failed to create driver: " + err.Error(),
		})
		return
	}

	if cp, ok := driver.(storage.CapacityProvider); ok {
		info, err := cp.GetCapacity(r.Context())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(storage.CapacityInfo{
				Available: false,
				Message:   err.Error(),
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storage.CapacityInfo{
			Available: false,
			Message:   "this storage type does not support capacity reporting",
		})
	}
}

// uploadStorageFile POST /api/admin/storage/files
func (h *Handler) uploadStorageFile(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse form", err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to get file from form", err)
		return
	}
	defer file.Close()

	configIDStr := r.FormValue("storage_config_id")
	if configIDStr == "" {
		respondError(w, http.StatusBadRequest, "storage_config_id is required", nil)
		return
	}

	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid storage_config_id", err)
		return
	}

	sysFile, err := storage.GlobalManager.UploadFile(r.Context(), uint(configID), userID, header.Filename, file, header.Size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to upload file", err)
		return
	}

	respondJSON(w, http.StatusCreated, sysFile)
}

// listStorageFiles GET /api/admin/storage/files
func (h *Handler) listStorageFiles(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configIDStr := r.URL.Query().Get("storage_config_id")
	var configID *uint
	if configIDStr != "" {
		id, err := strconv.ParseUint(configIDStr, 10, 32)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid storage_config_id", err)
			return
		}
		configID = (*uint)(&[]uint{uint(id)}[0])
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid limit", err)
			return
		}
		limit = l
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid offset", err)
			return
		}
		offset = o
	}

	files, total, err := storage.GlobalManager.ListFiles(r.Context(), configID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list files", err)
		return
	}

	response := map[string]interface{}{
		"files":  files,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	respondJSON(w, http.StatusOK, response)
}

// downloadStorageFile GET /api/admin/storage/files/{fileID}
func (h *Handler) downloadStorageFile(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid fileID", err)
		return
	}

	reader, sysFile, err := storage.GlobalManager.DownloadFile(r.Context(), uint(fileID))
	if err != nil {
		respondError(w, http.StatusNotFound, "failed to download file", err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", sysFile.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", sysFile.OriginalName))

	if _, err := io.Copy(w, reader); err != nil {
		fmt.Printf("error copying file to response: %v\n", err)
	}
}

// deleteStorageFile DELETE /api/admin/storage/files/{fileID}
func (h *Handler) deleteStorageFile(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid fileID", err)
		return
	}

	if err := storage.GlobalManager.DeleteFile(r.Context(), uint(fileID)); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete file", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

// listStorageModules GET /api/admin/storage/modules
func (h *Handler) listStorageModules(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var modules []models.StorageModuleConfig
	if err := h.db.Where("user_id = ? OR user_id IS NULL", userID).Order("module_code ASC").Find(&modules).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load storage modules", err)
		return
	}

	respondJSON(w, http.StatusOK, modules)
}

type createStorageModuleRequest struct {
	ModuleCode    string `json:"module_code"`
	ModuleName    string `json:"module_name"`
	BaseDirectory string `json:"base_directory"`
	Description   string `json:"description"`
	Enabled       bool   `json:"enabled"`
}

// createStorageModule POST /api/admin/storage/modules
func (h *Handler) createStorageModule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req createStorageModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	if req.ModuleCode == "" || req.ModuleName == "" {
		respondError(w, http.StatusBadRequest, "module_code and module_name are required", nil)
		return
	}

	module := models.StorageModuleConfig{
		UserID:        &userID,
		ModuleCode:    req.ModuleCode,
		ModuleName:    req.ModuleName,
		BaseDirectory: req.BaseDirectory,
		Description:   req.Description,
		Enabled:       req.Enabled,
	}

	if err := h.db.Create(&module).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create storage module", err)
		return
	}

	respondJSON(w, http.StatusCreated, module)
}

// updateStorageModule PUT /api/admin/storage/modules/{id}
func (h *Handler) updateStorageModule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var module models.StorageModuleConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&module).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage module not found", err)
		return
	}

	var req createStorageModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}

	updates := map[string]interface{}{
		"module_code":    req.ModuleCode,
		"module_name":    req.ModuleName,
		"base_directory": req.BaseDirectory,
		"description":    req.Description,
		"enabled":        req.Enabled,
	}

	if err := h.db.Model(&module).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update storage module", err)
		return
	}

	respondJSON(w, http.StatusOK, module)
}

// deleteStorageModule DELETE /api/admin/storage/modules/{id}
func (h *Handler) deleteStorageModule(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	var module models.StorageModuleConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&module).Error; err != nil {
		respondError(w, http.StatusNotFound, "storage module not found", err)
		return
	}

	if err := h.db.Delete(&module).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete storage module", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "storage module deleted"})
}
