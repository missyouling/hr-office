package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

var GlobalManager *StorageManager

type StorageManager struct {
	db       *gorm.DB
	registry *Registry
	drivers  map[uint]Driver
	mu       sync.RWMutex
}

func NewStorageManager(db *gorm.DB, registry *Registry) *StorageManager {
	return &StorageManager{
		db:       db,
		registry: registry,
		drivers:  make(map[uint]Driver),
	}
}

func (m *StorageManager) Init() error {
	var configs []models.StorageConfig
	if err := m.db.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to load storage configs: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, config := range configs {
		// 解密配置（若未加密则直接返回原文）
		rawConfig := []byte(config.Config)
		decryptedConfig, err := DecryptConfig(rawConfig, aesKey)
		if err != nil {
			log.Printf("[StorageManager] failed to decrypt config %d (%s): %v", config.ID, config.Name, err)
			continue
		}
		driver, err := m.registry.Create(config.Type, decryptedConfig)
		if err != nil {
			log.Printf("[StorageManager] failed to initialize driver for config %d (%s): %v", config.ID, config.Name, err)
			continue
		}
		m.drivers[config.ID] = driver
		log.Printf("[StorageManager] initialized driver for config %d (%s, type=%s)", config.ID, config.Name, config.Type)
	}

	return nil
}

func (m *StorageManager) GetDriver(configID uint) (Driver, error) {
	m.mu.RLock()
	driver, ok := m.drivers[configID]
	m.mu.RUnlock()

	if ok {
		return driver, nil
	}

	var config models.StorageConfig
	if err := m.db.First(&config, configID).Error; err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !config.Enabled {
		return nil, fmt.Errorf("storage config is disabled")
	}

	// 解密配置（若未加密则直接返回原文）
	rawConfig := []byte(config.Config)
	decryptedConfig, decErr := DecryptConfig(rawConfig, aesKey)
	if decErr != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", decErr)
	}
	driver, err := m.registry.Create(config.Type, decryptedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver: %w", err)
	}

	m.mu.Lock()
	m.drivers[configID] = driver
	m.mu.Unlock()

	return driver, nil
}

func (m *StorageManager) GetDefaultDriver(userID uint) (Driver, *models.StorageConfig, error) {
	var config models.StorageConfig
	if err := m.db.Where("user_id = ? AND is_default = ? AND enabled = ?", userID, true, true).
		First(&config).Error; err != nil {
		return nil, nil, fmt.Errorf("default storage config not found for user %d: %w", userID, err)
	}

	driver, err := m.GetDriver(config.ID)
	if err != nil {
		return nil, nil, err
	}

	return driver, &config, nil
}

func (m *StorageManager) UploadFile(ctx context.Context, configID uint, userID uint, filename string, reader io.Reader, size int64) (*models.SysFile, error) {
	driver, err := m.GetDriver(configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	timestamp := fmt.Sprintf("%d", now.UnixNano())
	storagePath := filepath.Join(year, month, day, fmt.Sprintf("%s_%s", timestamp, filename))

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Buffer the file content to allow retry on failover
	fileData := make([]byte, size)
	n, err := io.ReadFull(reader, fileData)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if int64(n) != size {
		return nil, fmt.Errorf("incomplete file read: expected %d bytes, got %d", size, n)
	}

	hash := md5.New()
	hash.Write(fileData)
	etag := hex.EncodeToString(hash.Sum(nil))

	// 尝试上传到主存储
	uploadErr := driver.Upload(ctx, storagePath, io.NopCloser(bytes.NewReader(fileData)), size)

	// 上传失败直接返回 error，不再静默降级到本地存储
	// 理由：静默 fallback 会导致用户无法感知文件被放到非预期存储中，造成数据分散与安全风险
	// 调用方应根据错误信息决定是否需要手动切换到备选存储重试
	if uploadErr != nil {
		log.Printf("[StorageManager] upload to config %d (%s) failed: %v", configID, driver.Type(), uploadErr)
		return nil, fmt.Errorf("failed to upload file to storage config %d: %w", configID, uploadErr)
	}

	sysFile := &models.SysFile{
		StorageType:     driver.Type(),
		Path:            storagePath,
		OriginalName:    filename,
		Size:            size,
		ContentType:     contentType,
		ETag:            etag,
		StorageConfigID: &configID,
		CreatedBy:       &userID,
		IsFallback:      false,
		MigrationStatus: "none",
	}

	if err := m.db.Create(sysFile).Error; err != nil {
		if delErr := driver.Delete(ctx, storagePath); delErr != nil {
			log.Printf("[StorageManager] failed to delete file after metadata creation error: %v", delErr)
		}
		return nil, fmt.Errorf("failed to create file metadata: %w", err)
	}

	return sysFile, nil
}

func (m *StorageManager) DownloadFile(ctx context.Context, fileID uint) (io.ReadCloser, *models.SysFile, error) {
	var sysFile models.SysFile
	if err := m.db.First(&sysFile, fileID).Error; err != nil {
		return nil, nil, fmt.Errorf("file not found: %w", err)
	}

	if sysFile.DeletedAt.Valid {
		return nil, nil, fmt.Errorf("file has been deleted")
	}

	if sysFile.StorageConfigID == nil {
		return nil, nil, fmt.Errorf("file has no storage config")
	}

	driver, err := m.GetDriver(*sysFile.StorageConfigID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get driver: %w", err)
	}

	reader, err := driver.Download(ctx, sysFile.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file: %w", err)
	}

	return reader, &sysFile, nil
}

func (m *StorageManager) DeleteFile(ctx context.Context, fileID uint) error {
	var sysFile models.SysFile
	if err := m.db.First(&sysFile, fileID).Error; err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if sysFile.StorageConfigID == nil {
		return fmt.Errorf("file has no storage config")
	}

	driver, err := m.GetDriver(*sysFile.StorageConfigID)
	if err != nil {
		return fmt.Errorf("failed to get driver: %w", err)
	}

	if err := driver.Delete(ctx, sysFile.Path); err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}

	if err := m.db.Delete(&sysFile).Error; err != nil {
		return fmt.Errorf("failed to delete file metadata: %w", err)
	}

	return nil
}

func (m *StorageManager) ListFiles(ctx context.Context, configID *uint, limit, offset int) ([]models.SysFile, int64, error) {
	var files []models.SysFile
	var total int64

	query := m.db

	if configID != nil {
		query = query.Where("storage_config_id = ?", *configID)
	}

	if err := query.Model(&models.SysFile{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	if err := query.Limit(limit).Offset(offset).Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}

	return files, total, nil
}

func (m *StorageManager) RefreshDriver(configID uint) error {
	var config models.StorageConfig
	if err := m.db.First(&config, configID).Error; err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	// 解密配置（若未加密则直接返回原文）
	rawConfig := []byte(config.Config)
	decryptedConfig, err := DecryptConfig(rawConfig, aesKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt config: %w", err)
	}
	driver, err := m.registry.Create(config.Type, decryptedConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize driver: %w", err)
	}

	m.mu.Lock()
	m.drivers[configID] = driver
	m.mu.Unlock()

	log.Printf("[StorageManager] refreshed driver for config %d (%s)", configID, config.Name)
	return nil
}
