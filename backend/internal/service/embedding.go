package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// EmbeddingService 向量化服务
type EmbeddingService struct {
	db         *gorm.DB
	httpClient *http.Client
}

// NewEmbeddingService 构造函数
func NewEmbeddingService(db *gorm.DB) *EmbeddingService {
	return &EmbeddingService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================
// 向量生成（保持原有 API 调用逻辑）
// ============================================================

// GenerateEmbedding 调用向量模型 API 生成 embedding
func (s *EmbeddingService) GenerateEmbedding(userID uint, text string) ([]float64, error) {
	startTime := time.Now()

	if strings.TrimSpace(text) == "" {
		return []float64{}, fmt.Errorf("text cannot be empty")
	}

	config, err := s.GetEmbeddingConfig(userID)
	if err != nil {
		log.Printf("[embedding] failed to get config: %v", err)
		return []float64{}, fmt.Errorf("failed to get embedding config: %v", err)
	}

	if config == nil {
		log.Printf("[embedding] no embedding config found for user %d", userID)
		return []float64{}, nil
	}

	reqBody := map[string]interface{}{
		"input": text,
		"model": config.ModelName,
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return []float64{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", config.APIEndpoint, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return []float64{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), err.Error())
		log.Printf("[embedding] API call failed: %v", err)
		return []float64{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
		log.Printf("[embedding] API returned %d: %s", resp.StatusCode, string(body))
		return []float64{}, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		go s.recordUsage(userID, config, "failed", 0, 0, int(elapsed.Milliseconds()), err.Error())
		log.Printf("[embedding] failed to decode response: %v", err)
		return []float64{}, nil
	}

	inputTokens := 0
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(pt)
		}
	}

	vec := []float64{}
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if item, ok := data[0].(map[string]interface{}); ok {
			if embedding, ok := item["embedding"].([]interface{}); ok {
				vec = make([]float64, len(embedding))
				for i, v := range embedding {
					if f, ok := v.(float64); ok {
						vec[i] = f
					}
				}
				go s.recordUsage(userID, config, "success", inputTokens, 0, int(elapsed.Milliseconds()), "")
				return vec, nil
			}
		}
	}

	go s.recordUsage(userID, config, "failed", inputTokens, 0, int(elapsed.Milliseconds()), "no embedding in response")
	return []float64{}, nil
}

// recordUsage 异步记录模型用量
func (s *EmbeddingService) recordUsage(userID uint, config *models.ModelConfig, status string, inputTokens, outputTokens, durationMs int, errMsg string) {
	usageLog := &models.ModelUsageLog{
		UserID:      userID,
		ConfigID:    config.ID,
		ModelName:   config.ModelName,
		Provider:    config.Provider,
		ConfigType:  "embedding",
		InputTokens: inputTokens,
		OutputTokens: outputTokens,
		TotalTokens: inputTokens + outputTokens,
		Status:      status,
		ErrorMsg:    errMsg,
		DurationMs:  durationMs,
	}
	usageLog.CostUSD = usageLog.CalculateCost()
	if err := s.db.Create(usageLog).Error; err != nil {
		log.Printf("[embedding] failed to record usage: %v", err)
	}
}

// GetEmbeddingConfig 获取用户的 embedding 模型配置
func (s *EmbeddingService) GetEmbeddingConfig(userID uint) (*models.ModelConfig, error) {
	var config models.ModelConfig
	if err := s.db.Where("user_id = ? AND config_type = ? AND enabled = ?", userID, "embedding", true).
		Order("is_default DESC, created_at DESC").
		First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// ============================================================
// 文档摄取（新分块策略 → document_chunks + pgvector）
// ============================================================

// ChunkConfig 分块配置
type ChunkConfig struct {
	MaxChunkSize      int  // 最大块字符数（默认 800）
	Overlap           int  // 重叠字符数（默认 100）
	MaxChunkSizeBytes int  // 最大块字节数（默认 4096）
	EnableParentChild bool // 启用父子分块（子块供检索，父块供上下文）
	ParentChunkSize   int  // 父块字符数（默认 3200）
	ChildChunkSize    int  // 子块字符数（默认 800）
}

// DefaultChunkConfig 返回默认分块配置
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize:      800,
		Overlap:           100,
		MaxChunkSizeBytes: 4096,
		EnableParentChild: false,
		ParentChunkSize:   3200,
		ChildChunkSize:    800,
	}
}

// IngestDocument 将文档内容分块并生成向量存入 document_chunks 表
// 改进点：
//  1. 检测 Markdown/HTML 标题作为 ContextHeader
//  2. 记录 StartAt/EndAt 原文坐标
//  3. 生成 ContentHash
//  4. 通过 raw SQL 写入 pgvector，同时存储 JSON 副本
func (s *EmbeddingService) IngestDocument(userID uint, docID uint, content string) error {
	return s.IngestDocumentWithConfig(userID, docID, content, DefaultChunkConfig())
}

// IngestDocumentWithConfig 使用自定义分块配置摄入文档
func (s *EmbeddingService) IngestDocumentWithConfig(userID uint, docID uint, content string, cfg ChunkConfig) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// 验证文档存在且属于该用户
	var doc models.Document
	if err := s.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		return fmt.Errorf("document not found or unauthorized: %v", err)
	}

	// 保存原始内容到 document_contents（保留兼容）
	docContent := &models.DocumentContent{
		DocID:       docID,
		Content:     content,
		OCRProvider: "ingestion",
		OCRModel:    "chunking",
		OCRVersion:  "2.0",
	}
	if err := s.db.Create(docContent).Error; err != nil {
		log.Printf("[embedding] failed to save document content: %v", err)
	}

	// 分块处理（带 ContextHeader 和坐标）
	chunks := s.splitContentWithContext(content, cfg)

	// 清除旧数据
	if err := s.db.Where("doc_id = ?", docID).Delete(&models.DocumentChunk{}).Error; err != nil {
		return fmt.Errorf("failed to delete old chunks: %v", err)
	}

	// 为每个块生成向量并写入
	for idx, chunk := range chunks {
		// 用 EmbeddingContent（ContextHeader + Content）做向量化
		vecText := chunk.EmbeddingContent()
		vec, err := s.GenerateEmbedding(userID, vecText)
		if err != nil {
			log.Printf("[embedding] failed to generate embedding for chunk %d: %v", idx, err)
			chunk.IndexStatus = models.IndexStatusFailed
		}
		if len(vec) == 0 {
			chunk.IndexStatus = models.IndexStatusFailed
		}

		// ContentHash
		hash := sha256.Sum256([]byte(chunk.Content))
		chunk.ContentHash = fmt.Sprintf("%x", hash)
		chunk.ModelName = "embedding-model"
		chunk.CreatedAt = time.Now()
		chunk.UpdatedAt = time.Now()

		// GORM 写入元数据（不包含 vector）
		if err := s.db.Create(chunk).Error; err != nil {
			log.Printf("[embedding] failed to create chunk %d: %v", idx, err)
			continue
		}

		// raw SQL 写入 pgvector + JSON 副本
		if len(vec) > 0 && chunk.IndexStatus == models.IndexStatusReady {
			if err := s.writeChunkVector(chunk.ID, vec); err != nil {
				log.Printf("[embedding] failed to write vector for chunk %d: %v", idx, err)
				chunk.IndexStatus = models.IndexStatusFailed
				s.db.Model(chunk).Update("index_status", models.IndexStatusFailed)
			}
		}

		// 父子分块：如果启用，创建父块的 embedding 子块
		if cfg.EnableParentChild && chunk.ChunkType == models.ChunkTypeParent {
			// 父块本身不参与向量索引（content 已存），只需要子块
			// 子块已在 splitContentWithContext 中生成，此处仅做关联
		}
	}

	// 更新文档解析状态
	s.db.Model(&doc).Updates(map[string]interface{}{
		"ocr_status": "completed",
	})

	// 尝试创建 HNSW 索引（首次摄入时）
	s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunk_embedding_hnsw
		ON document_chunks USING hnsw (embedding vector_cosine_ops)
		WITH (m = 16, ef_construction = 64)
	`)

	return nil
}

// writeChunkVector 通过 raw SQL 写入 pgvector 向量和 JSON 副本
func (s *EmbeddingService) writeChunkVector(chunkID uint, vec []float64) error {
	vecStr := vectorToPGString(vec)
	vecJSON, _ := json.Marshal(vec)

	return s.db.Exec(`
		UPDATE document_chunks
		SET embedding = $1::vector, embedding_json = $2, index_status = 'ready'
		WHERE id = $3
	`, vecStr, datatypes.JSON(vecJSON), chunkID).Error
}

// ============================================================
// 智能分块（ContextHeader 检测 + 坐标追踪）
// ============================================================

// headingPattern 匹配 Markdown 标题: #, ##, ### 等
var headingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)

// ChunkCandidate 分块中间态（带坐标和头部信息）
type ChunkCandidate struct {
	Content       string
	ContextHeader string // 当前块的标题面包屑，如 "## 第三章 > ### 3.1 概述"
	StartAt       int
	EndAt         int
}

// splitContentWithContext 智能分块：段落分割 + 标题面包屑 + 坐标
// 流程：
//  1. 扫描建立标题层级栈
//  2. 按 \n\n 段落分割
//  3. 短段落合并直到 maxChunkSize
//  4. 记录每块的 StartAt/EndAt
//  5. 为每块附上当前活动的标题面包屑作为 ContextHeader
func (s *EmbeddingService) splitContentWithContext(content string, cfg ChunkConfig) []*models.DocumentChunk {
	if cfg.MaxChunkSize <= 0 {
		cfg.MaxChunkSize = DefaultChunkConfig().MaxChunkSize
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = DefaultChunkConfig().Overlap
	}

	maxChunkSize := cfg.MaxChunkSize
	_ = cfg.Overlap // reserved for future sliding window implementation

	// 第一步：扫描标题（参考 WeKnora header_hook.py）
	headers := scanHeaders(content)

	// 第二步：按段落分割
	paragraphs := splitParagraphs(content)

	// 第三步：合并段落，记录标题上下文
	var currentHeader string
	headerStack := []string{}
	candidates := []ChunkCandidate{}
	pos := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			pos += 2 // 空段落消耗两个换行符
			continue
		}

		// 检查是否是标题行
		paraStart := pos
		paraEnd := pos + len(para)

		if h, ok := headers[paraStart]; ok {
			// 更新标题栈
			level := h.level
			// 剪掉同级及更深层的标题
			for len(headerStack) >= level {
				headerStack = headerStack[:len(headerStack)-1]
			}
			headerStack = append(headerStack, h.text)
			currentHeader = strings.Join(headerStack, " > ")
			pos = paraEnd + 1
			continue
		}

		// 合并短段落
		if len(candidates) > 0 {
			last := &candidates[len(candidates)-1]
			if len(last.Content)+len(para)+1 <= maxChunkSize {
				last.Content += "\n" + para
				last.EndAt = paraEnd
				pos = paraEnd + 1
				continue
			}
		}

		candidates = append(candidates, ChunkCandidate{
			Content:       para,
			ContextHeader: currentHeader,
			StartAt:       paraStart,
			EndAt:         paraEnd,
		})
		pos = paraEnd + 1
	}

	// 第四步：转换并建立块链
	chunks := make([]*models.DocumentChunk, 0, len(candidates))
	for i, cand := range candidates {
		content := cand.Content
		// 截断过长块
		if len(content) > maxChunkSize {
			content = content[:maxChunkSize]
		}

		chunk := &models.DocumentChunk{
			ChunkIndex:     i,
			ChunkType:      models.ChunkTypeText,
			Content:        content,
			SourceContent:  content, // 初始源内容 = 内容
			ContextHeader:  cand.ContextHeader,
			StartAt:        cand.StartAt,
			EndAt:          cand.EndAt,
			ContentRevision: 1,
			IndexStatus:     models.IndexStatusReady,
		}

		// 设置块链
		if i > 0 {
			chunks[i-1].NextChunkID = nil // 先占位，save 后修正
		}

		chunks = append(chunks, chunk)
	}

	// 父子分块（可选）
	if cfg.EnableParentChild && len(chunks) > 0 {
		chunks = s.buildParentChildChunks(chunks, content, cfg)
	}

	return chunks
}

// scanHeaders 扫描文本中的 Markdown 标题，返回 字符偏移 → 标题信息
func scanHeaders(content string) map[int]headerInfo {
	matches := headingPattern.FindAllStringSubmatchIndex(content, -1)
	headers := make(map[int]headerInfo, len(matches))

	for _, m := range matches {
		start := m[0]
		fullMatch := content[m[0]:m[1]]
		// 计算标题层级（# 的数量）
		level := 0
		for _, c := range fullMatch {
			if c == '#' {
				level++
			} else {
				break
			}
		}
		text := strings.TrimSpace(fullMatch[level:])
		headers[start] = headerInfo{level: level, text: text}
	}
	return headers
}

type headerInfo struct {
	level int
	text  string
}

// splitParagraphs 按双换行符分割段落
func splitParagraphs(text string) []string {
	return strings.Split(text, "\n\n")
}

// buildParentChildChunks 构建父子分块结构
// 父块 = 原文大段（提供上下文），子块 = 细粒度块（用于向量匹配）
func (s *EmbeddingService) buildParentChildChunks(chunks []*models.DocumentChunk, fullContent string, cfg ChunkConfig) []*models.DocumentChunk {
	parentSize := cfg.ParentChunkSize
	if parentSize <= 0 {
		parentSize = DefaultChunkConfig().ParentChunkSize
	}
	childSize := cfg.ChildChunkSize
	if childSize <= 0 {
		childSize = DefaultChunkConfig().ChildChunkSize
	}

	result := make([]*models.DocumentChunk, 0, len(chunks)*2)
	prevParentEnd := 0

	for i, chunk := range chunks {
		// 子块：缩短内容
		if len(chunk.Content) > childSize {
			chunk.Content = chunk.Content[:childSize]
		}
		result = append(result, chunk)

		// 父块：合并 N 个子块
		if (i+1)%3 == 0 || i == len(chunks)-1 {
			start := prevParentEnd
			parentContent := extractTextRange(fullContent, start, chunk.EndAt)
			if len(parentContent) > parentSize {
				parentContent = parentContent[:parentSize]
			}

			parentChunk := &models.DocumentChunk{
				ChunkIndex: -(i + 1), // 负数区分父块
				ChunkType:  models.ChunkTypeParent,
				Content:    parentContent,
				StartAt:    start,
				EndAt:      chunk.EndAt,
				ContentRevision: 1,
				IndexStatus:     models.IndexStatusReady,
			}
			result = append(result, parentChunk)
			prevParentEnd = chunk.EndAt
		}
	}

	return result
}

// extractTextRange 安全提取文本区间
func extractTextRange(text string, start, end int) string {
	runes := []rune(text)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

// ============================================================
// 旧版兼容（保留原 chunkContent 供回退使用）
// ============================================================

// chunkContent 按段落分块（旧版，保留作为 fallback）
func (s *EmbeddingService) chunkContent(content string, maxChunkSize, overlap int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 512
	}
	if overlap < 0 {
		overlap = 0
	}

	paragraphs := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if currentChunk.Len() > 0 && currentChunk.Len()+len(para)+1 > maxChunkSize {
			chunks = append(chunks, currentChunk.String())
			if overlap > 0 && currentChunk.Len() > overlap {
				lastPart := currentChunk.String()
				currentChunk.Reset()
				currentChunk.WriteString(lastPart[len(lastPart)-overlap:])
				currentChunk.WriteString("\n")
			} else {
				currentChunk.Reset()
			}
		}
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(para)
	}
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	if len(chunks) == 0 {
		chunks = append(chunks, content)
	}
	return chunks
}

// ============================================================
// Chunk 版本管理（P2 预备）
// ============================================================

// SaveChunkRevision 保存分块编辑快照
func (s *EmbeddingService) SaveChunkRevision(chunkID, editorID uint, oldContent, newContent, source string) error {
	var chunk models.DocumentChunk
	if err := s.db.First(&chunk, chunkID).Error; err != nil {
		return fmt.Errorf("chunk not found: %v", err)
	}

	rev := &models.ChunkRevision{
		ChunkID:    chunkID,
		Revision:   chunk.ContentRevision,
		Content:    oldContent,
		EditorID:   &editorID,
		EditSource: source,
	}

	if err := s.db.Create(rev).Error; err != nil {
		return fmt.Errorf("failed to save revision: %v", err)
	}

	// 更新 chunk 内容和版本号
	return s.db.Model(&chunk).Updates(map[string]interface{}{
		"content":          newContent,
		"content_revision": gorm.Expr("content_revision + 1"),
		"updated_at":       time.Now(),
	}).Error
}

// ============================================================
// 辅助函数
// ============================================================

// vectorToPGString 将 []float64 转为 pgvector 格式字符串
func vectorToPGString(vec []float64) string {
	if len(vec) == 0 {
		return "[]"
	}
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%.6f", math.Round(v*1e6)/1e6)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
