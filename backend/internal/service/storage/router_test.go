package storage

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.StorageConfig{},
		&models.StorageRule{},
		&models.StorageModuleConfig{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestResolve_ExactRuleMatch(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config := models.StorageConfig{
		Name:      "Local Storage",
		Type:      "local",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	rule := models.StorageRule{
		StorageID:    config.ID,
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Priority:     10,
		Enabled:      true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create storage rule: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.StorageID != config.ID {
		t.Errorf("expected storage ID %d, got %d", config.ID, result.StorageID)
	}

	if result.StorageType != "local" {
		t.Errorf("expected storage type 'local', got '%s'", result.StorageType)
	}

	if result.BasePath != "/archives/employee_photos" {
		t.Errorf("expected base path '/archives/employee_photos', got '%s'", result.BasePath)
	}

	if result.StorageConfig.ID != config.ID {
		t.Errorf("expected config ID %d, got %d", config.ID, result.StorageConfig.ID)
	}
}

func TestResolve_FallbackToGlobalDefault(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	defaultConfig := models.StorageConfig{
		Name:      "Default Storage",
		Type:      "s3",
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&defaultConfig).Error; err != nil {
		t.Fatalf("failed to create default storage config: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "unknown_module",
		ResourceType: "unknown_type",
		Filename:     "file.txt",
		FileSize:     512,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.StorageID != defaultConfig.ID {
		t.Errorf("expected storage ID %d, got %d", defaultConfig.ID, result.StorageID)
	}

	if result.StorageType != "s3" {
		t.Errorf("expected storage type 's3', got '%s'", result.StorageType)
	}

	if result.BasePath != "/unknown_module/unknown_type" {
		t.Errorf("expected base path '/unknown_module/unknown_type', got '%s'", result.BasePath)
	}
}

func TestResolve_NoDefaultConfig(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	result, err := router.Resolve(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when no default config exists")
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	if err.Error() != "no default storage config found: record not found" {
		t.Logf("error message: %v", err)
	}
}

func TestResolve_ModuleDefaultRule(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config := models.StorageConfig{
		Name:      "Module Storage",
		Type:      "webdav",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	moduleConfig := models.StorageModuleConfig{
		ModuleCode:    "archives",
		ModuleName:    "Archives Module",
		BaseDirectory: "/archives",
		Enabled:       true,
	}
	if err := db.Create(&moduleConfig).Error; err != nil {
		t.Fatalf("failed to create module config: %v", err)
	}

	rule := models.StorageRule{
		StorageID:    config.ID,
		ModuleCode:   "archives",
		ResourceType: "",
		Priority:     5,
		Enabled:      true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("failed to create storage rule: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "any_type",
		Filename:     "document.pdf",
		FileSize:     2048,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.StorageID != config.ID {
		t.Errorf("expected storage ID %d, got %d", config.ID, result.StorageID)
	}

	if result.StorageType != "webdav" {
		t.Errorf("expected storage type 'webdav', got '%s'", result.StorageType)
	}
}

func TestResolve_PriorityOrdering(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config1 := models.StorageConfig{
		Name:      "Low Priority Storage",
		Type:      "local",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&config1).Error; err != nil {
		t.Fatalf("failed to create storage config 1: %v", err)
	}

	config2 := models.StorageConfig{
		Name:      "High Priority Storage",
		Type:      "s3",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&config2).Error; err != nil {
		t.Fatalf("failed to create storage config 2: %v", err)
	}

	rule1 := models.StorageRule{
		StorageID:    config1.ID,
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Priority:     5,
		Enabled:      true,
	}
	if err := db.Create(&rule1).Error; err != nil {
		t.Fatalf("failed to create rule 1: %v", err)
	}

	rule2 := models.StorageRule{
		StorageID:    config2.ID,
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Priority:     10,
		Enabled:      true,
	}
	if err := db.Create(&rule2).Error; err != nil {
		t.Fatalf("failed to create rule 2: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.StorageID != config2.ID {
		t.Errorf("expected high priority storage ID %d, got %d", config2.ID, result.StorageID)
	}

	if result.StorageType != "s3" {
		t.Errorf("expected storage type 's3', got '%s'", result.StorageType)
	}
}

func TestResolve_DisabledRuleIgnored(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config := models.StorageConfig{
		Name:      "Default Storage",
		Type:      "s3",
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	disabledRule := models.StorageRule{
		StorageID:    config.ID,
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Priority:     10,
		Enabled:      false,
	}
	if err := db.Create(&disabledRule).Error; err != nil {
		t.Fatalf("failed to create disabled rule: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.StorageConfig.IsDefault {
		t.Errorf("expected default storage config (IsDefault=true), got IsDefault=%v", result.StorageConfig.IsDefault)
	}

	if result.StorageConfig.Name != "Default Storage" {
		t.Errorf("expected 'Default Storage', got '%s'", result.StorageConfig.Name)
	}
}

func TestBuildFullPath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		filename string
		validate func(string) bool
	}{
		{
			name:     "simple path",
			basePath: "/archives/employee_photos",
			filename: "photo.jpg",
			validate: func(path string) bool {
				return path == "/archives/employee_photos/"+time.Now().Format("2006-01-02")+"/photo.jpg"
			},
		},
		{
			name:     "path with special chars",
			basePath: "/module/resource",
			filename: "file-2025_01.pdf",
			validate: func(path string) bool {
				return path == "/module/resource/"+time.Now().Format("2006-01-02")+"/file-2025_01.pdf"
			},
		},
		{
			name:     "single level path",
			basePath: "/uploads",
			filename: "document.docx",
			validate: func(path string) bool {
				return path == "/uploads/"+time.Now().Format("2006-01-02")+"/document.docx"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFullPath(tt.basePath, tt.filename)
			if !tt.validate(result) {
				t.Errorf("buildFullPath(%q, %q) = %q, validation failed", tt.basePath, tt.filename, result)
			}
		})
	}
}

func TestResolve_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config := models.StorageConfig{
		Name:      "Test Storage",
		Type:      "local",
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	_, err := router.Resolve(ctx, req)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

func TestResolve_ExactRuleOverridesModuleDefault(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	moduleConfig := models.StorageConfig{
		Name:      "Module Default Storage",
		Type:      "local",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&moduleConfig).Error; err != nil {
		t.Fatalf("failed to create module config: %v", err)
	}

	exactConfig := models.StorageConfig{
		Name:      "Exact Rule Storage",
		Type:      "s3",
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&exactConfig).Error; err != nil {
		t.Fatalf("failed to create exact config: %v", err)
	}

	storageModuleConfig := models.StorageModuleConfig{
		ModuleCode:    "archives",
		ModuleName:    "Archives",
		BaseDirectory: "/archives",
		Enabled:       true,
	}
	if err := db.Create(&storageModuleConfig).Error; err != nil {
		t.Fatalf("failed to create storage module config: %v", err)
	}

	moduleRule := models.StorageRule{
		StorageID:    moduleConfig.ID,
		ModuleCode:   "archives",
		ResourceType: "",
		Priority:     5,
		Enabled:      true,
	}
	if err := db.Create(&moduleRule).Error; err != nil {
		t.Fatalf("failed to create module rule: %v", err)
	}

	exactRule := models.StorageRule{
		StorageID:    exactConfig.ID,
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Priority:     10,
		Enabled:      true,
	}
	if err := db.Create(&exactRule).Error; err != nil {
		t.Fatalf("failed to create exact rule: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "employee_photos",
		Filename:     "photo.jpg",
		FileSize:     1024,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.StorageID != exactConfig.ID {
		t.Errorf("expected exact rule storage ID %d, got %d", exactConfig.ID, result.StorageID)
	}

	if result.StorageType != "s3" {
		t.Errorf("expected storage type 's3', got '%s'", result.StorageType)
	}
}

func TestResolve_EmptyResourceTypeInRequest(t *testing.T) {
	db := setupTestDB(t)
	router := NewStorageRouter(db)

	config := models.StorageConfig{
		Name:      "Default Storage",
		Type:      "local",
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("failed to create storage config: %v", err)
	}

	req := ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "",
		Filename:     "file.txt",
		FileSize:     512,
	}

	result, err := router.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.BasePath != "/archives" {
		t.Errorf("expected base path '/archives', got '%s'", result.BasePath)
	}
}
