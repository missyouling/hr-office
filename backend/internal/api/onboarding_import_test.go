package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"siapp/internal/models"
	"siapp/internal/service"
)

// setupOnboardingImportTestEnv 初始化入职导入测试环境（含 WorkTodo/OnboardingImportRun 迁移）。
func setupOnboardingImportTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupOnboardingTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.WorkTodo{}, &models.OnboardingImportRun{}, &SystemLog{}))
	return env
}

// importOnboardingViaAPI 通过 API 手动 JSON 批量导入。
func importOnboardingViaAPI(t *testing.T, env *rbacTestEnv, token string, records []service.OnboardingImportRow) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]interface{}{"records": records}
	return doRBACRequest(t, env.router, http.MethodPost, "/api/onboarding-records/import", token, body)
}

// TestOnboardingImportManualSuccess 手动 JSON 批量导入成功（pending 落库）。
func TestOnboardingImportManualSuccess(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	rows := []service.OnboardingImportRow{
		{Name: "导入A", IDNumber: "110101199001011234", Phone: "13800000001", Department: "销售部", Position: "销售员", PlannedHireDate: "2026-09-01", EmploymentStatus: "试用"},
		{Name: "导入B", IDNumber: "110101199002022345", Department: "技术部", Position: "工程师", PlannedHireDate: "2026-09-15", EmploymentStatus: "formal"},
	}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusCreated, rec.Code, "批量导入应成功: %d %s", rec.Code, rec.Body.String())

	var resp struct {
		Imported int                       `json:"imported"`
		Records  []models.OnboardingRecord `json:"records"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Imported)
	require.Len(t, resp.Records, 2)
	assert.Equal(t, models.OnboardingStatusPending, resp.Records[0].Status)
	assert.Equal(t, models.EmploymentStatusTrial, resp.Records[0].EmploymentStatus, "中文'试用'应归一化为 trial")
	assert.Equal(t, models.EmploymentStatusFormal, resp.Records[1].EmploymentStatus)

	var count int64
	require.NoError(t, env.db.Model(&models.OnboardingRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// TestOnboardingImportValidationRollback 预校验失败整体拒绝，不落库（回滚语义）。
func TestOnboardingImportValidationRollback(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	// 第2行与已有员工冲突 → 整体拒绝，第1行也不落库
	createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011234", "active")
	rows := []service.OnboardingImportRow{
		{Name: "合法行", IDNumber: "110101199009091234", PlannedHireDate: "2026-09-01"},
		{Name: "冲突行", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"},
	}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusConflict, rec.Code, "含冲突行应整体拒绝: %d %s", rec.Code, rec.Body.String())

	var count int64
	require.NoError(t, env.db.Model(&models.OnboardingRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count, "预校验失败不应有任何记录落库")
}

// TestOnboardingImportDuplicateInFile 文件内身份证重复拒绝。
func TestOnboardingImportDuplicateInFile(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	rows := []service.OnboardingImportRow{
		{Name: "重复A", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"},
		{Name: "重复B", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"},
	}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusConflict, rec.Code, "文件内重复应拒绝: %d %s", rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "文件内身份证号与第 2 行重复")

	var count int64
	require.NoError(t, env.db.Model(&models.OnboardingRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count)
}

// TestOnboardingImportEmployeeConflict 身份证命中 employees 全量（active/resigned）统一拒绝。
func TestOnboardingImportEmployeeConflict(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199001011234", "active")
	createTestEmployeeWithID(t, env.db, env.admin.ID, "110101199002022345", "resigned")

	// active 冲突
	rows := []service.OnboardingImportRow{{Name: "冲突", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"}}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusConflict, rec.Code, "与 active 员工冲突应拒绝: %d %s", rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "该身份证号已存在员工记录")

	// resigned 冲突（禁止自动返聘/覆盖/恢复）
	rows[0].IDNumber = "110101199002022345"
	rec = importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusConflict, rec.Code, "与 resigned 员工冲突应拒绝: %d %s", rec.Code, rec.Body.String())
}

// TestOnboardingImportInvalidRows 必填缺失与日期格式错误拒绝。
func TestOnboardingImportInvalidRows(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	rows := []service.OnboardingImportRow{
		{Name: "", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"},                                // 姓名缺失
		{Name: "日期错", IDNumber: "110101199002022345", PlannedHireDate: "2026/09/01"},                             // 日期格式错
		{Name: "状态错", IDNumber: "110101199003033456", PlannedHireDate: "2026-09-01", EmploymentStatus: "intern"}, // 用工状态非法
	}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusConflict, rec.Code, "含非法行应拒绝: %d %s", rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "姓名必填")
	assert.Contains(t, rec.Body.String(), "计划入职日期格式应为 YYYY-MM-DD")
	assert.Contains(t, rec.Body.String(), "用工状态仅支持 trial/formal")

	var count int64
	require.NoError(t, env.db.Model(&models.OnboardingRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count)
}

// TestOnboardingImportExcel 通过 Excel 文件导入成功。
func TestOnboardingImportExcel(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	// 构造 Excel 内容
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "入职导入.xlsx")
	require.NoError(t, err)
	content, err := buildOnboardingTestExcel()
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := newMultipartRequest(t, "/api/onboarding-records/import/excel", token, writer.FormDataContentType(), &buf)
	rec := doRawRequest(t, env.router, req)
	assert.Equal(t, http.StatusCreated, rec.Code, "Excel 导入应成功: %d %s", rec.Code, rec.Body.String())

	var resp struct {
		Imported int `json:"imported"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Imported)

	var count int64
	require.NoError(t, env.db.Model(&models.OnboardingRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// TestOnboardingImportRBAC 无 employee.create 权限调用导入端点必须 403。
func TestOnboardingImportRBAC(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.viewer.SupabaseUID // viewer 无 employee 权限

	rows := []service.OnboardingImportRow{{Name: "无权限", IDNumber: "110101199001011234", PlannedHireDate: "2026-09-01"}}
	rec := importOnboardingViaAPI(t, env, token, rows)
	assert.Equal(t, http.StatusForbidden, rec.Code, "无权限导入应 403: %d %s", rec.Code, rec.Body.String())

	rec = doRBACRequest(t, env.router, http.MethodGet, "/api/onboarding-records/template", token, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code, "无权限下载模板应 403: %d %s", rec.Code, rec.Body.String())
}

// TestOnboardingImportTemplate 模板下载成功且为 xlsx。
func TestOnboardingImportTemplate(t *testing.T) {
	env := setupOnboardingImportTestEnv(t)
	token := env.admin.SupabaseUID

	rec := doRBACRequest(t, env.router, http.MethodGet, "/api/onboarding-records/template", token, nil)
	assert.Equal(t, http.StatusOK, rec.Code, "模板下载应成功: %d %s", rec.Code, rec.Body.String())
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", rec.Header().Get("Content-Type"))
	assert.NotEmpty(t, rec.Body.Bytes(), "模板内容不应为空")
}

// TestNormalizeEmploymentStatus 用工状态中英文归一化。
func TestNormalizeEmploymentStatus(t *testing.T) {
	cases := map[string]string{
		"trial":  models.EmploymentStatusTrial,
		"试用":     models.EmploymentStatusTrial,
		"试用期":    models.EmploymentStatusTrial,
		"formal": models.EmploymentStatusFormal,
		"正式":     models.EmploymentStatusFormal,
		"正式员工":   models.EmploymentStatusFormal,
		"":       "",
	}
	for input, want := range cases {
		assert.Equal(t, want, service.NormalizeEmploymentStatus(input), "归一化 %q", input)
	}
	// 非法值原样返回
	assert.Equal(t, "intern", service.NormalizeEmploymentStatus("intern"))
}

// buildOnboardingTestExcel 构造入职导入测试 Excel（表头 + 2 行数据）。
func buildOnboardingTestExcel() ([]byte, error) {
	f := newTestExcelFile()
	defer f.Close()
	sheet := "入职导入"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"姓名", "身份证号", "联系电话", "部门", "岗位", "计划入职日期", "用工状态", "备注"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	rows := [][]interface{}{
		{"ExcelA", "110101199001011234", "13800000001", "销售部", "销售员", "2026-09-01", "试用", ""},
		{"ExcelB", "110101199002022345", "", "技术部", "工程师", "2026-09-15", "正式", "备注"},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return nil, err
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// newTestExcelFile 创建内存 Excel 文件。
func newTestExcelFile() *excelize.File {
	return excelize.NewFile()
}

// newMultipartRequest 构造 multipart 请求。
func newMultipartRequest(t *testing.T, path, token, contentType string, body *bytes.Buffer) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, path, body)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", contentType)
	return req
}

// doRawRequest 执行原始请求。
func doRawRequest(t *testing.T, router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
