package main

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

// setupStorageDB 建立内存 SQLite 并迁移存储三张表（StorageConfig / StorageModuleConfig / StorageRule）
func setupStorageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&models.StorageConfig{},
		&models.StorageModuleConfig{},
		&models.StorageRule{},
	); err != nil {
		t.Fatalf("迁移存储表失败: %v", err)
	}
	return db
}

// storageRootPath 解析 StorageConfig.Config JSON 中的 root_path
func storageRootPath(t *testing.T, cfg models.StorageConfig) string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(cfg.Config, &m); err != nil {
		t.Fatalf("解析 StorageConfig.Config 失败: %v", err)
	}
	return m["root_path"]
}

// assertE2EStorageConfig 校验 E2E 存储配置处于目标状态（local + enabled + active + 隔离路径）
func assertE2EStorageConfig(t *testing.T, cfg models.StorageConfig) {
	t.Helper()
	if cfg.Name != e2eStorageConfigName {
		t.Errorf("配置名称应为 %q，实际为 %q", e2eStorageConfigName, cfg.Name)
	}
	if cfg.Type != "local" {
		t.Errorf("配置类型应为 local，实际为 %q", cfg.Type)
	}
	if !cfg.Enabled {
		t.Errorf("配置应启用，实际 enabled=false")
	}
	if cfg.IsDefault {
		t.Errorf("E2E 配置不应为全局默认，实际 is_default=true")
	}
	if cfg.Status != "active" {
		t.Errorf("配置状态应为 active，实际为 %q", cfg.Status)
	}
	if got := storageRootPath(t, cfg); got != e2eStorageRootPath {
		t.Errorf("root_path 应为 %q，实际为 %q", e2eStorageRootPath, got)
	}
}

// assertE2EStorageModule 校验 E2E archives 模块配置处于目标状态（enabled + 隔离目录）
func assertE2EStorageModule(t *testing.T, module models.StorageModuleConfig) {
	t.Helper()
	if module.ModuleCode != "archives" {
		t.Errorf("模块编码应为 archives，实际为 %q", module.ModuleCode)
	}
	if module.ModuleName != e2eStorageModuleName {
		t.Errorf("模块名称应为 %q，实际为 %q", e2eStorageModuleName, module.ModuleName)
	}
	if !module.Enabled {
		t.Errorf("模块配置应启用，实际 enabled=false")
	}
	if module.BaseDirectory != e2eStorageRootPath {
		t.Errorf("base_directory 应为 %q，实际为 %q", e2eStorageRootPath, module.BaseDirectory)
	}
}

// assertE2EStorageRule 校验 E2E archives/resign_proof 规则处于目标状态（enabled + 指向 E2E 配置）
func assertE2EStorageRule(t *testing.T, rule models.StorageRule, storageID uint) {
	t.Helper()
	if rule.ModuleCode != "archives" {
		t.Errorf("规则模块编码应为 archives，实际为 %q", rule.ModuleCode)
	}
	if rule.ResourceType != "resign_proof" {
		t.Errorf("规则资源类型应为 resign_proof，实际为 %q", rule.ResourceType)
	}
	if rule.Name != e2eStorageRuleName {
		t.Errorf("规则名称应为 %q，实际为 %q", e2eStorageRuleName, rule.Name)
	}
	if rule.StorageID != storageID {
		t.Errorf("规则 storage_id 应为 %d，实际为 %d", storageID, rule.StorageID)
	}
	if rule.Priority != e2eStorageRulePriority {
		t.Errorf("规则优先级应为 %d，实际为 %d", e2eStorageRulePriority, rule.Priority)
	}
	if !rule.Enabled {
		t.Errorf("规则应启用，实际 enabled=false")
	}
	if rule.TargetType != "document" {
		t.Errorf("规则目标类型应为 document，实际为 %q", rule.TargetType)
	}
}

// TestEnsureE2EStorage_CreatesIsolatedRecords 首次运行应创建三张表记录且字段符合目标状态
func TestEnsureE2EStorage_CreatesIsolatedRecords(t *testing.T) {
	db := setupStorageDB(t)

	storageID, err := ensureE2EStorage(db)
	if err != nil {
		t.Fatalf("首次初始化 E2E 存储失败: %v", err)
	}
	if storageID == 0 {
		t.Fatal("返回的 E2E 存储配置 ID 不应为 0")
	}

	var cfg models.StorageConfig
	if err := db.First(&cfg, storageID).Error; err != nil {
		t.Fatalf("查询 E2E 存储配置失败: %v", err)
	}
	assertE2EStorageConfig(t, cfg)

	var module models.StorageModuleConfig
	if err := db.Where("module_code = ? AND module_name = ?", "archives", e2eStorageModuleName).First(&module).Error; err != nil {
		t.Fatalf("查询 E2E archives 模块配置失败: %v", err)
	}
	assertE2EStorageModule(t, module)

	var rule models.StorageRule
	if err := db.Where("module_code = ? AND resource_type = ? AND name = ?", "archives", "resign_proof", e2eStorageRuleName).First(&rule).Error; err != nil {
		t.Fatalf("查询 E2E 存储规则失败: %v", err)
	}
	assertE2EStorageRule(t, rule, storageID)
}

// TestEnsureE2EStorage_Idempotent 重复运行不重复创建记录
func TestEnsureE2EStorage_Idempotent(t *testing.T) {
	db := setupStorageDB(t)

	firstID, err := ensureE2EStorage(db)
	if err != nil {
		t.Fatalf("首次初始化 E2E 存储失败: %v", err)
	}

	secondID, err := ensureE2EStorage(db)
	if err != nil {
		t.Fatalf("重复初始化 E2E 存储失败: %v", err)
	}
	if secondID != firstID {
		t.Errorf("重复运行应复用同一配置 ID，首次=%d 再次=%d", firstID, secondID)
	}

	var configCount, moduleCount, ruleCount int64
	db.Model(&models.StorageConfig{}).Count(&configCount)
	db.Model(&models.StorageModuleConfig{}).Count(&moduleCount)
	db.Model(&models.StorageRule{}).Count(&ruleCount)
	if configCount != 1 || moduleCount != 1 || ruleCount != 1 {
		t.Errorf("重复运行后记录数应为 1/1/1，实际 config=%d module=%d rule=%d",
			configCount, moduleCount, ruleCount)
	}
}

// TestEnsureE2EStorage_RestoresDamagedRecords 目标记录损坏后应恢复 enabled/规则指向/路径
func TestEnsureE2EStorage_RestoresDamagedRecords(t *testing.T) {
	db := setupStorageDB(t)

	storageID, err := ensureE2EStorage(db)
	if err != nil {
		t.Fatalf("首次初始化 E2E 存储失败: %v", err)
	}

	// 模拟损坏：配置禁用+状态异常+路径被改、模块禁用+目录被改、规则禁用+指向错误配置
	if err := db.Model(&models.StorageConfig{}).Where("id = ?", storageID).Updates(map[string]any{
		"enabled": false,
		"status":  "error",
		"config":  []byte(`{"root_path":"/tmp/evil-path"}`),
	}).Error; err != nil {
		t.Fatalf("模拟配置损坏失败: %v", err)
	}
	if err := db.Model(&models.StorageModuleConfig{}).Where("module_code = ? AND module_name = ?", "archives", e2eStorageModuleName).Updates(map[string]any{
		"enabled":        false,
		"base_directory": "/tmp/evil-path",
	}).Error; err != nil {
		t.Fatalf("模拟模块配置损坏失败: %v", err)
	}
	if err := db.Model(&models.StorageRule{}).Where("module_code = ? AND resource_type = ? AND name = ?", "archives", "resign_proof", e2eStorageRuleName).Updates(map[string]any{
		"enabled":    false,
		"storage_id": 99999,
	}).Error; err != nil {
		t.Fatalf("模拟规则损坏失败: %v", err)
	}

	// 再次执行：应恢复目标状态
	if _, err := ensureE2EStorage(db); err != nil {
		t.Fatalf("恢复 E2E 存储失败: %v", err)
	}

	var cfg models.StorageConfig
	if err := db.First(&cfg, storageID).Error; err != nil {
		t.Fatalf("重新查询 E2E 存储配置失败: %v", err)
	}
	assertE2EStorageConfig(t, cfg)

	var module models.StorageModuleConfig
	if err := db.Where("module_code = ? AND module_name = ?", "archives", e2eStorageModuleName).First(&module).Error; err != nil {
		t.Fatalf("重新查询 E2E 模块配置失败: %v", err)
	}
	assertE2EStorageModule(t, module)

	var rule models.StorageRule
	if err := db.Where("module_code = ? AND resource_type = ? AND name = ?", "archives", "resign_proof", e2eStorageRuleName).First(&rule).Error; err != nil {
		t.Fatalf("重新查询 E2E 存储规则失败: %v", err)
	}
	assertE2EStorageRule(t, rule, storageID)
}

// TestEnsureE2EStorage_DoesNotTouchNonE2E 非 E2E 配置/规则不应被覆盖或删除
func TestEnsureE2EStorage_DoesNotTouchNonE2E(t *testing.T) {
	db := setupStorageDB(t)

	// 预置非 E2E 存储配置（全局默认、生产路径）
	prodConfig := models.StorageConfig{
		Name:      "生产本地存储",
		Type:      "local",
		Enabled:   true,
		IsDefault: true,
		Status:    "active",
		Config:    []byte(`{"root_path":"/data/prod"}`),
	}
	if err := db.Create(&prodConfig).Error; err != nil {
		t.Fatalf("创建非 E2E 存储配置失败: %v", err)
	}
	// 预置非 E2E 模块配置（同模块 archives，但名称不同）
	prodModule := models.StorageModuleConfig{
		ModuleCode:    "archives",
		ModuleName:    "生产模块配置",
		BaseDirectory: "/data/prod",
		Enabled:       true,
	}
	if err := db.Create(&prodModule).Error; err != nil {
		t.Fatalf("创建非 E2E 模块配置失败: %v", err)
	}
	// 预置非 E2E 规则（同模块+资源类型，但名称不同、优先级更低）
	prodRule := models.StorageRule{
		StorageID:    prodConfig.ID,
		ModuleCode:   "archives",
		ResourceType: "resign_proof",
		Name:         "生产规则",
		Priority:     50,
		Enabled:      true,
		TargetType:   "document",
	}
	if err := db.Create(&prodRule).Error; err != nil {
		t.Fatalf("创建非 E2E 存储规则失败: %v", err)
	}

	if _, err := ensureE2EStorage(db); err != nil {
		t.Fatalf("执行 E2E 存储 seed 失败: %v", err)
	}

	// 非 E2E 配置保持原样
	var reloadedConfig models.StorageConfig
	if err := db.First(&reloadedConfig, prodConfig.ID).Error; err != nil {
		t.Fatalf("查询非 E2E 存储配置失败: %v", err)
	}
	if !reloadedConfig.Enabled || !reloadedConfig.IsDefault || reloadedConfig.Status != "active" {
		t.Errorf("非 E2E 存储配置不应被改动，实际: enabled=%v is_default=%v status=%q",
			reloadedConfig.Enabled, reloadedConfig.IsDefault, reloadedConfig.Status)
	}
	if got := storageRootPath(t, reloadedConfig); got != "/data/prod" {
		t.Errorf("非 E2E 存储配置 root_path 不应被改动，实际为 %q", got)
	}

	// 非 E2E 模块配置保持原样
	var reloadedModule models.StorageModuleConfig
	if err := db.First(&reloadedModule, prodModule.ID).Error; err != nil {
		t.Fatalf("查询非 E2E 模块配置失败: %v", err)
	}
	if !reloadedModule.Enabled || reloadedModule.BaseDirectory != "/data/prod" {
		t.Errorf("非 E2E 模块配置不应被改动，实际: enabled=%v base_directory=%q",
			reloadedModule.Enabled, reloadedModule.BaseDirectory)
	}

	// 非 E2E 规则保持原样（未被删除、未被改指向）
	var reloadedRule models.StorageRule
	if err := db.First(&reloadedRule, prodRule.ID).Error; err != nil {
		t.Fatalf("查询非 E2E 存储规则失败: %v", err)
	}
	if !reloadedRule.Enabled || reloadedRule.StorageID != prodConfig.ID || reloadedRule.Priority != 50 {
		t.Errorf("非 E2E 存储规则不应被改动，实际: enabled=%v storage_id=%d priority=%d",
			reloadedRule.Enabled, reloadedRule.StorageID, reloadedRule.Priority)
	}
}

// TestEnsureE2EStorage_ResolvesViaStorageRouter 实际 StorageRouter 应把 archives/resign_proof 解析到 E2E 配置
func TestEnsureE2EStorage_ResolvesViaStorageRouter(t *testing.T) {
	db := setupStorageDB(t)

	storageID, err := ensureE2EStorage(db)
	if err != nil {
		t.Fatalf("初始化 E2E 存储失败: %v", err)
	}

	router := storage.NewStorageRouter(db)
	route, err := router.Resolve(context.Background(), storage.ResolveRequest{
		ModuleCode:   "archives",
		ResourceType: "resign_proof",
	})
	if err != nil {
		t.Fatalf("StorageRouter 解析 archives/resign_proof 失败: %v", err)
	}
	if route.StorageID != storageID {
		t.Errorf("解析结果 storage_id 应为 %d，实际为 %d", storageID, route.StorageID)
	}
	if route.StorageType != "local" {
		t.Errorf("解析结果存储类型应为 local，实际为 %q", route.StorageType)
	}
	if route.StorageConfig == nil || route.StorageConfig.Name != e2eStorageConfigName {
		t.Errorf("解析结果应指向 E2E 存储配置，实际: %+v", route.StorageConfig)
	}
}
