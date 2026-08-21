package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// grantContractPermission 为指定角色补充 contract 模块指定动作权限。
func grantContractPermission(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "contract", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "contract", Action: action, Label: action, SortOrder: 90}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// setupContractTestEnv 初始化劳动合同测试环境（admin 具备 contract 全权限 + 档案文档表）
func setupContractTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.LaborContract{}, &models.Document{}, &models.Employee{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantContractPermission(t, env.db, env.roleIDs["admin"], action)
	}
	return env
}

// createTestDocument 创建测试档案文档（document_code 唯一）。
func createTestDocument(t *testing.T, db *gorm.DB, userID uint, code string) models.Document {
	t.Helper()
	doc := models.Document{
		UserID:          userID,
		DocumentCode:    code,
		CategoryCode:    "HT",
		SubCategoryCode: "01",
		Year:            2026,
		Sequence:        1,
		FileName:        "合同扫描件-" + code,
		Status:          "active",
	}
	require.NoError(t, db.Create(&doc).Error)
	return doc
}

// createContractViaAPI 通过 API 创建劳动合同草稿。
func createContractViaAPI(t *testing.T, env *rbacTestEnv, body map[string]interface{}) models.LaborContract {
	t.Helper()
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/contracts", env.admin.SupabaseUID, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建合同应成功: %d %s", rec.Code, rec.Body.String())
	var contract models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &contract))
	return contract
}

// TestContractRBACForbidden 无 contract 权限调用全部合同端点必须 403
func TestContractRBACForbidden(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.viewer.SupabaseUID // viewer 无 contract 权限
	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-2026-001",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
	})

	cases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/contracts", nil},
		{http.MethodGet, "/api/contracts/expiring", nil},
		{http.MethodPost, "/api/contracts", map[string]interface{}{"start_date": "2026-01-01", "end_date": "2028-12-31", "term_months": 36, "department": "测试部"}},
		{http.MethodPost, "/api/contracts/batch", []map[string]interface{}{{"start_date": "2026-01-01", "end_date": "2028-12-31", "term_months": 36, "department": "测试部"}}},
		{http.MethodPost, "/api/contracts/expire", nil},
		{http.MethodGet, fmt.Sprintf("/api/contracts/%d", contract.ID), nil},
		{http.MethodPut, fmt.Sprintf("/api/contracts/%d", contract.ID), map[string]interface{}{"start_date": "2026-02-01", "end_date": "2028-12-31", "term_months": 35, "department": "测试部"}},
		{http.MethodDelete, fmt.Sprintf("/api/contracts/%d", contract.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", contract.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/contracts/%d/cancel", contract.ID), map[string]interface{}{"reason": "作废"}},
	}
	for _, c := range cases {
		rec := doRBACRequest(t, env.router, c.method, c.path, token, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s 应 403: %d %s", c.method, c.path, rec.Code, rec.Body.String())
	}
}

// TestContractLifecycle 完整生命周期：创建草稿 → 列表 → 编辑 → 手动生效 → 到期标记 → 作废
func TestContractLifecycle(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.admin.SupabaseUID

	// 创建草稿
	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-2026-001",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
		"position":    "测试岗",
		"name":        "张三",
	})
	assert.Equal(t, models.ContractStatusDraft, contract.Status)
	assert.Equal(t, models.ContractTypeFixedTerm, contract.ContractType)
	assert.Equal(t, "测试部", contract.SnapshotDepartment)

	// 列表可见
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/contracts", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed, 1)

	// 编辑草稿
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/contracts/%d", contract.ID), token, map[string]interface{}{
		"contract_no": "HT-2026-001",
		"start_date":  "2026-02-01",
		"end_date":    "2029-01-31",
		"term_months": 36,
		"department":  "测试部",
	})
	require.Equal(t, http.StatusOK, rec.Code, "编辑草稿应成功: %s", rec.Body.String())
	var edited models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &edited))
	assert.Equal(t, "2026-02-01", edited.StartDate)
	assert.Equal(t, models.ContractStatusDraft, edited.Status)

	// 手动生效（draft → active）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", contract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code, "生效应成功: %s", rec.Body.String())
	var activated models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &activated))
	assert.Equal(t, models.ContractStatusActive, activated.Status)
	assert.NotNil(t, activated.ActivatedAt)

	// 到期标记（end_date 过去 → expired）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/contracts/expire", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	// 将合同到期日改为过去再触发
	require.NoError(t, env.db.Model(&models.LaborContract{}).Where("id = ?", contract.ID).Update("end_date", "2020-01-01").Error)
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/contracts/expire", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var expireResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &expireResp))
	assert.Equal(t, float64(1), expireResp["expired"])

	// 作废（expired 为终态不可作废）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/cancel", contract.ID), token, map[string]interface{}{"reason": "作废"})
	assert.Equal(t, http.StatusConflict, rec.Code, "已到期合同不可作废: %d %s", rec.Code, rec.Body.String())
}

// TestContractActiveNotEditableDeletable active 合同不可编辑/删除，只能作废后新建
func TestContractActiveNotEditableDeletable(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.admin.SupabaseUID

	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-2026-002",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
	})
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", contract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// active 不可编辑
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/contracts/%d", contract.ID), token, map[string]interface{}{
		"start_date": "2026-03-01", "end_date": "2029-02-28", "term_months": 36, "department": "测试部",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, "active 合同不可编辑")

	// active 不可删除
	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/contracts/%d", contract.ID), token, nil)
	assert.Equal(t, http.StatusConflict, rec.Code, "active 合同不可删除")

	// 作废后新建
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/cancel", contract.ID), token, map[string]interface{}{"reason": "合同内容有误，作废重签"})
	require.Equal(t, http.StatusOK, rec.Code, "active 合同可作废: %s", rec.Body.String())
	var cancelled models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelled))
	assert.Equal(t, models.ContractStatusCancelled, cancelled.Status)
	assert.NotNil(t, cancelled.CancelledAt)
	assert.Equal(t, "合同内容有误，作废重签", cancelled.CancelReason)

	// 作废后新建成功
	newContract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-2026-003",
		"start_date":  "2026-04-01",
		"end_date":    "2029-03-31",
		"term_months": 36,
		"department":  "测试部",
	})
	assert.Equal(t, models.ContractStatusDraft, newContract.Status)
}

// TestContractBatchCreate 批量创建：成功条目入库，失败条目返回明细
func TestContractBatchCreate(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.admin.SupabaseUID

	payloads := []map[string]interface{}{
		{"contract_no": "HT-B-001", "start_date": "2026-01-01", "end_date": "2028-12-31", "term_months": 36, "department": "测试部"},
		{"contract_no": "HT-B-002", "start_date": "2026-01-01", "end_date": "2028-12-31", "term_months": 36, "department": "测试部"},
		{"contract_no": "HT-B-003", "start_date": "bad-date", "end_date": "2028-12-31", "term_months": 36, "department": "测试部"},  // 失败：日期格式错误
		{"contract_no": "HT-B-004", "start_date": "2026-01-01", "end_date": "2028-12-31", "term_months": 0, "department": "测试部"}, // 失败：期限非正数
	}
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/contracts/batch", token, payloads)
	require.Equal(t, http.StatusOK, rec.Code, "批量创建应成功: %s", rec.Body.String())
	var resp struct {
		Created []models.LaborContract `json:"created"`
		Failed  []map[string]any       `json:"failed"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Created, 2, "应成功创建 2 条")
	assert.Len(t, resp.Failed, 2, "应返回 2 条失败明细")
	for _, f := range resp.Failed {
		assert.NotEmpty(t, f["error"])
	}
}

// TestContractExpiringReminder 到期提醒查询：active 且未来 N 天内到期返回，过期/作废不返回
func TestContractExpiringReminder(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.admin.SupabaseUID

	// 未来 10 天到期（active）
	soon := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-R-001",
		"start_date":  "2026-01-01",
		"end_date":    soon,
		"term_months": 12,
		"department":  "测试部",
	})
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", contract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// 已到期合同（active 但 end_date 过去，查询前应被惰性标记为 expired）
	past := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	pastContract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-R-002",
		"start_date":  "2020-01-01",
		"end_date":    past,
		"term_months": 12,
		"department":  "测试部",
	})
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", pastContract.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/contracts/expiring?days=30", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Days      int                    `json:"days"`
		Contracts []models.LaborContract `json:"contracts"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 30, resp.Days)
	require.Len(t, resp.Contracts, 1, "仅未来到期合同应出现在提醒中")
	assert.Equal(t, contract.ID, resp.Contracts[0].ID)

	// 已到期合同已被惰性标记为 expired
	var pastReload models.LaborContract
	require.NoError(t, env.db.First(&pastReload, pastContract.ID).Error)
	assert.Equal(t, models.ContractStatusExpired, pastReload.Status)
	assert.NotNil(t, pastReload.ExpiredAt)
}

// TestContractDepartmentIsolation 租户与部门快照隔离：
// 数据按创建者 user_id + 创建时部门快照双隔离，跨用户/跨部门互不可见。
func TestContractDepartmentIsolation(t *testing.T) {
	env := setupContractTestEnv(t)

	// 创建两个不同部门的用户并授予 contract.view + create
	deptA := createRBACTestUser(t, env.db, "deptA", "deptA-uuid", env.roleIDs["viewer"])
	deptB := createRBACTestUser(t, env.db, "deptB", "deptB-uuid", env.roleIDs["viewer"])
	require.NoError(t, env.db.Model(&deptA).Update("department", "测试部A").Error)
	require.NoError(t, env.db.Model(&deptB).Update("department", "测试部B").Error)
	grantContractPermission(t, env.db, env.roleIDs["viewer"], "view")
	grantContractPermission(t, env.db, env.roleIDs["viewer"], "create")

	// deptA 用户自己创建部门 A 的合同（快照部门 = 测试部A）
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/contracts", deptA.SupabaseUID, map[string]interface{}{
		"contract_no": "HT-D-001",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部A",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "deptA 创建合同应成功: %s", rec.Body.String())
	var contract models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &contract))
	assert.Equal(t, "测试部A", contract.SnapshotDepartment)

	// 同一租户（deptA 名下）插入一条部门快照为 B 的合同（模拟创建后部门快照冻结）
	otherDept := models.LaborContract{
		UserID:             deptA.ID,
		ContractNo:         "HT-D-002",
		ContractType:       models.ContractTypeFixedTerm,
		StartDate:          "2026-01-01",
		EndDate:            "2028-12-31",
		TermMonths:         36,
		SnapshotDepartment: "测试部B",
		Status:             models.ContractStatusDraft,
	}
	require.NoError(t, env.db.Create(&otherDept).Error)

	// 部门 B 用户列表不可见（user_id 不同 + 部门快照不匹配）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/contracts", deptB.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listedB []models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listedB))
	assert.Len(t, listedB, 0, "部门 B 用户不应看到部门 A 的合同")

	// 部门 B 用户详情不可见
	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/contracts/%d", contract.ID), deptB.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "部门 B 用户详情应 404")

	// 部门 A 用户列表仅可见部门快照 A 的合同（部门快照过滤生效）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/contracts", deptA.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listedA []models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listedA))
	require.Len(t, listedA, 1, "部门 A 用户应仅看到部门快照 A 的合同")
	assert.Equal(t, contract.ID, listedA[0].ID)
}

// TestContractEmployeeSnapshot 关联员工时快照从员工主表拷贝
func TestContractEmployeeSnapshot(t *testing.T) {
	env := setupContractTestEnv(t)

	emp := models.Employee{
		UserID:     env.admin.ID,
		Name:       "李四",
		IDNumber:   "110101199001011234",
		Department: "测试部",
		Position:   "开发岗",
		Status:     "active",
	}
	require.NoError(t, env.db.Create(&emp).Error)

	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-E-001",
		"employee_id": emp.ID,
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
	})
	assert.Equal(t, "李四", contract.SnapshotName)
	assert.Equal(t, "测试部", contract.SnapshotDepartment)
	assert.Equal(t, "开发岗", contract.SnapshotPosition)
	assert.Equal(t, "110101199001011234", contract.SnapshotIDNumber)
}

// TestContractDocumentLink 复用档案文档关联字段：关联文档校验租户归属
func TestContractDocumentLink(t *testing.T) {
	env := setupContractTestEnv(t)
	token := env.admin.SupabaseUID

	doc := createTestDocument(t, env.db, env.admin.ID, "HT-DOC-001")

	// 关联本租户文档成功
	contract := createContractViaAPI(t, env, map[string]interface{}{
		"contract_no": "HT-L-001",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
		"document_id": doc.ID,
	})
	assert.Equal(t, doc.ID, *contract.DocumentID)

	// 关联不存在文档失败
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/contracts", token, map[string]interface{}{
		"contract_no": "HT-L-002",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
		"document_id": 99999,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "关联不存在文档应失败: %s", rec.Body.String())
}

// TestContractCancelRequiresDeletePermission 作废合同权限必须为 contract.delete（而非 contract.edit）：
// 持有 delete 但无 edit 的角色可作废；持有 edit 但无 delete 的角色不可作废。
func TestContractCancelRequiresDeletePermission(t *testing.T) {
	env := setupContractTestEnv(t)

	// 角色 A：view+create+delete（无 edit）
	deleteOnlyRole := models.Role{Name: "contract_delete_only", Label: "仅作废", IsSystem: false}
	require.NoError(t, env.db.Create(&deleteOnlyRole).Error)
	for _, action := range []string{"view", "create", "delete"} {
		grantContractPermission(t, env.db, deleteOnlyRole.ID, action)
	}
	// 角色 B：view+create+edit（无 delete）
	editOnlyRole := models.Role{Name: "contract_edit_only", Label: "仅编辑", IsSystem: false}
	require.NoError(t, env.db.Create(&editOnlyRole).Error)
	for _, action := range []string{"view", "create", "edit"} {
		grantContractPermission(t, env.db, editOnlyRole.ID, action)
	}

	deleteOnlyUser := createRBACTestUser(t, env.db, "contractDeleteOnly", "contract-delete-only-uuid", deleteOnlyRole.ID)
	editOnlyUser := createRBACTestUser(t, env.db, "contractEditOnly", "contract-edit-only-uuid", editOnlyRole.ID)

	// delete-only：可创建草稿（create）
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/contracts", deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"contract_no": "HT-CL-001",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "delete-only 创建合同应成功: %d %s", rec.Code, rec.Body.String())
	var deleteOnlyContract models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleteOnlyContract))

	// delete-only：有 delete 无 edit，作废成功
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/cancel", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废重签"})
	require.Equal(t, http.StatusOK, rec.Code, "delete-only 作废合同应成功: %d %s", rec.Code, rec.Body.String())
	var cancelled models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelled))
	assert.Equal(t, models.ContractStatusCancelled, cancelled.Status)

	// delete-only：无 edit，编辑/生效应 403（证明作废走的是 delete 而非 edit）
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/contracts/%d", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"start_date": "2026-02-01", "end_date": "2029-01-31", "term_months": 36, "department": "测试部",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 编辑应 403: %d %s", rec.Code, rec.Body.String())
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", deleteOnlyContract.ID), deleteOnlyUser.SupabaseUID, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 生效应 403: %d %s", rec.Code, rec.Body.String())

	// edit-only：可创建并生效（有 edit）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/contracts", editOnlyUser.SupabaseUID, map[string]interface{}{
		"contract_no": "HT-CL-002",
		"start_date":  "2026-01-01",
		"end_date":    "2028-12-31",
		"term_months": 36,
		"department":  "测试部",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "edit-only 创建合同应成功: %d %s", rec.Code, rec.Body.String())
	var editOnlyContract models.LaborContract
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &editOnlyContract))
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/activate", editOnlyContract.ID), editOnlyUser.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code, "edit-only 生效合同应成功: %d %s", rec.Code, rec.Body.String())

	// edit-only：有 edit 无 delete，作废应 403
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/contracts/%d/cancel", editOnlyContract.ID), editOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废"})
	assert.Equal(t, http.StatusForbidden, rec.Code, "edit-only 无 delete 作废应 403: %d %s", rec.Code, rec.Body.String())
}

// TestContractExpiryWorker 到期扫描定时任务（确定性测试，不真实等待）：
// active 且 end_date 早于当日 → expired；当日/未来保持 active；draft/cancelled 不受影响；不修改员工状态。
func TestContractExpiryWorker(t *testing.T) {
	env := setupContractTestEnv(t)
	loc := shanghaiLoc(t)
	now := time.Date(2026, 1, 10, 2, 0, 0, 0, loc)

	// 关联员工：验证到期扫描不修改员工状态
	emp := models.Employee{
		UserID: env.admin.ID, Name: "合同员工", IDNumber: "110101199001011234",
		Department: "测试部", Position: "测试岗", Status: "active",
	}
	require.NoError(t, env.db.Create(&emp).Error)

	newContract := func(no, endDate, status string, empID *uint) models.LaborContract {
		c := models.LaborContract{
			UserID: env.admin.ID, ContractNo: no, ContractType: models.ContractTypeFixedTerm,
			StartDate: "2025-01-01", EndDate: endDate, TermMonths: 12,
			SnapshotDepartment: "测试部", Status: status, EmployeeID: empID,
		}
		require.NoError(t, env.db.Create(&c).Error)
		return c
	}

	activePast := newContract("HT-W-001", "2026-01-09", models.ContractStatusActive, &emp.ID)   // 早于当日 → expired
	activeToday := newContract("HT-W-002", "2026-01-10", models.ContractStatusActive, nil)      // 等于当日 → 保持 active
	activeFuture := newContract("HT-W-003", "2026-12-31", models.ContractStatusActive, nil)     // 未来 → 保持 active
	draftPast := newContract("HT-W-004", "2025-12-31", models.ContractStatusDraft, nil)         // 非 active → 不动
	cancelledPast := newContract("HT-W-005", "2025-12-31", models.ContractStatusCancelled, nil) // 非 active → 不动

	handler := NewHandler(env.db)
	handler.runContractExpiryOnce(loc, now)

	// 每次查询使用独立变量，避免 GORM First 复用 dest 时残留主键参与条件
	loadContract := func(id uint) models.LaborContract {
		var c models.LaborContract
		require.NoError(t, env.db.First(&c, id).Error)
		return c
	}

	got := loadContract(activePast.ID)
	assert.Equal(t, models.ContractStatusExpired, got.Status, "早于当日的 active 合同应标记 expired")
	assert.NotNil(t, got.ExpiredAt, "应记录到期标记时间")

	got = loadContract(activeToday.ID)
	assert.Equal(t, models.ContractStatusActive, got.Status, "end_date 等于当日不应标记过期")

	got = loadContract(activeFuture.ID)
	assert.Equal(t, models.ContractStatusActive, got.Status, "未来到期不应标记过期")

	got = loadContract(draftPast.ID)
	assert.Equal(t, models.ContractStatusDraft, got.Status, "草稿不应被扫描")

	got = loadContract(cancelledPast.ID)
	assert.Equal(t, models.ContractStatusCancelled, got.Status, "已作废合同不应被扫描")

	// 员工状态不变
	var empAfter models.Employee
	require.NoError(t, env.db.First(&empAfter, emp.ID).Error)
	assert.Equal(t, "active", empAfter.Status, "到期扫描不应修改员工状态")

	// 幂等：再次运行不改变已标记结果（批量更新天然幂等）
	handler.runContractExpiryOnce(loc, now)
	got = loadContract(activePast.ID)
	assert.Equal(t, models.ContractStatusExpired, got.Status, "重复运行应保持幂等")
}
