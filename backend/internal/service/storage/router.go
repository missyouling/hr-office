package storage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ResolveRequest 路由解析请求
type ResolveRequest struct {
	ModuleCode   string
	ResourceType string
	Filename     string
	FileSize     int64
}

// ResolvedRoute 解析结果
type ResolvedRoute struct {
	StorageConfig *models.StorageConfig
	StorageType   string // local/s3/webdav
	BasePath      string // /archives/employee_photos
	FullPath      string // /archives/employee_photos/2025-04-18/photo.jpg
	StorageID     uint
}

// StorageRouter 存储路由解析引擎
type StorageRouter struct {
	db *gorm.DB
}

// NewStorageRouter 创建路由解析器
func NewStorageRouter(db *gorm.DB) *StorageRouter {
	return &StorageRouter{db: db}
}

// Resolve 根据模块和资源类型解析存储目标
// 查询优先级：精确规则 → 模块默认 → 全局默认
func (r *StorageRouter) Resolve(ctx context.Context, req ResolveRequest) (*ResolvedRoute, error) {
	// 1. 查询该模块+资源类型的精确规则
	var rule models.StorageRule
	err := r.db.WithContext(ctx).
		Where("module_code = ? AND resource_type = ? AND enabled = ?", req.ModuleCode, req.ResourceType, true).
		Order("priority DESC").
		First(&rule).Error

	if err == nil {
		// 找到精确规则，查询对应的存储配置
		return r.resolveFromRule(ctx, rule, req)
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("query storage rule: %w", err)
	}

	// 2. 无精确规则，使用降级策略
	return r.resolveFallback(ctx, req)
}

// resolveFromRule 从规则解析存储路由
func (r *StorageRouter) resolveFromRule(ctx context.Context, rule models.StorageRule, req ResolveRequest) (*ResolvedRoute, error) {
	var config models.StorageConfig
	if err := r.db.WithContext(ctx).First(&config, rule.StorageID).Error; err != nil {
		return nil, fmt.Errorf("load storage config %d: %w", rule.StorageID, err)
	}

	basePath := "/" + req.ModuleCode + "/" + req.ResourceType
	fullPath := buildFullPath(basePath, req.Filename)

	return &ResolvedRoute{
		StorageConfig: &config,
		StorageType:   config.Type,
		BasePath:      basePath,
		FullPath:      fullPath,
		StorageID:     config.ID,
	}, nil
}

// resolveFallback 规则查询失败时的降级策略
// 优先级: 模块默认 → 全局默认 → 本地存储兜底
func (r *StorageRouter) resolveFallback(ctx context.Context, req ResolveRequest) (*ResolvedRoute, error) {
	// 1. 查询该模块的默认存储（通过 StorageModuleConfig）
	route, err := r.resolveModuleDefault(ctx, req)
	if err == nil {
		return route, nil
	}

	// 2. 使用全局默认存储
	route, err = r.resolveGlobalDefault(ctx, req)
	if err == nil {
		return route, nil
	}

	// 3. 兜底：本地存储（无需数据库配置）
	basePath := "/" + req.ModuleCode
	if req.ResourceType != "" {
		basePath += "/" + req.ResourceType
	}
	fullPath := buildFullPath(basePath, req.Filename)

	return &ResolvedRoute{
		StorageConfig: nil, // 表示使用本地默认存储
		StorageType:   "local",
		BasePath:      basePath,
		FullPath:      fullPath,
		StorageID:     0,
	}, nil
}

// resolveModuleDefault 使用模块默认存储
func (r *StorageRouter) resolveModuleDefault(ctx context.Context, req ResolveRequest) (*ResolvedRoute, error) {
	// 查询模块配置
	var moduleConfig models.StorageModuleConfig
	if err := r.db.WithContext(ctx).
		Where("module_code = ? AND enabled = ?", req.ModuleCode, true).
		First(&moduleConfig).Error; err != nil {
		return nil, fmt.Errorf("module config not found for %s: %w", req.ModuleCode, err)
	}

	// 查询该模块是否有不指定资源类型的通用规则
	var rule models.StorageRule
	err := r.db.WithContext(ctx).
		Where("module_code = ? AND (resource_type = '' OR resource_type IS NULL) AND enabled = ?", req.ModuleCode, true).
		Order("priority DESC").
		First(&rule).Error

	if err == nil {
		return r.resolveFromRule(ctx, rule, req)
	}

	// 没有通用规则，使用全局默认
	return nil, fmt.Errorf("no default rule for module %s", req.ModuleCode)
}

// resolveGlobalDefault 使用全局默认存储
func (r *StorageRouter) resolveGlobalDefault(ctx context.Context, req ResolveRequest) (*ResolvedRoute, error) {
	var config models.StorageConfig
	if err := r.db.WithContext(ctx).
		Where("is_default = ? AND enabled = ?", true, true).
		First(&config).Error; err != nil {
		return nil, fmt.Errorf("no default storage config found: %w", err)
	}

	basePath := "/" + req.ModuleCode
	if req.ResourceType != "" {
		basePath += "/" + req.ResourceType
	}
	fullPath := buildFullPath(basePath, req.Filename)

	return &ResolvedRoute{
		StorageConfig: &config,
		StorageType:   config.Type,
		BasePath:      basePath,
		FullPath:      fullPath,
		StorageID:     config.ID,
	}, nil
}

// buildFullPath 根据日期+文件名构建完整路径
func buildFullPath(basePath, filename string) string {
	dateDir := time.Now().Format("2006-01-02")
	return basePath + "/" + dateDir + "/" + filename
}
