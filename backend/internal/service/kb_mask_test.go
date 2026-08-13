package service

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// newMaskTestDB 创建仅含角色相关表的 SQLite 内存库（豁免查询测试用）
func newMaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.UserRole{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	return db
}

// maskRule 构造 KBFieldMask 测试对象
func maskRule(kbID uint, field, pattern string, exemptRole *string) *models.KBFieldMask {
	return &models.KBFieldMask{
		KnowledgeBaseID: kbID,
		FieldName:       field,
		MaskPattern:     pattern,
		ExemptRole:      exemptRole,
	}
}

func TestMaskFieldValueInText_KeyValue(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		text     string
		want     string
		wantHits int // 提取到的敏感值数量
	}{
		{
			name:     "中文标签冒号分隔",
			field:    "id_card",
			text:     "员工身份证号：110101199001011234，已登记",
			want:     "员工身份证号：110***********1234，已登记",
			wantHits: 1,
		},
		{
			name:     "英文标签等号分隔",
			field:    "id_card",
			text:     "id_card=110101199001011234",
			want:     "id_card=110***********1234",
			wantHits: 1,
		},
		{
			name:     "手机号空格分隔",
			field:    "phone",
			text:     "联系电话 13800138000 备注",
			want:     "联系电话 138****8000 备注",
			wantHits: 1,
		},
		{
			name:     "address 键值保留标签与上下文",
			field:    "address",
			text:     "家庭住址：北京市朝阳区望京SOHO，备注：正常",
			want:     "家庭住址：北京市*****SOHO，备注：正常",
			wantHits: 1,
		},
		{
			name:     "address 值遇到其他字段标签截断",
			field:    "address",
			text:     "地址：北京市朝阳区 电话：13800138000",
			want:     "地址：*** 电话：13800138000",
			wantHits: 1,
		},
		{
			name:     "金额只替换键值",
			field:    "amount",
			text:     "合同金额：45000.50元，其他数字 123456789012345678 保留",
			want:     "合同金额：450*0.50元，其他数字 123456789012345678 保留",
			wantHits: 1,
		},
		{
			name:     "自定义字段标签",
			field:    "employee_no",
			text:     "employee_no：A12345 归档",
			want:     "employee_no：A12**5 归档",
			wantHits: 1,
		},
		{
			name:     "无标签无模式保持原文",
			field:    "address",
			text:     "脱敏测试内容 无敏感信息",
			want:     "脱敏测试内容 无敏感信息",
			wantHits: 0,
		},
		{
			name:     "重复值多处替换且映射合并",
			field:    "phone",
			text:     "电话：13800138000 与 13800138000",
			want:     "电话：138****8000 与 138****8000",
			wantHits: 1, // 映射按原始值去重
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mask := maskRule(1, tt.field, "front3back4", nil)
			got, sensitive, _, err := maskFieldValueInText(nil, &models.User{ID: 9}, tt.text, tt.field, mask, nil)
			if err != nil {
				t.Fatalf("maskFieldValueInText 返回错误: %v", err)
			}
			if got != tt.want {
				t.Errorf("脱敏结果不匹配\n got: %s\nwant: %s", got, tt.want)
			}
			if len(sensitive) != tt.wantHits {
				t.Errorf("敏感值映射数量不匹配: got %d, want %d (映射=%v)", len(sensitive), tt.wantHits, sensitive)
			}
		})
	}
}

func TestMaskFieldValueInText_GlobalFallback(t *testing.T) {
	// 无标签的裸身份证号：数字型强模式字段全局替换兜底
	mask := maskRule(1, "id_card", "front3back4", nil)
	text := "脱敏测试内容 110101199001011234"
	got, sensitive, _, err := maskFieldValueInText(nil, &models.User{ID: 9}, text, "id_card", mask, nil)
	if err != nil {
		t.Fatalf("maskFieldValueInText 返回错误: %v", err)
	}
	want := "脱敏测试内容 110***********1234"
	if got != want {
		t.Errorf("全局兜底替换不匹配\n got: %s\nwant: %s", got, want)
	}
	if len(sensitive) != 1 {
		t.Errorf("应提取 1 个敏感值, got %d (%v)", len(sensitive), sensitive)
	}
}

func TestMaskFieldValueInText_AllStar(t *testing.T) {
	mask := maskRule(1, "address", "all_star", nil)
	got, sensitive, _, err := maskFieldValueInText(nil, &models.User{ID: 9}, "地址：北京市朝阳区望京SOHO", "address", mask, nil)
	if err != nil {
		t.Fatalf("maskFieldValueInText 返回错误: %v", err)
	}
	want := "地址：***"
	if got != want {
		t.Errorf("all_star 不匹配\n got: %s\nwant: %s", got, want)
	}
	if sensitive["北京市朝阳区望京SOHO"] != "***" {
		t.Errorf("敏感值映射应包含 北京市朝阳区望京SOHO→***, got %v", sensitive)
	}
}

func TestExtractFieldValue_LongTextFullExtract(t *testing.T) {
	// 中文+ASCII 混合超长字段（远超 60 字节）：必须完整提取到终止边界，不得按字节截断
	const mixed = "北京市朝阳区望京街道SOHO塔楼A座第66层6601室内部编号XZ8899"
	text := "家庭住址：" + mixed + "，备注：正常"
	got := extractFieldValue(text, len("家庭住址："), valueKindText)
	if got != mixed {
		t.Fatalf("extractFieldValue 未完整提取超长字段\n got: %q\nwant: %q", got, mixed)
	}
}

func TestMaskFieldValueInText_LongTextFieldNoTailLeak(t *testing.T) {
	// 中文+ASCII 混合超长字段：完整映射后，结果不得泄露中间/尾部原文
	const mixed = "北京市朝阳区望京街道SOHO塔楼A座第66层6601室内部编号XZ8899"
	mask := maskRule(1, "address", "front3back4", nil)
	got, sensitive, _, err := maskFieldValueInText(nil, &models.User{ID: 9},
		"家庭住址："+mixed+"，备注：正常", "address", mask, nil)
	if err != nil {
		t.Fatalf("maskFieldValueInText 返回错误: %v", err)
	}
	// 完整值进入映射（不再是截断后的前缀）
	if _, ok := sensitive[mixed]; !ok {
		t.Fatalf("敏感值映射应包含完整原文 %q，实际映射=%v", mixed, sensitive)
	}
	// 无中间/尾部原文泄露
	if strings.Contains(got, mixed) {
		t.Errorf("脱敏结果泄露完整原文: %q", got)
	}
	if strings.Contains(got, "内部编号XZ") {
		t.Errorf("脱敏结果泄露中间/尾部原文: %q", got)
	}
	// front3back4：保留前3字符（北京市）与后4字符（ASCII 尾部 8899）
	if !strings.HasPrefix(got, "家庭住址：北京市") {
		t.Errorf("脱敏结果应保留前3字符，实际: %q", got)
	}
	if !strings.Contains(got, "8899") {
		t.Errorf("脱敏结果应保留后4字符 8899，实际: %q", got)
	}
}

func TestMaskFieldValueInText_ExemptRoleQueryFailure(t *testing.T) {
	// ExemptRole 非空 + roles 查询注入失败 → 安全失败返回错误，不得降级返回原文
	db := newMaskTestDB(t)
	exemptRole := "admin"
	mask := maskRule(1, "address", "front3back4", &exemptRole)
	db.Callback().Query().Before("gorm:query").Register("fail_role_query", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "roles" {
			tx.AddError(errors.New("注入角色查询失败"))
		}
	})
	text := "地址：北京市朝阳区望京SOHO"
	_, _, _, err := maskFieldValueInText(db, &models.User{ID: 9}, text, "address", mask, nil)
	if err == nil {
		t.Fatal("豁免角色查询失败应返回错误（安全失败）")
	}
}

func TestApplyValueMap(t *testing.T) {
	sensitive := map[string]string{
		"13800138000": "138****8000",
		"北京市朝阳区":      "北京市*****",
	}
	got := applyValueMap("联系电话13800138000，地址北京市朝阳区", sensitive)
	want := "联系电话138****8000，地址北京市*****"
	if got != want {
		t.Errorf("applyValueMap 不匹配\n got: %s\nwant: %s", got, want)
	}
	if got := applyValueMap("", sensitive); got != "" {
		t.Errorf("空文本应返回空")
	}
	if got := applyValueMap("无敏感", nil); got != "无敏感" {
		t.Errorf("空映射应返回原文")
	}
}

func TestApplyFront3Back4(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "***"},   // 空串也走默认
		{"张三", "***"}, // 短于 7 全星
		{"13800138000", "138****8000"},
		{"北京市朝阳区望京SOHO", "北京市*****SOHO"},
		{"abcdefghij", "abc***ghij"},
	}
	for _, tt := range tests {
		if got := applyFront3Back4(tt.in); got != tt.want {
			t.Errorf("applyFront3Back4(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMaskSensitiveText(t *testing.T) {
	text := "电话 13800138000，身份证 110101199001011234，卡号 6222020202020202020"
	got := MaskSensitiveText(text)
	if got == text {
		t.Fatal("MaskSensitiveText 未做任何替换")
	}
	if containsAny(got, []string{"13800138000", "110101199001011234", "6222020202020202020"}) {
		t.Errorf("MaskSensitiveText 泄露原文: %s", got)
	}
	if !containsAll(got, []string{"电话", "身份证", "卡号"}) {
		t.Errorf("MaskSensitiveText 丢失非敏感上下文: %s", got)
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if sub != "" && containsStr(s, sub) {
			return true
		}
	}
	return false
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !containsStr(s, sub) {
			return false
		}
	}
	return true
}

func containsStr(s, sub string) bool {
	return len(sub) > 0 && (len(s) == len(sub) && s == sub || len(s) > len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestStreamMasker(t *testing.T) {
	m := &streamMasker{sensitive: map[string]string{"13800138000": "138****8000"}}
	// 跨 token 边界：数字被拆到两批
	out1 := m.Push("联系电话1380")
	out2 := m.Push("0138000确认")
	flush := m.Flush()
	joined := out1 + out2 + flush
	if containsStr(joined, "13800138000") {
		t.Errorf("流式输出泄露手机号: %q", joined)
	}
	if !containsStr(joined, "138****8000") {
		t.Errorf("流式输出应包含脱敏手机号: %q", joined)
	}
}
