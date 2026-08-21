package models

import "testing"

// TestSafetyInspectionValidate 字段约束校验：正常通过，异常边界逐项拒绝。
func TestSafetyInspectionValidate(t *testing.T) {
	valid := SafetyInspection{
		InspectionType:           SafetyInspectionTypeRoutine,
		InspectionDate:           "2026-08-19",
		Location:                 "一号车间",
		ResponsiblePerson:        "张三",
		IssueDescription:         "消防通道堆放杂物",
		RectificationRequirement: "立即清理并保持畅通",
		Status:                   SafetyInspectionStatusDraft,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效记录校验失败: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*SafetyInspection)
	}{
		{"类型非法", func(s *SafetyInspection) { s.InspectionType = "other" }},
		{"状态非法", func(s *SafetyInspection) { s.Status = "pending" }},
		{"日期缺失", func(s *SafetyInspection) { s.InspectionDate = "" }},
		{"日期格式错误", func(s *SafetyInspection) { s.InspectionDate = "2026/08/19" }},
		{"日期非法值", func(s *SafetyInspection) { s.InspectionDate = "2026-13-01" }},
		{"地点缺失", func(s *SafetyInspection) { s.Location = " " }},
		{"责任人缺失", func(s *SafetyInspection) { s.ResponsiblePerson = "" }},
		{"问题描述缺失", func(s *SafetyInspection) { s.IssueDescription = " " }},
		{"整改要求缺失", func(s *SafetyInspection) { s.RectificationRequirement = "" }},
	}
	for _, c := range cases {
		record := valid
		c.mutate(&record)
		if err := record.Validate(); err == nil {
			t.Errorf("%s 应被拒绝", c.name)
		}
	}
}
