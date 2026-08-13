package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// ======== 知识库管理路由注册 ========

// RegisterKnowledgeBaseRoutes 注册知识库所有路由（前缀 /api/knowledge-bases）
func RegisterKnowledgeBaseRoutes(r chi.Router, db *gorm.DB, kbIngestSvc *service.KBIngestService) {
	r.Route("/knowledge-bases", func(kbr chi.Router) {
		// 登录用户可访问（权限过滤在 handler 内处理）
		kbr.Get("/", listKnowledgeBases(db))
		kbr.Get("/{id}", getKnowledgeBase(db))
		kbr.Get("/{id}/rules", listKBAccessRules(db))
		kbr.Get("/{id}/masks", listKBFieldMasks(db))
		kbr.Post("/{id}/ingest", ingestKnowledgeBase(db, kbIngestSvc))

		// admin 专属操作
		kbr.Group(func(admin chi.Router) {
			admin.Use(middleware.RequireAdmin(db))
			admin.Post("/", createKnowledgeBase(db))
			admin.Put("/{id}", updateKnowledgeBase(db))
			admin.Delete("/{id}", deleteKnowledgeBase(db))
			admin.Post("/{id}/rules", addKBAccessRule(db))
			admin.Delete("/{id}/rules/{ruleId}", deleteKBAccessRule(db))
			admin.Post("/{id}/masks", addKBFieldMask(db))
			admin.Delete("/{id}/masks/{maskId}", deleteKBFieldMask(db))
			admin.Get("/stats", kbStats(db))
		})
	})
}

// ======== 权限检查 ========

// HasAccess 检查用户是否有权限访问指定知识库
func HasAccess(db *gorm.DB, user *models.User, kbID uint) bool {
	if user == nil {
		return false
	}
	var kb models.KnowledgeBase
	if err := db.First(&kb, kbID).Error; err != nil {
		return false
	}

	// admin / super_admin 统一全量放行（不限 visibility/owner/规则）
	if userIsSystemAdmin(db, user) {
		return true
	}

	// 公开 + 员工花名册模块：全员可见
	if kb.Visibility == "public" && kb.SourceModule == "employee" {
		return true
	}

	// 私有知识库：仅所有者可访问
	if kb.Visibility == "private" {
		return kb.OwnerID != nil && *kb.OwnerID == user.ID
	}

	// 受限知识库：检查访问规则（OR 组合）
	var rules []models.KBAccessRule
	db.Where("knowledge_base_id = ?", kbID).Find(&rules)
	for _, r := range rules {
		if r.RoleLevel != nil && *r.RoleLevel != "" {
			if !userHasRole(db, user, *r.RoleLevel) {
				continue
			}
		}
		if r.DepartmentID != nil {
			if user.DepartmentID == nil || *r.DepartmentID != *user.DepartmentID {
				continue
			}
		}
		if r.UserID != nil && *r.UserID != user.ID {
			continue
		}
		return true
	}
	return false
}

// userIsSystemAdmin 判断用户是否为 admin 或 super_admin（全量放行依据）
func userIsSystemAdmin(db *gorm.DB, user *models.User) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name IN (?, ?)", user.ID, models.RoleAdmin, "super_admin").
		Count(&count)
	return count > 0
}

// userHasRole 判断用户是否拥有指定角色级别（通过 user_roles 联表查询）
func userHasRole(db *gorm.DB, user *models.User, roleLevel string) bool {
	var count int64
	db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", user.ID, roleLevel).
		Count(&count)
	return count > 0
}

// getKBUser 从请求上下文提取用户并查库
func getKBUser(db *gorm.DB, r *http.Request) (*models.User, error) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// requireKBAccess 检查用户对指定知识库的访问权限，无权限返回 403
func requireKBAccess(db *gorm.DB, r *http.Request, kbID uint) (*models.User, error) {
	user, err := getKBUser(db, r)
	if err != nil {
		return nil, err
	}
	if !HasAccess(db, user, kbID) {
		return nil, gorm.ErrRecordNotFound // 用 NotFound 代表无权限，避免暴露是否存在
	}
	return user, nil
}

// ======== 知识库 CRUD ========

// listKnowledgeBases 列出用户可见的知识库（按权限过滤）
func listKnowledgeBases(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getKBUser(db, r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "未登录", err)
			return
		}

		// 查询所有知识库
		var allKBs []models.KnowledgeBase
		db.Order("id ASC").Find(&allKBs)

		// 对管理员返回全部
		if userHasRole(db, user, models.RoleAdmin) {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"items": allKBs,
				"total": len(allKBs),
			})
			return
		}

		// 过滤用户有权限的知识库
		visible := make([]models.KnowledgeBase, 0, len(allKBs))
		for _, kb := range allKBs {
			if HasAccess(db, user, kb.ID) {
				visible = append(visible, kb)
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items": visible,
			"total": len(visible),
		})
	}
}

// createKnowledgeBase 创建用户自定义知识库（IsSystem=false）
func createKnowledgeBase(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getKBUser(db, r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "未登录", err)
			return
		}

		var payload models.KnowledgeBase
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误", err)
			return
		}

		// 校验必填字段
		if payload.Name == "" {
			respondError(w, http.StatusBadRequest, "知识库名称不能为空", nil)
			return
		}

		// 强制设置为用户自定义知识库
		payload.ID = 0
		payload.IsSystem = false
		payload.OwnerID = &user.ID
		if payload.SourceModule == "" {
			payload.SourceModule = "custom"
		}
		if payload.Visibility == "" {
			payload.Visibility = "restricted"
		}

		if err := db.Create(&payload).Error; err != nil {
			respondError(w, http.StatusConflict, "创建知识库失败，名称可能重复", err)
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{"item": payload})
	}
}

// getKnowledgeBase 获取知识库详情（含规则与脱敏列表）
func getKnowledgeBase(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		_, err = requireKBAccess(db, r, kbID)
		if err != nil {
			respondError(w, http.StatusForbidden, "无权访问该知识库", err)
			return
		}

		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}

		// 加载脱敏规则
		var masks []models.KBFieldMask
		db.Where("knowledge_base_id = ?", kbID).Find(&masks)

		// 加载访问规则
		var rules []models.KBAccessRule
		db.Where("knowledge_base_id = ?", kbID).Find(&rules)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"item":  kb,
			"rules": rules,
			"masks": masks,
		})
	}
}

// updateKnowledgeBase 编辑知识库（系统模板只允许改名字/描述/脱敏/规则，不可改 IsSystem）
func updateKnowledgeBase(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}

		var payload models.KnowledgeBase
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误", err)
			return
		}

		// 构建更新字段（仅允许修改特定字段）
		updates := map[string]interface{}{
			"name":        payload.Name,
			"description": payload.Description,
		}

		// 非系统模板允许修改 SourceModule、Visibility
		if !kb.IsSystem {
			if payload.SourceModule != "" {
				updates["source_module"] = payload.SourceModule
			}
			if payload.Visibility != "" {
				updates["visibility"] = payload.Visibility
			}
			if payload.OwnerID != nil {
				updates["owner_id"] = *payload.OwnerID
			}
		}

		// 允许更新分块配置预留字段
		if payload.ChunkingConfig != nil {
			updates["chunking_config"] = payload.ChunkingConfig
		}
		if payload.EmbeddingModelID != nil {
			updates["embedding_model_id"] = *payload.EmbeddingModelID
		}

		if err := db.Model(&kb).Updates(updates).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "更新知识库失败", err)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{"item": kb})
	}
}

// deleteKnowledgeBase 删除知识库（IsSystem=true 返回 403）
func deleteKnowledgeBase(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}

		// 系统模板不可删除
		if kb.IsSystem {
			respondError(w, http.StatusForbidden, "系统模板不可删除", nil)
			return
		}

		if err := db.Delete(&kb).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "删除知识库失败", err)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
	}
}

// ======== 访问规则管理 ========

// listKBAccessRules 列出知识库的访问规则
func listKBAccessRules(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		_, err = requireKBAccess(db, r, kbID)
		if err != nil {
			respondError(w, http.StatusForbidden, "无权访问该知识库", err)
			return
		}

		var rules []models.KBAccessRule
		db.Where("knowledge_base_id = ?", kbID).Find(&rules)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items": rules,
			"total": len(rules),
		})
	}
}

// addKBAccessRule 添加访问规则
func addKBAccessRule(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		// 确认知识库存在
		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}

		var payload struct {
			RoleLevel    *string `json:"role_level"`
			DepartmentID *uint   `json:"department_id"`
			UserID       *uint   `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误", err)
			return
		}

		// 至少指定一种规则维度
		if (payload.RoleLevel == nil || *payload.RoleLevel == "") &&
			payload.DepartmentID == nil && payload.UserID == nil {
			respondError(w, http.StatusBadRequest, "至少需要指定 role_level、department_id 或 user_id 之一", nil)
			return
		}

		rule := models.KBAccessRule{
			KnowledgeBaseID: kbID,
			RoleLevel:       payload.RoleLevel,
			DepartmentID:    payload.DepartmentID,
			UserID:          payload.UserID,
		}

		if err := db.Create(&rule).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "添加访问规则失败", err)
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{"item": rule})
	}
}

// deleteKBAccessRule 删除访问规则
func deleteKBAccessRule(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		ruleID, err := parseRuleID(r, "ruleId")
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的规则ID", err)
			return
		}

		var rule models.KBAccessRule
		if err := db.Where("id = ? AND knowledge_base_id = ?", ruleID, kbID).First(&rule).Error; err != nil {
			respondError(w, http.StatusNotFound, "访问规则不存在", err)
			return
		}

		if err := db.Delete(&rule).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "删除访问规则失败", err)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
	}
}

// ======== 字段脱敏规则管理 ========

// listKBFieldMasks 列出知识库的字段脱敏规则
func listKBFieldMasks(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		_, err = requireKBAccess(db, r, kbID)
		if err != nil {
			respondError(w, http.StatusForbidden, "无权访问该知识库", err)
			return
		}

		var masks []models.KBFieldMask
		db.Where("knowledge_base_id = ?", kbID).Find(&masks)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items": masks,
			"total": len(masks),
		})
	}
}

// addKBFieldMask 添加脱敏规则
func addKBFieldMask(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		// 确认知识库存在
		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}

		var payload struct {
			FieldName   string  `json:"field_name"`
			MaskPattern string  `json:"mask_pattern"`
			ExemptRole  *string `json:"exempt_role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误", err)
			return
		}

		// 校验必填字段
		if payload.FieldName == "" {
			respondError(w, http.StatusBadRequest, "字段名不能为空", nil)
			return
		}

		// 默认脱敏模式
		if payload.MaskPattern == "" {
			payload.MaskPattern = "front3back4"
		}
		if payload.MaskPattern != "front3back4" && payload.MaskPattern != "all_star" {
			respondError(w, http.StatusBadRequest, "脱敏模式仅支持 front3back4 或 all_star", nil)
			return
		}

		mask := models.KBFieldMask{
			KnowledgeBaseID: kbID,
			FieldName:       payload.FieldName,
			MaskPattern:     payload.MaskPattern,
			ExemptRole:      payload.ExemptRole,
		}

		if err := db.Create(&mask).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "添加脱敏规则失败", err)
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{"item": mask})
	}
}

// deleteKBFieldMask 删除脱敏规则
func deleteKBFieldMask(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		maskID, err := parseRuleID(r, "maskId")
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的脱敏规则ID", err)
			return
		}

		var mask models.KBFieldMask
		if err := db.Where("id = ? AND knowledge_base_id = ?", maskID, kbID).First(&mask).Error; err != nil {
			respondError(w, http.StatusNotFound, "脱敏规则不存在", err)
			return
		}

		if err := db.Delete(&mask).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "删除脱敏规则失败", err)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
	}
}

// ======== 入库与统计 ========

// ingestKnowledgeBase 半自动入库触发
func ingestKnowledgeBase(db *gorm.DB, kbIngestSvc *service.KBIngestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID, err := parseKBID(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的知识库ID", err)
			return
		}

		user, err := getKBUser(db, r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "未登录", err)
			return
		}

		// 校验权限
		if !HasAccess(db, user, kbID) {
			respondError(w, http.StatusForbidden, "无权访问该知识库", nil)
			return
		}

		// 解析请求体
		var body struct {
			Since        string `json:"since"`
			SourceModule string `json:"source_module"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			// body 为空时使用知识库的默认 source_module
		}

		// 校验知识库并填充默认值
		var kb models.KnowledgeBase
		if err := db.First(&kb, kbID).Error; err != nil {
			respondError(w, http.StatusNotFound, "知识库不存在", err)
			return
		}
		if body.SourceModule == "" {
			body.SourceModule = kb.SourceModule
		}
		if body.SourceModule == "" {
			respondError(w, http.StatusBadRequest, "source_module 不能为空", nil)
			return
		}

		// 调用入库服务
		req := service.IngestRequest{
			KBID:         kbID,
			Since:        body.Since,
			SourceModule: body.SourceModule,
		}
		result, err := kbIngestSvc.Ingest(r.Context(), user.ID, req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "入库失败", err)
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// kbStats 知识库统计
func kbStats(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 总数
		var totalCount int64
		db.Model(&models.KnowledgeBase{}).Count(&totalCount)

		// 按可见性统计
		type visibilityStat struct {
			Visibility string `json:"visibility"`
			Count      int64  `json:"count"`
		}
		var visibilityStats []visibilityStat
		db.Model(&models.KnowledgeBase{}).
			Select("visibility, COUNT(*) as count").
			Group("visibility").
			Find(&visibilityStats)

		// 按来源模块统计
		type moduleStat struct {
			SourceModule string `json:"source_module"`
			Count        int64  `json:"count"`
		}
		var moduleStats []moduleStat
		db.Model(&models.KnowledgeBase{}).
			Select("source_module, COUNT(*) as count").
			Group("source_module").
			Find(&moduleStats)

		// 系统模板数量
		var systemCount int64
		db.Model(&models.KnowledgeBase{}).Where("is_system = ?", true).Count(&systemCount)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"total_count":      totalCount,
			"system_count":     systemCount,
			"custom_count":     totalCount - systemCount,
			"by_visibility":    visibilityStats,
			"by_source_module": moduleStats,
		})
	}
}

// ======== 工具函数 ========

// parseKBID 从 URL 路径参数中解析知识库 ID
func parseKBID(r *http.Request) (uint, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// parseRuleID 从 URL 路径参数中解析指定名称的 ID
func parseRuleID(r *http.Request, paramName string) (uint, error) {
	raw := chi.URLParam(r, paramName)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
