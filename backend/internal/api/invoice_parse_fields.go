package api

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"siapp/internal/models"
)

const invoiceTextMaximum = 200000

type parsedInvoice struct {
	invoiceNo, code, electronicNo, seller, sellerTaxNo string
	buyer, buyerTaxNo, remark, invoiceType             string
	date                                               time.Time
	dateProvided, invalidDate                          bool
	amount, tax, total                                 float64
	items                                              []models.InvoiceItem
	confidence                                         datatypes.JSON
}

var invoiceFieldPatterns = map[string]*regexp.Regexp{
	"code":       regexp.MustCompile(`(?i)(?:发票代码|invoice code)[：:\s]*([A-Z0-9]{8,30})`),
	"number":     regexp.MustCompile(`(?im)(?:^|\n)\s*(?:发票号码|票号)[：:\s]*([A-Z0-9-]{6,30})`),
	"electronic": regexp.MustCompile(`(?im)(?:^|\n)\s*电子(?:发票)?票号[：:\s]*([A-Z0-9-]{6,50})`),
	"date":       regexp.MustCompile(`(?:开票日期|日期)[：:\s]*(\d{4}[年/-]\d{1,2}[月/-]\d{1,2}日?)`),
	"total":      regexp.MustCompile(`(?:价税合计|合计)[：:\s￥¥]*([-\d,]+(?:\.\d{1,2})?)`),
	"amount":     regexp.MustCompile(`(?:不含税金额|金额)[：:\s￥¥]*([-\d,]+(?:\.\d{1,2})?)`),
	"tax":        regexp.MustCompile(`税额[：:\s￥¥]*([-\d,]+(?:\.\d{1,2})?)`),
	"rate":       regexp.MustCompile(`([-\d.]+)%`),
}

func invoiceTextQuality(text string) bool {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) < 20 || hasUnusableInvoiceText(runes) {
		return false
	}
	for _, signal := range []string{"发票", "购方", "销售方", "购买方", "金额", "税额", "价税", "付款", "收款", "行程单"} {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}

func hasUnusableInvoiceText(runes []rune) bool {
	invalid := 0
	for _, char := range runes {
		if char == '\uFFFD' || (char < 32 && char != '\n' && char != '\r' && char != '\t') {
			invalid++
		}
	}
	return invalid*5 > len(runes)
}
func limitInvoiceText(text string) string {
	if len(text) > invoiceTextMaximum {
		return text[:invoiceTextMaximum]
	}
	return text
}

func parseInvoiceFields(text string) parsedInvoice {
	p := parsedInvoice{code: parseField(text, "code"), invoiceNo: parseField(text, "number"), electronicNo: parseField(text, "electronic")}
	rawDate := parseField(text, "date")
	p.dateProvided, p.date = rawDate != "", parseInvoiceDate(rawDate)
	p.invalidDate = p.dateProvided && p.date.IsZero()
	p.total, p.amount, p.tax = parseInvoiceMoney(parseField(text, "total")), parseInvoiceMoney(parseField(text, "amount")), parseInvoiceMoney(parseField(text, "tax"))
	p.seller, p.sellerTaxNo = parseLine(text, "销售方名称"), parseLine(text, "销售方纳税人识别号")
	p.buyer, p.buyerTaxNo = parseLine(text, "购买方名称"), parseLine(text, "购买方纳税人识别号")
	p.remark, p.invoiceType = parseLine(text, "备注"), parseLine(text, "发票类型")
	p.items, p.confidence = parseInvoiceItems(text), invoiceConfidence(p)
	return p
}

func parseField(text, name string) string {
	match := invoiceFieldPatterns[name].FindStringSubmatch(text)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
func parseLine(text, label string) string {
	match := regexp.MustCompile(label + `[：:]?\s*([^\r\n]+)`).FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
func parseInvoiceDate(raw string) time.Time {
	raw = strings.NewReplacer("年", "-", "月", "-", "日", "", "/", "-").Replace(raw)
	value, _ := time.Parse("2006-1-2", raw)
	return value
}
func parseInvoiceMoney(raw string) float64 {
	value, _ := strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
	return value
}

func parseInvoiceItems(text string) []models.InvoiceItem {
	items := []models.InvoiceItem{}
	for _, line := range strings.Split(text, "\n") {
		columns := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(columns) < 7 || strings.Contains(line, "名称") {
			continue
		}
		item := models.InvoiceItem{Name: strings.TrimSpace(columns[0]), Specification: strings.TrimSpace(columns[1]), Unit: strings.TrimSpace(columns[2]), Quantity: parseInvoiceMoney(columns[3]), UnitPrice: parseInvoiceMoney(columns[4]), Amount: parseInvoiceMoney(columns[5]), TaxRate: parseInvoiceMoney(strings.TrimSuffix(strings.TrimSpace(columns[6]), "%"))}
		if len(columns) > 7 {
			item.TaxAmount = parseInvoiceMoney(columns[7])
		}
		if item.Name != "" {
			items = append(items, item)
		}
	}
	return items
}

func invoiceConfidence(p parsedInvoice) datatypes.JSON {
	missing := []string{}
	lowConfidence := []string{}
	levels := map[string]any{}
	for field, value := range map[string]string{"invoice_no": p.invoiceNo, "invoice_code": p.code, "seller": p.seller, "buyer": p.buyer, "seller_tax_no": p.sellerTaxNo, "buyer_tax_no": p.buyerTaxNo} {
		levels[field] = confidenceLevel(value)
		if value == "" {
			missing = append(missing, field)
		}
	}
	if p.invalidDate {
		lowConfidence = append(lowConfidence, "invoice_date")
	}
	if !validTaxNo(p.sellerTaxNo) && p.sellerTaxNo != "" {
		lowConfidence = append(lowConfidence, "seller_tax_no")
	}
	if !validTaxNo(p.buyerTaxNo) && p.buyerTaxNo != "" {
		lowConfidence = append(lowConfidence, "buyer_tax_no")
	}
	anomaly := amountAnomaly(p.amount, p.tax, p.total)
	if anomaly {
		lowConfidence = append(lowConfidence, "amount", "tax_amount", "total_amount")
	}
	levels["missing_fields"], levels["low_confidence_fields"], levels["amount_anomaly"] = missing, lowConfidence, anomaly
	encoded, _ := json.Marshal(levels)
	return datatypes.JSON(encoded)
}
func validTaxNo(value string) bool {
	return regexp.MustCompile(`^[A-Z0-9]{15,20}$`).MatchString(strings.ToUpper(value))
}

func (p parsedInvoice) invoiceUpdates(text, source string, voucherType models.InvoiceVoucherType) map[string]any {
	updates := map[string]any{
		"invoice_no": p.invoiceNo, "invoice_code": p.code, "electronic_invoice_no": p.electronicNo,
		"invoice_type": p.invoiceType, "seller": p.seller, "seller_tax_no": p.sellerTaxNo,
		"buyer": p.buyer, "buyer_tax_no": p.buyerTaxNo, "amount": p.amount, "tax_amount": p.tax,
		"total_amount": p.total, "remark": p.remark, "original_text": text,
		"recognition_source": source, "field_confidence": p.confidence,
		"identity_key": computeInvoiceIdentityKey(voucherType, p.invoiceNo, p.code, p.electronicNo),
	}
	if !p.date.IsZero() {
		updates["invoice_date"] = p.date
	}
	return updates
}

func confidenceLevel(value string) string {
	if value == "" {
		return "missing"
	}
	return "high"
}
func amountAnomaly(amount, tax, total float64) bool {
	if total == 0 || amount+tax == 0 {
		return false
	}
	difference := total - amount - tax
	if difference < 0 {
		difference = -difference
	}
	return difference > 0.02
}
