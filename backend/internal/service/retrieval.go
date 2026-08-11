package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// RetrievalService 混合检索服务（pgvector + tsvector 全文检索）
type RetrievalService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
}

// SearchResult 搜索结果（chunk 级别粒度）
type SearchResult struct {
	ChunkID uint    `json:"chunk_id"`
	DocID   uint    `json:"doc_id"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
	Title   string  `json:"title"`
	// 新增字段（供前端引用定位）
	Content    string `json:"content,omitempty"`     // chunk 完整内容
	MatchType  string `json:"match_type"`            // keyword / vector / both
	ChunkIndex int    `json:"chunk_index,omitempty"` // chunk 在文档中的序号
}

// GlobalSearchResult 全局搜索结果
type GlobalSearchResult struct {
	Module  string  `json:"module"`
	ID      uint    `json:"id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// NewRetrievalService 构造函数
func NewRetrievalService(db *gorm.DB, embeddingService *EmbeddingService) *RetrievalService {
	return &RetrievalService{
		db:               db,
		embeddingService: embeddingService,
	}
}

// ============================================================
// 混合检索（pgvector 向量 + tsvector 全文 + RRF 融合）
// ============================================================

// HybridSearch 混合检索
// 1. 向量检索: pgvector HNSW 索引，cosine 距离
// 2. 全文检索: PostgreSQL tsvector GIN 索引
// 3. RRF 融合合并
// kbID=0 表示搜索全部可见知识库；kbID>0 时限定在指定知识库范围内
func (s *RetrievalService) HybridSearch(ctx context.Context, userID uint, query string, limit int, kbID uint) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// 确定 KB 过滤所需的 user_id
	var kbUserID uint // 0 = 不过滤
	if kbID > 0 {
		var kb models.KnowledgeBase
		if err := s.db.First(&kb, kbID).Error; err != nil {
			return nil, fmt.Errorf("知识库不存在: %v", err)
		}
		if kb.UserID != nil {
			kbUserID = *kb.UserID
		}
	}

	// 并行执行两路检索
	type pair struct {
		results []SearchResult
		err     error
	}

	chFTS := make(chan pair, 1)
	chVec := make(chan pair, 1)

	go func() {
		r, e := s.FullTextSearch(userID, query, limit*2, kbUserID)
		chFTS <- pair{r, e}
	}()
	go func() {
		r, e := s.vectorSearch(userID, query, limit*2, kbUserID)
		chVec <- pair{r, e}
	}()

	ftsRes := <-chFTS
	vecRes := <-chVec

	if ftsRes.err != nil {
		log.Printf("[retrieval] full text search failed: %v", ftsRes.err)
		ftsRes.results = []SearchResult{}
	}
	if vecRes.err != nil {
		log.Printf("[retrieval] vector search failed: %v", vecRes.err)
		vecRes.results = []SearchResult{}
	}

	// RRF 融合
	merged := s.rrfMerge(ftsRes.results, vecRes.results)
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

// ============================================================
// 全文检索（tsvector）
// ============================================================

// FullTextSearch 使用 PostgreSQL tsvector 全文检索
// 搜索范围: documents.content_tsv + document_chunks.content_tsv
// kbUserID=0 表示不按知识库所有者过滤
func (s *RetrievalService) FullTextSearch(userID uint, query string, limit int, kbUserID uint) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// 使用 plainto_tsquery('simple', ...) 做纯分词匹配
	// 'simple' 配置不做词典归一化（中文友好：按字符边界分词）

	// 确定文档过滤的 user_id：kbUserID>0 时用知识库所有者，否则用请求者
	docUserID := userID
	if kbUserID > 0 {
		docUserID = kbUserID
	}
	sql := `
		SELECT 'doc' AS source, d.id AS doc_id, 0 AS chunk_id,
		       ts_rank(d.content_tsv, plainto_tsquery('simple', ?)) AS score,
		       d.file_name AS title,
		       LEFT(COALESCE(d.content_text, ''), 200) AS snippet
		FROM documents d
		WHERE d.user_id = ? AND d.content_tsv @@ plainto_tsquery('simple', ?)

		UNION ALL

		SELECT 'chunk' AS source, dc.doc_id, dc.id AS chunk_id,
		       ts_rank(dc.content_tsv, plainto_tsquery('simple', ?)) AS score,
		       '' AS title,
		       LEFT(dc.content, 200) AS snippet
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id AND d.user_id = ?
		WHERE dc.content_tsv @@ plainto_tsquery('simple', ?)

		ORDER BY score DESC
		LIMIT ?
	`

	var rows []struct {
		Source  string
		DocID   uint
		ChunkID uint
		Score   float64
		Title   string
		Snippet string
	}
	if err := s.db.Raw(sql, query, docUserID, query, query, docUserID, query, limit).Scan(&rows).Error; err != nil {
		// 降级：没有 tsvector 索引或匹配不到时用 ILIKE
		log.Printf("[retrieval] tsvector search failed, falling back to ILIKE: %v", err)
		return s.fallbackFullTextSearch(docUserID, query, limit)
	}

	results := make([]SearchResult, 0, len(rows))
	seenDocs := make(map[uint]bool)

	for _, row := range rows {
		snippet := row.Snippet
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		// 合并同一文档
		if row.ChunkID != 0 {
			if seenDocs[row.DocID] {
				continue
			}
			seenDocs[row.DocID] = true
		}

		results = append(results, SearchResult{
			ChunkID:   row.ChunkID,
			DocID:     row.DocID,
			Score:     row.Score,
			Snippet:   snippet,
			Title:     row.Title,
			MatchType: "keyword",
		})
	}

	return results, nil
}

// fallbackFullTextSearch ILIKE 降级方案
func (s *RetrievalService) fallbackFullTextSearch(userID uint, query string, limit int) ([]SearchResult, error) {
	searchPattern := "%" + strings.TrimSpace(query) + "%"

	var docs []models.Document
	if err := s.db.Where("user_id = ? AND (file_name ILIKE ? OR content_text ILIKE ?)", userID, searchPattern, searchPattern).
		Limit(limit).
		Find(&docs).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("fallback search: %v", err)
	}

	results := make([]SearchResult, 0, len(docs))
	for _, doc := range docs {
		snippet := doc.ContentText
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		results = append(results, SearchResult{
			DocID:     doc.ID,
			Score:     0.5,
			Snippet:   snippet,
			Title:     doc.FileName,
			MatchType: "keyword",
		})
	}
	return results, nil
}

// ============================================================
// 全局搜索（保持现有逻辑，ILKE）
// ============================================================

// GlobalSearch 跨模块全局搜索
func (s *RetrievalService) GlobalSearch(userID uint, query string, limit int) ([]GlobalSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []GlobalSearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}

	searchPattern := "%" + strings.TrimSpace(query) + "%"
	var results []GlobalSearchResult

	// 搜索档案文档
	var docs []models.Document
	if err := s.db.Where("user_id = ? AND (file_name ILIKE ? OR content_text ILIKE ?)", userID, searchPattern, searchPattern).
		Limit(limit).
		Find(&docs).Error; err == nil {
		for _, doc := range docs {
			snippet := doc.ContentText
			if len(snippet) > 150 {
				snippet = snippet[:150] + "..."
			}
			results = append(results, GlobalSearchResult{
				Module:  "archives",
				ID:      doc.ID,
				Title:   doc.FileName,
				Snippet: snippet,
				Score:   1.0,
			})
		}
	}

	// 搜索员工
	var employees []models.Employee
	if err := s.db.Where("user_id = ? AND (name ILIKE ? OR id_number ILIKE ? OR department ILIKE ?)", userID, searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&employees).Error; err == nil {
		for _, emp := range employees {
			snippet := fmt.Sprintf("部门: %s, 身份证: %s", emp.Department, emp.IDNumber)
			results = append(results, GlobalSearchResult{
				Module:  "employee",
				ID:      emp.ID,
				Title:   emp.Name,
				Snippet: snippet,
				Score:   0.9,
			})
		}
	}

	// 搜索宿舍房间
	var rooms []models.DormRoom
	if err := s.db.Where("user_id = ? AND room_number ILIKE ?", userID, searchPattern).
		Limit(limit).
		Find(&rooms).Error; err == nil {
		for _, room := range rooms {
			snippet := fmt.Sprintf("房间号: %s, 房间类型: %s", room.RoomNumber, room.RoomType)
			results = append(results, GlobalSearchResult{
				Module:  "dormitory",
				ID:      room.ID,
				Title:   room.RoomNumber,
				Snippet: snippet,
				Score:   0.8,
			})
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// ============================================================
// 向量检索（pgvector HNSW）
// ============================================================

// vectorSearch 使用 pgvector 做近似最近邻搜索
// 对比现有实现：不再全表加载到内存计算余弦相似度，
// 而是利用 pgvector HNSW 索引做 O(log n) 检索
// kbUserID=0 表示不按知识库所有者过滤
func (s *RetrievalService) vectorSearch(userID uint, query string, limit int, kbUserID uint) ([]SearchResult, error) {
	// 生成查询向量
	queryVec, err := s.embeddingService.GenerateEmbedding(userID, query)
	if err != nil {
		return []SearchResult{}, fmt.Errorf("failed to generate query embedding: %v", err)
	}
	if len(queryVec) == 0 {
		return []SearchResult{}, nil // 无配置时返回空
	}

	// 转换为 pgvector 格式: '[0.1,0.2,...]'
	vecStr := vectorToPGString(queryVec)

	// 确定文档过滤的 user_id：kbUserID>0 时用知识库所有者，否则用请求者
	docUserID := userID
	if kbUserID > 0 {
		docUserID = kbUserID
	}

	// pgvector HNSW ANN 检索（cosine 距离 <=>）
	sql := `
		SELECT dc.id AS chunk_id, dc.doc_id, dc.chunk_index,
		       1 - (dc.embedding <=> ?::vector) AS similarity, -- <=> = cosine distance; 1 - distance = similarity
		       dc.content AS content,
		       d.file_name AS title
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id AND d.user_id = ?
		WHERE dc.index_status = 'ready'
		  AND dc.chunk_type != 'parent_text'
		  AND dc.embedding IS NOT NULL
		ORDER BY dc.embedding <=> ?::vector
		LIMIT ?
	`

	type row struct {
		ChunkID    uint
		DocID      uint
		ChunkIndex int
		Similarity float64
		Content    string
		Title      string
	}

	var rows []row
	if err := s.db.Raw(sql, vecStr, docUserID, vecStr, limit).Scan(&rows).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("vector search: %v", err)
	}

	results := make([]SearchResult, 0, len(rows))
	seenDocs := make(map[uint]bool)

	for _, r := range rows {
		// 每个文档只返回最匹配的 chunk
		if seenDocs[r.DocID] {
			continue
		}
		seenDocs[r.DocID] = true

		snippet := r.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results = append(results, SearchResult{
			ChunkID:    r.ChunkID,
			DocID:      r.DocID,
			Score:      r.Similarity,
			Snippet:    snippet,
			Title:      r.Title,
			Content:    r.Content,
			MatchType:  "vector",
			ChunkIndex: r.ChunkIndex,
		})
	}

	return results, nil
}

// ============================================================
// 粗粒度搜索（按文档级聚合 chunk 结果）
// ============================================================

// SearchChunks 按 chunk 粒度搜索（不做文档级聚合），用于 chunk 级引用定位
func (s *RetrievalService) SearchChunks(userID uint, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}

	// 并行向量检索
	queryVec, err := s.embeddingService.GenerateEmbedding(userID, query)
	if err != nil {
		return []SearchResult{}, fmt.Errorf("failed to generate query embedding: %v", err)
	}
	if len(queryVec) == 0 {
		return []SearchResult{}, nil
	}

	vecStr := vectorToPGString(queryVec)

	sql := `
		SELECT dc.id AS chunk_id, dc.doc_id, dc.chunk_index,
		       1 - (dc.embedding <=> ?::vector) AS similarity,
		       LEFT(dc.content, 300) AS snippet,
		       dc.content AS content,
		       dc.start_at, dc.end_at,
		       d.file_name AS title
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id AND d.user_id = ?
		WHERE dc.index_status = 'ready'
		  AND dc.embedding IS NOT NULL
		ORDER BY dc.embedding <=> ?::vector
		LIMIT ?
	`

	type row struct {
		ChunkID    uint
		DocID      uint
		ChunkIndex int
		Similarity float64
		Snippet    string
		Content    string
		StartAt    int
		EndAt      int
		Title      string
	}

	var rows []row
	if err := s.db.Raw(sql, vecStr, userID, vecStr, limit).Scan(&rows).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("chunk search: %v", err)
	}

	results := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		snippet := r.Snippet
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		results = append(results, SearchResult{
			ChunkID:    r.ChunkID,
			DocID:      r.DocID,
			Score:      r.Similarity,
			Snippet:    snippet,
			Content:    r.Content,
			Title:      r.Title,
			MatchType:  "vector",
			ChunkIndex: r.ChunkIndex,
		})
	}

	return results, nil
}

// ============================================================
// RRF 融合
// ============================================================

// rrfMerge RRF (Reciprocal Rank Fusion) 合并两路检索结果
func (s *RetrievalService) rrfMerge(fullText, vector []SearchResult) []SearchResult {
	scoreMap := make(map[uint]float64)    // docID -> RRF score
	docMap := make(map[uint]SearchResult) // docID -> result
	typeMap := make(map[uint]string)      // docID -> match_type

	// 全文搜索结果（权重 0.5）
	for i, result := range fullText {
		rank := i + 1
		rrfScore := 1.0 / (60.0 + float64(rank))
		scoreMap[result.DocID] += rrfScore * 0.5
		if _, ok := docMap[result.DocID]; !ok {
			docMap[result.DocID] = result
			typeMap[result.DocID] = "keyword"
		}
	}

	// 向量搜索结果（权重 0.5）
	for i, result := range vector {
		rank := i + 1
		rrfScore := 1.0 / (60.0 + float64(rank))
		scoreMap[result.DocID] += rrfScore * 0.5
		if _, ok := docMap[result.DocID]; !ok {
			docMap[result.DocID] = result
			typeMap[result.DocID] = "vector"
		} else if typeMap[result.DocID] == "keyword" {
			typeMap[result.DocID] = "both"
		}
	}

	// 转换为结果列表
	var results []SearchResult
	for docID, score := range scoreMap {
		result := docMap[docID]
		result.Score = score
		result.MatchType = typeMap[docID]
		results = append(results, result)
	}

	// 按 RRF 分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// ============================================================
// 工具函数
// ============================================================

// VectorFromJSON 从 JSON 字节提取 []float64（用于迁移）
func VectorFromJSON(data []byte) ([]float64, error) {
	var vec []float64
	if err := json.Unmarshal(data, &vec); err != nil {
		return nil, err
	}
	return vec, nil
}

// ApplyMaskToResults 对检索结果应用知识库字段脱敏（P9.2）
// - admin 角色豁免（由 ApplyFieldMask 内部的 ExemptRole 机制处理）
// - 对 Content、Snippet、Title 字段按知识库脱敏规则处理
func (s *RetrievalService) ApplyMaskToResults(db *gorm.DB, user *models.User, kbID uint, results []SearchResult) []SearchResult {
	for i := range results {
		results[i].Content = ApplyFieldMask(db, user, kbID, "content", results[i].Content)
		results[i].Snippet = ApplyFieldMask(db, user, kbID, "snippet", results[i].Snippet)
		results[i].Title = ApplyFieldMask(db, user, kbID, "title", results[i].Title)
	}
	return results
}
