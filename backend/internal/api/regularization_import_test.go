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

func setupRegularizationImportTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	env := setupRegularizationWriteTestEnv(t)
	require.NoError(t, env.db.AutoMigrate(&models.RegularizationEffectRun{}))
	return env
}

func buildRegularizationExcel(rows [][]interface{}) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "转正导入"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"员工工号", "身份证号", "计划转正日期", "劳动合同期限（月）", "试用期结束日期", "员工自评", "备注"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
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

func doRegularizationImportRequest(t *testing.T, env *rbacTestEnv, token string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "转正导入.xlsx")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/regularization-records/import", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return doRawRequest(t, env.router, req)
}

func TestRegularizationTemplateDownload(t *testing.T) {
	env := setupRegularizationImportTestEnv(t)
	resp := doRBACRequest(t, env.router, http.MethodGet, "/api/regularization-records/template", env.admin.SupabaseUID, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
	f, err := excelize.OpenReader(bytes.NewReader(resp.Body.Bytes()))
	require.NoError(t, err)
	defer f.Close()
	rows, err := f.GetRows("转正导入")
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, []string{"员工工号", "身份证号", "计划转正日期", "劳动合同期限（月）", "试用期结束日期", "员工自评", "备注"}, rows[0])
}

func TestRegularizationImportSuccessAndWarnings(t *testing.T) {
	env := setupRegularizationImportTestEnv(t)
	fixRegularizationNow(t, 2026, 8, 18)
	emp1 := createRegularizationTestEmployee(t, env, "110101199001011234", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	emp2 := createRegularizationTestEmployee(t, env, "110101199002022345", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	require.NoError(t, env.db.Model(&models.Employee{}).Where("id = ?", emp1.ID).Updates(map[string]any{"hire_date": "2026-01-01", "probation_end_date": "2026-03-15"}).Error)
	require.NoError(t, env.db.Model(&models.Employee{}).Where("id = ?", emp2.ID).Updates(map[string]any{"hire_date": "2026-08-01", "probation_end_date": "2026-09-01"}).Error)

	content, err := buildRegularizationExcel([][]interface{}{
		{emp1.EmployeeID, emp1.IDNumber, "2026-08-18", 12, "2026-03-15", "自评A", "备注A"},
		{emp2.EmployeeID, emp2.IDNumber, "2026-08-20", 24, "2026-09-01", "自评B", "备注B"},
	})
	require.NoError(t, err)
	resp := doRegularizationImportRequest(t, env, env.admin.SupabaseUID, content)
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	var result struct {
		Imported int                                   `json:"imported"`
		Warnings []service.RegularizationImportWarning `json:"warnings"`
		Records  []models.RegularizationRecord         `json:"records"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))
	assert.Equal(t, 2, result.Imported)
	assert.NotEmpty(t, result.Warnings)
	require.Len(t, result.Records, 2)
	assert.Equal(t, models.RegularizationStatusEffective, result.Records[0].Status)
	assert.Equal(t, models.RegularizationStatusScheduled, result.Records[1].Status)
	assert.Equal(t, models.RegularizationSourceExcelDirect, result.Records[0].Source)
	assert.Nil(t, result.Records[0].SupervisorApproverUserID)
	assert.Nil(t, result.Records[0].HRReviewerUserID)

	var count int64
	require.NoError(t, env.db.Model(&models.RegularizationRecord{}).Where("user_id = ? AND source = ?", env.admin.ID, models.RegularizationSourceExcelDirect).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestRegularizationImportAtomicFailure(t *testing.T) {
	env := setupRegularizationImportTestEnv(t)
	createRegularizationTestEmployee(t, env, "110101199001011234", "研发部", models.EmployeeStatusActive, models.EmploymentStatusTrial)
	content, err := buildRegularizationExcel([][]interface{}{
		{"DEV001", "110101199001011234", "2026-08-18", 12, "", "", ""},
		{"DEV001", "110101199001011234", "2026-08-18", 12, "", "", ""},
	})
	require.NoError(t, err)
	resp := doRegularizationImportRequest(t, env, env.admin.SupabaseUID, content)
	assert.Equal(t, http.StatusConflict, resp.Code)
	var count int64
	require.NoError(t, env.db.Model(&models.RegularizationRecord{}).Where("user_id = ?", env.admin.ID).Count(&count).Error)
	assert.Zero(t, count)
}
