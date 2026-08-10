// Package migrate 提供从旧系统 (office-supply-analytics SQLite) 到 hr-office 新表的数据迁移。
package migrate

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/models"
)

// Options 迁移选项
type Options struct {
	OnlyDictionaries bool  // 仅迁字典，不迁业务单据
	DryRun           bool  // 只打印 INSERT 语句，不实际写入
	UserID           *uint // 多租户 ID（nil = 共享数据，不绑定用户）
}

// idMap 存储 oldID → newID 的映射
type idMap map[uint]uint

// ctx 迁移上下文，贯穿整个迁移流程
type ctx struct {
	target *gorm.DB
	source *gorm.DB
	opts   Options
	// maps[tableName] → oldID → newID
	maps map[string]idMap
}

// Run 执行完整的数据迁移。
// targetDB 为目标库（已 migrate 好表结构），sourceDSN 为源 SQLite 文件路径。
func Run(targetDB *gorm.DB, sourceDSN string, opts Options) error {
	src, err := gorm.Open(sqlite.Open(sourceDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("打开源数据库失败: %w", err)
	}

	c := &ctx{
		target: targetDB,
		source: src,
		opts:   opts,
		maps:   make(map[string]idMap),
	}

	// ====== 字典迁移 ======
	if err := skipMissingTable(c.migrateOfficeCategories()); err != nil {
		return fmt.Errorf("办公用品分类迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateOfficeSuppliers()); err != nil {
		return fmt.Errorf("供应商迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateOfficeSupplies()); err != nil {
		return fmt.Errorf("办公用品字典迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenCategories()); err != nil {
		return fmt.Errorf("食堂分类迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenSupplies()); err != nil {
		return fmt.Errorf("食堂食材字典迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenExpenseCategories()); err != nil {
		return fmt.Errorf("食堂费用科目迁移失败: %w", err)
	}

	if opts.OnlyDictionaries {
		return nil
	}

	// ====== 业务单据迁移 ======
	// 办公用品业务
	if err := skipMissingTable(c.migrateOfficePurchases()); err != nil {
		return fmt.Errorf("办公用品采购单迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateOfficePurchaseItems()); err != nil {
		return fmt.Errorf("办公用品采购明细迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateOfficePaymentRequests()); err != nil {
		return fmt.Errorf("请款单迁移失败: %w", err)
	}

	// 食堂业务
	if err := skipMissingTable(c.migrateCanteenPurchases()); err != nil {
		return fmt.Errorf("食堂采购单迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenPurchaseItems()); err != nil {
		return fmt.Errorf("食堂采购明细迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenOtherExpenses()); err != nil {
		return fmt.Errorf("食堂其他费用迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenDailyIncome()); err != nil {
		return fmt.Errorf("食堂每日收入迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenResourceFees()); err != nil {
		return fmt.Errorf("食堂资源占用费迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenWeeklyMenu()); err != nil {
		return fmt.Errorf("食堂每周菜单迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenMenuTemplates()); err != nil {
		return fmt.Errorf("食堂菜单模板迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenCardRecharges()); err != nil {
		return fmt.Errorf("饭卡充值记录迁移失败: %w", err)
	}
	if err := skipMissingTable(c.migrateCanteenCardRefunds()); err != nil {
		return fmt.Errorf("饭卡退费记录迁移失败: %w", err)
	}

	return nil
}

// ---------- 工具方法 ----------

// skipMissingTable 包装迁移步骤：源库缺少表时跳过而非报错。
func skipMissingTable(err error) error {
	if err != nil && strings.Contains(err.Error(), "no such table") {
		fmt.Printf("[INFO] 源库缺少表，跳过: %v\n", err)
		return nil
	}
	return err
}

// parseTime 解析源库 TEXT 时间字段，格式 "2006-01-02 15:04:05"
func parseTime(raw string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseTimePtr 解析可能为空的 TEXT 时间字段，返回 *time.Time
func parseTimePtr(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	t := parseTime(raw)
	return &t
}

// newID 从 idMap 中查找映射后的新 ID；未找到返回 0
func (c *ctx) newID(table string, oldID uint) uint {
	m, ok := c.maps[table]
	if !ok {
		return 0
	}
	n, ok := m[oldID]
	if !ok {
		return 0
	}
	return n
}

// save 执行写入：dry-run 模式打印语句，否则事务写入
func (c *ctx) save(tx *gorm.DB, model interface{}, desc string) {
	if c.opts.DryRun {
		fmt.Printf("[DRY-RUN] INSERT INTO %s: %+v\n", tx.Statement.Table, model)
		return
	}
	// 遇到唯一约束冲突则跳过（业务单据可能重复）
	if err := tx.Create(model).Error; err != nil {
		fmt.Printf("[SKIP] %s 插入失败(可能重复): %v\n", desc, err)
	}
}

// ---------- 源表中间结构体 ----------

// srcCategory 源库 categories 表
type srcCategory struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	SortOrder int    `gorm:"column:sort_order"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

func (srcCategory) TableName() string { return "categories" }

// srcSupplier 源库 suppliers 表
type srcSupplier struct {
	ID          uint   `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Contact     string `gorm:"column:contact"`
	Phone       string `gorm:"column:phone"`
	BankName    string `gorm:"column:bank_name"`
	BankAccount string `gorm:"column:bank_account"`
	IsDefault   int    `gorm:"column:is_default"`
	Remark      string `gorm:"column:remark"`
	CreatedAt   string `gorm:"column:created_at"`
	UpdatedAt   string `gorm:"column:updated_at"`
}

func (srcSupplier) TableName() string { return "suppliers" }

// srcSupply 源库 supplies 表
type srcSupply struct {
	ID             uint    `gorm:"column:id"`
	Name           string  `gorm:"column:name"`
	Spec           string  `gorm:"column:spec"`
	Unit           string  `gorm:"column:unit"`
	ReferencePrice float64 `gorm:"column:reference_price"`
	SafetyStock    int     `gorm:"column:safety_stock"`
	CategoryID     uint    `gorm:"column:category_id"`
	SupplierID     uint    `gorm:"column:supplier_id"`
	Status         string  `gorm:"column:status"`
	Remark         string  `gorm:"column:remark"`
	CreatedAt      string  `gorm:"column:created_at"`
	UpdatedAt      string  `gorm:"column:updated_at"`
}

func (srcSupply) TableName() string { return "supplies" }

// srcCanteenCategory 源库 canteen_categories 表
type srcCanteenCategory struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	SortOrder int    `gorm:"column:sort_order"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

func (srcCanteenCategory) TableName() string { return "canteen_categories" }

// srcCanteenSupply 源库 canteen_supplies 表
type srcCanteenSupply struct {
	ID             uint    `gorm:"column:id"`
	Name           string  `gorm:"column:name"`
	Spec           string  `gorm:"column:spec"`
	Unit           string  `gorm:"column:unit"`
	ReferencePrice float64 `gorm:"column:reference_price"`
	CategoryID     uint    `gorm:"column:category_id"`
	Status         string  `gorm:"column:status"`
	Remark         string  `gorm:"column:remark"`
	CreatedAt      string  `gorm:"column:created_at"`
	UpdatedAt      string  `gorm:"column:updated_at"`
}

func (srcCanteenSupply) TableName() string { return "canteen_supplies" }

// srcCanteenExpenseCategory 源库 canteen_expense_categories 表
type srcCanteenExpenseCategory struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	SortOrder int    `gorm:"column:sort_order"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

func (srcCanteenExpenseCategory) TableName() string { return "canteen_expense_categories" }

// srcPurchase 源库 purchases 表
type srcPurchase struct {
	ID            uint    `gorm:"column:id"`
	OrderNo       string  `gorm:"column:order_no"`
	PurchaseDate  string  `gorm:"column:purchase_date"`
	TotalAmount   float64 `gorm:"column:total_amount"`
	Status        string  `gorm:"column:status"`
	Remark        string  `gorm:"column:remark"`
	SupplierID    uint    `gorm:"column:supplier_id"`
	SupplierName  string  `gorm:"column:supplier_name"`
	PaymentStatus string  `gorm:"column:payment_status"`
	PaymentDate   string  `gorm:"column:payment_date"`
	CreatedAt     string  `gorm:"column:created_at"`
	UpdatedAt     string  `gorm:"column:updated_at"`
}

func (srcPurchase) TableName() string { return "purchases" }

// srcPurchaseItem 源库 purchase_items 表
type srcPurchaseItem struct {
	ID         uint    `gorm:"column:id"`
	PurchaseID uint    `gorm:"column:purchase_id"`
	SupplyID   uint    `gorm:"column:supply_id"`
	Quantity   int     `gorm:"column:quantity"`
	UnitPrice  float64 `gorm:"column:unit_price"`
	Subtotal   float64 `gorm:"column:subtotal"`
	Date       string  `gorm:"column:date"`
}

func (srcPurchaseItem) TableName() string { return "purchase_items" }

// srcPaymentRequest 源库 payment_requests 表
type srcPaymentRequest struct {
	ID              uint    `gorm:"column:id"`
	RequestNo       string  `gorm:"column:request_no"`
	PaymentUnit     string  `gorm:"column:payment_unit"`
	Department      string  `gorm:"column:department"`
	Applicant       string  `gorm:"column:applicant"`
	RequestDate     string  `gorm:"column:request_date"`
	Content         string  `gorm:"column:content"`
	Payee           string  `gorm:"column:payee"`
	PayeeSupplierID uint    `gorm:"column:payee_supplier_id"`
	BankName        string  `gorm:"column:bank_name"`
	BankAccount     string  `gorm:"column:bank_account"`
	Amount          float64 `gorm:"column:amount"`
	AmountCN        string  `gorm:"column:amount_cn"`
	PaymentMethod   string  `gorm:"column:payment_method"`
	Remark          string  `gorm:"column:remark"`
	CompanyHead     string  `gorm:"column:company_head"`
	FinanceHead     string  `gorm:"column:finance_head"`
	DeptHead        string  `gorm:"column:dept_head"`
	Handler         string  `gorm:"column:handler"`
	Status          string  `gorm:"column:status"`
	PurchaseIDs     string  `gorm:"column:purchase_ids"`
	CreatedAt       string  `gorm:"column:created_at"`
	UpdatedAt       string  `gorm:"column:updated_at"`
}

func (srcPaymentRequest) TableName() string { return "payment_requests" }

// srcCanteenPurchase 源库 canteen_purchases 表
type srcCanteenPurchase struct {
	ID           uint    `gorm:"column:id"`
	OrderNo      string  `gorm:"column:order_no"`
	PurchaseDate string  `gorm:"column:purchase_date"`
	TotalAmount  float64 `gorm:"column:total_amount"`
	SupplierID   uint    `gorm:"column:supplier_id"`
	SupplierName string  `gorm:"column:supplier_name"`
	Channel      string  `gorm:"column:channel"`
	ActualPay    float64 `gorm:"column:actual_pay"`
	Remark       string  `gorm:"column:remark"`
	CreatedAt    string  `gorm:"column:created_at"`
	UpdatedAt    string  `gorm:"column:updated_at"`
}

func (srcCanteenPurchase) TableName() string { return "canteen_purchases" }

// srcCanteenPurchaseItem 源库 canteen_purchase_items 表
type srcCanteenPurchaseItem struct {
	ID         uint    `gorm:"column:id"`
	PurchaseID uint    `gorm:"column:purchase_id"`
	SupplyID   uint    `gorm:"column:supply_id"`
	Quantity   float64 `gorm:"column:quantity"`
	UnitPrice  float64 `gorm:"column:unit_price"`
	Subtotal   float64 `gorm:"column:subtotal"`
	Remark     string  `gorm:"column:remark"`
}

func (srcCanteenPurchaseItem) TableName() string { return "canteen_purchase_items" }

// srcCanteenOtherExpense 源库 canteen_other_expenses 表
type srcCanteenOtherExpense struct {
	ID           uint    `gorm:"column:id"`
	ExpenseDate  string  `gorm:"column:expense_date"`
	Category     string  `gorm:"column:category"`
	Amount       float64 `gorm:"column:amount"`
	ActualAmount float64 `gorm:"column:actual_amount"`
	Params       string  `gorm:"column:params"`
	Remark       string  `gorm:"column:remark"`
	CreatedAt    string  `gorm:"column:created_at"`
	UpdatedAt    string  `gorm:"column:updated_at"`
}

func (srcCanteenOtherExpense) TableName() string { return "canteen_other_expenses" }

// srcCanteenDailyIncome 源库 canteen_daily_income 表
type srcCanteenDailyIncome struct {
	ID              uint    `gorm:"column:id"`
	IncomeDate      string  `gorm:"column:income_date"`
	BreakfastCount  int     `gorm:"column:breakfast_count"`
	BreakfastAmount float64 `gorm:"column:breakfast_amount"`
	LunchCount      int     `gorm:"column:lunch_count"`
	LunchAmount     float64 `gorm:"column:lunch_amount"`
	DinnerCount     int     `gorm:"column:dinner_count"`
	DinnerAmount    float64 `gorm:"column:dinner_amount"`
	TotalCount      int     `gorm:"column:total_count"`
	TotalAmount     float64 `gorm:"column:total_amount"`
	Remark          string  `gorm:"column:remark"`
	CreatedAt       string  `gorm:"column:created_at"`
	UpdatedAt       string  `gorm:"column:updated_at"`
}

func (srcCanteenDailyIncome) TableName() string { return "canteen_daily_income" }

// srcCanteenResourceFee 源库 canteen_resource_fees 表
type srcCanteenResourceFee struct {
	ID        uint    `gorm:"column:id"`
	FeeDate   string  `gorm:"column:fee_date"`
	MealType  string  `gorm:"column:meal_type"`
	Amount    float64 `gorm:"column:amount"`
	Payer     string  `gorm:"column:payer"`
	Reason    string  `gorm:"column:reason"`
	Remark    string  `gorm:"column:remark"`
	Handler   string  `gorm:"column:handler"`
	CreatedAt string  `gorm:"column:created_at"`
	UpdatedAt string  `gorm:"column:updated_at"`
}

func (srcCanteenResourceFee) TableName() string { return "canteen_resource_fees" }

// srcCanteenWeeklyMenu 源库 canteen_weekly_menu 表
type srcCanteenWeeklyMenu struct {
	ID            uint   `gorm:"column:id"`
	WeekStartDate string `gorm:"column:week_start_date"`
	DayOfWeek     int    `gorm:"column:day_of_week"`
	MealType      string `gorm:"column:meal_type"`
	Dishes        string `gorm:"column:dishes"`
	Remark        string `gorm:"column:remark"`
	CreatedAt     string `gorm:"column:created_at"`
	UpdatedAt     string `gorm:"column:updated_at"`
}

func (srcCanteenWeeklyMenu) TableName() string { return "canteen_weekly_menu" }

// srcCanteenMenuTemplate 源库 canteen_menu_templates 表
type srcCanteenMenuTemplate struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	Data      string `gorm:"column:data"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

func (srcCanteenMenuTemplate) TableName() string { return "canteen_menu_templates" }

// srcCanteenCardRecharge 源库 canteen_card_recharges 表
type srcCanteenCardRecharge struct {
	ID              uint     `gorm:"column:id"`
	ExternalSN      *string  `gorm:"column:external_sn"`
	CardNo          string   `gorm:"column:card_no"`
	UserID          string   `gorm:"column:user_id"` // 饭卡系统员工编号
	UserName        string   `gorm:"column:user_name"`
	DepartmentCode  string   `gorm:"column:department_code"`
	UserDepartment  string   `gorm:"column:user_department"`
	RechargeDate    string   `gorm:"column:recharge_date"`
	Amount          float64  `gorm:"column:amount"`
	BalanceRecorded *float64 `gorm:"column:balance_recorded"`
	PaymentMethod   string   `gorm:"column:payment_method"`
	Operator        string   `gorm:"column:operator"`
	MachineNo       string   `gorm:"column:machine_no"`
	BillNo          string   `gorm:"column:bill_no"`
	Remark          string   `gorm:"column:remark"`
	CreatedAt       string   `gorm:"column:created_at"`
	UpdatedAt       string   `gorm:"column:updated_at"`
}

func (srcCanteenCardRecharge) TableName() string { return "canteen_card_recharges" }

// srcCanteenCardRefund 源库 canteen_card_refunds 表
type srcCanteenCardRefund struct {
	ID              uint     `gorm:"column:id"`
	ExternalSN      *string  `gorm:"column:external_sn"`
	CardNo          string   `gorm:"column:card_no"`
	UserID          string   `gorm:"column:user_id"` // 饭卡系统员工编号
	UserName        string   `gorm:"column:user_name"`
	DepartmentCode  string   `gorm:"column:department_code"`
	UserDepartment  string   `gorm:"column:user_department"`
	RefundDate      string   `gorm:"column:refund_date"`
	Amount          float64  `gorm:"column:amount"`
	BalanceRecorded *float64 `gorm:"column:balance_recorded"`
	Operator        string   `gorm:"column:operator"`
	MachineNo       string   `gorm:"column:machine_no"`
	BillNo          string   `gorm:"column:bill_no"`
	Remark          string   `gorm:"column:remark"`
	CreatedAt       string   `gorm:"column:created_at"`
	UpdatedAt       string   `gorm:"column:updated_at"`
}

func (srcCanteenCardRefund) TableName() string { return "canteen_card_refunds" }

// ---------- 字典迁移函数 ----------

func (c *ctx) migrateOfficeCategories() error {
	var src []srcCategory
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["categories"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			// 去重：按 name 查找是否已存在
			var exist models.OfficeCategory
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["categories"][s.ID] = exist.ID
				continue
			}
			m := models.OfficeCategory{
				Name:      s.Name,
				SortOrder: s.SortOrder,
				UserID:    c.opts.UserID,
				CreatedAt: parseTime(s.CreatedAt),
				UpdatedAt: parseTime(s.UpdatedAt),
			}
			if c.opts.DryRun {
				fmt.Printf("[DRY-RUN] INSERT office_categories: name=%s sort=%d\n", m.Name, m.SortOrder)
				continue
			}
			if err := tx.Create(&m).Error; err != nil {
				// 名字重复则查找已存在记录并记录映射
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["categories"][s.ID] = exist.ID
					continue
				}
				return fmt.Errorf("插入分类 %q 失败: %w", s.Name, err)
			}
			c.maps["categories"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateOfficeSuppliers() error {
	var src []srcSupplier
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["suppliers"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var exist models.OfficeSupplier
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["suppliers"][s.ID] = exist.ID
				continue
			}
			m := models.OfficeSupplier{
				Name:        s.Name,
				Contact:     s.Contact,
				Phone:       s.Phone,
				BankName:    s.BankName,
				BankAccount: s.BankAccount,
				IsDefault:   s.IsDefault,
				Remark:      s.Remark,
				UserID:      c.opts.UserID,
				CreatedAt:   parseTime(s.CreatedAt),
				UpdatedAt:   parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("供应商 %s", s.Name))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["suppliers"][s.ID] = exist.ID
				}
				continue
			}
			c.maps["suppliers"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateOfficeSupplies() error {
	var src []srcSupply
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["supplies"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var exist models.OfficeSupply
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["supplies"][s.ID] = exist.ID
				continue
			}
			// 外键重映射
			var catID *uint
			if s.CategoryID > 0 {
				n := c.newID("categories", s.CategoryID)
				if n > 0 {
					catID = &n
				}
			}
			var supID *uint
			if s.SupplierID > 0 {
				n := c.newID("suppliers", s.SupplierID)
				if n > 0 {
					supID = &n
				}
			}
			m := models.OfficeSupply{
				Name:           s.Name,
				Spec:           s.Spec,
				Unit:           s.Unit,
				ReferencePrice: s.ReferencePrice,
				SafetyStock:    s.SafetyStock,
				CategoryID:     catID,
				SupplierID:     supID,
				Status:         s.Status,
				Remark:         s.Remark,
				UserID:         c.opts.UserID,
				CreatedAt:      parseTime(s.CreatedAt),
				UpdatedAt:      parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("办公用品 %s", s.Name))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["supplies"][s.ID] = exist.ID
				}
				continue
			}
			c.maps["supplies"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateCanteenCategories() error {
	var src []srcCanteenCategory
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["canteen_categories"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var exist models.CanteenCategory
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["canteen_categories"][s.ID] = exist.ID
				continue
			}
			m := models.CanteenCategory{
				Name:      s.Name,
				SortOrder: s.SortOrder,
				UserID:    c.opts.UserID,
				CreatedAt: parseTime(s.CreatedAt),
				UpdatedAt: parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂分类 %s", s.Name))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["canteen_categories"][s.ID] = exist.ID
				}
				continue
			}
			c.maps["canteen_categories"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateCanteenSupplies() error {
	var src []srcCanteenSupply
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["canteen_supplies"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var exist models.CanteenSupply
			// 按 name 去重（因为 uniqueIndex 是 name + category_id，所以仅用 name 不够精确）
			// 但在仅迁移模式下足够；实际以唯一约束冲突来兜底
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["canteen_supplies"][s.ID] = exist.ID
				continue
			}
			var catID *uint
			if s.CategoryID > 0 {
				n := c.newID("canteen_categories", s.CategoryID)
				if n > 0 {
					catID = &n
				}
			}
			m := models.CanteenSupply{
				Name:           s.Name,
				Spec:           s.Spec,
				Unit:           s.Unit,
				ReferencePrice: s.ReferencePrice,
				CategoryID:     catID,
				Status:         s.Status,
				Remark:         s.Remark,
				UserID:         c.opts.UserID,
				CreatedAt:      parseTime(s.CreatedAt),
				UpdatedAt:      parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂食材 %s", s.Name))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["canteen_supplies"][s.ID] = exist.ID
				}
				continue
			}
			c.maps["canteen_supplies"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateCanteenExpenseCategories() error {
	var src []srcCanteenExpenseCategory
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["canteen_expense_categories"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var exist models.CanteenExpenseCategory
			if err := tx.Where("name = ?", s.Name).First(&exist).Error; err == nil {
				c.maps["canteen_expense_categories"][s.ID] = exist.ID
				continue
			}
			m := models.CanteenExpenseCategory{
				Name:      s.Name,
				SortOrder: s.SortOrder,
				UserID:    c.opts.UserID,
				CreatedAt: parseTime(s.CreatedAt),
				UpdatedAt: parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂费用科目 %s", s.Name))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("name = ?", s.Name).First(&exist).Error == nil {
					c.maps["canteen_expense_categories"][s.ID] = exist.ID
				}
				continue
			}
			c.maps["canteen_expense_categories"][s.ID] = m.ID
		}
		return nil
	})
}

// ---------- 办公用品业务单据迁移 ----------

func (c *ctx) migrateOfficePurchases() error {
	var src []srcPurchase
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["purchases"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var supID *uint
			if s.SupplierID > 0 {
				n := c.newID("suppliers", s.SupplierID)
				if n > 0 {
					supID = &n
				}
			}
			m := models.OfficePurchase{
				OrderNo:       s.OrderNo,
				PurchaseDate:  parseTime(s.PurchaseDate),
				TotalAmount:   s.TotalAmount,
				Status:        s.Status,
				Remark:        s.Remark,
				SupplierID:    supID,
				SupplierName:  s.SupplierName,
				PaymentStatus: s.PaymentStatus,
				PaymentDate:   parseTimePtr(s.PaymentDate),
				UserID:        c.opts.UserID,
				CreatedAt:     parseTime(s.CreatedAt),
				UpdatedAt:     parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("采购单 %s", s.OrderNo))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("order_no = ?", s.OrderNo).First(&m).Error == nil {
					c.maps["purchases"][s.ID] = m.ID
				}
				continue
			}
			c.maps["purchases"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateOfficePurchaseItems() error {
	var src []srcPurchaseItem
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			purchaseID := c.newID("purchases", s.PurchaseID)
			supplyID := c.newID("supplies", s.SupplyID)
			if purchaseID == 0 || supplyID == 0 {
				fmt.Printf("[SKIP] 采购明细 id=%d: purchase_id=%d supply_id=%d 映射失败\n",
					s.ID, s.PurchaseID, s.SupplyID)
				continue
			}
			m := models.OfficePurchaseItem{
				PurchaseID: purchaseID,
				SupplyID:   supplyID,
				Quantity:   s.Quantity,
				UnitPrice:  s.UnitPrice,
				Subtotal:   s.Subtotal,
				Date:       parseTimePtr(s.Date),
				UserID:     c.opts.UserID,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			c.save(tx, &m, fmt.Sprintf("采购明细 purchase=%d supply=%d", purchaseID, supplyID))
		}
		return nil
	})
}

func (c *ctx) migrateOfficePaymentRequests() error {
	var src []srcPaymentRequest
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var supID *uint
			if s.PayeeSupplierID > 0 {
				n := c.newID("suppliers", s.PayeeSupplierID)
				if n > 0 {
					supID = &n
				}
			}
			m := models.OfficePaymentRequest{
				RequestNo:       s.RequestNo,
				PaymentUnit:     s.PaymentUnit,
				Department:      s.Department,
				Applicant:       s.Applicant,
				RequestDate:     parseTime(s.RequestDate),
				Content:         s.Content,
				Payee:           s.Payee,
				PayeeSupplierID: supID,
				BankName:        s.BankName,
				BankAccount:     s.BankAccount,
				Amount:          s.Amount,
				AmountCN:        s.AmountCN,
				PaymentMethod:   s.PaymentMethod,
				Remark:          s.Remark,
				CompanyHead:     s.CompanyHead,
				FinanceHead:     s.FinanceHead,
				DeptHead:        s.DeptHead,
				Handler:         s.Handler,
				Status:          s.Status,
				PurchaseIDs:     s.PurchaseIDs,
				UserID:          c.opts.UserID,
				CreatedAt:       parseTime(s.CreatedAt),
				UpdatedAt:       parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("请款单 %s", s.RequestNo))
		}
		return nil
	})
}

// ---------- 食堂业务单据迁移 ----------

func (c *ctx) migrateCanteenPurchases() error {
	var src []srcCanteenPurchase
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	c.maps["canteen_purchases"] = make(idMap)
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			var supID *uint
			if s.SupplierID > 0 {
				n := c.newID("suppliers", s.SupplierID)
				if n > 0 {
					supID = &n
				}
			}
			m := models.CanteenPurchase{
				OrderNo:      s.OrderNo,
				PurchaseDate: parseTime(s.PurchaseDate),
				TotalAmount:  s.TotalAmount,
				SupplierID:   supID,
				SupplierName: s.SupplierName,
				Channel:      s.Channel,
				ActualPay:    s.ActualPay,
				Remark:       s.Remark,
				UserID:       c.opts.UserID,
				CreatedAt:    parseTime(s.CreatedAt),
				UpdatedAt:    parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂采购单 %s", s.OrderNo))
			if c.opts.DryRun {
				continue
			}
			if m.ID == 0 {
				if tx.Where("order_no = ?", s.OrderNo).First(&m).Error == nil {
					c.maps["canteen_purchases"][s.ID] = m.ID
				}
				continue
			}
			c.maps["canteen_purchases"][s.ID] = m.ID
		}
		return nil
	})
}

func (c *ctx) migrateCanteenPurchaseItems() error {
	var src []srcCanteenPurchaseItem
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			purchaseID := c.newID("canteen_purchases", s.PurchaseID)
			supplyID := c.newID("canteen_supplies", s.SupplyID)
			if purchaseID == 0 || supplyID == 0 {
				fmt.Printf("[SKIP] 食堂采购明细 id=%d: purchase_id=%d supply_id=%d 映射失败\n",
					s.ID, s.PurchaseID, s.SupplyID)
				continue
			}
			m := models.CanteenPurchaseItem{
				PurchaseID: purchaseID,
				SupplyID:   supplyID,
				Quantity:   s.Quantity,
				UnitPrice:  s.UnitPrice,
				Subtotal:   s.Subtotal,
				Remark:     s.Remark,
				UserID:     c.opts.UserID,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			c.save(tx, &m, fmt.Sprintf("食堂采购明细 purchase=%d supply=%d", purchaseID, supplyID))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenOtherExpenses() error {
	var src []srcCanteenOtherExpense
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenOtherExpense{
				ExpenseDate:  parseTime(s.ExpenseDate),
				Category:     s.Category,
				Amount:       s.Amount,
				ActualAmount: s.ActualAmount,
				Params:       s.Params,
				Remark:       s.Remark,
				UserID:       c.opts.UserID,
				CreatedAt:    parseTime(s.CreatedAt),
				UpdatedAt:    parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂其他费用 %s %.2f", s.Category, s.Amount))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenDailyIncome() error {
	var src []srcCanteenDailyIncome
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenDailyIncome{
				IncomeDate:      parseTime(s.IncomeDate),
				BreakfastCount:  s.BreakfastCount,
				BreakfastAmount: s.BreakfastAmount,
				LunchCount:      s.LunchCount,
				LunchAmount:     s.LunchAmount,
				DinnerCount:     s.DinnerCount,
				DinnerAmount:    s.DinnerAmount,
				TotalCount:      s.TotalCount,
				TotalAmount:     s.TotalAmount,
				Remark:          s.Remark,
				UserID:          c.opts.UserID,
				CreatedAt:       parseTime(s.CreatedAt),
				UpdatedAt:       parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂每日收入 %s", s.IncomeDate))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenResourceFees() error {
	var src []srcCanteenResourceFee
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenResourceFee{
				FeeDate:   parseTime(s.FeeDate),
				MealType:  s.MealType,
				Amount:    s.Amount,
				Payer:     s.Payer,
				Reason:    s.Reason,
				Remark:    s.Remark,
				Handler:   s.Handler,
				UserID:    c.opts.UserID,
				CreatedAt: parseTime(s.CreatedAt),
				UpdatedAt: parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("食堂资源占用费 %s %.2f", s.FeeDate, s.Amount))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenWeeklyMenu() error {
	var src []srcCanteenWeeklyMenu
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenWeeklyMenu{
				WeekStartDate: s.WeekStartDate,
				DayOfWeek:     s.DayOfWeek,
				MealType:      s.MealType,
				Dishes:        s.Dishes,
				Remark:        s.Remark,
				UserID:        c.opts.UserID,
				CreatedAt:     parseTime(s.CreatedAt),
				UpdatedAt:     parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("每周菜单 %s day=%d %s", s.WeekStartDate, s.DayOfWeek, s.MealType))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenMenuTemplates() error {
	var src []srcCanteenMenuTemplate
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenMenuTemplate{
				Name:      s.Name,
				Data:      datatypes.JSON(s.Data),
				UserID:    c.opts.UserID,
				CreatedAt: parseTime(s.CreatedAt),
				UpdatedAt: parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("菜单模板 %s", s.Name))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenCardRecharges() error {
	var src []srcCanteenCardRecharge
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenCardRecharge{
				ExternalSN:      s.ExternalSN,
				CardNo:          s.CardNo,
				CardUserID:      s.UserID, // 源 user_id → 目标 card_user_id
				UserName:        s.UserName,
				DepartmentCode:  s.DepartmentCode,
				UserDepartment:  s.UserDepartment,
				RechargeDate:    parseTimePtr(s.RechargeDate),
				Amount:          s.Amount,
				BalanceRecorded: s.BalanceRecorded,
				PaymentMethod:   s.PaymentMethod,
				Operator:        s.Operator,
				MachineNo:       s.MachineNo,
				BillNo:          s.BillNo,
				Remark:          s.Remark,
				UserID:          c.opts.UserID,
				CreatedAt:       parseTime(s.CreatedAt),
				UpdatedAt:       parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("饭卡充值 %s", s.UserName))
		}
		return nil
	})
}

func (c *ctx) migrateCanteenCardRefunds() error {
	var src []srcCanteenCardRefund
	if err := c.source.Order("id").Find(&src).Error; err != nil {
		return err
	}
	return c.target.Transaction(func(tx *gorm.DB) error {
		for _, s := range src {
			m := models.CanteenCardRefund{
				ExternalSN:      s.ExternalSN,
				CardNo:          s.CardNo,
				CardUserID:      s.UserID, // 源 user_id → 目标 card_user_id
				UserName:        s.UserName,
				DepartmentCode:  s.DepartmentCode,
				UserDepartment:  s.UserDepartment,
				RefundDate:      parseTimePtr(s.RefundDate),
				Amount:          s.Amount,
				BalanceRecorded: s.BalanceRecorded,
				Operator:        s.Operator,
				MachineNo:       s.MachineNo,
				BillNo:          s.BillNo,
				Remark:          s.Remark,
				UserID:          c.opts.UserID,
				CreatedAt:       parseTime(s.CreatedAt),
				UpdatedAt:       parseTime(s.UpdatedAt),
			}
			c.save(tx, &m, fmt.Sprintf("饭卡退费 %s", s.UserName))
		}
		return nil
	})
}
