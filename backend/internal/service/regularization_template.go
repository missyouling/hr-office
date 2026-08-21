package service

import "path/filepath"

// regularizationTemplateConfig 转正导入 Excel 模板配置。
var regularizationTemplateConfig = templateConfig{
	sheetName: "转正导入",
	headers: []string{
		"员工工号", "身份证号", "计划转正日期", "劳动合同期限（月）", "试用期结束日期", "员工自评", "备注",
	},
	samples: [][]interface{}{
		{"DEV001", "110101199001011234", "2026-09-01", 36, "2026-08-31", "试用期表现良好", "示例数据，导入前请删除"},
		{"DEV002", "110101199002022345", "2026-09-15", 24, "", "", ""},
	},
	widths: []float64{14, 22, 16, 18, 16, 18, 24},
}

// GenerateRegularizationTemplate 生成转正导入模板。
func GenerateRegularizationTemplate(outputPath string) error {
	return generateTemplateFile(regularizationTemplateConfig, outputPath)
}

// GetRegularizationTemplatePath 返回转正导入模板路径。
func GetRegularizationTemplatePath() string {
	return filepath.Join(".", "data", "转正导入模板.xlsx")
}
