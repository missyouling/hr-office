package service

import (
	"strings"
	"testing"
)

func assertStreamNeverLeaks(t *testing.T, outputs []string, joined string, rawValues ...string) {
	t.Helper()
	for i, output := range outputs {
		for _, raw := range rawValues {
			if strings.Contains(output, raw) {
				t.Errorf("第 %d 个已输出片段泄露原文 %q: %q", i, raw, output)
			}
		}
	}
	for _, raw := range rawValues {
		if strings.Contains(joined, raw) {
			t.Errorf("拼接后的流式输出泄露原文 %q: %q", raw, joined)
		}
	}
}

func TestStreamMasker_LongAddress(t *testing.T) {
	raw := "北京市朝阳区望京街道望京SOHO塔楼A座第六十六层6601室"
	masked := "北京市***************6601室"
	m := &streamMasker{sensitive: map[string]string{raw: masked}}

	outputs := []string{
		m.Push("地址：" + raw[:20]),
		m.Push(raw[20:45]),
		m.Push(raw[45:] + "，已核验"),
		m.Flush(),
	}
	joined := strings.Join(outputs, "")
	assertStreamNeverLeaks(t, outputs, joined, raw)
	if !strings.Contains(joined, masked) {
		t.Fatalf("长地址应输出脱敏值 %q，实际: %q", masked, joined)
	}
}

func TestStreamMasker_LongCustomField(t *testing.T) {
	raw := "CUSTOM-员工档案-北京市朝阳区望京街道SOHO-A座-6601-内部编号-2026"
	m := &streamMasker{sensitive: map[string]string{raw: "***"}}

	outputs := []string{
		m.Push("自定义字段：" + raw[:15]),
		m.Push(raw[15:38]),
		m.Push(raw[38:] + "；后续内容"),
		m.Flush(),
	}
	joined := strings.Join(outputs, "")
	assertStreamNeverLeaks(t, outputs, joined, raw)
	if !strings.Contains(joined, "***") {
		t.Fatalf("自定义字段应输出脱敏值，实际: %q", joined)
	}
}

func TestStreamMasker_AllStarLongValue(t *testing.T) {
	raw := "这是一个长度超过十九字符的地址字段敏感值用于验证全星脱敏"
	m := &streamMasker{sensitive: map[string]string{raw: "***"}}

	outputs := []string{
		m.Push("地址：" + raw[:10]),
		m.Push(raw[10:] + "，备注保留"),
		m.Flush(),
	}
	joined := strings.Join(outputs, "")
	assertStreamNeverLeaks(t, outputs, joined, raw)
	if !strings.Contains(joined, "***") {
		t.Fatalf("all_star 应输出 ***，实际: %q", joined)
	}
}

func TestStreamMasker_AdjacentAndContainedValues(t *testing.T) {
	short := "北京市朝阳区"
	long := "北京市朝阳区望京SOHO"
	m := &streamMasker{sensitive: map[string]string{
		short: "***",
		long:  "北京市*****SOHO",
	}}

	outputs := []string{
		m.Push("地址1：" + long[:8]),
		m.Push(long[8:] + " 地址2：" + short),
		m.Flush(),
	}
	joined := strings.Join(outputs, "")
	assertStreamNeverLeaks(t, outputs, joined, short, long)
	if !strings.Contains(joined, "北京市*****SOHO") || !strings.Contains(joined, "***") {
		t.Fatalf("相邻/包含敏感值未按映射脱敏: %q", joined)
	}
}

func TestStreamSensitiveBufferLen(t *testing.T) {
	if got := streamSensitiveBufferLen(nil, nil); got != maxSensitiveLen {
		t.Fatalf("无字段映射时缓冲长度 = %d，期望 %d", got, maxSensitiveLen)
	}
	// 完整提取后缓冲长度应跟随最长字段映射值（不再受 60 字节截断限制）
	longRaw := strings.Repeat("长", 40)
	longerRaw := strings.Repeat("超", 80)
	got := streamSensitiveBufferLen(map[string]string{longRaw: "***", longerRaw: "***"}, nil)
	if got != 80 {
		t.Fatalf("字段映射缓冲长度 = %d，期望跟随最长值 80", got)
	}
	// 豁免值同样计入缓冲长度（保证完整留在缓冲内，避免被切开后误伤）
	exempt := map[string]struct{}{strings.Repeat("免", 100): {}}
	if got := streamSensitiveBufferLen(nil, exempt); got != 100 {
		t.Fatalf("豁免值缓冲长度 = %d，期望跟随最长豁免值 100", got)
	}
}

func TestMaskSensitiveTextExempt(t *testing.T) {
	// 豁免值整体（身份证）+ 非豁免值（手机号）：豁免保留原文，非豁免仍脱敏
	exempt := toExemptSet([]string{"110101199001011234", "110101199001011235"})
	text := "员工身份证号是 110101199001011234，电话 13800138000"
	got := maskSensitiveTextExempt(text, exempt)
	if !strings.Contains(got, "110101199001011234") {
		t.Errorf("豁免的身份证号应保留原文，实际: %q", got)
	}
	if strings.Contains(got, "13800138000") {
		t.Errorf("非豁免的手机号应被脱敏，实际: %q", got)
	}
	if !strings.Contains(got, "138****8000") {
		t.Errorf("手机号应按 front3back4 脱敏为 138****8000，实际: %q", got)
	}
}

func TestMaskSensitiveTextExempt_InternalSubstring(t *testing.T) {
	// 关键回归：非豁免字段的正则命中豁免值内部子串时必须跳过。
	// 手机号正则 1[3-9]\d{9} 会命中身份证 110101199001011234 内部的
	// 19900101123，若精确匹配判断则会破坏豁免原文。
	exempt := toExemptSet([]string{"110101199001011234"})
	text := "身份证 110101199001011234 已核验"
	got := maskSensitiveTextExempt(text, exempt)
	if got != text {
		t.Errorf("豁免身份证内部子串不得被非豁免正则破坏\n got: %q\nwant: %q", got, text)
	}
}

func TestCoveredByExempt(t *testing.T) {
	exemptSet := map[string]struct{}{
		"110101199001011234": {},
		"13800138000":        {},
	}
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"完全等于豁免值", "110101199001011234", true},
		{"豁免值内部子串", "19900101123", true},
		{"豁免值前缀", "1101011990010", true},
		{"无关数字", "6222020202020202020", false},
		{"空串", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := coveredByExempt(tt.raw, exemptSet); got != tt.want {
				t.Errorf("coveredByExempt(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
	if coveredByExempt("123", nil) {
		t.Error("空豁免集合应返回 false")
	}
}

func TestMaskSensitiveTextExempt_NonExemptStillMasked(t *testing.T) {
	// 豁免集合不影响其他模式：银行卡号/身份证（非豁免）防御仍生效
	exempt := toExemptSet([]string{"13800138000"}) // 仅豁免手机号
	text := "卡号 6222020202020202020，手机 13800138000"
	got := maskSensitiveTextExempt(text, exempt)
	if strings.Contains(got, "6222020202020202020") {
		t.Errorf("非豁免银行卡号应被脱敏，实际: %q", got)
	}
	if !strings.Contains(got, "13800138000") {
		t.Errorf("豁免手机号应保留原文，实际: %q", got)
	}
}
