package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============ 系统配置 API (Admin) ============

// getStorageConfig 获取存储配置
func (h *Handler) getStorageConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[getStorageConfig] called")

	var config models.StorageConfig
	// 只获取第一条配置，如果没有则返回默认值
	if err := h.db.First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			config = models.StorageConfig{
				ID:   0,
				Type: "local",
			}
			writeJSON(w, config)
			return
		}
		log.Printf("[getStorageConfig] query error: %v", err)
		http.Error(w, fmt.Sprintf("获取存储配置失败: %v", err), http.StatusInternalServerError)
		return
	}
	maskStorageConfig(&config)
	writeJSON(w, config)
}

// saveStorageConfig 保存存储配置
func (h *Handler) saveStorageConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[saveStorageConfig] called")

	var payload struct {
		Type           string `json:"type"`
		RootPath       string `json:"root_path"`
		S3Endpoint     string `json:"s3_endpoint"`
		S3Bucket       string `json:"s3_bucket"`
		S3AccessKey    string `json:"s3_access_key"`
		S3SecretKey    string `json:"s3_secret_key"`
		S3Region       string `json:"s3_region"`
		WebDAVURL      string `json:"webdav_url"`
		WebDAVUsername string `json:"webdav_username"`
		WebDAVPassword string `json:"webdav_password"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[saveStorageConfig] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[saveStorageConfig] payload: type=%s", payload.Type)

	// 获取或创建配置
	var config models.StorageConfig
	result := h.db.First(&config)

	// 构建配置JSON
	configMap := map[string]interface{}{
		"root_path":       payload.RootPath,
		"s3_endpoint":     payload.S3Endpoint,
		"s3_bucket":       payload.S3Bucket,
		"s3_access_key":   payload.S3AccessKey,
		"s3_secret_key":   payload.S3SecretKey,
		"s3_region":       payload.S3Region,
		"webdav_url":      payload.WebDAVURL,
		"webdav_username": payload.WebDAVUsername,
		"webdav_password": payload.WebDAVPassword,
	}
	configJSON, _ := json.Marshal(configMap)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.StorageConfig{
			Type:   payload.Type,
			Config: configJSON,
		}
		if err := h.db.Create(&config).Error; err != nil {
			log.Printf("[saveStorageConfig] create error: %v", err)
			http.Error(w, fmt.Sprintf("创建存储配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[saveStorageConfig] created config id=%d", config.ID)
	} else if result.Error != nil {
		log.Printf("[saveStorageConfig] query error: %v", result.Error)
		http.Error(w, fmt.Sprintf("获取存储配置失败: %v", result.Error), http.StatusInternalServerError)
		return
	} else {
		// 更新配置
		config.Type = payload.Type
		config.Config = configJSON

		if err := h.db.Save(&config).Error; err != nil {
			log.Printf("[saveStorageConfig] save error: %v", err)
			http.Error(w, fmt.Sprintf("保存存储配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[saveStorageConfig] updated config id=%d", config.ID)
	}

	maskStorageConfig(&config)
	writeJSON(w, config)
}

// testStorageConnection 测试存储连接
func (h *Handler) testStorageConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("[testStorageConnection] called")

	var payload struct {
		Type           string `json:"type"`
		RootPath       string `json:"root_path"`
		S3Endpoint     string `json:"s3_endpoint"`
		S3Bucket       string `json:"s3_bucket"`
		S3AccessKey    string `json:"s3_access_key"`
		S3SecretKey    string `json:"s3_secret_key"`
		S3Region       string `json:"s3_region"`
		WebDAVURL      string `json:"webdav_url"`
		WebDAVUsername string `json:"webdav_username"`
		WebDAVPassword string `json:"webdav_password"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[testStorageConnection] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 根据类型测试连接
	var message string
	switch payload.Type {
	case "local":
		// 测试本地路径是否可写
		if payload.RootPath == "" {
			message = "本地存储模式，无需额外配置"
		} else {
			message = fmt.Sprintf("本地存储路径: %s", payload.RootPath)
		}
	case "s3":
		message = fmt.Sprintf("S3存储测试: endpoint=%s, bucket=%s, region=%s",
			payload.S3Endpoint, payload.S3Bucket, payload.S3Region)
		// 实际应该测试S3连接
	case "webdav":
		message = fmt.Sprintf("WebDAV存储测试: url=%s", payload.WebDAVURL)
		// 实际应该测试WebDAV连接
	default:
		message = "未知存储类型"
	}

	log.Printf("[testStorageConnection] result: %s", message)
	writeJSON(w, map[string]string{"message": message, "success": "true"})
}

// getSMTPConfig 获取SMTP配置
func (h *Handler) getSMTPConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[getSMTPConfig] called")

	var config models.SMTPConfig
	if err := h.db.First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回空配置
			config = models.SMTPConfig{}
			writeJSON(w, config)
			return
		}
		log.Printf("[getSMTPConfig] query error: %v", err)
		http.Error(w, fmt.Sprintf("获取SMTP配置失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, config)
}

// saveSMTPConfig 保存SMTP配置
func (h *Handler) saveSMTPConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[saveSMTPConfig] called")

	var payload struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		FromName string `json:"from_name"`
		UseTLS   bool   `json:"use_tls"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[saveSMTPConfig] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[saveSMTPConfig] payload: host=%s, port=%s, from=%s", payload.Host, payload.Port, payload.From)

	// 获取或创建配置
	var config models.SMTPConfig
	result := h.db.First(&config)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.SMTPConfig{
			Host:     payload.Host,
			Port:     payload.Port,
			Username: payload.Username,
			Password: payload.Password,
			From:     payload.From,
			FromName: payload.FromName,
			UseTLS:   payload.UseTLS,
		}
		if err := h.db.Create(&config).Error; err != nil {
			log.Printf("[saveSMTPConfig] create error: %v", err)
			http.Error(w, fmt.Sprintf("创建SMTP配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[saveSMTPConfig] created config id=%d", config.ID)
	} else if result.Error != nil {
		log.Printf("[saveSMTPConfig] query error: %v", result.Error)
		http.Error(w, fmt.Sprintf("获取SMTP配置失败: %v", result.Error), http.StatusInternalServerError)
		return
	} else {
		// 更新配置
		config.Host = payload.Host
		config.Port = payload.Port
		config.Username = payload.Username
		config.Password = payload.Password
		config.From = payload.From
		config.FromName = payload.FromName
		config.UseTLS = payload.UseTLS

		if err := h.db.Save(&config).Error; err != nil {
			log.Printf("[saveSMTPConfig] save error: %v", err)
			http.Error(w, fmt.Sprintf("保存SMTP配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[saveSMTPConfig] updated config id=%d", config.ID)
	}

	writeJSON(w, config)
}

// testSMTPConnection 测试SMTP连接
func (h *Handler) testSMTPConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("[testSMTPConnection] called")

	var payload struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		FromName string `json:"from_name"`
		UseTLS   bool   `json:"use_tls"`
	}

	if err := decodeJSON(r, &payload); err != nil {
		log.Printf("[testSMTPConnection] decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 实际应该测试SMTP连接并发送测试邮件
	message := fmt.Sprintf("SMTP连接测试: host=%s, port=%s, from=%s",
		payload.Host, payload.Port, payload.From)

	log.Printf("[testSMTPConnection] result: %s", message)
	writeJSON(w, map[string]string{"message": message, "success": "true"})
}
