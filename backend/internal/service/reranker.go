package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// RerankerService 重排服务
// 对混合检索结果用 rerank 模型进行二次排序，提升语义相关性
type RerankerService struct {
	db       *gorm.DB
	modelCfg *models.ModelConfig // rerank 模型配置（首次加载后缓存）

	httpClient *http.Client
}

// NewRerankerService 构造函数
func NewRerankerService(db *gorm.DB) *RerankerService {
	return &RerankerService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================
// 公开方法
// ============================================================

// Rerank 对检索结果用 rerank 模型二次排序
// results: 混合检索返回的 top_k 结果
// query: 用户原始问句
// kbID: 所属知识库 ID（可选，0 表示不限制）
// 返回：重排后的结果（按 rerank_score 降序排列）
func (s *RerankerService) Rerank(ctx context.Context, results []SearchResult, query string, kbID uint) ([]SearchResult, error) {
	cfg, err := s.getRerankConfig()
	if err != nil {
		log.Printf("[reranker] 获取 rerank 模型配置失败: %v，fallback 到原始排序", err)
		return results, nil
	}
	if cfg == nil {
		log.Printf("[reranker] 未配置 rerank 模型，跳过重排")
		return results, nil
	}
	_ = kbID // 预留：未来可支持知识库级别重排模型选择
	return s.RerankWithConfig(ctx, results, query, cfg)
}

// RerankWithConfig 使用指定模型配置重排
func (s *RerankerService) RerankWithConfig(ctx context.Context, results []SearchResult, query string, config *models.ModelConfig) ([]SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}
	if config == nil {
		log.Printf("[reranker] 未提供重排模型配置，跳过重排")
		return results, nil
	}

	// 提取文档内容列表
	documents := make([]string, 0, len(results))
	for _, r := range results {
		text := r.Content
		if text == "" {
			text = r.Snippet
		}
		if text == "" {
			text = r.Title
		}
		documents = append(documents, text)
	}

	// 调用 rerank API
	scores, err := s.callRerankAPI(ctx, config, query, documents)
	if err != nil {
		log.Printf("[reranker] rerank API 调用失败: %v，fallback 到原始排序", err)
		return results, nil
	}

	// 用 rerank 分数替换 Score
	for i := range results {
		if i < len(scores) {
			results[i].Score = scores[i]
		}
	}

	// 按分数降序排列
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// ============================================================
// 内部方法
// ============================================================

// getRerankConfig 获取默认 rerank 模型配置
func (s *RerankerService) getRerankConfig() (*models.ModelConfig, error) {
	// 缓存命中直接返回
	if s.modelCfg != nil {
		return s.modelCfg, nil
	}

	var cfg models.ModelConfig
	if err := s.db.Where("config_type = ? AND is_default = ? AND enabled = ?", "rerank", true, true).
		First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	s.modelCfg = &cfg
	return s.modelCfg, nil
}

// rerankAPIRequest OpenAI 兼容的 rerank API 请求体
type rerankAPIRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// rerankAPIResponse OpenAI 兼容的 rerank API 响应体
type rerankAPIResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

// callRerankAPI 调用 rerank API 并返回每个文档的相关性分数
func (s *RerankerService) callRerankAPI(ctx context.Context, config *models.ModelConfig, query string, documents []string) ([]float64, error) {
	topN := len(documents)
	reqBody := rerankAPIRequest{
		Model:     config.ModelName,
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 拼接 rerank 端点：使用配置的 APIEndpoint 拼接 /rerank
	endpoint := config.APIEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/rerank", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("rerank API 返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apiResp rerankAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 构建 index -> score 映射，再按原始顺序还原分数
	scoreMap := make(map[int]float64, len(apiResp.Results))
	for _, r := range apiResp.Results {
		scoreMap[r.Index] = r.RelevanceScore
	}

	scores := make([]float64, len(documents))
	for i := range scores {
		if s, ok := scoreMap[i]; ok {
			scores[i] = s
		} else {
			scores[i] = 0 // 未被 rerank 覆盖的默认为 0
		}
	}

	return scores, nil
}
