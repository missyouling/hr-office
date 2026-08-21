package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"siapp/internal/models"
	"siapp/internal/service/storage"
)

// ---------- 测试辅助 ----------

// grantInsurancePerm 为指定角色补充 insurance.<action> 权限。
// 测试种子（seedRBACForTest）默认不含 insurance 模块权限，此处按需补齐。
func grantInsurancePerm(t *testing.T, db *gorm.DB, roleID uint, action string) {
	t.Helper()
	var perm models.Permission
	err := db.Where("module = ? AND action = ?", "insurance", action).First(&perm).Error
	if err == gorm.ErrRecordNotFound {
		perm = models.Permission{Module: "insurance", Action: action, Label: "社保管理", SortOrder: 60}
		require.NoError(t, db.Create(&perm).Error)
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, db.Create(&models.RolePermission{RoleID: roleID, PermissionID: perm.ID}).Error)
}

// grantAllInsurancePerms 为指定角色授予 insurance 模块全部操作权限
func grantAllInsurancePerms(t *testing.T, db *gorm.DB, roleID uint) {
	t.Helper()
	for _, action := range []string{"view", "create", "edit", "delete"} {
		grantInsurancePerm(t, db, roleID, action)
	}
}

// migrateInsuranceRBACTables 迁移社保 RBAC 测试所需的业务表
func migrateInsuranceRBACTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&models.SocialInsuranceRecord{},
		&models.SocialInsuranceBatch{},
		&models.CallbackRecord{},
		&models.CallbackUpload{},
	))
}

// setupInsuranceStorage 配置本地存储（供 import 上传路径使用），通过 t.Cleanup 还原 GlobalManager
func setupInsuranceStorage(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&models.StorageConfig{}, &models.StorageModuleConfig{}, &models.StorageRule{}, &models.SysFile{},
	))
	dir := t.TempDir()
	config := models.StorageConfig{
		Name:      "保险测试本地存储",
		Type:      "local",
		Enabled:   true,
		IsDefault: true,
		Config:    datatypes.JSON([]byte(`{"root_path":"` + dir + `"}`)),
	}
	require.NoError(t, db.Create(&config).Error)
	previous := storage.GlobalManager
	storage.GlobalManager = storage.NewStorageManager(db, storage.DefaultRegistry)
	t.Cleanup(func() { storage.GlobalManager = previous })
}

// insuranceIncreasePayload 构造合法的 increase 变更记录请求体
func insuranceIncreasePayload(employeeName, identityNumber string) map[string]interface{} {
	return map[string]interface{}{
		"change_type":     "increase",
		"employee_name":   employeeName,
		"identity_number": identityNumber,
		"effective_date":  "2026-01-01",
		"template_values": map[string]string{
			"personalIdentity": identityNumber,
			"householdType":    "城镇",
			"education":        "本科",
			"pensionStartDate": "2026-01-01",
			"baseSalary":       "5000",
		},
	}
}

// buildCallbackWorkbook 构造包含回盘数据的 xlsx 字节
func buildCallbackWorkbook(t *testing.T, name, identity, personal string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	require.NoError(t, f.SetCellValue(sheet, "A1", "姓名"))
	require.NoError(t, f.SetCellValue(sheet, "B1", "证件号码"))
	require.NoError(t, f.SetCellValue(sheet, "C1", "个人编号"))
	require.NoError(t, f.SetCellValue(sheet, "A2", name))
	require.NoError(t, f.SetCellValue(sheet, "B2", identity))
	require.NoError(t, f.SetCellValue(sheet, "C2", personal))
	data, err := f.WriteToBuffer()
	require.NoError(t, err)
	return data.Bytes()
}

// doInsuranceMultipartRequest 以 multipart/form-data 发送社保上传/导入请求
func doInsuranceMultipartRequest(t *testing.T, router http.Handler, method, path, token, fileField, fileName string, fileData []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for key, value := range fields {
		require.NoError(t, mw.WriteField(key, value))
	}
	if fileField != "" {
		part, err := mw.CreateFormFile(fileField, fileName)
		require.NoError(t, err)
		_, err = part.Write(fileData)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// ---------- 测试用例 ----------

// TestInsuranceRBACAdminAllowed 管理员拥有 insurance 全权限时，各正常路径均成功
func TestInsuranceRBACAdminAllowed(t *testing.T) {
	env := setupRBACTestEnv(t)
	migrateInsuranceRBACTables(t, env.db)
	grantAllInsurancePerms(t, env.db, env.roleIDs["admin"])
	token := env.admin.SupabaseUID

	// 读操作：选项 + 空列表
	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/social-insurance/options", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/social-insurance/changes", token, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 创建变更记录
	payload := insuranceIncreasePayload("张三", "110101199001011234")
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/social-insurance/changes", token, payload)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created socialInsuranceRecordResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "张三", created.EmployeeName)

	// 列表可见（user_id 归属当前用户）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/social-insurance/changes", token, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listResp struct {
		Records []socialInsuranceRecordResponse `json:"records"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Records, 1)
	assert.Equal(t, created.ID, listResp.Records[0].ID)

	// 更新变更记录
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/social-insurance/changes/%d", created.ID), token, payload)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// 删除变更记录
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/social-insurance/changes/delete", token, map[string]interface{}{"ids": []uint{created.ID}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var delResp struct {
		Deleted int64 `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &delResp))
	assert.Equal(t, int64(1), delResp.Deleted)

	// 模板下载：权限通过（模板文件缺失时业务层返回 404，但绝不因权限返回 403）
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/insurance-template?type=increase", token, nil)
	assert.NotEqual(t, http.StatusForbidden, rec.Code, "模板下载不应因权限被拒: %s", rec.Body.String())

	// callback：上传 -> 列表 -> 清空
	xlsx := buildCallbackWorkbook(t, "李四", "110101199001011235", "P88888")
	rec = doInsuranceMultipartRequest(t, env.router, http.MethodPost, "/api/callback-records/upload", token, "file", "回盘.xlsx", xlsx, nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/callback-records", token, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var cbResp callbackRecordsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cbResp))
	require.Len(t, cbResp.Records, 1)
	assert.Equal(t, "110101199001011235", cbResp.Records[0].IdentityNumber)
	rec = doRBACRequest(t, env.router, http.MethodDelete, "/api/callback-records", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// import：完整正常路径（需本地存储配置）
	setupInsuranceStorage(t, env.db)
	importPayload := map[string]interface{}{
		"records": []map[string]interface{}{
			{
				"employee_name":   "王五",
				"department":      "测试部",
				"identity_number": "110101199001011236",
				"personal_number": "P99999",
				"effective_date":  "2026-02-01",
				"template_values": map[string]string{
					"personalIdentity": "110101199001011236",
					"householdType":    "城镇",
					"education":        "本科",
					"pensionStartDate": "2026-02-01",
					"baseSalary":       "6000",
				},
			},
		},
	}
	payloadJSON, err := json.Marshal(importPayload)
	require.NoError(t, err)
	importXLSX := buildCallbackWorkbook(t, "王五", "110101199001011236", "P99999")
	rec = doInsuranceMultipartRequest(t, env.router, http.MethodPost, "/api/social-insurance/changes/import", token,
		"file", "import.xls", importXLSX,
		map[string]string{"change_type": "increase", "payload": string(payloadJSON)})
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

// TestInsuranceRBACViewerForbidden 无 insurance 权限的 viewer 调用全部保险路由一律 403
func TestInsuranceRBACViewerForbidden(t *testing.T) {
	env := setupRBACTestEnv(t)
	migrateInsuranceRBACTables(t, env.db)
	token := env.viewer.SupabaseUID // viewer 无任何 insurance 权限

	payload := insuranceIncreasePayload("张三", "110101199001011234")
	importPayloadJSON, err := json.Marshal(map[string]interface{}{
		"records": []map[string]interface{}{
			{"employee_name": "王五", "identity_number": "110101199001011236"},
		},
	})
	require.NoError(t, err)
	xlsx := buildCallbackWorkbook(t, "李四", "110101199001011235", "P88888")

	cases := []struct {
		name      string
		method    string
		path      string
		multipart bool
		fields    map[string]string
		fileData  []byte
		body      interface{}
	}{
		{"社保变更列表", http.MethodGet, "/api/social-insurance/changes", false, nil, nil, nil},
		{"社保选项", http.MethodGet, "/api/social-insurance/options", false, nil, nil, nil},
		{"社保变更创建", http.MethodPost, "/api/social-insurance/changes", false, nil, nil, payload},
		{"社保变更更新", http.MethodPut, "/api/social-insurance/changes/1", false, nil, nil, payload},
		{"社保变更导入", http.MethodPost, "/api/social-insurance/changes/import", true,
			map[string]string{"change_type": "increase", "payload": string(importPayloadJSON)}, xlsx, nil},
		{"社保变更删除", http.MethodPost, "/api/social-insurance/changes/delete", false, nil, nil,
			map[string]interface{}{"ids": []uint{1}}},
		{"回盘列表", http.MethodGet, "/api/callback-records", false, nil, nil, nil},
		{"回盘上传", http.MethodPost, "/api/callback-records/upload", true, nil, xlsx, nil},
		{"回盘清空", http.MethodDelete, "/api/callback-records", false, nil, nil, nil},
		{"保险模板下载", http.MethodGet, "/api/insurance-template?type=increase", false, nil, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tc.multipart {
				rec = doInsuranceMultipartRequest(t, env.router, tc.method, tc.path, token, "file", "upload.xls", tc.fileData, tc.fields)
			} else {
				rec = doRBACRequest(t, env.router, tc.method, tc.path, token, tc.body)
			}
			assert.Equal(t, http.StatusForbidden, rec.Code, "viewer 应 403: %s -> %d %s", tc.path, rec.Code, rec.Body.String())
		})
	}
}

// TestInsuranceRBACCrossTenantIsolation 跨租户隔离：有权限的用户也看不到/动不了他人（user_id 隔离）的数据
func TestInsuranceRBACCrossTenantIsolation(t *testing.T) {
	env := setupRBACTestEnv(t)
	migrateInsuranceRBACTables(t, env.db)
	grantAllInsurancePerms(t, env.db, env.roleIDs["admin"])
	// manager 授予读/改/删权限，用于验证跨租户隔离（有权限但数据归属不同用户）
	for _, action := range []string{"view", "edit", "delete"} {
		grantInsurancePerm(t, env.db, env.roleIDs["manager"], action)
	}

	// admin 创建一条变更记录
	rec := doRBACRequest(t, env.router, http.MethodPost, "/api/social-insurance/changes",
		env.admin.SupabaseUID, insuranceIncreasePayload("赵六", "110101199001011237"))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created socialInsuranceRecordResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// manager 有 insurance.view，但列表看不到 admin 的记录
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/social-insurance/changes", env.manager.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listResp struct {
		Records []socialInsuranceRecordResponse `json:"records"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	assert.Empty(t, listResp.Records, "manager 不应看到 admin 的社保记录")

	// manager 有 insurance.edit，但更新 admin 的记录 -> 404（记录归属校验）
	rec = doRBACRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/social-insurance/changes/%d", created.ID),
		env.manager.SupabaseUID, insuranceIncreasePayload("赵六", "110101199001011237"))
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	// manager 有 insurance.delete，但删除 admin 的记录 -> deleted=0
	rec = doRBACRequest(t, env.router, http.MethodPost, "/api/social-insurance/changes/delete",
		env.manager.SupabaseUID, map[string]interface{}{"ids": []uint{created.ID}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var delResp struct {
		Deleted int64 `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &delResp))
	assert.Zero(t, delResp.Deleted, "manager 不应删除 admin 的记录")

	// admin 的记录仍存在
	var count int64
	require.NoError(t, env.db.Model(&models.SocialInsuranceRecord{}).Where("id = ?", created.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// callback：admin 直接入库一条回盘记录，manager 列表不可见（user_id 隔离）
	cbUserID := env.admin.ID
	cbUpload := models.CallbackUpload{
		UserID:       &cbUserID,
		FileName:     "隔离测试.xlsx",
		FileSize:     1024,
		TotalRecords: 1,
	}
	require.NoError(t, env.db.Create(&cbUpload).Error)
	require.NoError(t, env.db.Create(&models.CallbackRecord{
		UploadID:       cbUpload.ID,
		UserID:         &cbUserID,
		PersonalNumber: "P77777",
		IdentityNumber: "110101199001011238",
		Name:           "孙七",
		Sequence:       1,
	}).Error)
	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/callback-records", env.manager.SupabaseUID, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var cbResp callbackRecordsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cbResp))
	assert.Empty(t, cbResp.Records, "manager 不应看到 admin 的回盘记录")
	assert.Empty(t, cbResp.PersonalMap)
}
