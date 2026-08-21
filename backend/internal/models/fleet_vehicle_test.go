package models

import "testing"

// 测试用正整数座位数辅助构造。
func intPtr(value int) *int {
	return &value
}

// TestFleetVehicleValidate 字段约束校验：正常通过，异常边界逐项拒绝。
func TestFleetVehicleValidate(t *testing.T) {
	valid := FleetVehicle{
		PlateNumber:  "京A12345",
		VehicleModel: "丰田考斯特",
		Status:       FleetVehicleStatusActive,
		Brand:        "丰田",
		SeatCount:    intPtr(19),
		PurchaseDate: "2024-05-01",
		Remarks:      "商务接待用车",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("有效记录校验失败: %v", err)
	}

	// 最小必填场景：仅 plate_number / vehicle_model / status
	minimal := FleetVehicle{
		PlateNumber:  "京B67890",
		VehicleModel: "别克GL8",
		Status:       FleetVehicleStatusInactive,
	}
	if err := minimal.Validate(); err != nil {
		t.Fatalf("最小必填场景校验失败: %v", err)
	}

	// 可选字段为空字符串/空指针均可通过（purchase_date 空表示未设置）
	emptyOptional := FleetVehicle{
		PlateNumber:  "京C13579",
		VehicleModel: "金杯面包",
		Status:       FleetVehicleStatusActive,
		Brand:        "  ",
		PurchaseDate: "",
		Remarks:      "",
	}
	if err := emptyOptional.Validate(); err != nil {
		t.Fatalf("可选字段为空时校验失败: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*FleetVehicle)
	}{
		{"车牌缺失", func(v *FleetVehicle) { v.PlateNumber = " " }},
		{"车辆型号缺失", func(v *FleetVehicle) { v.VehicleModel = "" }},
		{"状态非法", func(v *FleetVehicle) { v.Status = "pending" }},
		{"状态缺失", func(v *FleetVehicle) { v.Status = "" }},
		{"座位数为零", func(v *FleetVehicle) { v.SeatCount = intPtr(0) }},
		{"座位数为负", func(v *FleetVehicle) { v.SeatCount = intPtr(-1) }},
		{"购置日期格式错误", func(v *FleetVehicle) { v.PurchaseDate = "2024/05/01" }},
		{"购置日期非法值", func(v *FleetVehicle) { v.PurchaseDate = "2024-13-01" }},
	}
	for _, c := range cases {
		record := valid
		c.mutate(&record)
		if err := record.Validate(); err == nil {
			t.Errorf("%s 应被拒绝", c.name)
		}
	}
}

// TestIsValidFleetVehicleStatus 状态枚举校验。
func TestIsValidFleetVehicleStatus(t *testing.T) {
	if !IsValidFleetVehicleStatus(FleetVehicleStatusActive) {
		t.Error("active 应为合法状态")
	}
	if !IsValidFleetVehicleStatus(FleetVehicleStatusInactive) {
		t.Error("inactive 应为合法状态")
	}
	for _, invalid := range []string{"", "pending", "archived", "ACTIVE"} {
		if IsValidFleetVehicleStatus(invalid) {
			t.Errorf("%q 应为非法状态", invalid)
		}
	}
}
