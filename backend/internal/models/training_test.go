package models

import "testing"

func TestTrainingRecordValidate(t *testing.T) {
	valid := TrainingRecord{Topic: "安全培训", TrainingType: TrainingTypeInternal, TrainingDate: "2026-08-19", Status: TrainingStatusDraft}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效培训记录校验失败: %v", err)
	}
	cases := []TrainingRecord{
		{Topic: "主题", TrainingType: "other", TrainingDate: "2026-08-19", Status: TrainingStatusDraft},
		{Topic: " ", TrainingType: TrainingTypeOnline, TrainingDate: "2026-08-19", Status: TrainingStatusDraft},
		{Topic: "主题", TrainingType: TrainingTypeExternal, TrainingDate: "2026/08/19", Status: TrainingStatusDraft},
		{Topic: "主题", TrainingType: TrainingTypeInternal, TrainingDate: "2026-08-19", Status: "pending"},
	}
	for _, record := range cases {
		if err := record.Validate(); err == nil {
			t.Error("无效培训记录应被拒绝")
		}
	}
}
