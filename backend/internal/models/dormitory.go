package models

import (
	"time"

	"gorm.io/datatypes"
)

// Charge configuration types
type SiteChargeConfig struct {
	Items []ChargeItemConfig `json:"items"`
}

type ChargeItemConfig struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Enabled   bool    `json:"enabled"`
	UnitPrice float64 `json:"unit_price"`
	UnitLabel string  `json:"unit_label"`
	Mode      string  `json:"mode"` // meter/fixed
}

type RoomChargeRates struct {
	Items []ChargeRateEntry `json:"items"`
}

type ChargeRateEntry struct {
	Key       string  `json:"key"`
	UnitPrice float64 `json:"unit_price"`
	UnitLabel string  `json:"unit_label"`
	Mode      string  `json:"mode"`
}

type MeterChargeDetail struct {
	Key       string   `json:"key"`
	Label     string   `json:"label"`
	Start     *float64 `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	Usage     *float64 `json:"usage,omitempty"`
	UnitPrice *float64 `json:"unit_price,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
}

// DormSite represents a dormitory location/site
type DormSite struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          *uint          `json:"user_id" gorm:"index"`
	User            *User          `json:"-,omitempty" gorm:"foreignKey:UserID"`
	Name            string         `json:"name"`
	Address         string         `json:"address"`
	ContactName     string         `json:"contact_name"`
	ContactPhone    string         `json:"contact_phone"`
	BuildingNumber  string         `json:"building_number"`
	PropertyCompany string         `json:"property_company"`
	PropertyContact string         `json:"property_contact"`
	SupportWechat   string         `json:"support_wechat"`
	Description     string         `json:"description"`
	ChargeConfig    datatypes.JSON `json:"charge_config" gorm:"type:json"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Buildings       []DormBuilding `json:"buildings,omitempty" gorm:"foreignKey:SiteID;constraint:OnDelete:CASCADE;"`
}

// DormBuilding represents a building under a site
type DormBuilding struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      *uint      `json:"user_id" gorm:"index"`
	User        *User      `json:"-,omitempty" gorm:"foreignKey:UserID"`
	SiteID      uint       `json:"site_id" gorm:"index"`
	Site        *DormSite  `json:"-"`
	Name        string     `json:"name"`
	Floors      int        `json:"floors"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Rooms       []DormRoom `json:"rooms,omitempty" gorm:"foreignKey:BuildingID;constraint:OnDelete:CASCADE;"`
}

// DormRoom represents a room within a building
type DormRoom struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	UserID          *uint           `json:"user_id" gorm:"index"`
	User            *User           `json:"-,omitempty" gorm:"foreignKey:UserID"`
	BuildingID      uint            `json:"building_id" gorm:"index"`
	Building        *DormBuilding   `json:"-"`
	SiteID          *uint           `json:"site_id" gorm:"index"`
	RoomNumber      string          `json:"room_number"`
	RoomType        string          `json:"room_type"`
	RoomCategory    string          `json:"room_category"`
	HouseLayout     string          `json:"house_layout"`
	BedCount        int             `json:"bed_count"`
	AreaSquare      float64         `json:"area_square"`
	FirstMonthFee   float64         `json:"first_month_fee"`
	MonthlyRent     float64         `json:"monthly_rent"`
	QuarterlyRent   float64         `json:"quarterly_rent"`
	PropertyFee     float64         `json:"property_fee"`
	GuaranteeFee    float64         `json:"guarantee_fee"`
	DepositFee      float64         `json:"deposit_fee"`
	WaterBase       float64         `json:"water_base"`
	ElectricBase    float64         `json:"electric_base"`
	GasBase         float64         `json:"gas_base"`
	TrashFee        float64         `json:"trash_fee"`
	WaterSupplyFee  float64         `json:"water_supply_fee"`
	SewageFee       float64         `json:"sewage_fee"`
	InventoryNote   string          `json:"inventory_note"`
	Status          string          `json:"status" gorm:"index"`
	Notes           string          `json:"notes"`
	ChargeRates     datatypes.JSON  `json:"charge_rates" gorm:"type:json"`
	CostBearingMode string          `json:"cost_bearing_mode" gorm:"type:varchar(20);default:'company'"`
	CompanyName     string          `json:"company_name"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Beds            []DormBed       `json:"beds,omitempty" gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE;"`
	Assets          []DormRoomAsset `json:"assets,omitempty" gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE;"`
}

// DormBed represents a bed under a room
type DormBed struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	User      *User     `json:"-,omitempty" gorm:"foreignKey:UserID"`
	RoomID    uint      `json:"room_id" gorm:"index"`
	Room      *DormRoom `json:"-"`
	BedNumber string    `json:"bed_number"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DormRoomAsset describes assets assigned to a room
type DormRoomAsset struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	UserID        *uint      `json:"user_id" gorm:"index"`
	User          *User      `json:"-,omitempty" gorm:"foreignKey:UserID"`
	RoomID        uint       `json:"room_id" gorm:"index"`
	Room          *DormRoom  `json:"-"`
	AssetType     string     `json:"asset_type"`
	Identifier    string     `json:"identifier"`
	Status        string     `json:"status"`
	PurchasedAt   *time.Time `json:"purchased_at"`
	WarrantyUntil *time.Time `json:"warranty_until"`
	Notes         string     `json:"notes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DormContract represents a stay/lease contract
type DormContract struct {
	ID            uint                        `json:"id" gorm:"primaryKey"`
	UserID        *uint                       `json:"user_id" gorm:"index"`
	User          *User                       `json:"-" gorm:"foreignKey:UserID"`
	EmployeeID    *uint                       `json:"employee_id" gorm:"index"`
	EmployeeName  string                      `json:"employee_name"`
	EmployeeDept  string                      `json:"employee_department" gorm:"column:employee_dept"`
	EmployeePhone string                      `json:"employee_phone"`
	EmployeeIDNo  string                      `json:"employee_id_number" gorm:"column:employee_id_number"`
	ResidenceAddr string                      `json:"employee_residence" gorm:"column:employee_residence"`
	RoomID        uint                        `json:"room_id" gorm:"index"`
	Room          *DormRoom                   `json:"room,omitempty"`
	BedID         *uint                       `json:"bed_id" gorm:"index"`
	Bed           *DormBed                    `json:"bed,omitempty"`
	StartDate     time.Time                   `json:"start_date"`
	EndDate       time.Time                   `json:"end_date"`
	RentAmount    float64                     `json:"rent_amount"`
	DepositAmount float64                     `json:"deposit_amount"`
	PaymentMethod string                      `json:"payment_method"`
	Attachments   datatypes.JSONSlice[string] `json:"attachments" gorm:"type:json"`
	Status        string                      `json:"status" gorm:"index"`
	Notes         string                      `json:"notes"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
}

// DormCheckout represents a checkout/exit inspection
type DormCheckout struct {
	ID                  uint  `json:"id" gorm:"primaryKey"`
	UserID              *uint `json:"user_id" gorm:"index"`
	User                *User `json:"-" gorm:"foreignKey:UserID"`
	ContractID          uint  `json:"contract_id" gorm:"uniqueIndex"`
	Contract            DormContract
	CheckoutDate        time.Time                   `json:"checkout_date"`
	Inspector           string                      `json:"inspector"`
	DamageReport        string                      `json:"damage_report"`
	ItemsStatus         string                      `json:"items_status"`
	FeeSummary          string                      `json:"fee_summary"`
	DepositCollected    float64                     `json:"deposit_collected"`
	DepositReturn       float64                     `json:"deposit_return"`
	DepositDeduct       float64                     `json:"deposit_deduct"`
	GuaranteeCollected  float64                     `json:"guarantee_collected"`
	GuaranteeDeduct     float64                     `json:"guarantee_deduct"`
	GuaranteeReturn     float64                     `json:"guarantee_return"`
	GuaranteeReturnDate *time.Time                  `json:"guarantee_return_date"`
	Attachments         datatypes.JSONSlice[string] `json:"attachments" gorm:"type:json"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

// DormMeterItem describes meter definitions (water/electric etc.)
type DormMeterItem struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	UserID      *uint             `json:"user_id" gorm:"index"`
	User        *User             `json:"-" gorm:"foreignKey:UserID"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	BillingMode string            `json:"billing_mode"`
	PricingMeta datatypes.JSONMap `json:"pricing_meta" gorm:"type:json"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// DormMeterReading stores periodic meter readings
type DormMeterReading struct {
	ID            uint  `json:"id" gorm:"primaryKey"`
	UserID        *uint `json:"user_id" gorm:"index"`
	User          *User `json:"-" gorm:"foreignKey:UserID"`
	RoomID        uint  `json:"room_id" gorm:"index"`
	Room          *DormRoom
	MeterDate     time.Time      `json:"meter_date"`
	BillingStart  time.Time      `json:"billing_start"`
	BillingEnd    time.Time      `json:"billing_end"`
	Inspector     string         `json:"inspector"`
	Notes         string         `json:"notes"`
	ChargeDetails datatypes.JSON `json:"charge_details" gorm:"type:json"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// UserPreference stores per-user UI or configuration settings
type UserPreference struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    *uint          `json:"user_id" gorm:"index:user_pref_user_key;uniqueIndex:user_pref_user_pref_key_uidx"`
	User      *User          `json:"-" gorm:"foreignKey:UserID"`
	PrefKey   string         `json:"key" gorm:"size:100;index:user_pref_user_key;uniqueIndex:user_pref_user_pref_key_uidx"`
	Value     datatypes.JSON `json:"value" gorm:"type:json"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// DormBillingRule describes billing configuration per room/contract
type DormBillingRule struct {
	ID         uint              `json:"id" gorm:"primaryKey"`
	UserID     *uint             `json:"user_id" gorm:"index"`
	User       *User             `json:"-" gorm:"foreignKey:UserID"`
	RoomID     *uint             `json:"room_id" gorm:"index"`
	ContractID *uint             `json:"contract_id" gorm:"index"`
	Scope      string            `json:"scope"` // room / contract / global
	Cycle      string            `json:"cycle"` // monthly/quarter/custom
	RuleType   string            `json:"rule_type"`
	RuleConfig datatypes.JSONMap `json:"rule_config" gorm:"type:json"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// DormBill represents generated bills for rooms or employees
type DormBill struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	UserID       *uint             `json:"user_id" gorm:"index"`
	User         *User             `json:"-" gorm:"foreignKey:UserID"`
	BillCode     string            `json:"bill_code" gorm:"index"`
	RoomID       *uint             `json:"room_id" gorm:"index"`
	Room         *DormRoom         `json:"-"`
	ContractID   *uint             `json:"contract_id" gorm:"index"`
	Contract     *DormContract     `json:"-"`
	EmployeeID   *uint             `json:"employee_id" gorm:"index"`
	EmployeeName string            `json:"employee_name"`
	PeriodLabel  string            `json:"period_label"`
	DueDate      time.Time         `json:"due_date"`
	Status       string            `json:"status" gorm:"index"`
	Items        []DormBillItem    `json:"items" gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE;"`
	AmountDue    float64           `json:"amount_due"`
	AmountPaid   float64           `json:"amount_paid"`
	Metadata     datatypes.JSONMap `json:"metadata" gorm:"type:json"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// DormBillItem stores each cost component on a bill
type DormBillItem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BillID    uint      `json:"bill_id" gorm:"index"`
	Bill      DormBill  `json:"-"`
	ItemType  string    `json:"item_type"`
	Label     string    `json:"label"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
