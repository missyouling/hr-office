package models

import (
	"time"

	"gorm.io/datatypes"
)

// CanteenCategory 食堂食材分类表
type CanteenCategory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CanteenCategory) TableName() string { return "canteen_categories" }

// CanteenSupply 食堂食材字典表
type CanteenSupply struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	UserID         *uint            `json:"user_id" gorm:"index"`
	User           *User            `json:"-" gorm:"foreignKey:UserID"`
	Name           string           `json:"name" gorm:"not null;index;uniqueIndex:idx_canteen_supplies_name_cat,priority:1"`
	Spec           string           `json:"spec"`
	Unit           string           `json:"unit" gorm:"default:斤"`
	ReferencePrice float64          `json:"reference_price" gorm:"default:0"`
	CategoryID     *uint            `json:"category_id" gorm:"uniqueIndex:idx_canteen_supplies_name_cat,priority:2"`
	Category       *CanteenCategory `json:"-" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Status         string           `json:"status" gorm:"default:active"`
	Remark         string           `json:"remark"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func (CanteenSupply) TableName() string { return "canteen_supplies" }

// CanteenExpenseCategory 食堂费用科目字典表
type CanteenExpenseCategory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CanteenExpenseCategory) TableName() string { return "canteen_expense_categories" }

// CanteenPurchase 食堂采购主表（供应商关联办公用品供应商表 office_suppliers）
type CanteenPurchase struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	UserID       *uint           `json:"user_id" gorm:"index"`
	User         *User           `json:"-" gorm:"foreignKey:UserID"`
	OrderNo      string          `json:"order_no" gorm:"uniqueIndex;not null"`
	PurchaseDate time.Time       `json:"purchase_date" gorm:"not null"`
	TotalAmount  float64         `json:"total_amount" gorm:"default:0"`
	SupplierID   *uint           `json:"supplier_id"`
	Supplier     *OfficeSupplier `json:"-" gorm:"foreignKey:SupplierID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	SupplierName string          `json:"supplier_name"`
	Channel      string          `json:"channel"`
	ActualPay    float64         `json:"actual_pay" gorm:"default:0"`
	Remark       string          `json:"remark"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (CanteenPurchase) TableName() string { return "canteen_purchases" }

// CanteenPurchaseItem 食堂采购明细表
type CanteenPurchaseItem struct {
	ID         uint             `json:"id" gorm:"primaryKey"`
	UserID     *uint            `json:"user_id" gorm:"index"`
	User       *User            `json:"-" gorm:"foreignKey:UserID"`
	PurchaseID uint             `json:"purchase_id" gorm:"not null;index"`
	Purchase   *CanteenPurchase `json:"-" gorm:"foreignKey:PurchaseID;constraint:OnDelete:CASCADE"`
	SupplyID   uint             `json:"supply_id" gorm:"not null;index"`
	Supply     *CanteenSupply   `json:"-" gorm:"foreignKey:SupplyID;constraint:OnDelete:RESTRICT"`
	Quantity   float64          `json:"quantity" gorm:"not null;default:0"`
	UnitPrice  float64          `json:"unit_price" gorm:"not null;default:0"`
	Subtotal   float64          `json:"subtotal" gorm:"not null;default:0"`
	Remark     string           `json:"remark"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

func (CanteenPurchaseItem) TableName() string { return "canteen_purchase_items" }

// CanteenOtherExpense 食堂其他费用表（水电气、工资、设备维护等）
type CanteenOtherExpense struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       *uint     `json:"user_id" gorm:"index"`
	User         *User     `json:"-" gorm:"foreignKey:UserID"`
	ExpenseDate  time.Time `json:"expense_date" gorm:"not null;index"`
	Category     string    `json:"category" gorm:"not null"`
	Amount       float64   `json:"amount" gorm:"default:0"`
	ActualAmount float64   `json:"actual_amount" gorm:"default:0"`
	Params       string    `json:"params"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (CanteenOtherExpense) TableName() string { return "canteen_other_expenses" }

// CanteenDailyIncome 食堂每日收入表
type CanteenDailyIncome struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          *uint     `json:"user_id" gorm:"index"`
	User            *User     `json:"-" gorm:"foreignKey:UserID"`
	IncomeDate      time.Time `json:"income_date" gorm:"uniqueIndex;not null"`
	BreakfastCount  int       `json:"breakfast_count" gorm:"default:0"`
	BreakfastAmount float64   `json:"breakfast_amount" gorm:"default:0"`
	LunchCount      int       `json:"lunch_count" gorm:"default:0"`
	LunchAmount     float64   `json:"lunch_amount" gorm:"default:0"`
	DinnerCount     int       `json:"dinner_count" gorm:"default:0"`
	DinnerAmount    float64   `json:"dinner_amount" gorm:"default:0"`
	TotalCount      int       `json:"total_count" gorm:"default:0"`
	TotalAmount     float64   `json:"total_amount" gorm:"default:0"`
	Remark          string    `json:"remark"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (CanteenDailyIncome) TableName() string { return "canteen_daily_income" }

// CanteenResourceFee 食堂资源占用费收取表
type CanteenResourceFee struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	FeeDate   time.Time `json:"fee_date" gorm:"not null;index"`
	MealType  string    `json:"meal_type" gorm:"default:午餐"`
	Amount    float64   `json:"amount" gorm:"default:0"`
	Payer     string    `json:"payer"`
	Reason    string    `json:"reason"`
	Remark    string    `json:"remark"`
	Handler   string    `json:"handler"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CanteenResourceFee) TableName() string { return "canteen_resource_fees" }

// CanteenWeeklyMenu 食堂每周菜单表
type CanteenWeeklyMenu struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        *uint     `json:"user_id" gorm:"index"`
	User          *User     `json:"-" gorm:"foreignKey:UserID"`
	WeekStartDate string    `json:"week_start_date" gorm:"not null;uniqueIndex:idx_weekly_menu_week_day_meal,priority:1"`
	DayOfWeek     int       `json:"day_of_week" gorm:"not null;uniqueIndex:idx_weekly_menu_week_day_meal,priority:2"`
	MealType      string    `json:"meal_type" gorm:"not null;uniqueIndex:idx_weekly_menu_week_day_meal,priority:3"`
	Dishes        string    `json:"dishes"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CanteenWeeklyMenu) TableName() string { return "canteen_weekly_menu" }

// CanteenMenuTemplate 食堂菜单模板表
type CanteenMenuTemplate struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    *uint          `json:"user_id" gorm:"index"`
	User      *User          `json:"-" gorm:"foreignKey:UserID"`
	Name      string         `json:"name" gorm:"uniqueIndex;not null"`
	Data      datatypes.JSON `json:"data" gorm:"not null;type:json"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (CanteenMenuTemplate) TableName() string { return "canteen_menu_templates" }

// CanteenCardRecharge 饭卡充值记录表（CSV导入，external_sn 唯一去重）
type CanteenCardRecharge struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	UserID          *uint      `json:"user_id" gorm:"index"`
	User            *User      `json:"-" gorm:"foreignKey:UserID"`
	ExternalSN      *string    `json:"external_sn" gorm:"uniqueIndex"`
	CardNo          string     `json:"card_no"`
	CardUserID      string     `json:"card_user_id" gorm:"column:card_user_id"` // 饭卡系统员工编号（原 user_id 列重命名以避让多租户外键）
	UserName        string     `json:"user_name" gorm:"not null"`
	DepartmentCode  string     `json:"department_code"`
	UserDepartment  string     `json:"user_department"`
	RechargeDate    *time.Time `json:"recharge_date"`
	Amount          float64    `json:"amount" gorm:"default:0"`
	BalanceRecorded *float64   `json:"balance_recorded"`
	PaymentMethod   string     `json:"payment_method" gorm:"default:现金"`
	Operator        string     `json:"operator" gorm:"default:导入"`
	MachineNo       string     `json:"machine_no"`
	BillNo          string     `json:"bill_no"`
	Remark          string     `json:"remark"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (CanteenCardRecharge) TableName() string { return "canteen_card_recharges" }

// CanteenCardRefund 饭卡退费记录表（CSV导入，external_sn 唯一去重）
type CanteenCardRefund struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	UserID          *uint      `json:"user_id" gorm:"index"`
	User            *User      `json:"-" gorm:"foreignKey:UserID"`
	ExternalSN      *string    `json:"external_sn" gorm:"uniqueIndex"`
	CardNo          string     `json:"card_no"`
	CardUserID      string     `json:"card_user_id" gorm:"column:card_user_id"` // 饭卡系统员工编号（原 user_id 列重命名以避让多租户外键）
	UserName        string     `json:"user_name" gorm:"not null"`
	DepartmentCode  string     `json:"department_code"`
	UserDepartment  string     `json:"user_department"`
	RefundDate      *time.Time `json:"refund_date"`
	Amount          float64    `json:"amount" gorm:"default:0"`
	BalanceRecorded *float64   `json:"balance_recorded"`
	Operator        string     `json:"operator" gorm:"default:导入"`
	MachineNo       string     `json:"machine_no"`
	BillNo          string     `json:"bill_no"`
	Remark          string     `json:"remark"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (CanteenCardRefund) TableName() string { return "canteen_card_refunds" }
