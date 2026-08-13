package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
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
	Kbid      uint   `json:"kb_id,omitempty"` // 可选：限定知识库范围
}

// knowledgeStatsResponse 知识库统计响应
type knowledgeStatsResponse struct {
	TotalDocuments    int64 `json:"total_documents"`
	TotalEmbeddings   int64 `json:"total_embeddings"`
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
// GET /api/knowledge/search?q=xxx&limit=10&kb_id=1
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

	// 解析可选 kb_id 参数（缺省/0/null 视为检索全部可见知识库；非法值返回 400），
	// 校验权限，加载用户（用于后续脱敏）
	kbID, kbIDOK := parseKbIDFromQuery(r)
	if !kbIDOK {
		respondError(w, http.StatusBadRequest, "无效的 kb_id 参数", nil)
		return
	}
	var kbUser *models.User
	if kbID > 0 {
		var uErr error
		kbUser, uErr = getKBUser(h.db, r)
		if uErr != nil || !HasAccess(h.db, kbUser, kbID) {
			respondError(w, http.StatusForbidden, "无权访问该知识库", nil)
			return
		}
	} else {
		// kbID=0 时也需要用户对象用于脱敏（按每条结果所属 KB 应用规则）
		var uErr error
		kbUser, uErr = getKBUser(h.db, r)
		if uErr != nil {
			respondError(w, http.StatusInternalServerError, "加载用户失败", uErr)
			return
		}
	}

	// 执行混合搜索（kbID=0 表示搜索全部可见知识库）
	results, err := h.retrievalService.HybridSearch(r.Context(), userID, query, limit, kbID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "search failed", err)
		return
	}

	// 对结果应用字段脱敏（kbID>0 按指定 KB；kbID=0 按每条结果所属 KB；脱敏失败安全失败）
	// 豁免值列表对检索接口无答案防御层可复用，忽略
	masked, _, _, mErr := h.retrievalService.ApplyMaskToResults(h.db, kbUser, kbID, results)
	if mErr != nil {
		log.Printf("[knowledge] 检索结果脱敏失败: %v", mErr)
		respondError(w, http.StatusInternalServerError, "检索结果脱敏失败", mErr)
		return
	}
	results = masked

	respondJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// chatKnowledge 问答
// POST /api/knowledge/chat
// Body: { "question": string, "session_id": string(optional), "kb_id": uint(optional) }
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

	// 若指定 kb_id，校验权限
	if req.Kbid > 0 {
		user, uErr := getKBUser(h.db, r)
		if uErr != nil || !HasAccess(h.db, user, req.Kbid) {
			respondError(w, http.StatusForbidden, "无权访问该知识库", nil)
			return
		}
	}

	// 执行问答（透传 kb_id 用于检索范围限定与脱敏）
	response, err := h.chatService.Chat(userID, req.SessionID, req.Question, req.Kbid)
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

	// 统计向量数（新表 document_chunks）
	h.db.Table("document_chunks").
		Where("doc_id IN (SELECT id FROM documents WHERE user_id = ?)", userID).
		Count(&stats.TotalEmbeddings)

	// 统计聊天消息数
	h.db.Table("chat_messages").
		Where("user_id = ?", userID).
		Count(&stats.TotalChatMessages)

	respondJSON(w, http.StatusOK, stats)
}

// ============================================================
// SSE 流式问答（P2）
// ============================================================

// chatKnowledgeStream SSE 流式问答
// POST /api/knowledge/chat/stream
// Body: { "question": string, "session_id": string(optional), "kb_id": uint(optional) }
func (h *Handler) chatKnowledgeStream(w http.ResponseWriter, r *http.Request) {
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

	// 若指定 kb_id，校验权限（越权严格 403）
	if req.Kbid > 0 {
		user, uErr := getKBUser(h.db, r)
		if uErr != nil || !HasAccess(h.db, user, req.Kbid) {
			respondError(w, http.StatusForbidden, "无权访问该知识库", nil)
			return
		}
	}

	// 透传 kb_id 用于检索范围限定与脱敏（kbID=0 表示搜索全部可见知识库）
	h.chatService.StreamChat(w, userID, req.SessionID, req.Question, req.Kbid)
}

// ============================================================
// 会话管理（P2）
// ============================================================

// listSessions GET /api/knowledge/sessions
func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	var sessions []models.ChatSession
	if err := h.db.Where("user_id = ?", userID).
		Order("is_pinned DESC, updated_at DESC").
		Find(&sessions).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list sessions", err)
		return
	}

	respondJSON(w, http.StatusOK, sessions)
}

// updateSession PUT /api/knowledge/sessions/{sessionID}
func (h *Handler) updateSession(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	sessionID := chi.URLParam(r, "sessionID")

	var req struct {
		Title    *string `json:"title"`
		IsPinned *bool   `json:"is_pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	var session models.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		respondError(w, http.StatusNotFound, "session not found", err)
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
		if *req.IsPinned {
			now := time.Now()
			updates["pinned_at"] = &now
		} else {
			updates["pinned_at"] = nil
		}
	}

	if len(updates) > 0 {
		if err := h.db.Model(&session).Updates(updates).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update session", err)
			return
		}
	}

	respondJSON(w, http.StatusOK, session)
}

// deleteSession DELETE /api/knowledge/sessions/{sessionID}
func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	sessionID := chi.URLParam(r, "sessionID")

	result := h.db.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&models.ChatSession{})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete session", err)
		return
	}
	if result.RowsAffected == 0 {
		respondError(w, http.StatusNotFound, "session not found", nil)
		return
	}

	// 同时删除关联的消息
	h.db.Where("session_id = ?", sessionID).Delete(&models.ChatMessage{})

	respondJSON(w, http.StatusOK, map[string]string{"message": "session deleted"})
}

// ============================================================
// Chunk 级搜索（P2）
// ============================================================

// searchChunks GET /api/knowledge/search/chunks?q=xxx&limit=20
func (h *Handler) searchChunks(w http.ResponseWriter, r *http.Request) {
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

	results, err := h.retrievalService.SearchChunks(userID, query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "chunk search failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
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

// parseKbIDFromQuery 从请求 query 参数中解析可选的 kb_id
// 返回 (id, ok)：
//   - 缺省/空/null → (0, true)，表示检索全部可见知识库
//   - 合法正整数 → (id, true)
//   - 非数字/负数 → (0, false)，调用方应返回 400
func parseKbIDFromQuery(r *http.Request) (uint, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("kb_id"))
	if raw == "" || strings.EqualFold(raw, "null") {
		return 0, true
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	if id == 0 {
		return 0, true // 显式 kb_id=0 同缺省：检索全部可见知识库
	}
	return uint(id), true
}
