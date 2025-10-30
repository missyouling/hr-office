package service

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// GenerateRosterTemplate generates a roster Excel template file
func GenerateRosterTemplate(outputPath string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "花名册"
	f.SetSheetName("Sheet1", sheetName)

	// Set headers - 完整匹配实际Excel模板
	headers := []string{
		"工号", "姓名", "部门", "岗位", "性别", "入职时间", "年龄", "工龄",
		"出生月份", "文化程度", "政治面貌", "工作服", "劳保鞋", "户口性质",
		"民族", "籍贯", "身份证地址", "身份证号码", "婚姻状况", "社保",
		"是否生育", "联系电话", "紧急联系人", "家庭电话/紧急情况联系电话",
		"现居住地址", "毕业院校", "专业", "毕业时间",
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Add sample data - 完整字段示例
	sampleData := [][]interface{}{
		{"2001", "张三", "销售部", "销售员", "男", "2020-01-01", "25", "3", "1月", "大专", "群众", "L", "42", "城镇", "汉族", "四川", "四川省成都市", "110101199001011234", "未婚", "有", "否", "13800138000", "张父", "028-12345678", "四川省成都市高新区", "四川大学", "市场营销", "2019"},
		{"2002", "李四", "技术部", "工程师", "女", "2019-06-15", "28", "4", "6月", "本科", "团员", "M", "38", "农村", "汉族", "重庆", "重庆市沙坪坝区", "110101199002022345", "已婚", "有", "是", "13900139000", "李母", "023-87654321", "重庆市沙坪坝区大学城", "重庆大学", "计算机科学", "2018"},
		{"2003", "王五", "财务部", "会计", "男", "2021-03-10", "30", "2", "3月", "本科", "党员", "XL", "43", "城镇", "汉族", "广东", "广东省深圳市", "110101199003033456", "已婚", "有", "否", "13700137000", "王妻", "0755-12345678", "广东省深圳市南山区", "华南理工大学", "会计学", "2015"},
	}

	for rowIndex, row := range sampleData {
		for colIndex, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIndex, rowIndex+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Set column widths - 动态设置所有列宽
	colWidths := []float64{
		8,  // 工号
		10, // 姓名
		12, // 部门
		12, // 岗位
		6,  // 性别
		12, // 入职时间
		6,  // 年龄
		6,  // 工龄
		8,  // 出生月份
		10, // 文化程度
		10, // 政治面貌
		8,  // 工作服
		8,  // 劳保鞋
		8,  // 户口性质
		8,  // 民族
		10, // 籍贯
		25, // 身份证地址
		20, // 身份证号码
		8,  // 婚姻状况
		6,  // 社保
		8,  // 是否生育
		12, // 联系电话
		10, // 紧急联系人
		20, // 家庭电话/紧急情况联系电话
		25, // 现居住地址
		15, // 毕业院校
		12, // 专业
		10, // 毕业时间
	}

	for i, width := range colWidths {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, width)
	}

	// Style the header row
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})
	if err != nil {
		return fmt.Errorf("create style: %w", err)
	}

	for i := 0; i < len(headers); i++ {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// Save the file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save template file: %w", err)
	}

	return nil
}

// GetRosterTemplatePath returns the path to the roster template file
func GetRosterTemplatePath() string {
	return filepath.Join(".", "data", "花名册模板.xlsx")
}