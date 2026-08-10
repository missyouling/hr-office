package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCostSummaryPerCapita 验证人均成本口径：
// 人均成本 = (采购 + 分摊 - 早餐收入 - 资源费) / 人次（仅午餐+晚餐）
// 每天独立计算后取平均值，其他费用按月天数均摊
func TestCostSummaryPerCapita(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 创建所有需要的表
	db.Exec(`CREATE TABLE canteen_daily_income (
		id INTEGER PRIMARY KEY, user_id INTEGER,
		income_date TEXT, breakfast_count INTEGER, breakfast_amount REAL,
		lunch_count INTEGER, lunch_amount REAL, dinner_count INTEGER, dinner_amount REAL,
		total_count INTEGER, total_amount REAL, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_purchases (
		id INTEGER PRIMARY KEY, user_id INTEGER,
		order_no TEXT, purchase_date TEXT, total_amount REAL,
		supplier_id INTEGER, supplier_name TEXT, channel TEXT,
		actual_pay REAL, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_other_expenses (
		id INTEGER PRIMARY KEY, user_id INTEGER,
		expense_date TEXT, category TEXT, amount REAL,
		actual_amount REAL, params TEXT, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_resource_fees (
		id INTEGER PRIMARY KEY, user_id INTEGER,
		fee_date TEXT, meal_type TEXT, amount REAL,
		payer TEXT, reason TEXT, remark TEXT, handler TEXT
	)`)
	// CostSummary 内部需要的额外表
	db.Exec(`CREATE TABLE canteen_purchase_items (
		id INTEGER PRIMARY KEY, user_id INTEGER, purchase_id INTEGER,
		supply_id INTEGER, quantity REAL, unit_price REAL, subtotal REAL, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_supplies (
		id INTEGER PRIMARY KEY, user_id INTEGER, name TEXT, spec TEXT, unit TEXT,
		reference_price REAL, category_id INTEGER, status TEXT, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_categories (
		id INTEGER PRIMARY KEY, user_id INTEGER, name TEXT, sort_order INTEGER
	)`)
	db.Exec(`CREATE TABLE canteen_card_recharges (
		id INTEGER PRIMARY KEY, user_id INTEGER, external_sn TEXT,
		card_no TEXT, card_user_id TEXT, user_name TEXT,
		department_code TEXT, user_department TEXT,
		recharge_date TEXT, amount REAL, balance_recorded REAL,
		payment_method TEXT, operator TEXT, machine_no TEXT, bill_no TEXT, remark TEXT
	)`)
	db.Exec(`CREATE TABLE canteen_card_refunds (
		id INTEGER PRIMARY KEY, user_id INTEGER, external_sn TEXT,
		card_no TEXT, card_user_id TEXT, user_name TEXT,
		department_code TEXT, user_department TEXT,
		refund_date TEXT, amount REAL, balance_recorded REAL,
		operator TEXT, machine_no TEXT, bill_no TEXT, remark TEXT
	)`)

	userID := uint(1)
	month := "2026-08"

	// 场景：3天有收入数据
	// 8月共31天，其他费用 120 元 → 分摊 = 120/31 ≈ 3.871
	// 第1天: 午餐100人×10=1000, 晚餐50人×10=500, 早餐0, 采购300
	// 第2天: 午餐120人×10=1200, 晚餐60人×10=600, 早餐200, 采购400
	// 第3天: 午餐90人×10=900, 晚餐40人×10=400, 早餐0, 采购200, 资源费100
	// 人均 = AVG((300+3.871-0-0)/150, (400+3.871-200-0)/180, (200+3.871-0-100)/130)
	//     = AVG(2.026, 1.133, 0.799) = 1.319

	db.Exec(`INSERT INTO canteen_daily_income (user_id, income_date, breakfast_count, breakfast_amount, lunch_count, lunch_amount, dinner_count, dinner_amount, total_count, total_amount)
		VALUES (?, '2026-08-01', 0, 0, 100, 1000, 50, 500, 150, 1500)`, userID)
	db.Exec(`INSERT INTO canteen_daily_income (user_id, income_date, breakfast_count, breakfast_amount, lunch_count, lunch_amount, dinner_count, dinner_amount, total_count, total_amount)
		VALUES (?, '2026-08-02', 10, 200, 120, 1200, 60, 600, 180, 2000)`, userID)
	db.Exec(`INSERT INTO canteen_daily_income (user_id, income_date, breakfast_count, breakfast_amount, lunch_count, lunch_amount, dinner_count, dinner_amount, total_count, total_amount)
		VALUES (?, '2026-08-03', 0, 0, 90, 900, 40, 400, 130, 1300)`, userID)

	db.Exec(`INSERT INTO canteen_purchases (user_id, order_no, purchase_date, total_amount) VALUES (?, 'CT-01', '2026-08-01', 300)`, userID)
	db.Exec(`INSERT INTO canteen_purchases (user_id, order_no, purchase_date, total_amount) VALUES (?, 'CT-02', '2026-08-02', 400)`, userID)
	db.Exec(`INSERT INTO canteen_purchases (user_id, order_no, purchase_date, total_amount) VALUES (?, 'CT-03', '2026-08-03', 200)`, userID)

	db.Exec(`INSERT INTO canteen_other_expenses (user_id, expense_date, category, amount) VALUES (?, '2026-08-01', '水电', 120)`, userID)
	db.Exec(`INSERT INTO canteen_resource_fees (user_id, fee_date, meal_type, amount, payer) VALUES (?, '2026-08-03', '午餐', 100, '张三')`, userID)

	svc := NewCanteenAnalyticsService(db)

	// 测试人均成本计算
	perCapita := svc.calcPerCapita(userID, month)

	// 公式：AVG((300+3.871-0)/150, (400+3.871-200)/180, (200+3.871-100)/130) ≈ 1.319
	expected := 1.32
	if perCapita < expected-0.05 || perCapita > expected+0.05 {
		t.Errorf("人均成本计算错误: 实际=%f, 期望≈%f（±0.05容差）", perCapita, expected)
	}
	t.Logf("人均成本: %f (期望≈%f)", perCapita, expected)

	// 验证 DailyTrend 分摊正确
	trend, err := svc.DailyTrend(userID, month)
	if err != nil {
		t.Fatalf("DailyTrend 失败: %v", err)
	}
	daysInMonth := 31.0
	expectedShare := 120.0 / daysInMonth
	for _, d := range trend {
		share := d["share_expense"].(float64)
		if share < expectedShare-0.01 || share > expectedShare+0.01 {
			t.Errorf("分摊费用异常: 日期=%v, 期望≈%f, 实际=%f", d["date"], expectedShare, share)
		}
	}
	t.Logf("分摊费用: %f (期望=%f/31≈%f)", trend[0]["share_expense"], 120.0, expectedShare)
}

// TestCalcPerCapitaEdgeCases 测试人均成本的边界情况
func TestCalcPerCapitaEdgeCases(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	db.Exec(`CREATE TABLE canteen_daily_income (id INTEGER PRIMARY KEY, user_id INTEGER, income_date TEXT, breakfast_count INTEGER, breakfast_amount REAL, lunch_count INTEGER, lunch_amount REAL, dinner_count INTEGER, dinner_amount REAL, total_count INTEGER, total_amount REAL)`)
	db.Exec(`CREATE TABLE canteen_purchases (id INTEGER PRIMARY KEY, user_id INTEGER, order_no TEXT, purchase_date TEXT, total_amount REAL, supplier_name TEXT, channel TEXT, actual_pay REAL, remark TEXT)`)
	db.Exec(`CREATE TABLE canteen_other_expenses (id INTEGER PRIMARY KEY, user_id INTEGER, expense_date TEXT, category TEXT, amount REAL, actual_amount REAL, params TEXT, remark TEXT)`)
	db.Exec(`CREATE TABLE canteen_resource_fees (id INTEGER PRIMARY KEY, user_id INTEGER, fee_date TEXT, meal_type TEXT, amount REAL, payer TEXT, reason TEXT, remark TEXT, handler TEXT)`)

	svc := NewCanteenAnalyticsService(db)
	userID := uint(99)

	// 边界1: 无数据 → 人均成本应为 0
	pc := svc.calcPerCapita(userID, "2026-09")
	if pc != 0 {
		t.Errorf("无数据时人均成本应为0，实际=%f", pc)
	}

	// 边界2: 只有一天数据（9月30天）
	// 只有午餐10人×10=100，晚餐5人×10=50，采购80，无其他费用和资源费
	db.Exec(`INSERT INTO canteen_daily_income (user_id, income_date, lunch_count, lunch_amount, dinner_count, dinner_amount, total_count, total_amount) VALUES (?, '2026-09-01', 10, 100, 5, 50, 15, 150)`, userID)
	db.Exec(`INSERT INTO canteen_purchases (user_id, order_no, purchase_date, total_amount, supplier_name) VALUES (?, 'CT-01', '2026-09-01', 80, '')`, userID)

	pc = svc.calcPerCapita(userID, "2026-09")
	// 期望: 分摊=0, (80+0-0-0)/15 = 5.33
	if pc < 5.2 || pc > 5.5 {
		t.Errorf("单天人均成本异常: 实际=%f, 期望≈5.33", pc)
	}
	t.Logf("单天人均成本: %f (期望≈5.33)", pc)
}
