package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// RetrievalService 混合检索服务（pgvector + tsvector 全文检索）
type RetrievalService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
	dialect          dbDialect // 数据库方言能力（向量检索/模糊匹配策略）
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
		dialect:          newDBDialect(db),
	}
}

// ============================================================
// 混合检索（pgvector 向量 + tsvector 全文 + RRF 融合）
// ============================================================

// HybridSearch 混合检索
// 1. 向量检索: pgvector HNSW 索引（PostgreSQL）/ 应用层余弦（SQLite 降级）
// 2. 全文检索: PostgreSQL tsvector GIN 索引 / LIKE 降级
// 3. RRF 融合合并
// kbID>0 时限定在指定知识库范围内（source_kb_id = kbID，调用方已校验权限）；
// kbID=0 时解析当前用户有权限的 KB 集合，仅在这些 KB 范围内检索（空集合不放开全量）。
func (s *RetrievalService) HybridSearch(ctx context.Context, userID uint, query string, limit int, kbID uint) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// 校验指定 KB 存在（kbID>0 时）
	if kbID > 0 {
		var kb models.KnowledgeBase
		if err := s.db.First(&kb, kbID).Error; err != nil {
			return nil, fmt.Errorf("知识库不存在: %v", err)
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
		r, e := s.FullTextSearch(userID, query, limit*2, kbID)
		chFTS <- pair{r, e}
	}()
	go func() {
		r, e := s.vectorSearch(userID, query, limit*2, kbID)
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
// KB 范围过滤
// ============================================================

// docScopeSQL 生成 documents 表的 KB 范围过滤条件（含参数）
// alias 为 documents 表别名（空表示无别名，用于单表查询）。
// 语义：
//   - kbID>0：仅命中该 KB 的影子文档（source_kb_id = kbID，调用方已校验权限）
//   - kbID=0 管理员：全量可见（返回恒真条件，不做任何过滤）
//   - kbID=0 普通用户：可访问 KB 的影子文档（source_kb_id IN 有权限 KB，
//     不限制 user_id——跨用户授权 KB 默认搜索可命中）
//     OR 当前用户自己的档案文档（source_kb_id IS NULL AND user_id = 当前用户）
//     —— 空权限集合时仅档案文档可检索，绝不放开全量
func (s *RetrievalService) docScopeSQL(alias string, userID uint, kbID uint) (string, []interface{}) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	if kbID > 0 {
		return prefix + "source_kb_id = ?", []interface{}{kbID}
	}
	// 管理员：全量可见（不做任何过滤）
	if userIsAdmin(s.db, userID) {
		return "1 = 1", nil
	}
	kbIDs := AccessibleKBIDs(s.db, userID)
	if len(kbIDs) == 0 {
		return prefix + "source_kb_id IS NULL AND " + prefix + "user_id = ?", []interface{}{userID}
	}
	return "(" + prefix + "source_kb_id IN (?) OR (" + prefix + "source_kb_id IS NULL AND " + prefix + "user_id = ?))",
		[]interface{}{kbIDs, userID}
}

// ============================================================
// 全文检索（tsvector）
// ============================================================

// FullTextSearch 使用 PostgreSQL tsvector 全文检索
// 搜索范围: documents.content_tsv + document_chunks.content_tsv
// kbID>0 限定指定知识库；kbID=0 解析用户有权限 KB 集合（见 docScopeSQL）
func (s *RetrievalService) FullTextSearch(userID uint, query string, limit int, kbID uint) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// 使用 plainto_tsquery('simple', ...) 做纯分词匹配
	// 'simple' 配置不做词典归一化（中文友好：按字符边界分词）

	// 解析 KB 范围过滤条件（含参数）
	scopeSQL, scopeArgs := s.docScopeSQL("d", userID, kbID)

	sql := `
		SELECT 'doc' AS source, d.id AS doc_id, 0 AS chunk_id,
		       ts_rank(d.content_tsv, plainto_tsquery('simple', ?)) AS score,
		       d.file_name AS title,
		       LEFT(COALESCE(d.content_text, ''), 200) AS snippet
		FROM documents d
		WHERE ` + scopeSQL + ` AND d.content_tsv @@ plainto_tsquery('simple', ?)

		UNION ALL

		SELECT 'chunk' AS source, dc.doc_id, dc.id AS chunk_id,
		       ts_rank(dc.content_tsv, plainto_tsquery('simple', ?)) AS score,
		       '' AS title,
		       LEFT(dc.content, 200) AS snippet
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id
		WHERE ` + scopeSQL + ` AND dc.content_tsv @@ plainto_tsquery('simple', ?)

		ORDER BY score DESC
		LIMIT ?
	`

	// 参数顺序：SELECT1(query, scopeArgs..., query) + SELECT2(query, scopeArgs..., query) + limit
	args := []interface{}{query}
	args = append(args, scopeArgs...)
	args = append(args, query, query)
	args = append(args, scopeArgs...)
	args = append(args, query, limit)

	var rows []struct {
		Source  string
		DocID   uint
		ChunkID uint
		Score   float64
		Title   string
		Snippet string
	}
	if err := s.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		// 降级：没有 tsvector 索引或匹配不到时用 LIKE/ILIKE
		log.Printf("[retrieval] tsvector search failed, falling back to LIKE: %v", err)
		return s.fallbackFullTextSearch(userID, query, limit, kbID)
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

// fallbackFullTextSearch LIKE/ILIKE 降级方案（SQLite 无 tsvector 时使用）
func (s *RetrievalService) fallbackFullTextSearch(userID uint, query string, limit int, kbID uint) ([]SearchResult, error) {
	searchPattern := "%" + strings.TrimSpace(query) + "%"
	scopeSQL, scopeArgs := s.docScopeSQL("", userID, kbID)
	likeName := s.dialect.likeExpr("file_name")
	likeContent := s.dialect.likeExpr("content_text")

	var docs []models.Document
	if err := s.db.Where(scopeSQL, scopeArgs...).
		Where("("+likeName+" OR "+likeContent+")", searchPattern, searchPattern).
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
	likeName := s.dialect.likeExpr("file_name")
	likeContent := s.dialect.likeExpr("content_text")
	var docs []models.Document
	if err := s.db.Where("user_id = ? AND ("+likeName+" OR "+likeContent+")", userID, searchPattern, searchPattern).
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
	likeName2 := s.dialect.likeExpr("name")
	likeID := s.dialect.likeExpr("id_number")
	likeDept := s.dialect.likeExpr("department")
	var employees []models.Employee
	if err := s.db.Where("user_id = ? AND ("+likeName2+" OR "+likeID+" OR "+likeDept+")", userID, searchPattern, searchPattern, searchPattern).
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
	likeRoom := s.dialect.likeExpr("room_number")
	var rooms []models.DormRoom
	if err := s.db.Where("user_id = ? AND "+likeRoom, userID, searchPattern).
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

// vectorSearch 向量检索
// PostgreSQL：pgvector HNSW 近似最近邻（cosine 距离）
// SQLite：embedding_json 候选在应用层计算余弦相似度、排序、截断（降级，不静默吞错）
// kbID>0 限定指定知识库；kbID=0 解析用户有权限 KB 集合（见 docScopeSQL）
func (s *RetrievalService) vectorSearch(userID uint, query string, limit int, kbID uint) ([]SearchResult, error) {
	// 生成查询向量
	queryVec, err := s.embeddingService.GenerateEmbedding(userID, query)
	if err != nil {
		return []SearchResult{}, fmt.Errorf("failed to generate query embedding: %v", err)
	}
	if len(queryVec) == 0 {
		return []SearchResult{}, nil // 无配置时返回空
	}

	if s.dialect.isPostgres() {
		return s.vectorSearchPG(userID, queryVec, limit, kbID)
	}
	return s.vectorSearchSQLite(userID, queryVec, limit, kbID)
}

// vectorSearchPG pgvector HNSW ANN 检索（cosine 距离 <=>）
func (s *RetrievalService) vectorSearchPG(userID uint, queryVec []float64, limit int, kbID uint) ([]SearchResult, error) {
	vecStr := vectorToPGString(queryVec)
	scopeSQL, scopeArgs := s.docScopeSQL("d", userID, kbID)

	sql := `
		SELECT dc.id AS chunk_id, dc.doc_id, dc.chunk_index,
		       1 - (dc.embedding <=> ?::vector) AS similarity, -- <=> = cosine distance; 1 - distance = similarity
		       dc.content AS content,
		       d.file_name AS title
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id
		WHERE ` + scopeSQL + `
		  AND dc.index_status = 'ready'
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

	args := []interface{}{vecStr}
	args = append(args, scopeArgs...)
	args = append(args, vecStr, limit)

	var rows []row
	if err := s.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
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

// vectorSearchSQLite SQLite 向量降级检索：
// 读取 embedding_json 非空的 ready 分块，应用层计算余弦相似度、排序、截断。
// 与 pgvector 路径保持一致的 KB 范围过滤与去重语义。
func (s *RetrievalService) vectorSearchSQLite(userID uint, queryVec []float64, limit int, kbID uint) ([]SearchResult, error) {
	scopeSQL, scopeArgs := s.docScopeSQL("d", userID, kbID)

	type row struct {
		ChunkID       uint
		DocID         uint
		ChunkIndex    int
		Content       string
		Title         string
		EmbeddingJSON datatypes.JSON
	}

	var rows []row
	q := s.db.Table("document_chunks dc").
		Select("dc.id AS chunk_id, dc.doc_id, dc.chunk_index, dc.content, d.file_name AS title, dc.embedding_json").
		Joins("JOIN documents d ON d.id = dc.doc_id").
		Where(scopeSQL, scopeArgs...).
		Where("dc.index_status = ?", models.IndexStatusReady).
		Where("dc.chunk_type != ?", models.ChunkTypeParent).
		Where("dc.embedding_json IS NOT NULL")
	if err := q.Scan(&rows).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("sqlite vector search: %v", err)
	}

	// 应用层余弦相似度计算
	type scored struct {
		row row
		sim float64
	}
	scoredRows := make([]scored, 0, len(rows))
	for _, r := range rows {
		vec, err := VectorFromJSON(r.EmbeddingJSON)
		if err != nil {
			continue // 损坏的 JSON 跳过
		}
		sim := cosineSimilarity(queryVec, vec)
		if sim <= 0 {
			continue // 无相似度或负相似度不参与排序
		}
		scoredRows = append(scoredRows, scored{row: r, sim: sim})
	}

	// 按相似度降序排序
	for i := 0; i < len(scoredRows)-1; i++ {
		for j := i + 1; j < len(scoredRows); j++ {
			if scoredRows[j].sim > scoredRows[i].sim {
				scoredRows[i], scoredRows[j] = scoredRows[j], scoredRows[i]
			}
		}
	}

	// 截断 + 每文档去重（与 pgvector 路径一致）
	results := make([]SearchResult, 0, len(scoredRows))
	seenDocs := make(map[uint]bool)
	for _, sr := range scoredRows {
		if seenDocs[sr.row.DocID] {
			continue
		}
		seenDocs[sr.row.DocID] = true

		snippet := sr.row.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		results = append(results, SearchResult{
			ChunkID:    sr.row.ChunkID,
			DocID:      sr.row.DocID,
			Score:      sr.sim,
			Snippet:    snippet,
			Title:      sr.row.Title,
			Content:    sr.row.Content,
			MatchType:  "vector",
			ChunkIndex: sr.row.ChunkIndex,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// cosineSimilarity 计算两个向量的余弦相似度（维度不一致或零向量返回 0）
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ============================================================
// 粗粒度搜索（按文档级聚合 chunk 结果）
// ============================================================

// SearchChunks 按 chunk 粒度搜索（不做文档级聚合），用于 chunk 级引用定位
// 检索范围按 kb_id=0 语义解析用户有权限 KB 集合（见 docScopeSQL）
func (s *RetrievalService) SearchChunks(userID uint, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}

	// 生成查询向量
	queryVec, err := s.embeddingService.GenerateEmbedding(userID, query)
	if err != nil {
		return []SearchResult{}, fmt.Errorf("failed to generate query embedding: %v", err)
	}
	if len(queryVec) == 0 {
		return []SearchResult{}, nil
	}

	if s.dialect.isPostgres() {
		return s.searchChunksPG(userID, queryVec, limit)
	}
	return s.searchChunksSQLite(userID, queryVec, limit)
}

// searchChunksPG pgvector chunk 级检索
func (s *RetrievalService) searchChunksPG(userID uint, queryVec []float64, limit int) ([]SearchResult, error) {
	vecStr := vectorToPGString(queryVec)
	scopeSQL, scopeArgs := s.docScopeSQL("d", userID, 0)

	sql := `
		SELECT dc.id AS chunk_id, dc.doc_id, dc.chunk_index,
		       1 - (dc.embedding <=> ?::vector) AS similarity,
		       LEFT(dc.content, 300) AS snippet,
		       dc.content AS content,
		       dc.start_at, dc.end_at,
		       d.file_name AS title
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.doc_id
		WHERE ` + scopeSQL + `
		  AND dc.index_status = 'ready'
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

	args := []interface{}{vecStr}
	args = append(args, scopeArgs...)
	args = append(args, vecStr, limit)

	var rows []row
	if err := s.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
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

// searchChunksSQLite SQLite 向量降级 chunk 级检索（应用层余弦）
func (s *RetrievalService) searchChunksSQLite(userID uint, queryVec []float64, limit int) ([]SearchResult, error) {
	scopeSQL, scopeArgs := s.docScopeSQL("d", userID, 0)

	type row struct {
		ChunkID       uint
		DocID         uint
		ChunkIndex    int
		Content       string
		Title         string
		EmbeddingJSON datatypes.JSON
	}

	var rows []row
	q := s.db.Table("document_chunks dc").
		Select("dc.id AS chunk_id, dc.doc_id, dc.chunk_index, dc.content, d.file_name AS title, dc.embedding_json").
		Joins("JOIN documents d ON d.id = dc.doc_id").
		Where(scopeSQL, scopeArgs...).
		Where("dc.index_status = ?", models.IndexStatusReady).
		Where("dc.embedding_json IS NOT NULL")
	if err := q.Scan(&rows).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("sqlite chunk search: %v", err)
	}

	type scored struct {
		row row
		sim float64
	}
	scoredRows := make([]scored, 0, len(rows))
	for _, r := range rows {
		vec, err := VectorFromJSON(r.EmbeddingJSON)
		if err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, vec)
		if sim <= 0 {
			continue
		}
		scoredRows = append(scoredRows, scored{row: r, sim: sim})
	}

	for i := 0; i < len(scoredRows)-1; i++ {
		for j := i + 1; j < len(scoredRows); j++ {
			if scoredRows[j].sim > scoredRows[i].sim {
				scoredRows[i], scoredRows[j] = scoredRows[j], scoredRows[i]
			}
		}
	}

	results := make([]SearchResult, 0, len(scoredRows))
	for _, sr := range scoredRows {
		snippet := sr.row.Content
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		results = append(results, SearchResult{
			ChunkID:    sr.row.ChunkID,
			DocID:      sr.row.DocID,
			Score:      sr.sim,
			Snippet:    snippet,
			Content:    sr.row.Content,
			Title:      sr.row.Title,
			MatchType:  "vector",
			ChunkIndex: sr.row.ChunkIndex,
		})
		if len(results) >= limit {
			break
		}
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
