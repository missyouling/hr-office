package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

func setupChunkTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	// SQLite 内存数据库绑定到连接，限制连接池大小为 1，
	// 避免异步 goroutine 拿到空的新连接导致表不存在。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取数据库连接对象失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.AutoMigrate(&models.User{}, &models.Document{}, &models.DocumentChunk{}, &models.ChunkRevision{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

func TestChunk_Update_Success(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test doc", DocumentCode: "DOC-001"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "old content", ContentRevision: 1}
	db.Create(&chunk)

	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.UpdateChunk(chunk.ID, 1, "new content", 1)
	if err != nil {
		t.Fatalf("UpdateChunk 失败: %v", err)
	}

	var updated models.DocumentChunk
	db.First(&updated, chunk.ID)
	if updated.Content != "new content" {
		t.Errorf("content = %q, want %q", updated.Content, "new content")
	}
	if updated.ContentRevision != 2 {
		t.Errorf("revision = %d, want 2", updated.ContentRevision)
	}

	var count int64
	db.Model(&models.ChunkRevision{}).Where("chunk_id = ?", chunk.ID).Count(&count)
	if count != 1 {
		t.Errorf("revision count = %d, want 1", count)
	}
}

func TestChunk_Update_RevisionConflict(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-002"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "content", ContentRevision: 5}
	db.Create(&chunk)

	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.UpdateChunk(chunk.ID, 1, "new", 3)
	if err == nil {
		t.Error("期望 revision conflict 错误")
	}
}

func TestChunk_Update_ChunkNotFound(t *testing.T) {
	db := setupChunkTestDB(t)
	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.UpdateChunk(999, 1, "content", 0)
	if err == nil {
		t.Error("期望 chunk not found 错误")
	}
}

func TestChunk_Revert_Success(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-003"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "v2", ContentRevision: 2}
	db.Create(&chunk)
	rev := models.ChunkRevision{ChunkID: chunk.ID, Revision: 1, Content: "v1", EditorID: ptr(uint(1)), EditSource: "manual"}
	db.Create(&rev)

	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.RevertChunk(chunk.ID, 1, 1)
	if err != nil {
		t.Fatalf("RevertChunk 失败: %v", err)
	}
	var updated models.DocumentChunk
	db.First(&updated, chunk.ID)
	if updated.Content != "v1" {
		t.Errorf("content = %q, want %q", updated.Content, "v1")
	}
}

func TestChunk_Revert_TargetRevisionNotFound(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-004"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "content", ContentRevision: 1}
	db.Create(&chunk)

	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.RevertChunk(chunk.ID, 1, 999)
	if err == nil {
		t.Error("期望 target revision not found 错误")
	}
}

func TestChunk_ListRevisions_Success(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-005"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "v3", ContentRevision: 3}
	db.Create(&chunk)
	db.Create(&models.ChunkRevision{ChunkID: chunk.ID, Revision: 1, Content: "v1", EditSource: "manual"})
	db.Create(&models.ChunkRevision{ChunkID: chunk.ID, Revision: 2, Content: "v2", EditSource: "manual"})

	svc := NewChunkService(db, NewEmbeddingService(db))
	revisions, err := svc.ListChunkRevisions(chunk.ID)
	if err != nil {
		t.Fatalf("ListChunkRevisions 失败: %v", err)
	}
	if len(revisions) != 2 {
		t.Errorf("revisions count = %d, want 2", len(revisions))
	}
	if revisions[0].Revision < revisions[1].Revision {
		t.Error("revisions 应该按 revision DESC 排列")
	}
}

func TestChunk_ListRevisions_Empty(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-006"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "content"}
	db.Create(&chunk)

	svc := NewChunkService(db, NewEmbeddingService(db))
	revisions, err := svc.ListChunkRevisions(chunk.ID)
	if err != nil {
		t.Fatalf("ListChunkRevisions 失败: %v", err)
	}
	if len(revisions) != 0 {
		t.Errorf("revisions count = %d, want 0", len(revisions))
	}
}

func TestChunk_Reindex_NotFound(t *testing.T) {
	db := setupChunkTestDB(t)
	svc := NewChunkService(db, NewEmbeddingService(db))
	err := svc.ReindexChunk(999)
	if err == nil {
		t.Error("期望 chunk not found 错误")
	}
}

func TestChunk_EditorIDPreserved(t *testing.T) {
	db := setupChunkTestDB(t)
	doc := models.Document{UserID: 1, FileName: "test", DocumentCode: "DOC-007"}
	db.Create(&doc)
	chunk := models.DocumentChunk{DocID: doc.ID, ChunkIndex: 0, Content: "old", ContentRevision: 1}
	db.Create(&chunk)

	svc := NewChunkService(db, NewEmbeddingService(db))
	editorID := uint(42)
	svc.UpdateChunk(chunk.ID, editorID, "new", 1)

	var rev models.ChunkRevision
	db.Where("chunk_id = ?", chunk.ID).First(&rev)
	if rev.EditorID == nil || *rev.EditorID != editorID {
		t.Errorf("EditorID = %v, want %d", rev.EditorID, editorID)
	}
}

func ptr[T any](v T) *T { return &v }
