package models

import "testing"

func TestPersonnelChangeValidate(t *testing.T) {
	valid := PersonnelChange{EmployeeID: 1, ChangeType: PersonnelChangeTypeTransfer, EffectiveDate: "2026-08-19", Reason: "业务调整", Status: PersonnelChangeStatusDraft}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效记录校验失败: %v", err)
	}
	cases := []PersonnelChange{
		{EmployeeID: 1, ChangeType: "other", EffectiveDate: "2026-08-19", Reason: "原因", Status: PersonnelChangeStatusDraft},
		{EmployeeID: 1, ChangeType: PersonnelChangeTypeTransfer, EffectiveDate: "2026/08/19", Reason: "原因", Status: PersonnelChangeStatusDraft},
		{EmployeeID: 1, ChangeType: PersonnelChangeTypeTransfer, EffectiveDate: "2026-08-19", Reason: " ", Status: PersonnelChangeStatusDraft},
		{EmployeeID: 0, ChangeType: PersonnelChangeTypeTransfer, EffectiveDate: "2026-08-19", Reason: "原因", Status: PersonnelChangeStatusDraft},
	}
	for _, record := range cases {
		if err := record.Validate(); err == nil {
			t.Error("无效记录应被拒绝")
		}
	}
}
