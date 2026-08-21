package models

import (
	"errors"
	"strings"
	"time"
)

// 车队车辆生命周期状态常量。
const (
	FleetVehicleStatusActive   = "active"   // 在用
	FleetVehicleStatusInactive = "inactive" // 停用
)

// IsValidFleetVehicleStatus 校验车辆状态是否合法（仅 active/inactive）。
func IsValidFleetVehicleStatus(value string) bool {
	return value == FleetVehicleStatusActive || value == FleetVehicleStatusInactive
}

// FleetVehicle 车队车辆档案（P12 车队管理最小真实功能）。
//
// 已确认规则：
//   - 仅车辆档案，不做调度、加油、维修、审批、轨迹、附件、定时任务、员工联动；
//   - 登录态注入租户（UserID），按租户隔离；
//   - plate_number 必填且租户内唯一（复合唯一索引兜底）；
//   - status 必填，仅 active/inactive；active 可编辑/删除，inactive 仅可 PUT 恢复为 active；
//   - seat_count 可选正整数；purchase_date 可选 YYYY-MM-DD。
type FleetVehicle struct {
	ID     uint  `json:"id" gorm:"primaryKey"`
	UserID uint  `json:"-" gorm:"not null;uniqueIndex:idx_fleet_vehicle_user_plate"` // 租户隔离归属（仅由服务端从登录态注入）
	User   *User `json:"-" gorm:"foreignKey:UserID"`

	PlateNumber  string `json:"plate_number" gorm:"size:20;not null;uniqueIndex:idx_fleet_vehicle_user_plate"` // 车牌号（租户内唯一）
	VehicleModel string `json:"vehicle_model" gorm:"size:100;not null"`                                        // 车辆型号
	Status       string `json:"status" gorm:"size:20;not null;index;default:active"`                           // 状态（active/inactive）
	Brand        string `json:"brand" gorm:"size:100"`                                                         // 品牌（可选）
	SeatCount    *int   `json:"seat_count"`                                                                    // 座位数（可选，正整数）
	PurchaseDate string `json:"purchase_date" gorm:"size:10"`                                                  // 购置日期（可选，YYYY-MM-DD）
	Remarks      string `json:"remarks" gorm:"type:text"`                                                      // 备注（可选）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 校验车辆档案字段约束（数据底座层）。
func (v *FleetVehicle) Validate() error {
	if strings.TrimSpace(v.PlateNumber) == "" {
		return errors.New("车牌号必填")
	}
	if strings.TrimSpace(v.VehicleModel) == "" {
		return errors.New("车辆型号必填")
	}
	if !IsValidFleetVehicleStatus(v.Status) {
		return errors.New("无效的车辆状态")
	}
	if v.SeatCount != nil && *v.SeatCount <= 0 {
		return errors.New("座位数必须为正整数")
	}
	if v.PurchaseDate != "" {
		if _, err := time.Parse("2006-01-02", v.PurchaseDate); err != nil {
			return errors.New("购置日期格式必须为 YYYY-MM-DD")
		}
	}
	return nil
}
