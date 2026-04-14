package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

// GenerateEmbedding 调用向量模型 API 生成 embedding
// 从 model_configs 表读取 embedding 类型配置
// 请求体: { "input": text, "model": modelName }
// 返回 []float64
func (s *EmbeddingService) GenerateEmbedding(userID uint, text string) ([]float64, error) {
	if strings.TrimSpace(text) == "" {
		return []float64{}, fmt.Errorf("text cannot be empty")
	}

	// 获取 embedding 配置
	config, err := s.GetEmbeddingConfig(userID)
	if err != nil {
		log.Printf("[embedding] failed to get config: %v", err)
		return []float64{}, fmt.Errorf("failed to get embedding config: %v", err)
	}

	if config == nil {
		// 返回占位向量（768 维零向量）
		log.Printf("[embedding] no embedding config found for user %d, returning placeholder", userID)
		return generatePlaceholderEmbedding(768), nil
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"input": text,
		"model": config.ModelName,
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return []float64{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 调用 API
	req, err := http.NewRequest("POST", config.APIEndpoint, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return []float64{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[embedding] API call failed: %v", err)
		return generatePlaceholderEmbedding(768), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[embedding] API returned %d: %s", resp.StatusCode, string(body))
		return generatePlaceholderEmbedding(768), nil
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[embedding] failed to decode response: %v", err)
		return generatePlaceholderEmbedding(768), nil
	}

	// 提取向量（假设响应格式为 { "data": [{ "embedding": [...] }] }）
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if item, ok := data[0].(map[string]interface{}); ok {
			if embedding, ok := item["embedding"].([]interface{}); ok {
				vec := make([]float64, len(embedding))
				for i, v := range embedding {
					if f, ok := v.(float64); ok {
						vec[i] = f
					}
				}
				return vec, nil
			}
		}
	}

	// 占位响应
	return generatePlaceholderEmbedding(768), nil
}

// IngestDocument 将文档内容分块并生成向量存入 document_embeddings 表
// 分块策略: 按段落分割，每块最大 512 字符，重叠 50 字符
// 同时存储 document_contents 记录
func (s *EmbeddingService) IngestDocument(userID uint, docID uint, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// 验证文档存在且属于该用户
	var doc models.Document
	if err := s.db.Where("id = ? AND user_id = ?", docID, userID).First(&doc).Error; err != nil {
		return fmt.Errorf("document not found or unauthorized: %v", err)
	}

	// 保存原始内容到 document_contents
	docContent := &models.DocumentContent{
		DocID:       docID,
		Content:     content,
		OCRProvider: "manual",
		OCRModel:    "ingestion",
		OCRVersion:  "1.0",
	}

	if err := s.db.Create(docContent).Error; err != nil {
		return fmt.Errorf("failed to save document content: %v", err)
	}

	// 分块处理
	chunks := s.chunkContent(content, 512, 50)

	// 为每个分块生成向量
	embeddings := make([]models.DocumentEmbedding, 0, len(chunks))
	for idx, chunk := range chunks {
		vec, err := s.GenerateEmbedding(userID, chunk)
		if err != nil {
			log.Printf("[embedding] failed to generate embedding for chunk %d: %v", idx, err)
			vec = generatePlaceholderEmbedding(768)
		}

		// 转换为 JSON
		vecJSON, _ := json.Marshal(vec)

		embedding := models.DocumentEmbedding{
			DocID:        docID,
			ChunkIndex:   idx,
			ChunkContent: chunk,
			Embedding:    datatypes.JSON(vecJSON),
			ModelName:    "embedding-model",
			ModelVersion: "1.0",
		}
		embeddings = append(embeddings, embedding)
	}

	// 批量保存向量
	if len(embeddings) > 0 {
		if err := s.db.CreateInBatches(embeddings, 100).Error; err != nil {
			return fmt.Errorf("failed to save embeddings: %v", err)
		}
	}

	return nil
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

// chunkContent 将内容分块
// maxChunkSize: 最大块大小（字符数）
// overlap: 重叠字符数
func (s *EmbeddingService) chunkContent(content string, maxChunkSize, overlap int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 512
	}
	if overlap < 0 {
		overlap = 0
	}

	// 按段落分割
	paragraphs := strings.Split(content, "\n")

	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 如果当前块加上新段落超过限制，保存当前块
		if currentChunk.Len() > 0 && currentChunk.Len()+len(para)+1 > maxChunkSize {
			chunks = append(chunks, currentChunk.String())

			// 处理重叠：保留最后 overlap 个字符
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

	// 保存最后一个块
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	// 如果没有分块，返回整个内容
	if len(chunks) == 0 {
		chunks = append(chunks, content)
	}

	return chunks
}

// generatePlaceholderEmbedding 生成占位向量
func generatePlaceholderEmbedding(dim int) []float64 {
	vec := make([]float64, dim)
	for i := 0; i < dim; i++ {
		vec[i] = 0.0
	}
	return vec
}
