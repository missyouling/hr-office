package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- 档案配置路由 RBAC 测试 ----------
// 覆盖 /archives 下配置类接口（categories/sub-categories/shared-fields/field-groups/
// field-definitions/retention-periods/storage-locations/code-rules/column-config）
// 权限约定：读 archives.view / 增 archives.create / 改 archives.edit / 删 archives.delete

// TestRBACArchivesConfigViewerReadOnly viewer（仅 archives.view）读成功、写一律 403
func TestRBACArchivesConfigViewerReadOnly(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.viewer.SupabaseUID

	// 读成功（archives.view）
	readCases := []struct {
		name string
		path string
	}{
		{"一级分类列表", "/api/archives/categories"},
		{"共享字段列表", "/api/archives/shared-fields"},
		{"字段分组列表", "/api/archives/field-groups"},
		{"字段定义列表", "/api/archives/field-definitions"},
		{"保管期限列表", "/api/archives/retention-periods"},
		{"存档地点列表", "/api/archives/storage-locations"},
		{"编码规则列表", "/api/archives/code-rules"},
		{"编码规则预览", "/api/archives/code-rules/preview?category_code=WS&sub_category_code=GW"},
		{"列配置读取", "/api/archives/column-config?sub_category_id=1"},
		{"二级分类字段", "/api/archives/sub-categories/1/fields"},
	}
	for _, tc := range readCases {
		t.Run("读-"+tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, http.MethodGet, tc.path, token, nil)
			assert.Equal(t, http.StatusOK, rec.Code, "%s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}

	// 写失败（viewer 无 archives.create/edit/delete）
	writeCases := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"一级分类创建", http.MethodPost, "/api/archives/categories", map[string]interface{}{"code": "WS", "name": "文书"}},
		{"一级分类修改", http.MethodPut, "/api/archives/categories/1", map[string]interface{}{"code": "WS", "name": "文书"}},
		{"一级分类删除", http.MethodDelete, "/api/archives/categories/1", nil},
		{"二级分类创建", http.MethodPost, "/api/archives/sub-categories", map[string]interface{}{"category_id": 1, "code": "GW", "name": "公文"}},
		{"二级分类修改", http.MethodPut, "/api/archives/sub-categories/1", map[string]interface{}{"code": "GW", "name": "公文"}},
		{"二级分类删除", http.MethodDelete, "/api/archives/sub-categories/1", nil},
		{"字段分组创建", http.MethodPost, "/api/archives/field-groups", map[string]interface{}{"name": "分组"}},
		{"字段分组修改", http.MethodPut, "/api/archives/field-groups/1", map[string]interface{}{"name": "分组2"}},
		{"字段分组删除", http.MethodDelete, "/api/archives/field-groups/1", nil},
		{"字段定义创建", http.MethodPost, "/api/archives/field-definitions", map[string]interface{}{"sub_category_id": 1, "field_name": "f1", "field_label": "字段1", "field_type": "text"}},
		{"字段定义修改", http.MethodPut, "/api/archives/field-definitions/1", map[string]interface{}{"field_label": "字段1改"}},
		{"字段定义删除", http.MethodDelete, "/api/archives/field-definitions/1", nil},
		{"保管期限创建", http.MethodPost, "/api/archives/retention-periods", map[string]interface{}{"name": "永久", "years": 100}},
		{"保管期限修改", http.MethodPut, "/api/archives/retention-periods/1", map[string]interface{}{"name": "永久2"}},
		{"保管期限删除", http.MethodDelete, "/api/archives/retention-periods/1", nil},
		{"存档地点创建", http.MethodPost, "/api/archives/storage-locations", map[string]interface{}{"name": "档案室A"}},
		{"存档地点修改", http.MethodPut, "/api/archives/storage-locations/1", map[string]interface{}{"name": "档案室B"}},
		{"存档地点删除", http.MethodDelete, "/api/archives/storage-locations/1", nil},
		{"编码规则创建", http.MethodPost, "/api/archives/code-rules", map[string]interface{}{"name": "规则1", "code_format": "{CATEGORY}-{SEQ}"}},
		{"编码规则修改", http.MethodPut, "/api/archives/code-rules/1", map[string]interface{}{"name": "规则1改"}},
		{"编码规则删除", http.MethodDelete, "/api/archives/code-rules/1", nil},
		{"列配置保存", http.MethodPost, "/api/archives/column-config", map[string]interface{}{"sub_category_id": 1, "column_keys": []string{"a"}}},
		{"列配置更新", http.MethodPut, "/api/archives/column-config", map[string]interface{}{"sub_category_id": 1, "column_keys": []string{"a"}}},
	}
	for _, tc := range writeCases {
		t.Run("写-"+tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, tc.method, tc.path, token, tc.body)
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}
}

// TestRBACArchivesConfigAuthorizedWrite 授权角色对应写操作成功，越权操作被拒
func TestRBACArchivesConfigAuthorizedWrite(t *testing.T) {
	env := setupRBACTestEnv(t)
	managerToken := env.manager.SupabaseUID

	// --- manager（archives.view/create/edit，无 delete）创建+修改成功 ---

	// 1. 一级分类：创建 + 修改
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/archives/categories", managerToken,
		map[string]interface{}{"code": "WS", "name": "文书"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var category struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &category))
	require.NotZero(t, category.ID, "一级分类应创建成功并返回 ID")

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/categories/%d", category.ID), managerToken,
		map[string]interface{}{"code": "WS", "name": "文书档案"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 2. 二级分类：创建（后续字段分组/字段定义依赖其 ID）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/sub-categories", managerToken,
		map[string]interface{}{"category_id": category.ID, "code": "GW", "name": "公文"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var subCategory struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &subCategory))
	require.NotZero(t, subCategory.ID, "二级分类应创建成功并返回 ID")

	// 3. 字段分组：创建 + 修改（sub_category_id 外键指向已建二级分类）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/field-groups", managerToken,
		map[string]interface{}{"sub_category_id": subCategory.ID, "name": "基础信息", "sort_order": 1})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var group struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &group))
	require.NotZero(t, group.ID, "字段分组应创建成功并返回 ID")

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/field-groups/%d", group.ID), managerToken,
		map[string]interface{}{"name": "基础信息改"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 4. 字段定义：创建 + 修改
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/field-definitions", managerToken,
		map[string]interface{}{"sub_category_id": subCategory.ID, "field_name": "custom_field", "field_label": "自定义字段", "field_type": "text"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var fieldDef struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fieldDef))
	require.NotZero(t, fieldDef.ID, "字段定义应创建成功并返回 ID")

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/field-definitions/%d", fieldDef.ID), managerToken,
		map[string]interface{}{"field_label": "自定义字段改"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 5. 保管期限：创建 + 修改
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/retention-periods", managerToken,
		map[string]interface{}{"name": "永久", "years": 100})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var period struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &period))
	require.NotZero(t, period.ID, "保管期限应创建成功并返回 ID")

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/retention-periods/%d", period.ID), managerToken,
		map[string]interface{}{"name": "永久改", "years": 100})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 6. 存档地点：创建 + 修改
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/storage-locations", managerToken,
		map[string]interface{}{"name": "档案室A"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var location struct{ ID uint }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &location))
	require.NotZero(t, location.ID, "存档地点应创建成功并返回 ID")

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/storage-locations/%d", location.ID), managerToken,
		map[string]interface{}{"name": "档案室B"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 7. 编码规则：创建成功
	// 注意：updateCodeRule/deleteCodeRule 业务层以 user_id 过滤，而 createCodeRule 不落 user_id，
	// 属既有遗留问题（本任务不改业务行为），因此此处仅验证创建（权限放行）。
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/code-rules", managerToken,
		map[string]interface{}{"name": "规则1", "code_format": "{CATEGORY}-{SEQ}"})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 8. 列配置保存（POST = archives.create）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/column-config", managerToken,
		map[string]interface{}{"sub_category_id": subCategory.ID, "column_keys": []string{"field_name"}})
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 9. manager 无 archives.delete → 删除 403
	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/archives/storage-locations/%d", location.ID), managerToken, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "manager 无删除权限应 403: %s", rec.Body.String())

	// --- editor（archives.view/edit，无 create/delete）：修改成功、创建 403 ---
	editorToken := env.editor.SupabaseUID

	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/archives/field-groups/%d", group.ID), editorToken,
		map[string]interface{}{"name": "编辑者改"})
	assert.Equal(t, http.StatusOK, rec.Code, "editor 有 edit 权限应修改成功: %s", rec.Body.String())

	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/archives/field-groups", editorToken,
		map[string]interface{}{"name": "编辑者创建"})
	assert.Equal(t, http.StatusForbidden, rec.Code, "editor 无 create 权限应 403: %s", rec.Body.String())

	// --- admin（全量权限）：删除成功 ---
	adminToken := env.admin.SupabaseUID

	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/archives/storage-locations/%d", location.ID), adminToken, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "admin 删除存档地点应成功: %s", rec.Body.String())

	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/archives/field-groups/%d", group.ID), adminToken, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "admin 删除字段分组应成功: %s", rec.Body.String())
}

// TestRBACArchivesCodeRulePreviewNotShadowed 静态路径 /code-rules/preview 不被 /{ruleID} 吞掉
func TestRBACArchivesCodeRulePreviewNotShadowed(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.viewer.SupabaseUID

	// 带参数：正常返回预览 200（证明路由匹配到 /code-rules/preview）
	rec := doRBACRequest(t, env.router, http.MethodGet,
		"/api/archives/code-rules/preview?category_code=WS&sub_category_code=GW", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "preview 带参数应 200: %s", rec.Body.String())

	// 不带参数：业务层 400（证明路由正确匹配 preview handler；
	// 若被 /{ruleID} 吞掉，GET 方法未注册会返回 404/405）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/archives/code-rules/preview", token, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "preview 缺参数应业务 400（而非 404/405）: %s", rec.Body.String())
}

// TestRBACArchivesPersonalEndpointsNotAffected 个人用户接口不受档案配置权限影响
func TestRBACArchivesPersonalEndpointsNotAffected(t *testing.T) {
	env := setupRBACTestEnv(t)
	token := env.viewer.SupabaseUID

	// 个人文档列表（未挂配置权限）不受影响
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/archives/documents", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "个人文档列表应可访问: %s", rec.Body.String())

	// 个人通知不受影响
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/notifications", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "个人通知应可访问: %s", rec.Body.String())
}
