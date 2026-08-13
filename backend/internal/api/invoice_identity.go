package api

import (
	"strings"

	"siapp/internal/models"
)

// identityKey 前缀常量：区分增值税发票与电子票号回退场景，避免跨类型碰撞。
const (
	identityKeyPrefixVAT        = "vat:"
	identityKeyPrefixElectronic = "electronic:"
)

// normalizeIdentityPart 稳定规范化身份键字段：
//   - 去除首尾空白；
//   - 全角数字/字母转半角（OCR 常见输出）；
//   - ASCII 统一大写。
//
// 注意：不做任何 OCR 字符纠错（如 O→0、I→1），避免误判真实票号。
func normalizeIdentityPart(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	builder.Grow(len(value))
	for _, char := range value {
		switch {
		case char >= '０' && char <= '９':
			builder.WriteRune(char - '０' + '0')
		case char >= 'Ａ' && char <= 'Ｚ':
			builder.WriteRune(char - 'Ａ' + 'A')
		case char >= 'ａ' && char <= 'ｚ':
			builder.WriteRune(char - 'ａ' + 'a')
		default:
			builder.WriteRune(char)
		}
	}
	return strings.ToUpper(builder.String())
}

// computeInvoiceIdentityKey 服务端统一计算发票身份键（创建/更新/解析写回/更正共用）。
// 规则（严禁使用金额作为身份键）：
//   - 增值税发票（vat_input）：优先规范化 invoice_code + invoice_no，前缀 vat:；
//     缺代码时回退电子票号 electronic_invoice_no，前缀 electronic:；
//   - 其他凭证类型：返回 nil（NULL），不参与全局去重；
//   - 空值一律不存空字符串（返回 nil）。
func computeInvoiceIdentityKey(voucherType models.InvoiceVoucherType, invoiceNo, invoiceCode, electronicNo string) *string {
	if voucherType != models.InvoiceVoucherTypeVATInput {
		return nil
	}
	code := normalizeIdentityPart(invoiceCode)
	no := normalizeIdentityPart(invoiceNo)
	if code != "" && no != "" {
		key := identityKeyPrefixVAT + code + "|" + no
		return &key
	}
	if electronic := normalizeIdentityPart(electronicNo); electronic != "" {
		key := identityKeyPrefixElectronic + electronic
		return &key
	}
	return nil
}
