package service

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupOCRTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.ModelConfig{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func createOCRConfig(t *testing.T, db *gorm.DB, userID *uint, enabled, isDefault bool, key string) {
	t.Helper()
	config := models.ModelConfig{UserID: userID, ConfigType: "ocr", Enabled: enabled, IsDefault: isDefault, APIKey: key, APIEndpoint: "http://invalid.example"}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled {
		if err := db.Model(&config).Update("enabled", false).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetGlobalDefaultOCRConfig_OnlySelectsEnabledGlobalDefault(t *testing.T) {
	db := setupOCRTestDB(t)
	userA, userB := uint(1), uint(2)
	createOCRConfig(t, db, &userA, true, true, "user-a-key")
	createOCRConfig(t, db, &userB, true, true, "user-b-key")
	createOCRConfig(t, db, nil, true, true, "global-key")
	config, err := NewOCRService(db).GetGlobalDefaultOCRConfig()
	if err != nil || config == nil || config.APIKey != "global-key" || config.UserID != nil {
		t.Fatalf("必须只选择启用的全局默认配置: config=%+v err=%v", config, err)
	}
}

func TestGetGlobalDefaultOCRConfig_RejectsUserAndDisabledConfigs(t *testing.T) {
	db := setupOCRTestDB(t)
	userID := uint(1)
	createOCRConfig(t, db, &userID, true, true, "user-key")
	createOCRConfig(t, db, nil, false, true, "disabled-global-key")
	config, err := NewOCRService(db).GetGlobalDefaultOCRConfig()
	if config != nil || err == nil || strings.Contains(err.Error(), "key") {
		t.Fatalf("无启用全局默认时不得选用户或泄露密钥: config=%+v err=%v", config, err)
	}
}
