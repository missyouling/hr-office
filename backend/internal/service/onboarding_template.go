package service

import (
	"path/filepath"
)

// onboardingTemplateConfig 入职导入 Excel 模板配置（P12.3.2.3）。
// 独立于旧员工导入模板，仅包含入职登记所需最小字段。
var onboardingTemplateConfig = templateConfig{
	sheetName: "入职导入",
	headers: []string{
		"姓名", "身份证号", "联系电话", "部门", "岗位", "计划入职日期", "用工状态", "备注",
	},
	samples: [][]interface{}{
		{"张三", "110101199001011234", "13800138000", "销售部", "销售员", "2026-09-01", "试用", "示例数据，导入前请删除"},
		{"李四", "110101199002022345", "13900139000", "技术部", "工程师", "2026-09-15", "正式", ""},
	},
	widths: []float64{10, 22, 14, 14, 14, 16, 12, 24},
}

// GenerateOnboardingTemplate 生成入职导入 Excel 模板。
func GenerateOnboardingTemplate(outputPath string) error {
	return generateTemplateFile(onboardingTemplateConfig, outputPath)
}

// GetOnboardingTemplatePath 返回入职导入模板路径。
func GetOnboardingTemplatePath() string {
	return filepath.Join(".", "data", "入职导入模板.xlsx")
}
