package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// CanteenCSVService 食堂模块 CSV 导入导出服务
type CanteenCSVService struct {
	db *gorm.DB
}

// NewCanteenCSVService 创建 CSV 服务实例
func NewCanteenCSVService(db *gorm.DB) *CanteenCSVService {
	return &CanteenCSVService{db: db}
}

// ========== CSV 行解析 ==========

// parseCSVContent 解析 CSV 文本（支持双引号转义、GBK/UTF-8 自适应、BOM 去除）
func parseCSVContent(raw []byte) ([][]string, error) {
	// 尝试 UTF-8
	text, isUTF8 := tryUTF8(raw)
	if !isUTF8 {
		// 尝试 GBK
		reader := transform.NewReader(strings.NewReader(string(raw)), simplifiedchinese.GBK.NewDecoder())
		buf, err := io.ReadAll(reader)
		if err == nil {
			text = string(buf)
		}
	}
	// 去除 BOM
	text = strings.TrimPrefix(text, "\uFEFF")
	// 按行分割
	lines := strings.Split(text, "\n")
	var nonEmpty []string
	for _, l := range lines {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) < 2 {
		return nil, fmt.Errorf("CSV 数据不足，至少需要表头和一行数据")
	}
	// 解析每行
	var result [][]string
	for _, line := range nonEmpty {
		cols := parseCSVLine(line)
		result = append(result, cols)
	}
	return result, nil
}

// tryUTF8 尝试 UTF-8 解码
func tryUTF8(raw []byte) (string, bool) {
	text := string(raw)
	// 简单的 UTF-8 有效性检查
	for i := 0; i < len(raw); {
		if raw[i] < 0x80 {
			i++
			continue
		}
		// 多字节序列检查
		r, size := decodeUTF8Rune(raw[i:])
		if r == 0xFFFD && size <= 1 {
			return text, false
		}
		i += size
	}
	return text, true
}

func decodeUTF8Rune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0xFFFD, 0
	}
	c := b[0]
	switch {
	case c < 0x80:
		return rune(c), 1
	case c < 0xC0:
		return 0xFFFD, 1
	case c < 0xE0:
		if len(b) < 2 {
			return 0xFFFD, 1
		}
		return rune((uint(c)&0x1F)<<6 | (uint(b[1]) & 0x3F)), 2
	case c < 0xF0:
		if len(b) < 3 {
			return 0xFFFD, 1
		}
		return rune((uint(c)&0x0F)<<12 | (uint(b[1])&0x3F)<<6 | (uint(b[2]) & 0x3F)), 3
	default:
		return 0xFFFD, 1
	}
}

// parseCSVLine 解析单行 CSV（支持双引号转义）
func parseCSVLine(line string) []string {
	var cols []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			if inQuote && i+1 < len(line) && line[i+1] == '"' {
				cur.WriteByte('"')
				i++
			} else {
				inQuote = !inQuote
			}
		} else if ch == ',' && !inQuote {
			cols = append(cols, strings.TrimSpace(strings.Trim(cur.String(), "\"")))
			cur.Reset()
		} else {
			cur.WriteByte(ch)
		}
	}
	cols = append(cols, strings.TrimSpace(strings.Trim(cur.String(), "\"")))
	return cols
}

// ========== 表头智能映射 ==========

// autoMapHeader 根据目标关键字列表自动匹配表头列名
// 规范化：去除空格、|、｜，转小写
func autoMapHeader(header []string, targets []string) string {
	norm := func(h string) string {
		h = strings.ReplaceAll(h, "|", "")
		h = strings.ReplaceAll(h, "｜", "")
		h = strings.ReplaceAll(h, " ", "")
		return strings.ToLower(h)
	}
	headerNorm := make([]string, len(header))
	for i, h := range header {
		headerNorm[i] = norm(h)
	}
	for _, t := range targets {
		tn := norm(t)
		for i, hn := range headerNorm {
			if strings.Contains(hn, tn) {
				return header[i]
			}
		}
	}
	return ""
}

// isSummaryRow 判断是否为汇总/合计行
func isSummaryRow(firstCol string) bool {
	matched, _ := regexp.MatchString("汇总|合计|总计|小计", firstCol)
	return matched
}

// cleanMoney 清洗金额字符串（去除￥¥等符号和千位分隔逗号）
func cleanMoney(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, "￥", "")
	s = strings.ReplaceAll(s, "¥", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err == nil
}

// cleanDate 清洗日期字符串（支持 YYYY-MM-DD、YYYY/MM/DD、含时分秒等格式）
func cleanDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 匹配 YYYY[-/]MM[-/]DD
	re := regexp.MustCompile(`(\d{4})[-/](\d{1,2})[-/](\d{1,2})`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return fmt.Sprintf("%s-%02s-%02s", m[1], m[2], m[3])
}

// ========== 饭卡充值导入 ==========

// CardImportResult 饭卡导入结果
type CardImportResult struct {
	Total    int               `json:"total"`
	Inserted int               `json:"inserted"`
	Updated  int               `json:"updated"`
	Skipped  int               `json:"skipped"`
	Errors   []CardImportError `json:"errors"`
}

// CardImportError 导入错误行
type CardImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// buildCardMapping 构建充值导入字段映射
func buildCardMapping(header []string, existing map[string]string) map[string]string {
	mapping := make(map[string]string)
	for k, v := range existing {
		if v != "" {
			mapping[k] = v
		}
	}
	autoSet := func(key string, targets []string) {
		if _, ok := mapping[key]; !ok {
			mapping[key] = autoMapHeader(header, targets)
		}
	}
	autoSet("external_sn", []string{"卡流水号", "流水号", "externalsn"})
	autoSet("user_name", []string{"姓名", "用户名", "username"})
	autoSet("card_user_id", []string{"工号", "userid", "员工编号"})
	autoSet("card_no", []string{"卡号", "cardno"})
	autoSet("department_code", []string{"部门编号", "departmentcode"})
	autoSet("user_department", []string{"部门名称", "部门", "department"})
	autoSet("recharge_date", []string{"充值时间", "充值日期", "时间", "rechargedate"})
	autoSet("amount", []string{"充值金额", "金额", "amount"})
	autoSet("balance_recorded", []string{"卡余额", "余额", "balance"})
	autoSet("payment_method", []string{"类型", "支付方式", "paymentmethod"})
	autoSet("operator", []string{"操作员", "operator"})
	autoSet("machine_no", []string{"机号", "machineno"})
	autoSet("bill_no", []string{"账单号", "billno"})
	return mapping
}

// ImportRecharges 导入饭卡充值记录
func (s *CanteenCSVService) ImportRecharges(userID uint, file multipart.File, mode string, rawMapping string) (*CardImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	parsed, err := parseCSVContent(raw)
	if err != nil {
		return nil, err
	}

	header := parsed[0]
	// 解析 mapping
	mapping := make(map[string]string)
	if rawMapping != "" {
		for _, pair := range strings.Split(rawMapping, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				mapping[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	mapping = buildCardMapping(header, mapping)

	// 必填列检查
	required := []string{"external_sn", "user_name", "recharge_date", "amount"}
	var missing []string
	for _, k := range required {
		if mapping[k] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("缺少必填列映射：%s", strings.Join(missing, "、"))
	}

	// 构建列索引
	colIndex := make(map[string]int)
	for k, col := range mapping {
		if col == "" {
			continue
		}
		for i, h := range header {
			if h == col {
				colIndex[k] = i
				break
			}
		}
	}

	result := &CardImportResult{Total: len(parsed) - 1}

	for i := 1; i < len(parsed); i++ {
		row := parsed[i]
		if len(row) == 0 || isSummaryRow(row[0]) {
			continue
		}
		getVal := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		extSN := getVal("external_sn")
		userName := getVal("user_name")
		rechargeDate := cleanDate(getVal("recharge_date"))
		amount, amountOK := cleanMoney(getVal("amount"))

		var missingFields []string
		if extSN == "" {
			missingFields = append(missingFields, "外部编号缺失")
		}
		if userName == "" {
			missingFields = append(missingFields, "姓名缺失")
		}
		if rechargeDate == "" {
			missingFields = append(missingFields, "日期缺失")
		}
		if !amountOK {
			missingFields = append(missingFields, "金额格式错误")
		}
		if len(missingFields) > 0 {
			result.Errors = append(result.Errors, CardImportError{Row: i + 1, Reason: strings.Join(missingFields, "、")})
			continue
		}

		balance, _ := cleanMoney(getVal("balance_recorded"))
		rechargeDateParsed, _ := time.Parse("2006-01-02", rechargeDate)

		record := models.CanteenCardRecharge{
			UserID:         &userID,
			CardNo:         getVal("card_no"),
			CardUserID:     getVal("card_user_id"),
			UserName:       userName,
			DepartmentCode: getVal("department_code"),
			UserDepartment: getVal("user_department"),
			RechargeDate:   &rechargeDateParsed,
			Amount:         amount,
			PaymentMethod:  defaultStr(getVal("payment_method"), "现金"),
			Operator:       defaultStr(getVal("operator"), "导入"),
			MachineNo:      getVal("machine_no"),
			BillNo:         getVal("bill_no"),
			Remark:         getVal("remark"),
		}
		extSNVal := extSN
		record.ExternalSN = &extSNVal
		if balance > 0 {
			balanceVal := balance
			record.BalanceRecorded = &balanceVal
		}

		// 检查 external_sn 是否已存在
		var existing models.CanteenCardRecharge
		err := s.db.Where("external_sn = ? AND user_id = ?", extSN, userID).First(&existing).Error
		if err == nil {
			// 已存在
			if mode == "skip" {
				result.Skipped++
				continue
			}
			// upsert: 更新
			record.ID = existing.ID
			if s.db.Save(&record).Error == nil {
				result.Updated++
			}
		} else if err == gorm.ErrRecordNotFound {
			// 新增
			if s.db.Create(&record).Error == nil {
				result.Inserted++
			}
		} else {
			result.Errors = append(result.Errors, CardImportError{Row: i + 1, Reason: err.Error()})
		}
	}
	return result, nil
}

// ========== 饭卡退费导入 ==========

// buildRefundMapping 构建退费导入字段映射
func buildRefundMapping(header []string, existing map[string]string) map[string]string {
	mapping := make(map[string]string)
	for k, v := range existing {
		if v != "" {
			mapping[k] = v
		}
	}
	autoSet := func(key string, targets []string) {
		if _, ok := mapping[key]; !ok {
			mapping[key] = autoMapHeader(header, targets)
		}
	}
	autoSet("external_sn", []string{"卡流水号", "流水号", "externalsn"})
	autoSet("user_name", []string{"姓名", "用户名", "username"})
	autoSet("card_user_id", []string{"工号", "userid", "员工编号"})
	autoSet("card_no", []string{"卡号", "cardno"})
	autoSet("department_code", []string{"部门编号", "departmentcode"})
	autoSet("user_department", []string{"部门名称", "部门", "department"})
	autoSet("refund_date", []string{"退款时间", "退款日期", "时间", "refunddate"})
	autoSet("amount", []string{"退款金额", "金额", "amount"})
	autoSet("balance_recorded", []string{"卡上余额", "卡余额", "余额", "balance"})
	autoSet("operator", []string{"操作员", "operator"})
	autoSet("machine_no", []string{"机号", "machineno"})
	autoSet("bill_no", []string{"收支统计账单号", "账单号", "billno"})
	return mapping
}

// ImportRefunds 导入饭卡退费记录
func (s *CanteenCSVService) ImportRefunds(userID uint, file multipart.File, mode string, rawMapping string) (*CardImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	parsed, err := parseCSVContent(raw)
	if err != nil {
		return nil, err
	}

	header := parsed[0]
	mapping := make(map[string]string)
	if rawMapping != "" {
		for _, pair := range strings.Split(rawMapping, ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				mapping[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	mapping = buildRefundMapping(header, mapping)

	required := []string{"external_sn", "user_name", "refund_date", "amount"}
	var missing []string
	for _, k := range required {
		if mapping[k] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("缺少必填列映射：%s", strings.Join(missing, "、"))
	}

	colIndex := make(map[string]int)
	for k, col := range mapping {
		if col == "" {
			continue
		}
		for i, h := range header {
			if h == col {
				colIndex[k] = i
				break
			}
		}
	}

	result := &CardImportResult{Total: len(parsed) - 1}

	for i := 1; i < len(parsed); i++ {
		row := parsed[i]
		if len(row) == 0 || isSummaryRow(row[0]) {
			continue
		}
		getVal := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		extSN := getVal("external_sn")
		userName := getVal("user_name")
		refundDate := cleanDate(getVal("refund_date"))
		amount, amountOK := cleanMoney(getVal("amount"))

		var missingFields []string
		if extSN == "" {
			missingFields = append(missingFields, "外部编号缺失")
		}
		if userName == "" {
			missingFields = append(missingFields, "姓名缺失")
		}
		if refundDate == "" {
			missingFields = append(missingFields, "日期缺失")
		}
		if !amountOK {
			missingFields = append(missingFields, "金额格式错误")
		}
		if len(missingFields) > 0 {
			result.Errors = append(result.Errors, CardImportError{Row: i + 1, Reason: strings.Join(missingFields, "、")})
			continue
		}

		balance, _ := cleanMoney(getVal("balance_recorded"))
		refundDateParsed, _ := time.Parse("2006-01-02", refundDate)

		record := models.CanteenCardRefund{
			UserID:         &userID,
			CardNo:         getVal("card_no"),
			CardUserID:     getVal("card_user_id"),
			UserName:       userName,
			DepartmentCode: getVal("department_code"),
			UserDepartment: getVal("user_department"),
			RefundDate:     &refundDateParsed,
			Amount:         amount,
			Operator:       defaultStr(getVal("operator"), "导入"),
			MachineNo:      getVal("machine_no"),
			BillNo:         getVal("bill_no"),
			Remark:         getVal("remark"),
		}
		extSNVal := extSN
		record.ExternalSN = &extSNVal
		if balance > 0 {
			balanceVal := balance
			record.BalanceRecorded = &balanceVal
		}

		var existing models.CanteenCardRefund
		err := s.db.Where("external_sn = ? AND user_id = ?", extSN, userID).First(&existing).Error
		if err == nil {
			if mode == "skip" {
				result.Skipped++
				continue
			}
			record.ID = existing.ID
			if s.db.Save(&record).Error == nil {
				result.Updated++
			}
		} else if err == gorm.ErrRecordNotFound {
			if s.db.Create(&record).Error == nil {
				result.Inserted++
			}
		} else {
			result.Errors = append(result.Errors, CardImportError{Row: i + 1, Reason: err.Error()})
		}
	}
	return result, nil
}

// ========== 采购 CSV 导出 ==========

// ExportPurchasesCSV 导出采购明细为 CSV（UTF-8 BOM）
func (s *CanteenCSVService) ExportPurchasesCSV(userID uint, dateFrom, dateTo string) (string, error) {
	type PurchaseExportRow struct {
		OrderNo      string
		PurchaseDate string
		SupplierName string
		Channel      string
		SupplyName   string
		CategoryName string
		Quantity     float64
		Unit         string
		UnitPrice    float64
		Subtotal     float64
		ActualPay    float64
		Remark       string
	}

	query := s.db.Table("canteen_purchase_items pi").
		Select(`p.order_no, p.purchase_date, p.supplier_name, p.channel,
			s2.name as supply_name, c.name as category_name,
			pi.quantity, s2.unit, pi.unit_price, pi.subtotal,
			p.actual_pay, p.remark`).
		Joins("LEFT JOIN canteen_purchases p ON pi.purchase_id = p.id").
		Joins("LEFT JOIN canteen_supplies s2 ON pi.supply_id = s2.id").
		Joins("LEFT JOIN canteen_categories c ON s2.category_id = c.id").
		Where("pi.user_id = ?", userID)

	if dateFrom != "" {
		query = query.Where("p.purchase_date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("p.purchase_date <= ?", dateTo)
	}
	query = query.Order("p.purchase_date, p.id")

	var rows []PurchaseExportRow
	if err := query.Scan(&rows).Error; err != nil {
		return "", err
	}

	header := []string{"采购单号", "采购日期", "供应商", "渠道", "品名", "分类", "数量", "单位", "单价", "小计", "实支金额", "备注"}
	var buf strings.Builder
	buf.WriteString("\uFEFF") // UTF-8 BOM
	buf.WriteString(csvEscape(header))
	buf.WriteByte('\n')

	for _, r := range rows {
		vals := []string{
			r.OrderNo,
			r.PurchaseDate,
			r.SupplierName,
			r.Channel,
			r.SupplyName,
			r.CategoryName,
			fmt.Sprintf("%.2f", r.Quantity),
			r.Unit,
			fmt.Sprintf("%.2f", r.UnitPrice),
			fmt.Sprintf("%.2f", r.Subtotal),
			fmt.Sprintf("%.2f", r.ActualPay),
			strings.ReplaceAll(r.Remark, ",", "，"),
		}
		buf.WriteString(csvEscape(vals))
		buf.WriteByte('\n')
	}
	return buf.String(), nil
}

// csvEscape 将字符串数组转为 CSV 行（逗号分隔，必要时加双引号）
func csvEscape(vals []string) string {
	var out []string
	for _, v := range vals {
		if strings.ContainsAny(v, ",\"\n\r") {
			v = strings.ReplaceAll(v, "\"", "\"\"")
			v = "\"" + v + "\""
		}
		out = append(out, v)
	}
	return strings.Join(out, ",")
}

func defaultStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
