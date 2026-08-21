package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// grantAdminContractPermission 为指定角色补充 admin_contract 模块指定动作权限。
func grantAdminContractPermission(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "admin_contract", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "admin_contract", Action: action, Label: action, SortOrder: 94}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// setupAdminContractTestEnv 初始化行政合同测试环境（admin 具备 admin_contract 全权限 + 档案文档表）。
func setupAdminContractTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.AdminContract{}, &models.Document{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantAdminContractPermission(t, env.db, env.roleIDs["admin"], action)
	}
	return env
}

// createTestAdminDocument 创建测试档案文档（document_code 唯一）。
func createTestAdminDocument(t *testing.T, db *gorm.DB, userID uint, code string) models.Document {
	t.Helper()
	doc := models.Document{
		UserID:          userID,
		DocumentCode:    code,
		CategoryCode:    "HT",
		SubCategoryCode: "01",
		Year:            2026,
		Sequence:        1,
		FileName:        "行政合同扫描件-" + code,
		Status:          "active",
	}
	require.NoError(t, db.Create(&doc).Error)
	return doc
}

// createAdminContractViaAPI 通过 API 创建行政合同草稿。
func createAdminContractViaAPI(t *testing.T, env *rbacTestEnv, body map[string]interface{}) models.AdminContract {
	t.Helper()
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/admin-contracts", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建行政合同应成功: %d %s", rec.Code, rec.Body.String())
	var contract models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &contract))
	return contract
}

// TestAdminContractRBACForbidden 无 admin_contract 权限调用全部合同端点必须 403。
func TestAdminContractRBACForbidden(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	token := env.viewer.SupabaseUID // viewer 无 admin_contract 权限
	contract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no":   "XZ-2026-001",
		"name":          "保洁服务合同",
		"counterparty":  "某某保洁公司",
		"contract_type": "服务合同",
		"start_date":    "2026-01-01",
		"end_date":      "2026-12-31",
	})

	cases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/admin-contracts", nil},
		{http.MethodGet, "/api/admin-contracts/expiring", nil},
		{http.MethodPost, "/api/admin-contracts", map[string]interface{}{"contract_no": "XZ-2026-002", "name": "n", "counterparty": "c", "contract_type": "t", "start_date": "2026-01-01", "end_date": "2026-12-31"}},
		{http.MethodPost, "/api/admin-contracts/expire", nil},
		{http.MethodGet, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), nil},
		{http.MethodPut, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), map[string]interface{}{"contract_no": "XZ-2026-001", "name": "n", "counterparty": "c", "contract_type": "t", "start_date": "2026-02-01", "end_date": "2026-12-31"}},
		{http.MethodDelete, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", contract.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", contract.ID), map[string]interface{}{"reason": "作废"}},
	}
	for _, c := range cases {
		rec := doRBACRequest(t, env.router, c.method, c.path, token, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s 应 403: %d %s", c.method, c.path, rec.Code, rec.Body.String())
	}
}

// TestAdminContractLifecycle 完整生命周期：创建草稿 → 列表 → 编辑 → 手动生效 → 到期标记 → 作废限制。
func TestAdminContractLifecycle(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	token := env.admin.SupabaseUID

	// 创建草稿（含选填字段）
	amount := 9999.5
	contract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no":     "XZ-2026-001",
		"name":            "保洁服务合同",
		"counterparty":    "某某保洁公司",
		"contract_type":   "服务合同",
		"start_date":      "2026-01-01",
		"end_date":        "2026-12-31",
		"amount_incl_tax": amount,
		"currency":        "CNY",
		"owner":           "张三",
		"remarks":         "季度结算",
	})
	assert.Equal(t, models.AdminContractStatusDraft, contract.Status)
	assert.Equal(t, "某某保洁公司", contract.Counterparty)
	assert.NotNil(t, contract.AmountInclTax)
	assert.Equal(t, "CNY", contract.Currency)

	// 列表可见
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/admin-contracts", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed, 1)

	// 编辑草稿
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), token, map[string]interface{}{
		"contract_no":   "XZ-2026-001",
		"name":          "保洁服务合同（续）",
		"counterparty":  "某某保洁公司",
		"contract_type": "服务合同",
		"start_date":    "2026-02-01",
		"end_date":      "2027-01-31",
	})
	require.Equal(t, http.StatusOK, rec.Code, "编辑草稿应成功: %s", rec.Body.String())
	var edited models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &edited))
	assert.Equal(t, "2026-02-01", edited.StartDate)
	assert.Equal(t, models.AdminContractStatusDraft, edited.Status)

	// 手动生效（draft → active）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", contract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code, "生效应成功: %s", rec.Body.String())
	var activated models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &activated))
	assert.Equal(t, models.AdminContractStatusActive, activated.Status)
	assert.NotNil(t, activated.ActivatedAt)

	// active 不可编辑/删除
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), token, map[string]interface{}{
		"contract_no": "XZ-2026-001", "name": "n", "counterparty": "c", "contract_type": "t", "start_date": "2026-03-01", "end_date": "2027-02-28",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, "active 合同不可编辑")
	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), token, nil)
	assert.Equal(t, http.StatusConflict, rec.Code, "active 合同不可删除")

	// 到期标记（end_date 过去 → expired）
	require.NoError(t, env.db.Model(&models.AdminContract{}).Where("id = ?", contract.ID).Update("end_date", "2020-01-01").Error)
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/admin-contracts/expire", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var expireResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &expireResp))
	assert.Equal(t, float64(1), expireResp["expired"])

	// expired 为终态不可作废
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", contract.ID), token, map[string]interface{}{"reason": "作废"})
	assert.Equal(t, http.StatusConflict, rec.Code, "已到期合同不可作废: %d %s", rec.Code, rec.Body.String())
}

// TestAdminContractCancelDraftAndActive draft/active 均可由 admin_contract.delete 填写原因作废，作废后可新建替代。
func TestAdminContractCancelDraftAndActive(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	token := env.admin.SupabaseUID

	// 草稿作废
	draft := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-010", "name": "草稿合同", "counterparty": "丙方公司", "contract_type": "服务合同",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", draft.ID), token, map[string]interface{}{"reason": "草稿作废"})
	require.Equal(t, http.StatusOK, rec.Code, "草稿作废应成功: %s", rec.Body.String())
	var cancelledDraft models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelledDraft))
	assert.Equal(t, models.AdminContractStatusCancelled, cancelledDraft.Status)
	assert.Equal(t, "草稿作废", cancelledDraft.CancelReason)

	// active 作废
	active := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-011", "name": "生效合同", "counterparty": "丁方公司", "contract_type": "采购合同",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", active.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", active.ID), token, map[string]interface{}{"reason": "供应商违约，作废重签"})
	require.Equal(t, http.StatusOK, rec.Code, "active 作废应成功: %s", rec.Body.String())
	var cancelledActive models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelledActive))
	assert.Equal(t, models.AdminContractStatusCancelled, cancelledActive.Status)
	assert.Equal(t, "供应商违约，作废重签", cancelledActive.CancelReason)

	// 作废原因必填（用新草稿验证，作废后的合同为终态不可再次作废）
	noReason := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-013", "name": "原因校验合同", "counterparty": "己方公司", "contract_type": "服务合同",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", noReason.ID), token, map[string]interface{}{"reason": "  "})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "作废原因必填: %d %s", rec.Code, rec.Body.String())

	// 作废后新建替代成功
	replacement := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-012", "name": "替代合同", "counterparty": "戊方公司", "contract_type": "采购合同",
		"start_date": "2026-03-01", "end_date": "2027-02-28",
	})
	assert.Equal(t, models.AdminContractStatusDraft, replacement.Status)
}
