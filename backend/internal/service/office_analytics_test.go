package service

import (
	"testing"
)

// TestGetPeriodFilter_Monthly 测试月度过滤区间
func TestGetPeriodFilter_Monthly(t *testing.T) {
	pf := GetPeriodFilter("monthly", "2025-03")

	// 当前月：2025-03-01 ~ 2025-03-31
	if pf.Current.StartDate != "2025-03-01" {
		t.Errorf("期望 Current.StartDate 为 2025-03-01，实际为 %s", pf.Current.StartDate)
	}
	if pf.Current.EndDate != "2025-03-31" {
		t.Errorf("期望 Current.EndDate 为 2025-03-31，实际为 %s", pf.Current.EndDate)
	}
	// 上月：2025-02-01 ~ 2025-02-28
	if pf.Prev.StartDate != "2025-02-01" {
		t.Errorf("期望 Prev.StartDate 为 2025-02-01，实际为 %s", pf.Prev.StartDate)
	}
	if pf.Prev.EndDate != "2025-02-28" {
		t.Errorf("期望 Prev.EndDate 为 2025-02-28，实际为 %s", pf.Prev.EndDate)
	}
}

// TestGetPeriodFilter_Yearly 测试年度过滤区间
func TestGetPeriodFilter_Yearly(t *testing.T) {
	pf := GetPeriodFilter("yearly", "2025")

	if pf.Current.StartDate != "2025-01-01" {
		t.Errorf("期望 Current.StartDate 为 2025-01-01，实际为 %s", pf.Current.StartDate)
	}
	if pf.Current.EndDate != "2025-12-31" {
		t.Errorf("期望 Current.EndDate 为 2025-12-31，实际为 %s", pf.Current.EndDate)
	}
	if pf.Prev.StartDate != "2024-01-01" {
		t.Errorf("期望 Prev.StartDate 为 2024-01-01，实际为 %s", pf.Prev.StartDate)
	}
	if pf.Prev.EndDate != "2024-12-31" {
		t.Errorf("期望 Prev.EndDate 为 2024-12-31，实际为 %s", pf.Prev.EndDate)
	}
}

// TestGetPeriodFilter_HalfYearly 测试半年度过滤区间
func TestGetPeriodFilter_HalfYearly(t *testing.T) {
	// 上半年
	pf := GetPeriodFilter("half-yearly", "2025-3")
	if pf.Current.StartDate != "2025-01-01" {
		t.Errorf("上半年 Current.StartDate 期望 2025-01-01，实际 %s", pf.Current.StartDate)
	}
	if pf.Current.EndDate != "2025-06-30" {
		t.Errorf("上半年 Current.EndDate 期望 2025-06-30，实际 %s", pf.Current.EndDate)
	}
	if pf.Prev.StartDate != "2024-07-01" {
		t.Errorf("上半年 Prev.StartDate 期望 2024-07-01，实际 %s", pf.Prev.StartDate)
	}

	// 下半年
	pf2 := GetPeriodFilter("half-yearly", "2025-9")
	if pf2.Current.StartDate != "2025-07-01" {
		t.Errorf("下半年 Current.StartDate 期望 2025-07-01，实际 %s", pf2.Current.StartDate)
	}
	if pf2.Current.EndDate != "2025-12-31" {
		t.Errorf("下半年 Current.EndDate 期望 2025-12-31，实际 %s", pf2.Current.EndDate)
	}
	if pf2.Prev.StartDate != "2025-01-01" {
		t.Errorf("下半年 Prev.StartDate 期望 2025-01-01，实际 %s", pf2.Prev.StartDate)
	}
}

// TestGetPeriodFilter_Default 测试默认（无类型）也为月度
func TestGetPeriodFilter_Default(t *testing.T) {
	pf := GetPeriodFilter("", "2025-01")

	if pf.Current.StartDate != "2025-01-01" {
		t.Errorf("期望 Current.StartDate 为 2025-01-01，实际为 %s", pf.Current.StartDate)
	}
	if pf.Current.EndDate != "2025-01-31" {
		t.Errorf("期望 Current.EndDate 为 2025-01-31，实际为 %s", pf.Current.EndDate)
	}
}

// TestGenerateRequestNo 测试请款单号格式
func TestGenerateRequestNo(t *testing.T) {
	no := GenerateRequestNo()
	if len(no) == 0 {
		t.Fatal("请款单号不应为空")
	}
	if no[:3] != "PR-" {
		t.Errorf("请款单号应以 PR- 开头，实际为 %s", no)
	}
}

// TestAmountToCN 测试金额大写（复用已有 amount_cn.go）
func TestAmountToCN(t *testing.T) {
	result := AmountToCN(0)
	if result != "零元整" {
		t.Errorf("0 应转为 零元整，实际为 %s", result)
	}

	result = AmountToCN(1234.56)
	if len(result) == 0 {
		t.Fatal("金额大写不应为空")
	}
	// 应包含"元"、"角"、"分" 等关键字
	if !containsChineseUnit(result) {
		t.Errorf("金额大写格式异常: %s", result)
	}
}

func containsChineseUnit(s string) bool {
	units := []string{"元", "角", "分", "整", "零", "壹", "贰", "叁", "肆", "伍"}
	for _, u := range units {
		for _, r := range s {
			if string(r) == u {
				return true
			}
		}
	}
	return false
}
