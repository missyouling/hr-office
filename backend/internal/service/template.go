package service

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

type templateConfig struct {
	sheetName string
	headers   []string
	samples   [][]interface{}
	widths    []float64
}

var rosterTemplateConfig = templateConfig{
	sheetName: "花名册",
	headers: []string{
		"工号", "姓名", "部门", "岗位", "性别", "入职时间", "年龄", "工龄",
		"出生月份", "文化程度", "政治面貌", "工作服", "劳保鞋", "户口性质",
		"民族", "籍贯", "身份证地址", "身份证号码", "婚姻状况", "社保",
		"是否生育", "联系电话", "紧急联系人", "家庭电话/紧急情况联系电话",
		"现居住地址", "毕业院校", "专业", "毕业时间",
	},
	samples: [][]interface{}{
		{"2001", "张三", "销售部", "销售员", "男", "2020-01-01", "25", "3", "1月", "大专", "群众", "L", "42", "城镇", "汉族", "四川", "四川省成都市", "110101199001011234", "未婚", "有", "否", "13800138000", "张父", "028-12345678", "四川省成都市高新区", "四川大学", "市场营销", "2019"},
		{"2002", "李四", "技术部", "工程师", "女", "2019-06-15", "28", "4", "6月", "本科", "团员", "M", "38", "农村", "汉族", "重庆", "重庆市沙坪坝区", "110101199002022345", "已婚", "有", "是", "13900139000", "李母", "023-87654321", "重庆市沙坪坝区大学城", "重庆大学", "计算机科学", "2018"},
		{"2003", "王五", "财务部", "会计", "男", "2021-03-10", "30", "2", "3月", "本科", "党员", "XL", "43", "城镇", "汉族", "广东", "广东省深圳市", "110101199003033456", "已婚", "有", "否", "13700137000", "王妻", "0755-12345678", "广东省深圳市南山区", "华南理工大学", "会计学", "2015"},
	},
	widths: []float64{
		8, 10, 12, 12, 6, 12, 6, 6,
		8, 10, 10, 8, 8, 8,
		8, 10, 25, 20, 8, 6,
		8, 12, 10, 20,
		25, 15, 12, 10,
	},
}

var employeeTemplateConfig = templateConfig{
	sheetName: "员工导入",
	headers: []string{
		"工号", "姓名", "部门", "岗位", "性别", "入职时间", "年龄", "工龄",
		"出生月份", "文化程度", "政治面貌", "工作服", "劳保鞋", "户口性质",
		"民族", "籍贯", "身份证地址", "身份证号码", "婚姻状况", "社保",
		"是否生育", "联系电话", "紧急联系人", "家庭电话/紧急情况联系电话",
		"现居住地址", "毕业院校", "专业", "毕业时间", "社保编号", "公积金编号", "邮箱", "备注", "状态", "离职日期",
	},
	samples: [][]interface{}{
		{"2001", "张三", "销售部", "销售员", "男", "2020-01-01", "25", "3", "01月", "大专", "群众", "L", "42", "城镇", "汉族", "四川", "四川省成都市", "110101199001011234", "未婚", "有", "否", "13800138000", "张父", "028-12345678", "四川省成都市高新区", "四川大学", "市场营销", "2019", "SI00001", "无", "zhangsan@example.com", "试岗员工", "active", ""},
		{"2002", "李四", "技术部", "工程师", "女", "2019-06-15", "28", "4", "06月", "本科", "团员", "M", "38", "农村", "汉族", "重庆", "重庆市沙坪坝区", "110101199002022345", "已婚", "有", "是", "13900139000", "李母", "023-87654321", "重庆市沙坪坝区大学城", "重庆大学", "计算机科学", "2018", "SI00002", "无", "lisi@example.com", "核心骨干", "active", ""},
		{"2003", "王五", "财务部", "会计", "男", "2021-03-10", "30", "2", "03月", "本科", "党员", "XL", "43", "城镇", "汉族", "广东", "广东省深圳市", "110101199003033456", "已婚", "有", "否", "13700137000", "王妻", "0755-12345678", "广东省深圳市南山区", "华南理工大学", "会计学", "2015", "SI00003", "无", "wangwu@example.com", "已离职", "resigned", "2024-12-31"},
	},
	widths: []float64{
		10, 10, 14, 14, 6, 14, 8, 8,
		10, 12, 12, 8, 8, 10,
		10, 12, 28, 22, 10, 8,
		10, 14, 12, 24,
		28, 18, 14, 12, 18, 18, 22, 18, 10, 14,
	},
}

// EmployeeTemplateHeaders returns a copy of employee template headers for external use
func EmployeeTemplateHeaders() []string {
	headers := make([]string, len(employeeTemplateConfig.headers))
	copy(headers, employeeTemplateConfig.headers)
	return headers
}

// EmployeeTemplateColumnWidths returns a copy of column widths for the template layout.
func EmployeeTemplateColumnWidths() []float64 {
	widths := make([]float64, len(employeeTemplateConfig.widths))
	copy(widths, employeeTemplateConfig.widths)
	return widths
}

func generateTemplateFile(cfg templateConfig, outputPath string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	f.SetSheetName("Sheet1", cfg.sheetName)

	for i, header := range cfg.headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("resolve header cell: %w", err)
		}
		if err := f.SetCellValue(cfg.sheetName, cell, header); err != nil {
			return fmt.Errorf("set header: %w", err)
		}
	}

	for rowIndex, row := range cfg.samples {
		for colIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err != nil {
				return fmt.Errorf("resolve body cell: %w", err)
			}
			if err := f.SetCellValue(cfg.sheetName, cell, value); err != nil {
				return fmt.Errorf("set sample data: %w", err)
			}
		}
	}

	for i, width := range cfg.widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return fmt.Errorf("resolve column name: %w", err)
		}
		if err := f.SetColWidth(cfg.sheetName, col, col, width); err != nil {
			return fmt.Errorf("set column width: %w", err)
		}
	}

	style, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return fmt.Errorf("create style: %w", err)
	}

	for i := 0; i < len(cfg.headers); i++ {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("resolve style cell: %w", err)
		}
		if err := f.SetCellStyle(cfg.sheetName, cell, cell, style); err != nil {
			return fmt.Errorf("set header style: %w", err)
		}
	}

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save template file: %w", err)
	}

	return nil
}

// GenerateRosterTemplate generates the roster Excel template.
func GenerateRosterTemplate(outputPath string) error {
	return generateTemplateFile(rosterTemplateConfig, outputPath)
}

// GenerateEmployeeTemplate generates the employee import template.
func GenerateEmployeeTemplate(outputPath string) error {
	return generateTemplateFile(employeeTemplateConfig, outputPath)
}

func GenerateResignedEmployeeTemplate(outputPath string) error {
	return generateTemplateFile(employeeTemplateConfig, outputPath)
}

// GetRosterTemplatePath returns the path to the roster template file.
func GetRosterTemplatePath() string {
	return filepath.Join(".", "data", "花名册模板.xlsx")
}

// GetEmployeeTemplatePath returns the path to the employee template file.
func GetEmployeeTemplatePath() string {
	return filepath.Join(".", "data", "员工导入模板.xlsx")
}

func GetResignedEmployeeTemplatePath() string {
	return filepath.Join(".", "data", "离职员工导入模板.xlsx")
}
