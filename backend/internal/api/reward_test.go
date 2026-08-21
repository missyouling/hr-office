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

// grantRewardPermission 为指定角色补充 reward 模块指定动作权限。
func grantRewardPermission(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "reward", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "reward", Action: action, Label: action, SortOrder: 98}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// setupRewardTestEnv 初始化奖惩记录测试环境（admin 具备 reward 全权限 + 员工/档案文档表）。
func setupRewardTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRBACTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.RewardRecord{}, &models.Document{}, &models.Employee{}))
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantRewardPermission(t, env.db, env.roleIDs["admin"], action)
	}
	return env
}

// createTestRewardEmployee 创建测试员工（供奖惩记录关联与快照）。
func createTestRewardEmployee(t *testing.T, db *gorm.DB, userID uint, name, department, position string) models.Employee {
	t.Helper()
	emp := models.Employee{
		UserID:     userID,
		Name:       name,
		Department: department,
		Position:   position,
		Status:     models.EmployeeStatusActive,
	}
	require.NoError(t, db.Create(&emp).Error)
	return emp
}

// createRewardViaAPI 通过 API 创建奖惩记录草稿。
func createRewardViaAPI(t *testing.T, env *rbacTestEnv, token string, body map[string]interface{}) models.RewardRecord {
	t.Helper()
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", token, body)
	require.Equal(t, http.StatusCreated, rec.Code, "创建奖惩记录应成功: %d %s", rec.Code, rec.Body.String())
	var record models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &record))
	return record
}

// TestRewardRBACForbidden 无 reward 权限调用全部端点必须 403。
func TestRewardRBACForbidden(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.viewer.SupabaseUID // viewer 无 reward 权限
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "张三", "业务部", "销售")
	record := createRewardViaAPI(t, env, env.admin.SupabaseUID, map[string]interface{}{
		"employee_id":   emp.ID,
		"record_type":   models.RewardTypeReward,
		"occurred_date": "2026-08-01",
		"reason":        "季度优秀员工",
		"level":         "嘉奖",
	})

	cases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/rewards", nil},
		{http.MethodGet, fmt.Sprintf("/api/rewards/%d", record.ID), nil},
		{http.MethodPost, "/api/rewards", map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l"}},
		{http.MethodPut, fmt.Sprintf("/api/rewards/%d", record.ID), map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-02", "reason": "r", "level": "l"}},
		{http.MethodDelete, fmt.Sprintf("/api/rewards/%d", record.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", record.ID), nil},
		{http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", record.ID), map[string]interface{}{"reason": "作废"}},
	}
	for _, c := range cases {
		rec := doRBACRequest(t, env.router, c.method, c.path, token, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s 应 403: %d %s", c.method, c.path, rec.Code, rec.Body.String())
	}
}

// TestRewardLifecycle 完整生命周期：创建草稿 → 列表/详情 → 编辑 → 手动生效 → 生效后不可编辑/删除 → 作废。
func TestRewardLifecycle(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "李四", "总经办", "助理")

	// 创建草稿（含选填字段）
	score := 5.0
	amount := 500.0
	record := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id":   emp.ID,
		"record_type":   models.RewardTypeReward,
		"occurred_date": "2026-08-01",
		"reason":        "季度优秀员工",
		"level":         "嘉奖",
		"score":         score,
		"amount":        amount,
		"owner":         "王五",
		"remarks":       "已公示",
	})
	assert.Equal(t, models.RewardStatusDraft, record.Status)
	assert.Equal(t, "李四", record.SnapshotName)
	assert.Equal(t, "总经办", record.SnapshotDepartment)
	assert.Equal(t, "助理", record.SnapshotPosition)
	assert.NotNil(t, record.Score)
	assert.Equal(t, score, *record.Score)

	// 列表可见
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/rewards", token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed, 1)

	// 详情可见
	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/rewards/%d", record.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// 编辑草稿
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rewards/%d", record.ID), token, map[string]interface{}{
		"employee_id":   emp.ID,
		"record_type":   models.RewardTypeReward,
		"occurred_date": "2026-08-02",
		"reason":        "季度优秀员工（补录）",
		"level":         "记功",
	})
	require.Equal(t, http.StatusOK, rec.Code, "编辑草稿应成功: %s", rec.Body.String())
	var edited models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &edited))
	assert.Equal(t, "2026-08-02", edited.OccurredDate)
	assert.Equal(t, "记功", edited.Level)
	assert.Equal(t, models.RewardStatusDraft, edited.Status)

	// 手动生效（draft → effective）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", record.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code, "生效应成功: %s", rec.Body.String())
	var activated models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &activated))
	assert.Equal(t, models.RewardStatusEffective, activated.Status)
	assert.NotNil(t, activated.EffectiveAt)

	// effective 不可编辑/删除
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rewards/%d", record.ID), token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-03", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, "生效记录不可编辑")
	rec = doRBACRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/rewards/%d", record.ID), token, nil)
	assert.Equal(t, http.StatusConflict, rec.Code, "生效记录不可删除")

	// 生效记录可作废（原因必填）
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", record.ID), token, map[string]interface{}{"reason": "  "})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "作废原因必填")
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", record.ID), token, map[string]interface{}{"reason": "记录有误"})
	require.Equal(t, http.StatusOK, rec.Code, "作废应成功: %s", rec.Body.String())
	var voided models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &voided))
	assert.Equal(t, models.RewardStatusVoided, voided.Status)
	assert.Equal(t, "记录有误", voided.VoidReason)
	assert.NotNil(t, voided.VoidedAt)

	// voided 为终态不可再次作废
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", record.ID), token, map[string]interface{}{"reason": "再次作废"})
	assert.Equal(t, http.StatusConflict, rec.Code, "已作废记录不可再次作废")
}

// TestRewardVoidDraftAndEffective draft/effective 均可作废，作废后可新建。
func TestRewardVoidDraftAndEffective(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "赵六", "业务部", "销售")

	// 草稿作废
	draft := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypePunishment, "occurred_date": "2026-08-01", "reason": "迟到", "level": "警告",
	})
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", draft.ID), token, map[string]interface{}{"reason": "误录"})
	require.Equal(t, http.StatusOK, rec.Code, "草稿作废应成功: %s", rec.Body.String())
	var cancelledDraft models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelledDraft))
	assert.Equal(t, models.RewardStatusVoided, cancelledDraft.Status)
	assert.Equal(t, "误录", cancelledDraft.VoidReason)

	// effective 作废
	active := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-02", "reason": "突出贡献", "level": "记功",
	})
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", active.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", active.ID), token, map[string]interface{}{"reason": "复核不符"})
	require.Equal(t, http.StatusOK, rec.Code, "生效记录作废应成功: %s", rec.Body.String())
	var cancelledActive models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cancelledActive))
	assert.Equal(t, models.RewardStatusVoided, cancelledActive.Status)
	assert.Equal(t, "复核不符", cancelledActive.VoidReason)

	// 作废后新建成功
	replacement := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-03", "reason": "重新申报", "level": "嘉奖",
	})
	assert.Equal(t, models.RewardStatusDraft, replacement.Status)
}

// TestRewardVoidRequiresDeletePermission 作废权限必须为 reward.delete（而非 edit）：
// 持有 delete 但无 edit 的角色可作废；持有 edit 但无 delete 的角色不可作废。
func TestRewardVoidRequiresDeletePermission(t *testing.T) {
	env := setupRewardTestEnv(t)
	adminEmp := createTestRewardEmployee(t, env.db, env.admin.ID, "钱七", "总经办", "专员")

	// 角色 A：view+create+delete（无 edit）
	deleteOnlyRole := models.Role{Name: "reward_delete_only", Label: "仅作废", IsSystem: false}
	require.NoError(t, env.db.Create(&deleteOnlyRole).Error)
	for _, action := range []string{"view", "create", "delete"} {
		grantRewardPermission(t, env.db, deleteOnlyRole.ID, action)
	}
	// 角色 B：view+create+edit（无 delete）
	editOnlyRole := models.Role{Name: "reward_edit_only", Label: "仅编辑", IsSystem: false}
	require.NoError(t, env.db.Create(&editOnlyRole).Error)
	for _, action := range []string{"view", "create", "edit"} {
		grantRewardPermission(t, env.db, editOnlyRole.ID, action)
	}

	deleteOnlyUser := createRBACTestUser(t, env.db, "rwDeleteOnly", "rw-delete-only-uuid", deleteOnlyRole.ID)
	editOnlyUser := createRBACTestUser(t, env.db, "rwEditOnly", "rw-edit-only-uuid", editOnlyRole.ID)
	// 各角色使用自己租户的员工（员工租户隔离）
	deleteOnlyEmp := createTestRewardEmployee(t, env.db, deleteOnlyUser.ID, "钱七", "总经办", "专员")
	editOnlyEmp := createTestRewardEmployee(t, env.db, editOnlyUser.ID, "钱七", "总经办", "专员")

	// delete-only：可创建草稿（create）
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"employee_id": deleteOnlyEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "delete-only 创建应成功: %d %s", rec.Code, rec.Body.String())
	var deleteOnlyRecord models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleteOnlyRecord))

	// delete-only：有 delete 无 edit，作废成功
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", deleteOnlyRecord.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废"})
	require.Equal(t, http.StatusOK, rec.Code, "delete-only 作废应成功: %d %s", rec.Code, rec.Body.String())

	// delete-only：无 edit，编辑/生效应 403（证明作废走的是 delete 而非 edit）
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rewards/%d", deleteOnlyRecord.ID), deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"employee_id": deleteOnlyEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-02", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 编辑应 403: %d %s", rec.Code, rec.Body.String())
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", deleteOnlyRecord.ID), deleteOnlyUser.SupabaseUID, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "delete-only 无 edit 生效应 403: %d %s", rec.Code, rec.Body.String())

	// edit-only：可创建并生效（有 edit）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", editOnlyUser.SupabaseUID, map[string]interface{}{
		"employee_id": editOnlyEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	require.Equal(t, http.StatusCreated, rec.Code, "edit-only 创建应成功: %d %s", rec.Code, rec.Body.String())
	var editOnlyRecord models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &editOnlyRecord))
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", editOnlyRecord.ID), editOnlyUser.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code, "edit-only 生效应成功: %d %s", rec.Code, rec.Body.String())

	// edit-only：有 edit 无 delete，作废应 403
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", editOnlyRecord.ID), editOnlyUser.SupabaseUID, map[string]interface{}{"reason": "作废"})
	assert.Equal(t, http.StatusForbidden, rec.Code, "edit-only 无 delete 作废应 403: %d %s", rec.Code, rec.Body.String())

	// 跨租户关联员工应被拒绝（delete-only 不能关联 admin 租户员工）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", deleteOnlyUser.SupabaseUID, map[string]interface{}{
		"employee_id": adminEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "跨租户关联员工应 400: %s", rec.Body.String())
}

// TestRewardDocumentLink 复用档案文档关联字段：关联文档校验租户归属。
func TestRewardDocumentLink(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "孙八", "业务部", "销售")

	doc := createTestAdminDocument(t, env.db, env.admin.ID, "RW-DOC-001")

	// 关联本租户文档成功
	record := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
		"document_id": doc.ID,
	})
	assert.Equal(t, doc.ID, *record.DocumentID)

	// 关联不存在文档失败
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
		"document_id": 99999,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "关联不存在文档应失败: %s", rec.Body.String())
}

// TestRewardSnapshotAndEmployeeValidation 员工快照拷贝 + 员工租户校验：
// 关联员工必须属于当前租户，快照从员工主表拷贝。
func TestRewardSnapshotAndEmployeeValidation(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "周九", "总经办", "经理")

	// 快照正确拷贝
	record := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypePunishment, "occurred_date": "2026-08-01", "reason": "违规", "level": "记过",
	})
	assert.Equal(t, "周九", record.SnapshotName)
	assert.Equal(t, "总经办", record.SnapshotDepartment)
	assert.Equal(t, "经理", record.SnapshotPosition)

	// 关联不存在员工失败
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", token, map[string]interface{}{
		"employee_id": 99999, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "关联不存在员工应失败: %s", rec.Body.String())

	// 必填字段缺失逐项拒绝
	for _, tc := range []struct {
		name string
		body map[string]interface{}
	}{
		{"员工缺失", map[string]interface{}{"record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l"}},
		{"类型缺失", map[string]interface{}{"employee_id": emp.ID, "occurred_date": "2026-08-01", "reason": "r", "level": "l"}},
		{"类型非法", map[string]interface{}{"employee_id": emp.ID, "record_type": "bonus", "occurred_date": "2026-08-01", "reason": "r", "level": "l"}},
		{"日期缺失", map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "reason": "r", "level": "l"}},
		{"日期格式错误", map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026/08/01", "reason": "r", "level": "l"}},
		{"事由缺失", map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "level": "l"}},
		{"等级缺失", map[string]interface{}{"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", token, tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "%s 应 400: %d %s", tc.name, rec.Code, rec.Body.String())
		})
	}
}

// TestRewardTenantIsolation 租户隔离：数据按 user_id 隔离，跨用户互不可见。
func TestRewardTenantIsolation(t *testing.T) {
	env := setupRewardTestEnv(t)

	// 为 viewer 补充 view+create+edit 权限（edit 确保请求能通过权限中间件，验证数据层隔离而非权限拦截）
	for _, action := range []string{"view", "create", "edit"} {
		grantRewardPermission(t, env.db, env.roleIDs["viewer"], action)
	}
	other := createRBACTestUser(t, env.db, "rwOther", "rw-other-uuid", env.roleIDs["viewer"])
	otherEmp := createTestRewardEmployee(t, env.db, other.ID, "吴十", "业务部", "销售")

	// admin 创建记录
	adminEmp := createTestRewardEmployee(t, env.db, env.admin.ID, "郑一", "总经办", "经理")
	record := createRewardViaAPI(t, env, env.admin.SupabaseUID, map[string]interface{}{
		"employee_id": adminEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})

	// 其他用户列表不可见
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/rewards", other.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []models.RewardRecord
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Len(t, listed, 0, "其他用户不应看到管理员创建的奖惩记录")

	// 其他用户详情不可见
	rec = doRBACRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/rewards/%d", record.ID), other.SupabaseUID, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "其他用户详情应 404")

	// 其他用户不能修改/作废管理员的记录
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/rewards/%d", record.ID), other.SupabaseUID, map[string]interface{}{
		"employee_id": otherEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-02", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusNotFound, rec.Code, "其他用户编辑应 404")

	// 其他用户不能关联管理员租户的员工（员工租户校验）
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/rewards", other.SupabaseUID, map[string]interface{}{
		"employee_id": adminEmp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "跨租户关联员工应 400: %s", rec.Body.String())
}

// TestRewardDoesNotChangeEmployee 生效/作废不改变员工状态或薪资（仅台账记录）。
func TestRewardDoesNotChangeEmployee(t *testing.T) {
	env := setupRewardTestEnv(t)
	token := env.admin.SupabaseUID
	emp := createTestRewardEmployee(t, env.db, env.admin.ID, "冯二", "业务部", "销售")

	record := createRewardViaAPI(t, env, token, map[string]interface{}{
		"employee_id": emp.ID, "record_type": models.RewardTypeReward, "occurred_date": "2026-08-01", "reason": "r", "level": "l",
	})
	// 生效
	rec := doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/activate", record.ID), token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	// 作废
	rec = doRBACRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/rewards/%d/void", record.ID), token, map[string]interface{}{"reason": "作废"})
	require.Equal(t, http.StatusOK, rec.Code)

	// 员工状态与就业状态保持不变
	var reloaded models.Employee
	require.NoError(t, env.db.First(&reloaded, emp.ID).Error)
	assert.Equal(t, models.EmployeeStatusActive, reloaded.Status)
	assert.Equal(t, models.EmploymentStatusFormal, reloaded.EmploymentStatus)
}
