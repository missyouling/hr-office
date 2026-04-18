package service

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// RetrievalService 混合检索服务
type RetrievalService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
}

// SearchResult 搜索结果
type SearchResult struct {
	DocID   uint    `json:"doc_id"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
	Title   string  `json:"title"`
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

// HybridSearch 混合检索（全文 + 向量）
// 1. 全文检索: 在 documents.content_text + document_contents.content 中 LIKE 搜索
// 2. 向量检索: 将 query 向量化，与 document_embeddings 计算余弦相似度（JSON 向量）
// 3. 融合: RRF (Reciprocal Rank Fusion) 合并两路结果
// 返回 []SearchResult（docID, score, snippet）
func (s *RetrievalService) HybridSearch(userID uint, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	// 全文检索
	fullTextResults, err := s.FullTextSearch(userID, query, limit*2)
	if err != nil {
		log.Printf("[retrieval] full text search failed: %v", err)
		fullTextResults = []SearchResult{}
	}

	// 向量检索
	vectorResults, err := s.vectorSearch(userID, query, limit*2)
	if err != nil {
		log.Printf("[retrieval] vector search failed: %v", err)
		vectorResults = []SearchResult{}
	}

	// RRF 融合
	merged := s.rrfMerge(fullTextResults, vectorResults)

	// 限制结果数量
	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged, nil
}

// FullTextSearch 纯全文检索
func (s *RetrievalService) FullTextSearch(userID uint, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []SearchResult{}, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	searchPattern := "%" + strings.TrimSpace(query) + "%"

	var results []SearchResult

	// 搜索 documents 表
	var docs []models.Document
	if err := s.db.Where("user_id = ? AND (file_name LIKE ? OR content_text LIKE ?)", userID, searchPattern, searchPattern).
		Limit(limit).
		Find(&docs).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("failed to search documents: %v", err)
	}

	for _, doc := range docs {
		snippet := doc.ContentText
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results = append(results, SearchResult{
			DocID:   doc.ID,
			Score:   1.0,
			Snippet: snippet,
			Title:   doc.FileName,
		})
	}

	// 搜索 document_contents 表
	var contents []models.DocumentContent
	if err := s.db.Where("doc_id IN (SELECT id FROM documents WHERE user_id = ?) AND content LIKE ?", userID, searchPattern).
		Limit(limit).
		Find(&contents).Error; err != nil {
		log.Printf("[retrieval] failed to search document contents: %v", err)
	} else {
		for _, content := range contents {
			snippet := content.Content
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}

			// 查找对应的文档标题
			var doc models.Document
			s.db.Where("id = ?", content.DocID).First(&doc)

			results = append(results, SearchResult{
				DocID:   content.DocID,
				Score:   0.8,
				Snippet: snippet,
				Title:   doc.FileName,
			})
		}
	}

	return results, nil
}

// GlobalSearch 跨模块全局搜索
// 搜索范围: documents + employees + dormitory rooms
// 返回 []GlobalSearchResult（module, id, title, snippet, score）
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
	if err := s.db.Where("user_id = ? AND (file_name LIKE ? OR content_text LIKE ?)", userID, searchPattern, searchPattern).
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
	if err := s.db.Where("user_id = ? AND (name LIKE ? OR id_number LIKE ? OR department LIKE ?)", userID, searchPattern, searchPattern, searchPattern).
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
	if err := s.db.Where("user_id = ? AND room_number LIKE ?", userID, searchPattern).
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

	// 按分数排序并限制结果
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// vectorSearch 向量检索
func (s *RetrievalService) vectorSearch(userID uint, query string, limit int) ([]SearchResult, error) {
	// 生成查询向量
	queryVec, err := s.embeddingService.GenerateEmbedding(userID, query)
	if err != nil {
		return []SearchResult{}, fmt.Errorf("failed to generate query embedding: %v", err)
	}

	// 获取所有文档的向量
	var embeddings []models.DocumentEmbedding
	if err := s.db.Where("doc_id IN (SELECT id FROM documents WHERE user_id = ?)", userID).
		Find(&embeddings).Error; err != nil {
		return []SearchResult{}, fmt.Errorf("failed to fetch embeddings: %v", err)
	}

	scoreMap := make(map[uint]float64)
	for _, emb := range embeddings {
		var vec []float64
		if err := json.Unmarshal(emb.Embedding, &vec); err != nil {
			continue
		}

		similarity := cosineSimilarity(queryVec, vec)
		if similarity > scoreMap[emb.DocID] {
			scoreMap[emb.DocID] = similarity
		}
	}

	// 转换为结果列表
	var results []SearchResult
	for docID, score := range scoreMap {
		if score > 0.3 {
			var doc models.Document
			if err := s.db.Where("id = ?", docID).First(&doc).Error; err == nil {
				snippet := doc.ContentText
				if len(snippet) > 200 {
					snippet = snippet[:200] + "..."
				}
				results = append(results, SearchResult{
					DocID:   docID,
					Score:   score,
					Snippet: snippet,
					Title:   doc.FileName,
				})
			}
		}
	}

	// 按分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// rrfMerge RRF (Reciprocal Rank Fusion) 融合
func (s *RetrievalService) rrfMerge(fullText, vector []SearchResult) []SearchResult {
	scoreMap := make(map[uint]float64)
	docMap := make(map[uint]SearchResult)

	// 全文搜索结果
	for i, result := range fullText {
		rank := i + 1
		score := 1.0 / (60.0 + float64(rank))
		scoreMap[result.DocID] += score * 0.5
		if _, exists := docMap[result.DocID]; !exists {
			docMap[result.DocID] = result
		}
	}

	// 向量搜索结果
	for i, result := range vector {
		rank := i + 1
		score := 1.0 / (60.0 + float64(rank))
		scoreMap[result.DocID] += score * 0.5
		if _, exists := docMap[result.DocID]; !exists {
			docMap[result.DocID] = result
		}
	}

	// 转换为结果列表
	var results []SearchResult
	for docID, score := range scoreMap {
		result := docMap[docID]
		result.Score = score
		results = append(results, result)
	}

	// 按分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
