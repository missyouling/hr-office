package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"siapp/internal/models"
)

// CSVSupplyItem CSV 导入的用品行数据
type CSVSupplyItem struct {
	Name           string
	Spec           string
	Unit           string
	ReferencePrice float64
	CategoryName   string
	SupplierName   string
	Status         string
	Remark         string
	SafetyStock    int
}

// DecodeCSVContent 自适应解码 CSV 内容（UTF-8 / GBK 自动识别）
func DecodeCSVContent(raw []byte) string {
	// 首先尝试 UTF-8
	body := string(raw)
	// 如果包含 Unicode 替换字符 U+FFFD，尝试 GBK 解码
	if strings.ContainsRune(body, '\uFFFD') || bytes.Contains(raw, []byte{0xEF, 0xBF, 0xBD}) {
		reader := transform.NewReader(bytes.NewReader(raw), simplifiedchinese.GBK.NewDecoder())
		decoded, err := io.ReadAll(reader)
		if err == nil {
			body = string(decoded)
		}
	}
	// 移除 UTF-8 BOM
	if strings.HasPrefix(body, "\uFEFF") {
		body = body[len("\uFEFF"):]
	}
	return body
}

// ParseCSVSupplies 按表头智能识别列解析用品 CSV
func ParseCSVSupplies(content string) []CSVSupplyItem {
	reader := csv.NewReader(strings.NewReader(content))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil
	}

	header := records[0]
	// 查找各列的索引位置（兼容中英文表头）
	nameIdx := findHeaderIndex(header, "品名", "name")
	specIdx := findHeaderIndex(header, "规格", "spec")
	unitIdx := findHeaderIndex(header, "单位", "unit")
	priceIdx := findHeaderIndex(header, "参考单价", "单价", "reference_price", "price")
	catIdx := findHeaderIndex(header, "分类", "分类名称", "category_name", "category")
	supplierIdx := findHeaderIndex(header, "供应商", "supplier_name", "supplier")
	statusIdx := findHeaderIndex(header, "状态", "status")
	remarkIdx := findHeaderIndex(header, "备注", "remark")
	safetyIdx := findHeaderIndex(header, "安全库存", "safety_stock")

	// 兜底：使用默认位置
	if nameIdx < 0 {
		nameIdx = 0
	}
	if specIdx < 0 {
		specIdx = 1
	}
	if unitIdx < 0 {
		unitIdx = 2
	}
	if priceIdx < 0 {
		priceIdx = 3
	}
	if catIdx < 0 {
		catIdx = 4
	}
	if remarkIdx < 0 {
		remarkIdx = 5
	}
	if statusIdx < 0 {
		statusIdx = 6
	}

	var items []CSVSupplyItem
	for i := 1; i < len(records); i++ {
		row := records[i]
		if getCol(row, nameIdx) == "" {
			continue
		}
		price, _ := strconv.ParseFloat(getCol(row, priceIdx), 64)
		safety := 0
		if safetyIdx >= 0 {
			safety, _ = strconv.Atoi(getCol(row, safetyIdx))
		}
		items = append(items, CSVSupplyItem{
			Name:           getCol(row, nameIdx),
			Spec:           getCol(row, specIdx),
			Unit:           getCol(row, unitIdx),
			ReferencePrice: price,
			CategoryName:   getCol(row, catIdx),
			SupplierName:   getCol(row, supplierIdx),
			Status:         getCol(row, statusIdx),
			Remark:         getCol(row, remarkIdx),
			SafetyStock:    safety,
		})
	}
	return items
}

// ExportSuppliesCSV 导出用品列表为 CSV 字符串
func ExportSuppliesCSV(items []models.OfficeSupply, catMap map[uint]string, supMap map[uint]string) string {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	writer := csv.NewWriter(&buf)

	writer.Write([]string{"品名", "规格", "单位", "参考单价", "分类", "供应商", "状态", "备注"})
	for _, s := range items {
		catName := catMap[uintVal(s.CategoryID)]
		supName := supMap[uintVal(s.SupplierID)]
		writer.Write([]string{
			s.Name, s.Spec, s.Unit,
			fmt.Sprintf("%.2f", s.ReferencePrice),
			catName, supName, s.Status, s.Remark,
		})
	}
	writer.Flush()
	return buf.String()
}

// ExportPurchasesCSV 导出采购单列表为 CSV 字符串
func ExportPurchasesCSV(items interface{}) string {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"单号", "日期", "品项数", "总金额", "状态", "备注", "创建时间"})
	// 这里应该解析 items，但在 API 层已直接处理
	writer.Flush()
	return buf.String()
}

// ExportPurchaseExcel 导出单个采购单为 Excel
func ExportPurchaseExcel(purchase *models.OfficePurchase, items []models.OfficePurchaseItem) ([]byte, string, error) {
	f := excelize.NewFile()
	sheet := "采购单"
	f.SetSheetName("Sheet1", sheet)

	// 设置列宽
	f.SetColWidth(sheet, "A", "A", 12)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 15)
	f.SetColWidth(sheet, "D", "D", 10)
	f.SetColWidth(sheet, "E", "E", 12)
	f.SetColWidth(sheet, "F", "F", 8)
	f.SetColWidth(sheet, "G", "G", 12)

	// 标题行
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	f.SetCellValue(sheet, "A1", fmt.Sprintf("采购单 - %s", purchase.OrderNo))
	f.MergeCell(sheet, "A1", "G1")
	f.SetCellStyle(sheet, "A1", "G1", style)

	f.SetCellValue(sheet, "A2", "日期：")
	f.SetCellValue(sheet, "B2", purchase.PurchaseDate.Format("2006-01-02"))
	f.SetCellValue(sheet, "A3", "状态：")
	f.SetCellValue(sheet, "B3", purchase.Status)
	f.SetCellValue(sheet, "A4", "供应商：")
	f.SetCellValue(sheet, "B4", purchase.SupplierName)

	// 表头
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#1E40AF"}, Pattern: 1},
	})
	headers := []string{"序号", "品名", "规格", "单位", "单价", "数量", "小计"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// 数据行
	for i, item := range items {
		row := 7 + i
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "用品 #"+strconv.Itoa(int(item.SupplyID)))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "")
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "")
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.UnitPrice)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.Subtotal)
	}

	// 合计行
	totalRow := 7 + len(items)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", totalRow), "合计：")
	f.SetCellValue(sheet, fmt.Sprintf("G%d", totalRow), purchase.TotalAmount)
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "#DC2626"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", totalRow), fmt.Sprintf("G%d", totalRow), totalStyle)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("%s-%s.xlsx", purchase.OrderNo, time.Now().Format("20060102"))
	return buf.Bytes(), filename, nil
}

// ExportPurchasesExcel 批量导出采购单列表为 Excel
func ExportPurchasesExcel(items []PurchaseExportItem, startDate, endDate, keyword string) ([]byte, string, error) {
	f := excelize.NewFile()
	sheet := "采购单列表"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"单号", "日期", "品项数", "总金额", "状态", "供应商", "付款状态", "备注", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#1E40AF"}, Pattern: 1},
		})
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}

	for i, p := range items {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), p.OrderNo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.PurchaseDate)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.ItemCount)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.TotalAmount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), p.Status)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), p.SupplierName)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), p.PaymentStatus)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.Remark)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.CreatedAt)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("purchases-%s.xlsx", time.Now().Format("20060102"))
	return buf.Bytes(), filename, nil
}

// PurchaseExportItem 采购单导出条目（导出给 API 层使用）
type PurchaseExportItem struct {
	ID            uint    `json:"id"`
	OrderNo       string  `json:"order_no"`
	PurchaseDate  string  `json:"purchase_date"`
	TotalAmount   float64 `json:"total_amount"`
	ItemCount     int64   `json:"item_count"`
	Status        string  `json:"status"`
	SupplierName  string  `json:"supplier_name"`
	PaymentStatus string  `json:"payment_status"`
	Remark        string  `json:"remark"`
	CreatedAt     string  `json:"created_at"`
}

// ---- 辅助函数 ----

func findHeaderIndex(header []string, aliases ...string) int {
	for i, h := range header {
		trimmed := strings.TrimSpace(h)
		for _, a := range aliases {
			if strings.Contains(trimmed, a) {
				return i
			}
		}
	}
	return -1
}

func getCol(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func uintVal(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}
