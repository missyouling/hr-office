package service

import "regexp"

// ============================================================
// 敏感模式检测脱敏（答案/SSE 增量防御层，P9.2）
// ============================================================

var (
	// 身份证号：18 位（17 位数字 + 数字或 X/x）
	idCardRe = regexp.MustCompile(`\d{17}[\dXx]`)
	// 手机号：11 位，1 开头
	phoneRe = regexp.MustCompile(`1[3-9]\d{9}`)
	// 银行卡号：16-19 位连续数字
	bankCardRe = regexp.MustCompile(`\d{16,19}`)
)

// MaskSensitiveText 对文本中的敏感模式（身份证号/手机号/银行卡号）应用 front3back4 脱敏
// 纯函数不依赖数据库，用于 LLM 答案与 SSE 增量内容的防御性脱敏
func MaskSensitiveText(text string) string {
	return maskSensitiveTextExempt(text, nil)
}

// maskSensitiveTextExempt 同 MaskSensitiveText，但 exempt 中的原始值被跳过（保留原文）。
// 用于 ExemptRole 豁免：豁免字段的原文在答案/SSE 中必须保留，
// 不能被防御层二次脱敏（避免"先替换后无法恢复"），其他数字模式防御仍生效。
// 采用包含匹配：非豁免字段的正则可能命中豁免值内部子串
// （如手机号正则 1[3-9]\d{9} 命中身份证 110101199001011234 内的 19900101123），
// 这些子串同样跳过，保证豁免字段原文完整。
func maskSensitiveTextExempt(text string, exempt map[string]struct{}) string {
	if text == "" {
		return text
	}
	replace := func(raw string) string {
		if coveredByExempt(raw, exempt) {
			return raw
		}
		return applyFront3Back4(raw)
	}
	// 顺序：先替换身份证号（18 位），避免被银行卡号正则二次匹配
	text = idCardRe.ReplaceAllStringFunc(text, replace)
	text = phoneRe.ReplaceAllStringFunc(text, replace)
	text = bankCardRe.ReplaceAllStringFunc(text, replace)
	return text
}

// toExemptSet 将豁免原始值列表转为去重集合（供防御层跳过）
func toExemptSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

// streamMasker 流式增量脱敏器
// 维护尾部缓冲，处理跨 token 边界的敏感模式（身份证号/手机号/银行卡号）；
// sensitive 为 KB 字段规则提取的 原始值→脱敏值 映射，保证 SSE 增量
// 与最终答案使用同一套字段规则脱敏结果；
// exempt 为 ExemptRole 豁免的原始值集合，这些值在输出中保留原文（跳过防御层）。
type streamMasker struct {
	pending   string
	sensitive map[string]string
	exempt    map[string]struct{}
}

// maxSensitiveLen 最长数字敏感模式长度（银行卡号 19 位）。
const maxSensitiveLen = 19

// streamSensitiveBufferLen 计算流式安全缓冲长度：
// 数字模式至少保留 19 个字符；字段映射与豁免值按当前最长原始值保留，
// 确保超长文本字段的完整值始终留在缓冲内，避免跨 token 边界泄露。
func streamSensitiveBufferLen(sensitive map[string]string, exempt map[string]struct{}) int {
	bufferLen := maxSensitiveLen
	for raw := range sensitive {
		valueLen := len([]rune(raw))
		if valueLen > bufferLen {
			bufferLen = valueLen
		}
	}
	for raw := range exempt {
		valueLen := len([]rune(raw))
		if valueLen > bufferLen {
			bufferLen = valueLen
		}
	}
	return bufferLen
}

// safeStreamBoundary 将输出边界回退到敏感值之前，避免敏感值跨多个
// SSE（服务器推送事件）增量被拆开后重新拼成原文。
// 当边界切入完整敏感值，或左侧是敏感值前缀时，都必须回退。
func safeStreamBoundary(runes []rune, boundary int, sensitive map[string]string) int {
	if boundary <= 0 || len(sensitive) == 0 {
		return boundary
	}
	changed := true
	for changed && boundary > 0 {
		changed = false
		for raw := range sensitive {
			rawRunes := []rune(raw)
			if len(rawRunes) == 0 {
				continue
			}
			// 当前缓冲中已有完整敏感值且边界切入其中。
			for start := 0; start+len(rawRunes) <= len(runes); start++ {
				if start >= boundary || start+len(rawRunes) <= boundary {
					continue
				}
				if runesEqual(runes[start:start+len(rawRunes)], rawRunes) {
					boundary = start
					changed = true
					break
				}
			}
			if changed {
				break
			}
			// 当前缓冲尚未形成完整值，但安全区末尾是敏感值前缀。
			maxPrefix := len(rawRunes) - 1
			if maxPrefix > boundary {
				maxPrefix = boundary
			}
			for prefixLen := maxPrefix; prefixLen > 0; prefixLen-- {
				start := boundary - prefixLen
				if runesEqual(runes[start:boundary], rawRunes[:prefixLen]) {
					boundary = start
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
	}
	return boundary
}

func runesEqual(left, right []rune) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// Push 接收新 token，返回可安全输出的脱敏文本
// 尾部保留 maxSensitiveLen 字符作为缓冲；若缓冲前末尾是数字（可能是敏感模式前缀），
// 一并回退到上一个非数字字符，确保跨边界的敏感模式完整保留在缓冲中
func (m *streamMasker) Push(token string) string {
	m.pending += token
	runes := []rune(m.pending)
	bufferLen := streamSensitiveBufferLen(m.sensitive, m.exempt)
	if len(runes) <= bufferLen {
		return ""
	}

	// 可安全输出的部分：去掉动态敏感值缓冲
	safeLen := len(runes) - bufferLen
	safeLen = safeStreamBoundary(runes, safeLen, m.sensitive)
	// 若 safe 末尾是数字（可能是敏感模式前缀），回退到上一个非数字字符
	for safeLen > 0 && runes[safeLen-1] >= '0' && runes[safeLen-1] <= '9' {
		safeLen--
	}
	if safeLen == 0 {
		return ""
	}

	safe := maskSensitiveTextExempt(string(runes[:safeLen]), m.exempt)
	safe = applyValueMap(safe, m.sensitive)
	m.pending = string(runes[safeLen:])
	return safe
}

// Flush 输出剩余缓冲（整体脱敏）
func (m *streamMasker) Flush() string {
	out := maskSensitiveTextExempt(m.pending, m.exempt)
	out = applyValueMap(out, m.sensitive)
	m.pending = ""
	return out
}
