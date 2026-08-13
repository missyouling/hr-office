package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============================================================
// 字段级精准脱敏（P9.2）
//
// 业务字段规则（id_card/phone/address/bank_card/amount/自定义字段）
// 不允许把整段 Content/Snippet 当单字段值整体 mask，而是：
//   - 键值对替换：定位"字段标签（中文/英文别名）+ 分隔符 + 值"，仅替换值；
//   - 数字型强模式字段（身份证/手机号/银行卡号）额外对文本任意位置的
//     匹配值做全局替换（裸值兜底）；
//   - 提取的 原始值→脱敏值 映射供最终答案/SSE 增量复用，保证同一规则
//     在检索结果、Sources、Prompt、最终答案全链路生效。
//
// 未能可靠匹配（文本中无标签无模式）时保持原文，不整段破坏内容。
// 豁免角色查询失败视为脱敏失败（安全失败，调用方不得返回原文）。
// ============================================================

// fieldValueKind 字段值类型：决定值匹配模式与是否全局替换
type fieldValueKind int

const (
	valueKindText     fieldValueKind = iota // 文本值（address/自定义字段）：仅键值替换
	valueKindIDCard                         // 身份证号：键值 + 全局 18 位
	valueKindPhone                          // 手机号：键值 + 全局 11 位
	valueKindBankCard                       // 银行卡号：键值 + 全局 16-19 位
	valueKindAmount                         // 金额：仅键值（数字过于宽泛，不做全局替换）
)

// fieldKindByRule 规则字段名 → 值类型（未登记的字段视为自定义文本字段）
var fieldKindByRule = map[string]fieldValueKind{
	"id_card":   valueKindIDCard,
	"phone":     valueKindPhone,
	"bank_card": valueKindBankCard,
	"amount":    valueKindAmount,
	"address":   valueKindText,
}

// fieldLabelAliases 规则字段名 → 常见标签别名（英文 + 中文，键值替换用）
var fieldLabelAliases = map[string][]string{
	"id_card":   {"身份证号", "身份证", "证件号码", "证件号", "id_card", "idcard", "id number", "id_number", "identity no"},
	"phone":     {"手机号", "手机号码", "联系电话", "电话", "phone", "tel", "mobile", "cellphone"},
	"address":   {"地址", "住址", "家庭住址", "现居地址", "address", "addr"},
	"bank_card": {"银行卡号", "银行卡", "卡号", "银行账号", "bank_card", "bankcard", "card_no", "account no"},
	"amount":    {"金额", "总额", "合同金额", "amount", "money", "total amount"},
}

// 数字型强模式值正则（全局替换与键值提取共用）
var (
	idCardValueRe   = regexp.MustCompile(`\d{17}[\dXx]`)
	phoneValueRe    = regexp.MustCompile(`1[3-9]\d{9}`)
	bankCardValueRe = regexp.MustCompile(`\d{16,19}`)
)

// allFieldLabelsRe 全部已知字段标签合集（文本类值截断用：值遇到其他字段标签即停止）
var allFieldLabelsRe = func() *regexp.Regexp {
	seen := make(map[string]bool)
	parts := make([]string, 0, 16)
	for _, labels := range fieldLabelAliases {
		for _, l := range labels {
			key := strings.ToLower(l)
			if !seen[key] {
				seen[key] = true
				parts = append(parts, regexp.QuoteMeta(l))
			}
		}
	}
	sort.Slice(parts, func(i, j int) bool { return len([]rune(parts[i])) > len([]rune(parts[j])) })
	return regexp.MustCompile(`(?i)(?:` + strings.Join(parts, "|") + `)`)
}()

// userExemptFromMask 检查用户是否命中规则的 ExemptRole 豁免
// 豁免角色联表查询失败返回错误（安全失败：不得降级返回原值）
func userExemptFromMask(db *gorm.DB, user *models.User, mask *models.KBFieldMask) (bool, error) {
	if mask.ExemptRole == nil {
		return false, nil
	}
	var exemptCount int64
	err := db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", user.ID, *mask.ExemptRole).
		Count(&exemptCount).Error
	if err != nil {
		logMaskFailure(db, user, mask.KnowledgeBaseID, mask.FieldName, err)
		return false, fmt.Errorf("豁免角色查询失败(%s): %w", mask.FieldName, err)
	}
	return exemptCount > 0, nil
}

// maskFieldValueInText 在文本中精准替换字段规则命中的敏感值：
// 1. 键值对替换：字段标签（中文/英文别名）+ 分隔符 + 值，仅替换值部分；
// 2. 数字型强模式字段额外做全局值替换（文本任意位置的裸值）。
// 返回替换后的文本、原始值→脱敏值 映射（用于最终答案/SSE 防御层）与
// 豁免原始值列表（ExemptRole 命中时提取，答案/SSE 防御层须跳过这些值）。
// 用户命中 ExemptRole 豁免时整条规则跳过（返回原文）。
// exemptSet 为已收集的其他字段豁免原始值集合：非豁免字段的全局替换
// 会跳过被豁免值包含的匹配串，避免误伤豁免字段原文的内部子串
// （如手机号正则 1[3-9]\d{9} 会命中身份证内部子串 19900101123）。
func maskFieldValueInText(db *gorm.DB, user *models.User, text, fieldName string, mask *models.KBFieldMask, exemptSet map[string]struct{}) (string, map[string]string, []string, error) {
	exempt, err := userExemptFromMask(db, user, mask)
	if err != nil {
		return text, nil, nil, err
	}
	kind := fieldKindByRule[fieldName]
	labels := fieldLabelPattern(fieldName)
	if exempt {
		// 豁免：保留原文，提取受该规则保护的原始值供答案/SSE 层跳过
		return text, nil, collectExemptValues(text, kind, labels), nil
	}

	sensitive := map[string]string{}

	// 键值对替换：标签 + 分隔符 + 值
	if labels != "" {
		text = replaceKeyValues(text, labels, kind, mask.MaskPattern, sensitive)
	}

	// 数字型强模式字段：全局值替换（兜底处理无标签的裸敏感值）
	globalRe := valueGlobalRe(kind)
	if globalRe != nil {
		text = globalRe.ReplaceAllStringFunc(text, func(raw string) string {
			if coveredByExempt(raw, exemptSet) {
				// 命中值被其他字段豁免值包含（如身份证内的手机号子串）：
				// 不得替换，保证豁免字段原文完整保留
				return raw
			}
			maskedVal := applyMaskValue(mask.MaskPattern, raw)
			if maskedVal == raw {
				return raw
			}
			sensitive[raw] = maskedVal
			return maskedVal
		})
	}
	return text, sensitive, nil, nil
}

// coveredByExempt 判断 raw 是否被豁免集合中的某个原始值包含
// （豁免字段原文整体保留，其内部数字子串不得被其他规则误伤）
func coveredByExempt(raw string, exemptSet map[string]struct{}) bool {
	if raw == "" || len(exemptSet) == 0 {
		return false
	}
	for ex := range exemptSet {
		if ex != "" && strings.Contains(ex, raw) {
			return true
		}
	}
	return false
}

// collectExemptValues 豁免命中时提取文本中受该字段规则保护的原始值：
// - 键值标签后的值（该字段标签 + 分隔符 + 值）；
// - 数字型强模式字段额外提取全局匹配的裸值。
// 这些值在最终答案/SSE 防御层中被跳过，保证豁免角色看到原文，
// 同时其他未豁免字段的数字模式防御仍然生效。
func collectExemptValues(text string, kind fieldValueKind, labels string) []string {
	if text == "" {
		return nil
	}
	var values []string
	if labels != "" {
		values = append(values, collectKeyValues(text, labels, kind)...)
	}
	if globalRe := valueGlobalRe(kind); globalRe != nil {
		values = append(values, globalRe.FindAllString(text, -1)...)
	}
	return values
}

// collectKeyValues 收集文本中"字段标签 + 分隔符 + 值"形式的值（仅提取，不替换）
func collectKeyValues(text, labels string, kind fieldValueKind) []string {
	labelRe := regexp.MustCompile(`(?i)(` + labels + `)\s*(?:[:：=]\s*|\s+)`)
	var values []string
	for _, loc := range labelRe.FindAllStringIndex(text, -1) {
		if raw := extractFieldValue(text, loc[1], kind); raw != "" {
			values = append(values, raw)
		}
	}
	return values
}

// wholeTextSensitiveRe 整段文本中全部数字敏感模式（身份证号/手机号/银行卡号）
var wholeTextSensitiveRe = regexp.MustCompile(`\d{17}[\dXx]|1[3-9]\d{9}|\d{16,19}`)

// collectWholeTextExemptValues 整段规则（content/snippet/title）豁免时，
// 提取文本中全部数字敏感模式值（这些值在答案/SSE 层跳过脱敏，保留原文）
func collectWholeTextExemptValues(text string) []string {
	if text == "" {
		return nil
	}
	return wholeTextSensitiveRe.FindAllString(text, -1)
}

// replaceKeyValues 扫描文本中的"字段标签 + 分隔符"，仅替换其后的敏感值
// （Go 正则不支持 lookahead，故采用两次正则：定位标签 + 提取/截断值）
func replaceKeyValues(text, labels string, kind fieldValueKind, pattern string, sensitive map[string]string) string {
	labelRe := regexp.MustCompile(`(?i)(` + labels + `)\s*(?:[:：=]\s*|\s+)`)
	var sb strings.Builder
	last := 0
	for _, loc := range labelRe.FindAllStringIndex(text, -1) {
		sb.WriteString(text[last:loc[0]])
		labelPart := text[loc[0]:loc[1]]
		raw := extractFieldValue(text, loc[1], kind)
		if raw == "" {
			sb.WriteString(labelPart)
			last = loc[1]
			continue
		}
		maskedVal := applyMaskValue(pattern, raw)
		if maskedVal != raw {
			sensitive[raw] = maskedVal
		}
		sb.WriteString(labelPart)
		sb.WriteString(maskedVal)
		last = loc[1] + len(raw)
	}
	sb.WriteString(text[last:])
	return sb.String()
}

// extractFieldValue 从位置 start 提取字段值
// - 数字型：从 start 起必须紧邻匹配对应值正则（否则视为无值，跳过）
// - 文本型：完整提取到行尾/常见分隔符/下一个已知标签（不做长度截断，避免尾部原文泄露）
func extractFieldValue(text string, start int, kind fieldValueKind) string {
	rest := text[start:]
	switch kind {
	case valueKindIDCard:
		if m := idCardValueRe.FindStringIndex(rest); m != nil && m[0] == 0 {
			return rest[m[0]:m[1]]
		}
	case valueKindPhone:
		if m := phoneValueRe.FindStringIndex(rest); m != nil && m[0] == 0 {
			return rest[m[0]:m[1]]
		}
	case valueKindBankCard:
		if m := bankCardValueRe.FindStringIndex(rest); m != nil && m[0] == 0 {
			return rest[m[0]:m[1]]
		}
	case valueKindAmount:
		if m := amountValueRe.FindStringIndex(rest); m != nil && m[0] == 0 {
			return rest[m[0]:m[1]]
		}
	default: // 文本型（address/自定义字段）
		// 完整提取到字段终止边界（行尾/常见分隔符/下一个已知标签），不做长度截断：
		// 超长文本字段必须整体进入敏感值映射，否则只脱敏前缀会泄露中间/尾部原文。
		cut := len(rest)
		if idx := strings.IndexAny(rest, "\n，。；;,"); idx >= 0 && idx < cut {
			cut = idx
		}
		if m := allFieldLabelsRe.FindStringIndex(rest); m != nil && m[0] > 0 && m[0] < cut {
			cut = m[0]
		}
		return strings.TrimSpace(rest[:cut])
	}
	return ""
}

// amountValueRe 金额值正则（支持千分位与小数，如 45,000.50 / 45000.50）
var amountValueRe = regexp.MustCompile(`\d+(?:[.,]\d+)*`)

// fieldLabelPattern 生成字段标签正则（规则字段名 + 别名，按长度降序避免短标签吞前缀）
// 未登记别名的字段（自定义字段）直接使用字段名本身作为标签
func fieldLabelPattern(fieldName string) string {
	labels := fieldLabelAliases[fieldName]
	seen := make(map[string]bool, len(labels)+1)
	all := make([]string, 0, len(labels)+1)
	for _, l := range labels {
		key := strings.ToLower(l)
		if !seen[key] {
			seen[key] = true
			all = append(all, l)
		}
	}
	if !seen[strings.ToLower(fieldName)] {
		all = append(all, fieldName)
	}
	sort.Slice(all, func(i, j int) bool { return len([]rune(all[i])) > len([]rune(all[j])) })
	escaped := make([]string, len(all))
	for i, l := range all {
		escaped[i] = regexp.QuoteMeta(l)
	}
	return strings.Join(escaped, "|")
}

// valueGlobalRe 返回数字型强模式字段的全局值正则（其余字段返回 nil，不做全局替换）
func valueGlobalRe(kind fieldValueKind) *regexp.Regexp {
	switch kind {
	case valueKindIDCard:
		return idCardValueRe
	case valueKindPhone:
		return phoneValueRe
	case valueKindBankCard:
		return bankCardValueRe
	default:
		return nil
	}
}

// applyValueMap 将文本中的敏感原始值替换为脱敏值（长值优先，避免短值先替换误伤长值）
// 用于最终答案与 SSE 增量：模型若复述检索原文中的敏感值，在此被替换为脱敏形式
func applyValueMap(text string, sensitive map[string]string) string {
	if text == "" || len(sensitive) == 0 {
		return text
	}
	keys := make([]string, 0, len(sensitive))
	for k := range sensitive {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len([]rune(keys[i])) > len([]rune(keys[j])) })
	for _, k := range keys {
		text = strings.ReplaceAll(text, k, sensitive[k])
	}
	return text
}

// mergeSensitive 合并敏感值映射（后写覆盖）
func mergeSensitive(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}
