package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"siapp/internal/models"
)

// TestAdminContractCancelRequiresDeletePermission 作废合同权限必须为 admin_contract.delete（而非 edit）：
// 持有 delete 但无 edit 的角色可作废；持有 edit 但无 delete 的角色不可作废。
func TestAdminContractCancelRequiresDeletePermission(t *testing.T) {
	env := setupAdminContractTestEnv(t)

	// 角色 A：view+create+delete（无 edit）
	deleteOnlyRole := models.Role{Name: "admin_contract_delete_only", Label: "仅作废", IsSystem: false}
	require.NoError(t, env.db.Create(&deleteOnlyRole).Error)
	for _, action := range []string{"view", "create", "delete"} {
		grantAdminContractPermission(t, env.db, deleteOnlyRole.ID, action)
	}
	// 角色 B：view+create+edit（无 delete）
	editOnlyRole := models.Role{Name: "admin_contract_edit_only", Label: "仅编辑", IsSystem: false}
	require.NoError(t, env.db.Create(&editOnlyRole).Error)
	for _, action := range []string{"view", "create", "edit"} {
		grantAdminContractPermission(t, env.db, editOnlyRole.ID, action)
	}

	deleteOnlyUser := createRBACTestUser(t, env.db, "acDeleteOnly", "ac-delete-only-uuid", deleteOnlyRole.ID)
	editOnlyUser := createRBACTestUser(t, env.db, "acEditOnly", "ac-edit-only-uuid", editOnlyRole.ID)

	// delete-only：可创建草稿（create）
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/admin-contracts", deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"contract_no": "XZ-2026-020", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "delete-only 创建应成功: %d %s", rec.Code, rec.Body.String())
	var deleteOnlyContract models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleteOnlyContract))

	// delete-only：有 delete 无 edit，作废成功
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废重签"})
	require.Equal(t, http.StatusOK, rec.Code, "delete-only 作废应成功: %d %s", rec.Code, rec.Body.String())

	// delete-only：无 edit，编辑/生效应 403（证明作废走的是 delete 而非 edit）
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/admin-contracts/%d", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"contract_no": "XZ-2026-020", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-02-01", "end_date": "2026-12-31",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 编辑应 403: %d %s", rec.Code, rec.Body.String())
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 生效应 403: %d %s", rec.Code, rec.Body.String())

	// edit-only：可创建并生效（有 edit）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/admin-contracts", editOnlyUser.SupabaseUID, map[string]interface{}{
		"contract_no": "XZ-2026-021", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "edit-only 创建应成功: %d %s", rec.Code, rec.Body.String())
	var editOnlyContract models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &editOnlyContract))
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", editOnlyContract.ID), editOnlyUser.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code, "edit-only 生效应成功: %d %s", rec.Code, rec.Body.String())

	// edit-only：有 edit 无 delete，作废应 403
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/cancel", editOnlyContract.ID), editOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废"})
	assert.Equal(t, http.StatusForbidden, rec.Code, "edit-only 无 delete 作废应 403: %d %s", rec.Code, rec.Body.String())
}

// TestAdminContractDocumentLink 复用档案文档关联字段：关联文档校验租户归属。
func TestAdminContractDocumentLink(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	token := env.admin.SupabaseUID

	doc := createTestAdminDocument(t, env.db, env.admin.ID, "XZ-DOC-001")

	// 关联本租户文档成功
	contract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-030", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": "2026-12-31", "document_id": doc.ID,
	})
	assert.Equal(t, doc.ID, *contract.DocumentID)

	// 关联不存在文档失败
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/admin-contracts", token, map[string]interface{}{
		"contract_no": "XZ-2026-031", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": "2026-12-31", "document_id": 99999,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "关联不存在文档应失败: %s", rec.Body.String())
}

// TestAdminContractExpiringReminder 到期提醒查询：active 且未来 N 天内到期返回，过期/作废不返回。
func TestAdminContractExpiringReminder(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	token := env.admin.SupabaseUID

	// 未来 10 天到期（active）
	soon := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	contract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-R-001", "name": "即将到期合同", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": soon,
	})
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", contract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// 已到期合同（active 但 end_date 过去，查询前应被惰性标记为 expired）
	past := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	pastContract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-R-002", "name": "已到期合同", "counterparty": "c", "contract_type": "t",
		"start_date": "2020-01-01", "end_date": past,
	})
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/admin-contracts/%d/activate", pastContract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/admin-contracts/expiring?days=30", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Days      int                    `json:"days"`
		Contracts []models.AdminContract `json:"contracts"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.Days)
	require.Len(t, resp.Contracts, 1, "仅未来到期合同应出现在提醒中")
	assert.Equal(t, contract.ID, resp.Contracts[0].ID)

	// 已到期合同已被惰性标记为 expired
	var pastReload models.AdminContract
	require.NoError(t, env.db.First(&pastReload, pastContract.ID).Error)
	assert.Equal(t, models.AdminContractStatusExpired, pastReload.Status)
	assert.NotNil(t, pastReload.ExpiredAt)

	// 非法 days 参数返回 400
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/admin-contracts/expiring?days=0", token, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "非法 days 应 400: %d %s", rec.Code, rec.Body.String())
}

// TestAdminContractTenantIsolation 租户隔离：数据按 user_id 隔离，跨用户互不可见。
func TestAdminContractTenantIsolation(t *testing.T) {
	env := setupAdminContractTestEnv(t)

	// 为 viewer 补充 view+create+edit 权限（edit 确保请求能通过权限中间件，验证数据层隔离而非权限拦截）
	for _, action := range []string{"view", "create", "edit"} {
		grantAdminContractPermission(t, env.db, env.roleIDs["viewer"], action)
	}
	other := createRBACTestUser(t, env.db, "acOther", "ac-other-uuid", env.roleIDs["viewer"])

	// admin 创建合同
	contract := createAdminContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "XZ-2026-040", "name": "管理员合同", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-01-01", "end_date": "2026-12-31",
	})

	// 其他用户列表不可见
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/admin-contracts", other.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.AdminContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Len(t, listed, 0, "其他用户不应看到管理员创建的行政合同")

	// 其他用户详情不可见
	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), other.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "其他用户详情应 404")

	// 其他用户不能修改/作废管理员的合同
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/admin-contracts/%d", contract.ID), other.SupabaseUID, map[string]interface{}{
		"contract_no": "XZ-2026-040", "name": "n", "counterparty": "c", "contract_type": "t",
		"start_date": "2026-02-01", "end_date": "2026-12-31",
	})
	assert.Equal(t, http.StatusNotFound, rec.Code, "其他用户编辑应 404")
}

// TestAdminContractExpiryWorker 到期扫描定时任务（确定性测试，不真实等待）：
// active 且 end_date 早于当日 → expired；当日/未来保持 active；draft/cancelled 不受影响。
func TestAdminContractExpiryWorker(t *testing.T) {
	env := setupAdminContractTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 1, 10, 2, 0, 0, 0, loc)

	newContract := func(no, endDate, status string) models.AdminContract {
		c := models.AdminContract{
			UserID:       env.admin.ID,
			ContractNo:   no,
			Name:         "合同-" + no,
			Counterparty: "外部主体",
			ContractType: "服务合同",
			StartDate:    "2025-01-01",
			EndDate:      endDate,
			Status:       status,
		}
		require.NoError(t, env.db.Create(&c).Error)
		return c
	}

	activePast := newContract("XZ-W-001", "2026-01-09", models.AdminContractStatusActive)       // 早于当日 → expired
	activeToday := newContract("XZ-W-002", "2026-01-10", models.AdminContractStatusActive)      // 等于当日 → 保持 active
	activeFuture := newContract("XZ-W-003", "2026-12-31", models.AdminContractStatusActive)     // 未来 → 保持 active
	draftPast := newContract("XZ-W-004", "2025-12-31", models.AdminContractStatusDraft)         // 非 active → 不动
	cancelledPast := newContract("XZ-W-005", "2025-12-31", models.AdminContractStatusCancelled) // 非 active → 不动

	handler := NewHandler(env.db)
	handler.runAdminContractExpiryOnce(loc, now)

	// 每次查询使用独立变量，避免 GORM First 复用 dest 时残留主键参与条件
	loadContract := func(id uint) models.AdminContract {
		var c models.AdminContract
		require.NoError(t, env.db.First(&c, id).Error)
		return c
	}

	got := loadContract(activePast.ID)
	assert.Equal(t, models.AdminContractStatusExpired, got.Status, "早于当日的 active 合同应标记 expired")
	assert.NotNil(t, got.ExpiredAt, "应记录到期标记时间")

	got = loadContract(activeToday.ID)
	assert.Equal(t, models.AdminContractStatusActive, got.Status, "end_date 等于当日不应标记过期")

	got = loadContract(activeFuture.ID)
	assert.Equal(t, models.AdminContractStatusActive, got.Status, "未来到期不应标记过期")

	got = loadContract(draftPast.ID)
	assert.Equal(t, models.AdminContractStatusDraft, got.Status, "草稿不应被扫描")

	got = loadContract(cancelledPast.ID)
	assert.Equal(t, models.AdminContractStatusCancelled, got.Status, "已作废合同不应被扫描")

	// 幂等：再次运行不改变已标记结果（批量更新天然幂等）
	handler.runAdminContractExpiryOnce(loc, now)
	got = loadContract(activePast.ID)
	assert.Equal(t, models.AdminContractStatusExpired, got.Status, "重复运行应保持幂等")
}
