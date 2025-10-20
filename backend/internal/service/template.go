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

	// Set headers
	headers := []string{"姓名", "证件号码", "部门", "岗位", "备注"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Add sample data
	sampleData := [][]interface{}{
		{"张三", "110101199001011234", "销售部", "销售员", ""},
		{"李四", "110101199002022345", "技术部", "工程师", ""},
		{"王五", "110101199003033456", "财务部", "会计", ""},
	}

	for rowIndex, row := range sampleData {
		for colIndex, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIndex, rowIndex+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 12) // 姓名
	f.SetColWidth(sheetName, "B", "B", 20) // 证件号码
	f.SetColWidth(sheetName, "C", "C", 15) // 部门
	f.SetColWidth(sheetName, "D", "D", 12) // 岗位
	f.SetColWidth(sheetName, "E", "E", 15) // 备注

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