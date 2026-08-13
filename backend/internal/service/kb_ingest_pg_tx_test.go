package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"siapp/internal/models"
)

// ============================================================
// 5. SavePoint 唯一冲突恢复（PostgreSQL 事务 aborted 语义验证）
//    PG 中事务内语句失败后整个事务进入 aborted，后续命令被拒绝，
//    必须 RollbackTo 恢复；findOrCreateShadowDocument 依赖此机制。
// ============================================================

func TestP9PG_SavePointRecovery(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	// documents.user_id 存在外键约束（fk_documents_user），需先创建测试用户
	user := seedP9PGUser(t, db, prefix, 1)

	// 预插入占位文档（占用 document_code 唯一键，模拟并发方已提交）
	now := time.Now()
	conflict := models.Document{
		UserID:       user.ID,
		DocumentCode: prefix + "-CONFLICT",
		FileName:     "占位",
		SourceType:   "other",
		SourceID:     999,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&conflict).Error; err != nil {
		t.Fatalf("预插入冲突记录失败: %v", err)
	}

	// 事务内 Create 触发 document_code 唯一冲突 → RollbackTo 恢复 → 事务可继续
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}
	doc := models.Document{
		UserID:       user.ID,
		DocumentCode: prefix + "-CONFLICT", // 与占位记录冲突
		FileName:     "触发冲突",
		SourceType:   "employee",
		SourceID:     1,
		Status:       "active",
		OCRStatus:    "completed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	tx.SavePoint("sp_shadow_create")
	createErr := tx.Create(&doc).Error
	if createErr == nil {
		tx.Rollback()
		t.Fatal("预期 Create 触发唯一冲突，实际成功")
	}
	if !isUniqueViolation(createErr) {
		tx.Rollback()
		t.Fatalf("预期唯一冲突错误，实际: %v", createErr)
	}
	// PG 关键语义：失败语句后事务 aborted，必须 RollbackTo 才能继续
	if rb := tx.RollbackTo("sp_shadow_create"); rb.Error != nil {
		tx.Rollback()
		t.Fatalf("RollbackTo 失败: %v", rb.Error)
	}
	if err := tx.Exec(
		"INSERT INTO documents (user_id, document_code, file_name, status, ocr_status, created_at, updated_at) VALUES (?, ?, '恢复后写入', 'active', 'completed', ?, ?)",
		user.ID, prefix+"-AFTER-SP", now, now,
	).Error; err != nil {
		tx.Rollback()
		t.Fatalf("RollbackTo 后事务应可继续写入，实际失败: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	// 验证：恢复后写入的记录存在
	var after int64
	db.Model(&models.Document{}).Where("document_code = ?", prefix+"-AFTER-SP").Count(&after)
	if after != 1 {
		t.Fatalf("恢复后写入的记录期望 1，实际 %d", after)
	}
}

// ============================================================
// 6. 并发同 source_type/source_id/source_kb_id 入库：
//    最终一个影子文档、两个请求均成功语义
// ============================================================

func TestP9PG_ConcurrentIngest(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	user := seedP9PGUser(t, db, prefix, 1)
	kb := seedP9PGKB(t, db, prefix)
	seedP9PGEmployee(t, db, user.ID, prefix, "周伯通")

	svc := newP9PGService(t, db, p9FixedParser())
	req := IngestRequest{KBID: kb.ID, SourceModule: "employee"}

	// 两个 goroutine 并发 ingest 同源同 KB
	var wg sync.WaitGroup
	results := make([]*IngestResult, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.Ingest(context.Background(), user.ID, req)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发 ingest #%d 失败: %v", i, err)
		}
		// 冲突后必须复用成功：两个请求都计入 ingested=1（而非一个失败 Skipped）
		if results[i].Ingested != 1 {
			t.Fatalf("并发 ingest #%d 期望 Ingested=1（冲突后复用），实际 %d（Skipped=%d）",
				i, results[i].Ingested, results[i].Skipped)
		}
	}

	// 唯一键兜底：最终只有 1 个影子文档（限定本 KB，避免共享库其他数据干扰）
	var shadowCount int64
	db.Model(&models.Document{}).
		Where("source_type = ? AND source_kb_id = ?", "employee", kb.ID).
		Count(&shadowCount)
	if shadowCount != 1 {
		t.Fatalf("并发同源同 KB 后影子文档期望 1，实际 %d", shadowCount)
	}
	// chunks 不翻倍：1 个 section → 1 个 chunk
	var chunkCount int64
	db.Model(&models.DocumentChunk{}).
		Joins("JOIN documents ON documents.id = document_chunks.doc_id").
		Where("documents.source_kb_id = ?", kb.ID).
		Count(&chunkCount)
	if chunkCount != 1 {
		t.Fatalf("并发后 chunk 期望 1，实际 %d", chunkCount)
	}
}

// ============================================================
// 7. 可重复运行：独立前缀 + 清理后无残留，同前缀可再次入库
// ============================================================

func TestP9PG_RepeatableRunAndCleanup(t *testing.T) {
	db, prefix := setupP9PGDB(t)
	if db == nil {
		return
	}
	user := seedP9PGUser(t, db, prefix, 1)
	kb := seedP9PGKB(t, db, prefix)
	seedP9PGEmployee(t, db, user.ID, prefix, "洪七公")

	svc := newP9PGService(t, db, p9FixedParser())
	req := IngestRequest{KBID: kb.ID, SourceModule: "employee"}
	if _, err := svc.Ingest(context.Background(), user.ID, req); err != nil {
		t.Fatalf("首次 Ingest 失败: %v", err)
	}

	// 模拟测试结束清理：按前缀删除后无残留
	p9PGCleanup(t, db, prefix)
	var leftover int64
	db.Model(&models.Document{}).Where("source_kb_id = ?", kb.ID).Count(&leftover)
	if leftover != 0 {
		t.Fatalf("清理后影子文档应无残留，实际 %d", leftover)
	}

	// 同前缀再次入库（可重复运行）：重新 seed 后 ingest 成功
	user2 := seedP9PGUser(t, db, prefix, 1)
	kb2 := seedP9PGKB(t, db, prefix)
	seedP9PGEmployee(t, db, user2.ID, prefix, "洪七公")
	if _, err := svc.Ingest(context.Background(), user2.ID, IngestRequest{KBID: kb2.ID, SourceModule: "employee"}); err != nil {
		t.Fatalf("二次 Ingest 失败: %v", err)
	}
	var shadowCount int64
	db.Model(&models.Document{}).
		Where("source_type = ? AND source_kb_id = ?", "employee", kb2.ID).
		Count(&shadowCount)
	if shadowCount != 1 {
		t.Fatalf("二次入库后影子文档期望 1，实际 %d", shadowCount)
	}
}
