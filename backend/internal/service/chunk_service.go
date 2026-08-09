package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// ChunkService 分块编辑与版本管理服务
// 负责 DocumentChunk 的内容更新、版本快照、回滚以及向量索引重排。
type ChunkService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
}

// NewChunkService 构造函数
func NewChunkService(db *gorm.DB, embeddingService *EmbeddingService) *ChunkService {
	return &ChunkService{
		db:               db,
		embeddingService: embeddingService,
	}
}

// ============================================================
// 分块内容编辑
// ============================================================

// UpdateChunk 更新分块内容并触发异步向量重索引
// 使用乐观锁（ContentRevision）防止并发覆盖；首次编辑会惰性回填 SourceContent。
func (s *ChunkService) UpdateChunk(chunkID uint, editorID uint, newContent string, expectedRevision int) error {
	var chunk models.DocumentChunk
	if err := s.db.First(&chunk, chunkID).Error; err != nil {
		return fmt.Errorf("分块不存在: %v", err)
	}

	// 乐观锁校验
	if chunk.ContentRevision != expectedRevision {
		return fmt.Errorf("revision conflict: expected %d, got %d", expectedRevision, chunk.ContentRevision)
	}

	// 首次编辑时惰性回填 SourceContent
	if chunk.SourceContent == "" {
		chunk.SourceContent = chunk.Content
	}

	oldContent := chunk.Content
	oldRevision := chunk.ContentRevision

	// 生成新内容哈希
	hash := sha256.Sum256([]byte(newContent))
	newHash := fmt.Sprintf("%x", hash)

	// 保存版本快照（旧内容、旧版本号）
	revision := &models.ChunkRevision{
		ChunkID:    chunkID,
		Revision:   oldRevision,
		Content:    oldContent,
		EditorID:   &editorID,
		EditSource: "manual",
	}

	// 在事务内完成快照保存与分块更新
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(revision).Error; err != nil {
			return fmt.Errorf("保存版本快照失败: %v", err)
		}

		updates := map[string]interface{}{
			"content":           newContent,
			"source_content":    chunk.SourceContent,
			"content_hash":      newHash,
			"content_revision":  oldRevision + 1,
			"index_status":      models.IndexStatusProcessing,
			"updated_at":        time.Now(),
		}
		if err := tx.Model(&chunk).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新分块失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 异步重索引：重新生成向量并写入 pgvector
	go s.reindexChunk(chunkID, chunk.DocID)

	return nil
}

// RevertChunk 将分块回滚到指定历史版本
// 会先保存当前内容为新的版本快照，再写入目标版本内容。
func (s *ChunkService) RevertChunk(chunkID uint, editorID uint, targetRevision int) error {
	var revision models.ChunkRevision
	if err := s.db.Where("chunk_id = ? AND revision = ?", chunkID, targetRevision).First(&revision).Error; err != nil {
		return fmt.Errorf("目标版本不存在: %v", err)
	}

	var chunk models.DocumentChunk
	if err := s.db.First(&chunk, chunkID).Error; err != nil {
		return fmt.Errorf("分块不存在: %v", err)
	}

	// 使用当前版本号作为乐观锁期望值
	return s.UpdateChunk(chunkID, editorID, revision.Content, chunk.ContentRevision)
}

// ============================================================
// 版本历史查询
// ============================================================

// ListChunkRevisions 列出分块的所有版本快照，按版本号降序排列
func (s *ChunkService) ListChunkRevisions(chunkID uint) ([]models.ChunkRevision, error) {
	var revisions []models.ChunkRevision
	if err := s.db.Where("chunk_id = ?", chunkID).Order("revision DESC").Find(&revisions).Error; err != nil {
		return nil, fmt.Errorf("查询版本历史失败: %v", err)
	}
	return revisions, nil
}

// ============================================================
// 向量索引重试
// ============================================================

// ReindexChunk 手动触发指定分块的向量重索引（用户级重试入口）
func (s *ChunkService) ReindexChunk(chunkID uint) error {
	var chunk models.DocumentChunk
	if err := s.db.First(&chunk, chunkID).Error; err != nil {
		return fmt.Errorf("分块不存在: %v", err)
	}

	// 标记为处理中
	if err := s.db.Model(&chunk).Update("index_status", models.IndexStatusProcessing).Error; err != nil {
		return fmt.Errorf("更新索引状态失败: %v", err)
	}

	go s.reindexChunk(chunkID, chunk.DocID)
	return nil
}

// reindexChunk 私有重索引逻辑：生成 embedding 并写入 pgvector
// 成功后 IndexStatus=ready，失败则置为 failed。
func (s *ChunkService) reindexChunk(chunkID uint, docID uint) {
	// 获取文档所属用户，用于读取 embedding 配置
	var userID uint
	if err := s.db.Model(&models.Document{}).Select("user_id").Where("id = ?", docID).Scan(&userID).Error; err != nil {
		log.Printf("[chunk_service] 获取文档 %d 用户失败: %v", docID, err)
		s.markIndexFailed(chunkID)
		return
	}

	// 重新加载最新分块内容
	var chunk models.DocumentChunk
	if err := s.db.First(&chunk, chunkID).Error; err != nil {
		log.Printf("[chunk_service] 加载分块 %d 失败: %v", chunkID, err)
		s.markIndexFailed(chunkID)
		return
	}

	text := chunk.EmbeddingContent()
	if text == "" {
		log.Printf("[chunk_service] 分块 %d 向量化文本为空", chunkID)
		s.markIndexFailed(chunkID)
		return
	}

	vec, err := s.embeddingService.GenerateEmbedding(userID, text)
	if err != nil || len(vec) == 0 {
		if err != nil {
			log.Printf("[chunk_service] 分块 %d 生成向量失败: %v", chunkID, err)
		} else {
			log.Printf("[chunk_service] 分块 %d 生成向量为空", chunkID)
		}
		s.markIndexFailed(chunkID)
		return
	}

	// 写入 pgvector 与 JSON 副本
	if err := s.writeChunkVector(chunkID, vec); err != nil {
		log.Printf("[chunk_service] 分块 %d 写入向量失败: %v", chunkID, err)
		s.markIndexFailed(chunkID)
		return
	}

	log.Printf("[chunk_service] 分块 %d 重索引完成", chunkID)
}

// writeChunkVector 通过 raw SQL 写入 pgvector 向量和 JSON 副本
func (s *ChunkService) writeChunkVector(chunkID uint, vec []float64) error {
	vecStr := vectorToPGString(vec)
	vecJSON, _ := json.Marshal(vec)

	return s.db.Exec(`
		UPDATE document_chunks
		SET embedding = $1::vector, embedding_json = $2, index_status = 'ready', updated_at = $3
		WHERE id = $4
	`, vecStr, datatypes.JSON(vecJSON), time.Now(), chunkID).Error
}

// markIndexFailed 将分块索引状态标记为失败
func (s *ChunkService) markIndexFailed(chunkID uint) {
	if err := s.db.Model(&models.DocumentChunk{}).Where("id = ?", chunkID).
		Update("index_status", models.IndexStatusFailed).Error; err != nil {
		log.Printf("[chunk_service] 标记分块 %d 索引失败状态失败: %v", chunkID, err)
	}
}
