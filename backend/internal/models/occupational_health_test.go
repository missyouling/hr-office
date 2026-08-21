package models

import (
	"reflect"
	"testing"
)

func TestOccupationalHealthCheckJSONTags(t *testing.T) {
	record := OccupationalHealthCheck{}
	if got := record.SnapshotName; got != "" {
		t.Fatal("测试初始化异常")
	}
	if tag := getStructJSONTag(t, OccupationalHealthCheck{}, "SnapshotName"); tag != "employee_name" {
		t.Fatalf("SnapshotName 的 JSON 标签应为 employee_name，实际为 %s", tag)
	}
	if tag := getStructJSONTag(t, OccupationalHealthCheck{}, "SnapshotDepartment"); tag != "employee_department" {
		t.Fatalf("SnapshotDepartment 的 JSON 标签应为 employee_department，实际为 %s", tag)
	}
	if tag := getStructJSONTag(t, OccupationalHealthCheck{}, "SnapshotPosition"); tag != "employee_position" {
		t.Fatalf("SnapshotPosition 的 JSON 标签应为 employee_position，实际为 %s", tag)
	}
	if tag := getStructJSONTag(t, OccupationalHealthCheck{}, "CheckInstitution"); tag != "medical_institution" {
		t.Fatalf("CheckInstitution 的 JSON 标签应为 medical_institution，实际为 %s", tag)
	}
	if tag := getStructJSONTag(t, OccupationalHealthCheck{}, "Conclusion"); tag != "check_conclusion" {
		t.Fatalf("Conclusion 的 JSON 标签应为 check_conclusion，实际为 %s", tag)
	}
}

func TestOccupationalHealthCheckValidate(t *testing.T) {
	valid := OccupationalHealthCheck{
		EmployeeID:         1,
		CheckDate:          "2026-08-19",
		CheckInstitution:   "市职业病防治院",
		CheckCategory:      "上岗前",
		Status:             OccupationalHealthStatusDraft,
		SnapshotName:       "张三",
		SnapshotDepartment: "业务部",
		SnapshotPosition:   "专员",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效职业健康检查记录校验失败: %v", err)
	}

	cases := []OccupationalHealthCheck{
		{EmployeeID: 1, CheckDate: "2026-08-19", CheckInstitution: "机构", CheckCategory: "类别", Status: "bad"},
		{CheckDate: "2026-08-19", CheckInstitution: "机构", CheckCategory: "类别", Status: OccupationalHealthStatusDraft},
		{EmployeeID: 1, CheckInstitution: "机构", CheckCategory: "类别", Status: OccupationalHealthStatusDraft},
		{EmployeeID: 1, CheckDate: "2026/08/19", CheckInstitution: "机构", CheckCategory: "类别", Status: OccupationalHealthStatusDraft},
		{EmployeeID: 1, CheckDate: "2026-08-19", CheckInstitution: " ", CheckCategory: "类别", Status: OccupationalHealthStatusDraft},
		{EmployeeID: 1, CheckDate: "2026-08-19", CheckInstitution: "机构", CheckCategory: " ", Status: OccupationalHealthStatusDraft},
		{EmployeeID: 1, CheckDate: "2026-08-19", CheckInstitution: "机构", CheckCategory: "类别", NextCheckDate: "2026/09/01", Status: OccupationalHealthStatusDraft},
	}
	for _, record := range cases {
		if err := record.Validate(); err == nil {
			t.Error("无效职业健康检查记录应被拒绝")
		}
	}
}

func getStructJSONTag(t *testing.T, value any, fieldName string) string {
	t.Helper()
	field, ok := reflect.TypeOf(value).FieldByName(fieldName)
	if !ok {
		t.Fatalf("未找到字段 %s", fieldName)
	}
	return field.Tag.Get("json")
}
