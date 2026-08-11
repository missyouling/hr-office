package service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// setupRerankTestDB 创建内存 SQLite 数据库用于测试
func setupRerankTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	// SQLite 内存数据库绑定到连接，限制连接池大小为 1
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取数据库连接对象失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// 迁移 ModelConfig 表
	if err := db.AutoMigrate(&models.ModelConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

// TestNewRerankerService 验证创建实例不崩溃
func TestNewRerankerService(t *testing.T) {
	db := setupRerankTestDB(t)
	svc := NewRerankerService(db)
	if svc == nil {
		t.Fatal("NewRerankerService 返回 nil")
	}
	if svc.db == nil {
		t.Fatal("RerankerService.db 为 nil")
	}
	if svc.httpClient == nil {
		t.Fatal("RerankerService.httpClient 为 nil")
	}
}

// TestRerank_NoConfig 未配置 rerank 模型时 fallback 到原始排序
func TestRerank_NoConfig(t *testing.T) {
	db := setupRerankTestDB(t)
	svc := NewRerankerService(db)

	original := []SearchResult{
		{ChunkID: 1, DocID: 10, Score: 0.9, Content: "原始第一名", MatchType: "vector"},
		{ChunkID: 2, DocID: 20, Score: 0.5, Content: "原始第二名", MatchType: "keyword"},
	}

	result, err := svc.Rerank(context.Background(), original, "测试查询", 0)
	if err != nil {
		t.Fatalf("Rerank 返回意外错误: %v", err)
	}

	// 没有 rerank 模型配置时，应保持原始顺序不变
	if len(result) != len(original) {
		t.Errorf("结果数量 = %d, 期望 %d", len(result), len(original))
	}
	for i := range result {
		if result[i].ChunkID != original[i].ChunkID {
			t.Errorf("索引 %d: ChunkID = %d, 期望 %d（fallback 应保持原始顺序）", i, result[i].ChunkID, original[i].ChunkID)
		}
	}
}

// TestRerank_EmptyResults 空结果数组不崩溃
func TestRerank_EmptyResults(t *testing.T) {
	db := setupRerankTestDB(t)
	svc := NewRerankerService(db)

	result, err := svc.Rerank(context.Background(), []SearchResult{}, "测试查询", 0)
	if err != nil {
		t.Fatalf("空结果 Rerank 返回意外错误: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("结果数量 = %d, 期望 0", len(result))
	}
}

// TestRerankWithConfig_NilConfig 传入 nil 配置时安全跳过
func TestRerankWithConfig_NilConfig(t *testing.T) {
	db := setupRerankTestDB(t)
	svc := NewRerankerService(db)

	original := []SearchResult{
		{ChunkID: 1, DocID: 10, Score: 0.9, Content: "文档A"},
		{ChunkID: 2, DocID: 20, Score: 0.8, Content: "文档B"},
	}

	result, err := svc.RerankWithConfig(context.Background(), original, "查询", nil)
	if err != nil {
		t.Fatalf("nil config RerankWithConfig 返回意外错误: %v", err)
	}

	if len(result) != len(original) {
		t.Errorf("结果数量 = %d, 期望 %d", len(result), len(original))
	}
}
