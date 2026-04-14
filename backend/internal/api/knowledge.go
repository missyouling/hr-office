package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/auth"
)

// ingestDocumentRequest 文档入知识库请求
type ingestDocumentRequest struct {
	DocumentID uint `json:"document_id"`
}

// searchKnowledgeRequest 知识库搜索请求
type searchKnowledgeRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// chatKnowledgeRequest 知识库问答请求
type chatKnowledgeRequest struct {
	Question  string `json:"question"`
	SessionID string `json:"session_id"`
}

// knowledgeStatsResponse 知识库统计响应
type knowledgeStatsResponse struct {
	TotalDocuments   int64 `json:"total_documents"`
	TotalEmbeddings  int64 `json:"total_embeddings"`
	TotalChatMessages int64 `json:"total_chat_messages"`
}

// ingestDocument 文档入知识库
// POST /api/knowledge/ingest
// Body: { "document_id": uint }
func (h *Handler) ingestDocument(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req ingestDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if req.DocumentID == 0 {
		respondError(w, http.StatusBadRequest, "document_id is required", nil)
		return
	}

	// 验证文档存在且属于该用户
	var doc struct {
		ID          uint
		Title       string
		ContentText string
	}
	if err := h.db.Table("documents").
		Where("id = ? AND user_id = ?", req.DocumentID, userID).
		First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "document not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load document", err)
		return
	}

	// 调用 embedding 服务
	if err := h.embeddingService.IngestDocument(userID, req.DocumentID, doc.ContentText); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to ingest document", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message":     "document ingested successfully",
		"document_id": req.DocumentID,
		"title":       doc.Title,
	})
}

// searchKnowledge 混合检索
// GET /api/knowledge/search?q=xxx&limit=10
func (h *Handler) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'q' is required", nil)
		return
	}

	limitStr := strings.TrimSpace(r.URL.Query().Get("limit"))
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// 执行混合搜索
	results, err := h.retrievalService.HybridSearch(userID, query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "search failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// chatKnowledge 问答
// POST /api/knowledge/chat
// Body: { "question": string, "session_id": string(optional) }
func (h *Handler) chatKnowledge(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var req chatKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if strings.TrimSpace(req.Question) == "" {
		respondError(w, http.StatusBadRequest, "question is required", nil)
		return
	}

	// 执行问答
	response, err := h.chatService.Chat(userID, req.SessionID, req.Question)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "chat failed", err)
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// knowledgeStats 知识库统计
// GET /api/knowledge/stats
func (h *Handler) knowledgeStats(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var stats knowledgeStatsResponse

	// 统计文档数
	h.db.Table("documents").
		Where("user_id = ?", userID).
		Count(&stats.TotalDocuments)

	// 统计向量数
	h.db.Table("document_embeddings").
		Where("doc_id IN (SELECT id FROM documents WHERE user_id = ?)", userID).
		Count(&stats.TotalEmbeddings)

	// 统计聊天消息数
	h.db.Table("chat_messages").
		Where("user_id = ?", userID).
		Count(&stats.TotalChatMessages)

	respondJSON(w, http.StatusOK, stats)
}

// globalSearch 全局搜索
// GET /api/search/global?q=xxx&limit=20
func (h *Handler) globalSearch(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'q' is required", nil)
		return
	}

	limitStr := strings.TrimSpace(r.URL.Query().Get("limit"))
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// 执行全局搜索
	results, err := h.retrievalService.GlobalSearch(userID, query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "search failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}
