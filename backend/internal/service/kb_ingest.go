// Package service 提供知识库入库服务
// 负责将各业务模块（员工/宿舍/社保/档案/办公用品/食堂/发票）的源数据
// 经 docreader 解析 → 分块 → 向量化后入库到知识库对应的 DocumentChunk 中。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/datatypes"
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
	dialect      dbDialect // 数据库方言能力（向量写入策略）
}

// NewKBIngestService 创建知识库入库服务
// docClient 采用懒连接模式：创建时不验证连接，首次 Parse 调用时才建立连接
func NewKBIngestService(db *gorm.DB, docClient *docreader.Client, embeddingSvc *EmbeddingService) *KBIngestService {
	return &KBIngestService{
		db:           db,
		parser:       docClient, // *docreader.Client 实现了 docParser 接口
		embeddingSvc: embeddingSvc,
		dialect:      newDBDialect(db),
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
	// P9.1：向量化失败的分块数（维度不符/API 失败/空向量），不影响 ingested 统计
	EmbeddingFailed int `json:"embedding_failed,omitempty"`
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
		embedFailed, err := s.ingestRecord(ctx, userID, req.KBID, req.SourceModule, rec)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Skipped++
			log.Printf("[kb_ingest] 记录 %d 入库失败: %v", rec.ID, err)
			continue
		}
		result.Ingested++
		result.EmbeddingFailed += embedFailed
	}

	return result, nil
}

// ============================================================
// 源数据查询
// ============================================================

// sourceRecord 统一源记录视图
type sourceRecord struct {
	ID         uint      // 源记录主键
	UserID     uint      // 源记录所属用户（影子文档 user_id 元数据）
	Title      string    // 源记录可读标题（影子文档 file_name）
	Department string    // 源记录部门（影子文档部门元数据）
	Text       string    // 序列化后的文本
	CreatedAt  time.Time // 创建时间
}

// sourceUserID 兼容 *uint 用户字段：nil 视为 0（全局数据）
func sourceUserID(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
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
		return toSourceRecords(items, func(e models.Employee) sourceRecord {
			return sourceRecord{ID: e.ID, UserID: e.UserID, Title: "员工-" + e.Name, Department: e.Department, CreatedAt: e.CreatedAt}
		}), nil

	case "dormitory":
		var items []models.DormContract
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询宿舍合同失败: %w", err)
		}
		return toSourceRecords(items, func(c models.DormContract) sourceRecord {
			return sourceRecord{ID: c.ID, UserID: sourceUserID(c.UserID), Title: "宿舍合同-" + c.EmployeeName, Department: c.EmployeeDept, CreatedAt: c.CreatedAt}
		}), nil

	case "insurance":
		var items []models.SocialInsuranceRecord
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询社保记录失败: %w", err)
		}
		return toSourceRecords(items, func(r models.SocialInsuranceRecord) sourceRecord {
			return sourceRecord{ID: r.ID, UserID: r.UserID, Title: "社保-" + r.EmployeeName, Department: r.Department, CreatedAt: r.CreatedAt}
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
				UserID:    d.UserID,
				Title:     d.FileName,
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
		return toSourceRecords(items, func(o models.OfficePurchase) sourceRecord {
			return sourceRecord{ID: o.ID, UserID: sourceUserID(o.UserID), Title: "办公采购-" + o.OrderNo, CreatedAt: o.CreatedAt}
		}), nil

	case "canteen":
		var items []models.CanteenPurchase
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询食堂采购失败: %w", err)
		}
		return toSourceRecords(items, func(c models.CanteenPurchase) sourceRecord {
			return sourceRecord{ID: c.ID, UserID: sourceUserID(c.UserID), Title: "食堂采购-" + c.OrderNo, CreatedAt: c.CreatedAt}
		}), nil

	case "invoice":
		var items []models.Invoice
		if err := baseQuery(s.db).Find(&items).Error; err != nil {
			return nil, fmt.Errorf("查询发票数据失败: %w", err)
		}
		return toSourceRecords(items, func(i models.Invoice) sourceRecord {
			return sourceRecord{ID: i.ID, UserID: sourceUserID(i.UserID), Title: "发票-" + i.InvoiceNo, CreatedAt: i.CreatedAt}
		}), nil

	default:
		return nil, fmt.Errorf("未知的 source_module: %s", module)
	}
}

// toSourceRecords 泛型转换函数：将模型切片转为统一 sourceRecord 切片
func toSourceRecords[T any](items []T, extract func(T) sourceRecord) []sourceRecord {
	records := make([]sourceRecord, 0, len(items))
	for _, item := range items {
		rec := extract(item)
		text, err := json.Marshal(item)
		if err != nil {
			log.Printf("[kb_ingest] 序列化记录 %d 失败: %v", rec.ID, err)
			continue
		}
		rec.Text = string(text)
		records = append(records, rec)
	}
	return records
}

// ============================================================
// 单条记录入库
// ============================================================

// ingestRecord 处理单条源记录的完整入库链路
// sourceModule 用于区分 archives（真实文档，直接复用其 ID）与非 archives（创建/复用影子 Document）
// 顺序：先解析（失败不触碰任何表）→ 事务内创建/复用影子文档 + 清理旧分块 + 写入新分块
// 返回：向量化失败的分块数（不影响 ingested 统计）
func (s *KBIngestService) ingestRecord(ctx context.Context, userID uint, kbID uint, sourceModule string, rec sourceRecord) (int, error) {
	// 1. 写临时文件 → docreader 解析（失败时不留任何残留）
	result, err := s.parseRecord(ctx, rec)
	if err != nil {
		return 0, fmt.Errorf("文档解析失败: %w", err)
	}

	// 2-4. 事务内完成：影子文档 + 旧分块清理 + 新分块写入，任一失败整体回滚
	tx := s.db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	docID := rec.ID
	if sourceModule != "archives" {
		doc, err := s.findOrCreateShadowDocument(tx, kbID, sourceModule, rec)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("创建影子文档失败: %w", err)
		}
		docID = doc.ID
	}

	// 幂等重建：清理该文档旧分块，避免重复 ingest 堆积孤儿 chunk
	if err := tx.Where("doc_id = ?", docID).Delete(&models.DocumentChunk{}).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("清理旧分块失败: %w", err)
	}

	// 解析结果写入 DocumentChunk（DocID 始终指向真实 documents.id）
	chunkCount, err := docreader.IngestToChunks(tx, result, docID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("写入分块失败: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("[kb_ingest] 记录 %d 解析完成，写入 %d 个分块", rec.ID, chunkCount)

	// 5. 向量化已创建的分块（事务外，失败不中断，返回失败数）
	embedFailed := s.embedChunks(userID, docID)
	return embedFailed, nil
}

// findOrCreateShadowDocument 为源记录幂等创建/复用影子 Document
// 唯一键为 (source_type, source_id, source_kb_id)：同一业务源记录可同时进入多个知识库，
// 每个 KB 拥有独立的影子文档与 chunks，互不覆盖。
// 并发安全：依赖 documents 表上的联合唯一部分索引（见 ensureShadowDocumentUniqueIndex），
// 创建冲突时回查复用已存在的影子文档。
func (s *KBIngestService) findOrCreateShadowDocument(tx *gorm.DB, kbID uint, sourceModule string, rec sourceRecord) (*models.Document, error) {
	var doc models.Document
	err := tx.Where("source_type = ? AND source_id = ? AND source_kb_id = ?", sourceModule, rec.ID, kbID).First(&doc).Error
	if err == nil {
		// 复用：刷新内容与元数据（源记录可能已更新）
		if err := tx.Model(&doc).Updates(map[string]interface{}{
			"user_id":      rec.UserID,
			"file_name":    rec.Title,
			"content_text": rec.Text,
			"source_dept":  rec.Department,
			"source_kb_id": kbID,
			"updated_at":   time.Now(),
		}).Error; err != nil {
			return nil, err
		}
		return &doc, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	doc = models.Document{
		UserID:       rec.UserID,
		DocumentCode: fmt.Sprintf("INGEST-%s-%d-%d", sourceModule, kbID, rec.ID), // 含 kbID：同源记录可进入多个 KB，避免 document_code 唯一冲突
		FileName:     rec.Title,
		ContentText:  rec.Text,
		OCRStatus:    "completed",
		Status:       "active",
		SourceType:   sourceModule,
		SourceID:     rec.ID,
		SourceKBID:   &kbID,
		SourceDept:   rec.Department,
	}
	// SavePoint：Create 冲突后回滚到该点恢复事务可用状态。
	// PostgreSQL 中事务内语句失败后整个事务进入 aborted 状态，后续命令会被拒绝
	// （"current transaction is aborted"），因此必须在失败事务内回查前先 RollbackTo。
	// SQLite 同样支持 SAVEPOINT，行为一致。
	tx.SavePoint("sp_shadow_create")
	if err := tx.Create(&doc).Error; err != nil {
		// 并发下唯一索引冲突：另一事务已创建同键影子文档
		if isUniqueViolation(err) {
			if rb := tx.RollbackTo("sp_shadow_create"); rb.Error != nil {
				return nil, fmt.Errorf("回滚 SavePoint 失败: %w", rb.Error)
			}
			if err2 := tx.Where("source_type = ? AND source_id = ? AND source_kb_id = ?", sourceModule, rec.ID, kbID).First(&doc).Error; err2 == nil {
				return &doc, nil
			}
		}
		return nil, err
	}
	return &doc, nil
}

// isUniqueViolation 判断错误是否为唯一约束冲突（SQLite/PostgreSQL 通用）
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique index")
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

// embeddingDim 当前系统支持的向量维度（与迁移 vector(768) 一致）
const embeddingDim = 768

// validateVectorDim 校验向量维度与系统支持一致
// PostgreSQL 原生向量列固定维度，维度不符时拒绝写入并计入 embedding_failed/pending
func validateVectorDim(vec []float64) error {
	if len(vec) != embeddingDim {
		return fmt.Errorf("向量维度 %d 与系统支持的 %d 不符", len(vec), embeddingDim)
	}
	return nil
}

// embedChunks 对指定 docID 的所有分块生成向量
// 向量化失败不中断流程，标记 index_status="pending" 等待后续重试；
// 返回失败的分块数（维度不符/API 失败/空向量），供 IngestResult.EmbeddingFailed 统计。
func (s *KBIngestService) embedChunks(userID uint, docID uint) int {
	var chunks []models.DocumentChunk
	if err := s.db.Where("doc_id = ? AND index_status = ?", docID, models.IndexStatusProcessing).Find(&chunks).Error; err != nil {
		log.Printf("[kb_ingest] 查询分块失败: %v", err)
		return 0
	}

	failed := 0
	postgres := s.dialect.isPostgres()
	for i := range chunks {
		chunk := &chunks[i]
		embedText := chunk.EmbeddingContent()
		if embedText == "" {
			chunk.IndexStatus = models.IndexStatusFailed
			s.db.Model(chunk).Update("index_status", models.IndexStatusFailed)
			failed++
			continue
		}

		// 生成嵌入向量
		vec, err := s.embeddingSvc.GenerateEmbedding(userID, embedText)
		if err != nil {
			log.Printf("[kb_ingest] 分块 %d 向量化失败: %v", chunk.ID, err)
			chunk.IndexStatus = "pending" // 标记待重试
			s.db.Model(chunk).Update("index_status", "pending")
			failed++
			continue
		}
		if len(vec) == 0 {
			log.Printf("[kb_ingest] 分块 %d 向量为空，标记 pending", chunk.ID)
			chunk.IndexStatus = "pending"
			s.db.Model(chunk).Update("index_status", "pending")
			failed++
			continue
		}

		// 按数据库能力写入向量（PostgreSQL 原生向量列 / SQLite JSON 降级）
		if err := s.writeChunkVector(chunk.ID, vec, postgres); err != nil {
			log.Printf("[kb_ingest] 分块 %d 向量写入失败: %v", chunk.ID, err)
			chunk.IndexStatus = "pending"
			s.db.Model(chunk).Update("index_status", "pending")
			failed++
			continue
		}
	}
	return failed
}

// writeChunkVector 按数据库能力写入向量：
//   - PostgreSQL：写原生 embedding 向量列（raw SQL）+ embedding_json 副本；写入前校验维度
//   - SQLite：仅写 embedding_json 副本（无原生向量类型，降级存储，不校验维度）
//
// 不硬编码数据库 ID，始终以 chunkID 定位目标行。
func (s *KBIngestService) writeChunkVector(chunkID uint, vec []float64, postgres bool) error {
	vecJSON, _ := json.Marshal(vec)
	if postgres {
		if err := validateVectorDim(vec); err != nil {
			return err
		}
		return s.db.Exec(
			"UPDATE document_chunks SET embedding = ?::vector, embedding_json = ?, index_status = ? WHERE id = ?",
			vectorToPGString(vec), datatypes.JSON(vecJSON), models.IndexStatusReady, chunkID,
		).Error
	}
	return s.db.Model(&models.DocumentChunk{}).Where("id = ?", chunkID).Updates(map[string]interface{}{
		"embedding_json": vecJSON,
		"index_status":   models.IndexStatusReady,
	}).Error
}
