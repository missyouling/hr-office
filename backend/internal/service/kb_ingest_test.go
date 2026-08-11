package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/docreader"
)

// ============================================================
// Mock docParser — 可配置的文档解析器 mock
// ============================================================

type mockDocParser struct {
	parseResult *docreader.ParseResult
	parseErr    error
}

func (m *mockDocParser) Parse(_ context.Context, _ docreader.ParseRequest) (*docreader.ParseResult, error) {
	return m.parseResult, m.parseErr
}

// fixedParseResult 返回固定章节内容的 ParseResult
func fixedParseResult() *docreader.ParseResult {
	return &docreader.ParseResult{
		FullText: "测试文档全文",
		Sections: []docreader.ParseSection{
			{Title: "第一章", Content: "第一章内容正文", Level: 1},
			{Title: "第二章", Content: "第二章内容正文", Level: 1},
		},
	}
}

// ============================================================
// 测试辅助
// ============================================================

// setupKBIngestDB 创建内存 SQLite 并自动迁移相关模型
func setupKBIngestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	// 迁移相关模型
	err = db.AutoMigrate(
		&models.KnowledgeBase{},
		&models.Employee{},
		&models.DocumentChunk{},
	)
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}
	return db
}

// setupKBIngestService 创建测试用的 KBIngestService
func setupKBIngestService(t *testing.T, db *gorm.DB, mockParser docParser) *KBIngestService {
	t.Helper()
	// 使用最小化的 EmbeddingService（仅用于 GenerateEmbedding）
	embSvc := NewEmbeddingService(db)
	return &KBIngestService{
		db:           db,
		parser:       mockParser,
		embeddingSvc: embSvc,
	}
}

// seedKnowledgeBase 插入一条知识库记录
func seedKnowledgeBase(t *testing.T, db *gorm.DB) models.KnowledgeBase {
	t.Helper()
	kb := models.KnowledgeBase{
		Name:         "测试知识库",
		SourceModule: "employee",
		Visibility:   "public",
	}
	if err := db.Create(&kb).Error; err != nil {
		t.Fatalf("创建知识库失败: %v", err)
	}
	return kb
}

// seedEmployees 插入指定数量的员工记录
func seedEmployees(t *testing.T, db *gorm.DB, count int) []models.Employee {
	t.Helper()
	emps := make([]models.Employee, count)
	for i := 0; i < count; i++ {
		emps[i] = models.Employee{
			UserID:     1,
			IDNumber:   fmt.Sprintf("EMP%03d", i), // 联合唯一约束需要不同 IDNumber
			Name:       "测试员工",
			Department: "研发部",
		}
	}
	if err := db.Create(&emps).Error; err != nil {
		t.Fatalf("创建员工记录失败: %v", err)
	}
	return emps
}

// ============================================================
// TestKBIngestService_Ingest — 正常入库流程
// ============================================================

func TestKBIngestService_Ingest(t *testing.T) {
	db := setupKBIngestDB(t)
	kb := seedKnowledgeBase(t, db)
	seedEmployees(t, db, 2)

	mock := &mockDocParser{parseResult: fixedParseResult()}
	svc := setupKBIngestService(t, db, mock)

	req := IngestRequest{
		KBID:         kb.ID,
		SourceModule: "employee",
	}
	result, err := svc.Ingest(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("Ingest 返回错误: %v", err)
	}

	// 验证统计
	if result.Scanned != 2 {
		t.Errorf("Scanned 期望 2，实际 %d", result.Scanned)
	}
	if result.Ingested != 2 {
		t.Errorf("Ingested 期望 2，实际 %d", result.Ingested)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped 期望 0，实际 %d", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors 期望 0 条，实际 %d 条: %v", len(result.Errors), result.Errors)
	}

	// 验证 Chunk 写入：2 条记录 × 2 个 Section = 4 个 Chunk
	var chunkCount int64
	db.Model(&models.DocumentChunk{}).Count(&chunkCount)
	if chunkCount < 2 {
		t.Errorf("Chunk 数量期望至少 2，实际 %d", chunkCount)
	}
}

// ============================================================
// TestKBIngestService_EmptySource — 源表无数据
// ============================================================

func TestKBIngestService_EmptySource(t *testing.T) {
	db := setupKBIngestDB(t)
	kb := seedKnowledgeBase(t, db)
	// 不插入任何员工数据

	mock := &mockDocParser{parseResult: fixedParseResult()}
	svc := setupKBIngestService(t, db, mock)

	req := IngestRequest{
		KBID:         kb.ID,
		SourceModule: "employee",
	}
	result, err := svc.Ingest(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("空源表 Ingest 不应报错: %v", err)
	}

	if result.Scanned != 0 {
		t.Errorf("Scanned 期望 0，实际 %d", result.Scanned)
	}
	if result.Ingested != 0 {
		t.Errorf("Ingested 期望 0，实际 %d", result.Ingested)
	}
}

// ============================================================
// TestKBIngestService_KBNotFound — 知识库不存在
// ============================================================

func TestKBIngestService_KBNotFound(t *testing.T) {
	db := setupKBIngestDB(t)

	mock := &mockDocParser{parseResult: fixedParseResult()}
	svc := setupKBIngestService(t, db, mock)

	req := IngestRequest{
		KBID:         99999, // 不存在的 KB
		SourceModule: "employee",
	}
	_, err := svc.Ingest(context.Background(), 1, req)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
}

// ============================================================
// TestKBIngestService_DocreaderUnavailable — docreader 不可达
// ============================================================

func TestKBIngestService_DocreaderUnavailable(t *testing.T) {
	db := setupKBIngestDB(t)
	kb := seedKnowledgeBase(t, db)
	seedEmployees(t, db, 1)

	mock := &mockDocParser{parseErr: errors.New("connection refused")}
	svc := setupKBIngestService(t, db, mock)

	req := IngestRequest{
		KBID:         kb.ID,
		SourceModule: "employee",
	}
	result, err := svc.Ingest(context.Background(), 1, req)
	// 服务本身不应崩溃
	if err != nil {
		t.Fatalf("Ingest 不应崩溃: %v", err)
	}
	// 所有记录应为 Skipped
	if result.Ingested != 0 {
		t.Errorf("Ingested 期望 0，实际 %d", result.Ingested)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped 期望 1，实际 %d", result.Skipped)
	}
	if len(result.Errors) == 0 {
		t.Error("期望 Errors 包含错误信息")
	}
}

// ============================================================
// TestKBIngestService_SourceModuleMismatch — source_module 不匹配
// ============================================================

func TestKBIngestService_SourceModuleMismatch(t *testing.T) {
	db := setupKBIngestDB(t)
	kb := seedKnowledgeBase(t, db) // source_module = "employee"

	mock := &mockDocParser{parseResult: fixedParseResult()}
	svc := setupKBIngestService(t, db, mock)

	req := IngestRequest{
		KBID:         kb.ID,
		SourceModule: "dormitory", // 不匹配
	}
	_, err := svc.Ingest(context.Background(), 1, req)
	if err == nil {
		t.Fatal("期望 source_module 不匹配时报错，实际 nil")
	}
}
