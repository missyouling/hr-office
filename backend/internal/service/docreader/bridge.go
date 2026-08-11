package docreader

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// IngestToChunks 将 docreader 解析结果转换为 DocumentChunk 并写入数据库。
//
// 逻辑：
//  1. 遍历 ParseResult.Sections，每个章节生成一个 chunk
//  2. ContextHeader 使用章节标题，Content 写入章节文本
//  3. IndexStatus 置为 "pending"，等待外部触发向量索引
//  4. 返回实际写入的 chunk 数量
//
// 注意：解析结果中的 FullText 仅作参考，不直接入库（由 sections 分块后写入）。
// embedding 回调暂未实现，调用方需自行触发向量化。
func IngestToChunks(db *gorm.DB, result *ParseResult, documentID uint) (chunkCount int, err error) {
	if result == nil {
		return 0, fmt.Errorf("解析结果为空")
	}

	sections := result.Sections
	if len(sections) == 0 {
		// 无章节时，将全文作为单个 chunk 写入
		sections = []ParseSection{{
			Title:   "",
			Content: result.FullText,
			Level:   0,
		}}
	}

	chunks := make([]models.DocumentChunk, 0, len(sections))
	now := time.Now()

	for i, sec := range sections {
		hash := sha256.Sum256([]byte(sec.Content))
		chunk := models.DocumentChunk{
			DocID:         documentID,
			ChunkIndex:    i,
			ChunkType:     models.ChunkTypeText,
			Content:       sec.Content,
			SourceContent: sec.Content,
			ContentHash:   fmt.Sprintf("%x", hash),
			ContextHeader: sec.Title,
			StartAt:       0, // 占位：暂不追踪原文精确位置
			EndAt:         len(sec.Content),
			IndexStatus:   models.IndexStatusProcessing, // 标记待向量化
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		chunks = append(chunks, chunk)
	}

	// 批量写入并更新块链（前后指针）
	if err := db.Create(&chunks).Error; err != nil {
		return 0, fmt.Errorf("写入分块失败: %w", err)
	}

	// 建立块链表（PreChunkID / NextChunkID），便于上下文扩展
	for i := range chunks {
		updates := map[string]interface{}{}
		if i > 0 {
			updates["pre_chunk_id"] = chunks[i-1].ID
		}
		if i < len(chunks)-1 {
			updates["next_chunk_id"] = chunks[i+1].ID
		}
		if len(updates) > 0 {
			if err := db.Model(&chunks[i]).Updates(updates).Error; err != nil {
				log.Printf("[docreader] 更新分块 %d 块链失败: %v", chunks[i].ID, err)
			}
		}
	}

	log.Printf("[docreader] 文档 %d 解析完成，写入 %d 个分块", documentID, len(chunks))

	// TODO: trigger embedding after chunk creation
	// 调用方需自行触发 embedding service 对新分块进行向量化

	return len(chunks), nil
}
