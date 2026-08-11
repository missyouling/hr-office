// Package service 提供知识库入库服务
// 负责将各业务模块（员工/宿舍/社保/档案/办公用品/食堂/发票）的源数据
// 经 docreader 解析 → 分块 → 向量化后入库到知识库对应的 DocumentChunk 中。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/docreader"
)

// 每次入库扫描的最大记录数，避免单次事务过长
const maxBatchSize = 100

// docParser 文档解析器接口，便于测试 mock
type docParser interface {
	Parse(ctx context.Context, req docreader.ParseRequest) (*docreader.ParseResult, error)
}

// KBIngestService 知识库入库服务
type KBIngestService struct {
	db           *gorm.DB
	parser       docParser // 文档解析器（*docreader.Client 实现该接口）
	embeddingSvc *EmbeddingService
}

// NewKBIngestService 创建知识库入库服务
// docClient 采用懒连接模式：创建时不验证连接，首次 Parse 调用时才建立连接
func NewKBIngestService(db *gorm.DB, docClient *docreader.Client, embeddingSvc *EmbeddingService) *KBIngestService {
	return &KBIngestService{
		db:           db,
		parser:       docClient, // *docreader.Client 实现了 docParser 接口
		embeddingSvc: embeddingSvc,
	}
}

// IngestRequest 知识库入库请求
type IngestRequest struct {
	KBID         uint   `json:"kb_id"`
	Since        string `json:"since,omitempty"` // 日期过滤 YYYY-MM-DD，可选
	SourceModule string `json:"source_module"`   // 来源模块标识
}

// IngestResult 入库执行结果
type IngestResult struct {
	Scanned  int      `json:"scanned"`
	Ingested int      `json:"ingested"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// ============================================================
// Ingest 主入口
// ============================================================

// Ingest 执行半自动入库：按 KB 的 source_module 扫描源数据 → 解析 → 分块 → 向量化。
// userID 用于 embedding 配置查找；ctx 用于超时控制。
func (s *KBIngestService) Ingest(ctx context.Context, userID uint, req IngestRequest) (*IngestResult, error) {
	// 1. 校验知识库存在 + source_module 匹配
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, req.KBID).Error; err != nil {
		return nil, fmt.Errorf("知识库 %d 不存在: %w", req.KBID, err)
	}
	if kb.SourceModule != "" && kb.SourceModule != req.SourceModule {
		return nil, fmt.Errorf("source_module 不匹配: 知识库为 %s，请求为 %s", kb.SourceModule, req.SourceModule)
	}

	// 2. 查询源数据
	records, err := s.querySourceModule(req.SourceModule, req.Since)
	if err != nil {
		return nil, fmt.Errorf("查询源数据失败: %w", err)
	}

	result := &IngestResult{Scanned: len(records), Errors: make([]string, 0)}

	// 3. 逐条入库
	for _, rec := range records {
		if err := s.ingestRecord(ctx, userID, req.KBID, rec); err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Skipped++
			log.Printf("[kb_ingest] 记录 %d 入库失败: %v", rec.ID, err)
			continue
		}
		result.Ingested++
	}

	return result, nil
}

// ============================================================
// 源数据查询
// ============================================================

// sourceRecord 统一源记录视图
type sourceRecord struct {
	ID        uint      // 源记录主键
	Text      string    // 序列化后的文本
	CreatedAt time.Time // 创建时间
}

// querySourceModule 按 source_module 查对应源表，返回统一视图切片
func (s *KBIngestService) querySourceModule(module, since string) ([]sourceRecord, error) {
	// 构建带 since 过滤的基础查询
	baseQuery := func(db *gorm.DB) *gorm.DB {
		q := db.Order("created_at DESC").Limit(maxBatchSize)
		if since != "" {
			q = q.Where("created_at >= ?", since)
		}
		return q
	}

	switch module {
	case "employee":
		var items []models.Employee
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询员工数据失败: %w", err)
		}
		return toSourceRecords(items, func(e models.Employee) (uint, time.Time, any) {
			return e.ID, e.CreatedAt, e
		}), nil

	case "dormitory":
		var items []models.DormContract
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询宿舍合同失败: %w", err)
		}
		return toSourceRecords(items, func(c models.DormContract) (uint, time.Time, any) {
			return c.ID, c.CreatedAt, c
		}), nil

	case "insurance":
		var items []models.SocialInsuranceRecord
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询社保记录失败: %w", err)
		}
		return toSourceRecords(items, func(r models.SocialInsuranceRecord) (uint, time.Time, any) {
			return r.ID, r.CreatedAt, r
		}), nil

	case "archives":
		var items []models.Document
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询档案数据失败: %w", err)
		}
		// archives 特殊处理：使用 ContentText 作为文本内容
		records := make([]sourceRecord, 0, len(items))
		for _, d := range items {
			text := d.ContentText
			if text == "" {
				// ContentText 为空时仍用 JSON 序列化作为兜底
				b, _ := json.Marshal(d)
				text = string(b)
			}
			records = append(records, sourceRecord{
				ID:        d.ID,
				Text:      text,
				CreatedAt: d.CreatedAt,
			})
		}
		return records, nil

	case "office":
		var items []models.OfficePurchase
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询办公用品采购失败: %w", err)
		}
		return toSourceRecords(items, func(o models.OfficePurchase) (uint, time.Time, any) {
			return o.ID, o.CreatedAt, o
		}), nil

	case "canteen":
		var items []models.CanteenPurchase
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询食堂采购失败: %w", err)
		}
		return toSourceRecords(items, func(c models.CanteenPurchase) (uint, time.Time, any) {
			return c.ID, c.CreatedAt, c
		}), nil

	case "invoice":
		var items []models.Invoice
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询发票数据失败: %w", err)
		}
		return toSourceRecords(items, func(i models.Invoice) (uint, time.Time, any) {
			return i.ID, i.CreatedAt, i
		}), nil

	default:
		return nil, fmt.Errorf("未知的 source_module: %s", module)
	}
}

// toSourceRecords 泛型转换函数：将模型切片转为统一 sourceRecord 切片
func toSourceRecords[T any](items []T, extract func(T) (uint, time.Time, any)) []sourceRecord {
	records := make([]sourceRecord, 0, len(items))
	for _, item := range items {
		id, createdAt, value := extract(item)
		text, err := json.Marshal(value)
		if err != nil {
			log.Printf("[kb_ingest] 序列化记录 %d 失败: %v", id, err)
			continue
		}
		records = append(records, sourceRecord{
			ID:        id,
			Text:      string(text),
			CreatedAt: createdAt,
		})
	}
	return records
}

// ============================================================
// 单条记录入库
// ============================================================

// ingestRecord 处理单条源记录的完整入库链路
func (s *KBIngestService) ingestRecord(ctx context.Context, userID uint, kbID uint, rec sourceRecord) error {
	// 1. 写临时文件 → docreader 解析
	result, err := s.parseRecord(ctx, rec)
	if err != nil {
		return fmt.Errorf("文档解析失败: %w", err)
	}

	// 2. 解析结果写入 DocumentChunk
	docID := rec.ID // 用源记录 ID 作为 DocID
	chunkCount, err := docreader.IngestToChunks(s.db, result, docID)
	if err != nil {
		return fmt.Errorf("写入分块失败: %w", err)
	}
	log.Printf("[kb_ingest] 记录 %d 解析完成，写入 %d 个分块", rec.ID, chunkCount)

	// 3. 向量化已创建的分块（失败不中断）
	s.embedChunks(userID, docID)

	return nil
}

// parseRecord 将源记录文本转为 ParseResult
// archives 模块使用 ContentText 直接构造 ParseResult，跳过 docreader
func (s *KBIngestService) parseRecord(ctx context.Context, rec sourceRecord) (*docreader.ParseResult, error) {
	// 写临时文件供 docreader 解析
	tmpFile, err := os.CreateTemp("", "kb-ingest-*.txt")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(rec.Text); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	// 调用 docreader 解析
	parseReq := docreader.ParseRequest{
		FilePath: tmpFile.Name(),
		FileType: filepath.Ext(tmpFile.Name()),
	}
	return s.parser.Parse(ctx, parseReq)
}

// embedChunks 对指定 docID 的所有分块生成向量
// 向量化失败不中断流程，标记 index_status="pending" 等待后续重试
func (s *KBIngestService) embedChunks(userID uint, docID uint) {
	var chunks []models.DocumentChunk
	if err := s.db.Where("doc_id = ? AND index_status = ?", docID, models.IndexStatusProcessing).Find(&chunks).Error; err != nil {
		log.Printf("[kb_ingest] 查询分块失败: %v", err)
		return
	}

	for i := range chunks {
		chunk := &chunks[i]
		embedText := chunk.EmbeddingContent()
		if embedText == "" {
			chunk.IndexStatus = models.IndexStatusFailed
			s.db.Model(chunk).Update("index_status", models.IndexStatusFailed)
			continue
		}

		// 生成嵌入向量
		vec, err := s.embeddingSvc.GenerateEmbedding(userID, embedText)
		if err != nil {
			log.Printf("[kb_ingest] 分块 %d 向量化失败: %v", chunk.ID, err)
			chunk.IndexStatus = "pending" // 标记待重试
			s.db.Model(chunk).Update("index_status", "pending")
			continue
		}
		if len(vec) == 0 {
			log.Printf("[kb_ingest] 分块 %d 向量为空，标记 pending", chunk.ID)
			chunk.IndexStatus = "pending"
			s.db.Model(chunk).Update("index_status", "pending")
			continue
		}

		// 写入向量 JSON 副本并更新状态
		vecJSON, _ := json.Marshal(vec)
		s.db.Model(chunk).Updates(map[string]interface{}{
			"embedding_json": vecJSON,
			"index_status":   models.IndexStatusReady,
		})
	}
}
