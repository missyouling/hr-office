package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CanteenAnalyticsService 食堂数据分析服务
type CanteenAnalyticsService struct {
	db *gorm.DB
}

// NewCanteenAnalyticsService 创建分析服务实例
func NewCanteenAnalyticsService(db *gorm.DB) *CanteenAnalyticsService {
	return &CanteenAnalyticsService{db: db}
}

// ========== 月度收支总览 ==========

// MonthlySummary 月度收支总览
// 收入=餐费+资源费，支出=食材采购+其他费用（优先实际金额）
func (s *CanteenAnalyticsService) MonthlySummary(userID uint, month string) (map[string]any, error) {
	m := defaultMonth(month)

	var income struct {
		Amount    float64
		Count     int
		Breakfast float64
		Lunch     float64
		Dinner    float64
	}
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) as amount,
		COALESCE(SUM(lunch_count + dinner_count),0) as count,
		COALESCE(SUM(breakfast_amount),0) as breakfast,
		COALESCE(SUM(lunch_amount),0) as lunch,
		COALESCE(SUM(dinner_amount),0) as dinner
		FROM canteen_daily_income WHERE user_id=? AND substr(income_date,1,7)=?`, userID, m).Scan(&income)

	var foodAmt float64
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) FROM canteen_purchases WHERE user_id=? AND substr(purchase_date,1,7)=?`, userID, m).Scan(&foodAmt)

	var otherAmt float64
	s.db.Raw(`SELECT COALESCE(SUM(CASE WHEN actual_amount>0 THEN actual_amount ELSE amount END),0)
		FROM canteen_other_expenses WHERE user_id=? AND substr(expense_date,1,7)=?`, userID, m).Scan(&otherAmt)

	var resAmt float64
	s.db.Raw(`SELECT COALESCE(SUM(amount),0) FROM canteen_resource_fees WHERE user_id=? AND substr(fee_date,1,7)=?`, userID, m).Scan(&resAmt)

	totalIncome := income.Amount + resAmt
	totalExpense := foodAmt + otherAmt

	return map[string]any{
		"month": m,
		"income": map[string]any{
			"total":     totalIncome,
			"meal":      income.Amount,
			"breakfast": income.Breakfast,
			"lunch":     income.Lunch,
			"dinner":    income.Dinner,
			"resource":  resAmt,
			"count":     income.Count,
		},
		"expense": map[string]any{
			"total": totalExpense,
			"food":  foodAmt,
			"other": otherAmt,
		},
		"profit": totalIncome - totalExpense,
	}, nil
}

// ========== 每日收支趋势 ==========

// DailyTrend 每日收支趋势（月度）
// 其他费用按当月天数均摊到每天
func (s *CanteenAnalyticsService) DailyTrend(userID uint, month string) ([]map[string]any, error) {
	m := defaultMonth(month)

	type IncomeRow struct {
		Date    string
		Amount  float64
		Count   int
		Bfast   float64
		LunchA  float64
		DinnerA float64
	}
	var incomes []IncomeRow
	s.db.Raw(`SELECT income_date as date, total_amount as amount,
		(lunch_count+dinner_count) as count, breakfast_amount as bfast,
		lunch_amount as lunch_a, dinner_amount as dinner_a
		FROM canteen_daily_income WHERE user_id=? AND substr(income_date,1,7)=? ORDER BY income_date`, userID, m).Scan(&incomes)

	type ExpRow struct {
		Date   string
		Amount float64
	}
	var expenses []ExpRow
	s.db.Raw(`SELECT purchase_date as date, COALESCE(SUM(total_amount),0) as amount
		FROM canteen_purchases WHERE user_id=? AND substr(purchase_date,1,7)=?
		GROUP BY purchase_date ORDER BY purchase_date`, userID, m).Scan(&expenses)

	var resources []ExpRow
	s.db.Raw(`SELECT fee_date as date, COALESCE(SUM(amount),0) as amount
		FROM canteen_resource_fees WHERE user_id=? AND substr(fee_date,1,7)=?
		GROUP BY fee_date ORDER BY fee_date`, userID, m).Scan(&resources)

	var otherTotal float64
	s.db.Raw(`SELECT COALESCE(SUM(CASE WHEN actual_amount>0 THEN actual_amount ELSE amount END),0)
		FROM canteen_other_expenses WHERE user_id=? AND substr(expense_date,1,7)=?`, userID, m).Scan(&otherTotal)

	daysInMonth := daysInMonthFromPrefix(m)
	share := 0.0
	if daysInMonth > 0 {
		share = roundTo2(otherTotal / float64(daysInMonth))
	}

	dateMap := make(map[string]map[string]any)
	for _, r := range incomes {
		dateMap[r.Date] = map[string]any{
			"date":          r.Date,
			"income":        r.Amount,
			"count":         r.Count,
			"breakfast":     r.Bfast,
			"lunch":         r.LunchA,
			"dinner":        r.DinnerA,
			"resource":      0.0,
			"expense":       0.0,
			"share_expense": share,
			"profit":        r.Amount - share,
		}
	}
	ensureDay := func(date string) map[string]any {
		if _, ok := dateMap[date]; !ok {
			dateMap[date] = map[string]any{
				"date": date, "income": 0.0, "count": 0, "breakfast": 0.0,
				"lunch": 0.0, "dinner": 0.0, "resource": 0.0, "expense": 0.0,
				"share_expense": share, "profit": -share,
			}
		}
		return dateMap[date]
	}
	for _, r := range expenses {
		d := ensureDay(r.Date)
		d["expense"] = d["expense"].(float64) + r.Amount
		d["profit"] = d["income"].(float64) - d["expense"].(float64) - share
	}
	for _, r := range resources {
		d := ensureDay(r.Date)
		d["resource"] = d["resource"].(float64) + r.Amount
		d["income"] = d["income"].(float64) + r.Amount
		d["profit"] = d["income"].(float64) - d["expense"].(float64) - share
	}
	result := make([]map[string]any, 0, len(dateMap))
	for _, v := range dateMap {
		result = append(result, v)
	}
	sortByDate(result, "date")
	return result, nil
}

// ========== 支出构成 ==========

// ExpenseBreakdown 支出构成（食材总额 + 其他费用按科目明细）
func (s *CanteenAnalyticsService) ExpenseBreakdown(userID uint, month string) (map[string]any, error) {
	m := defaultMonth(month)
	var foodAmt float64
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) FROM canteen_purchases WHERE user_id=? AND substr(purchase_date,1,7)=?`, userID, m).Scan(&foodAmt)

	type OtherRow struct {
		Category string
		Amount   float64
	}
	var others []OtherRow
	s.db.Raw(`SELECT category, COALESCE(SUM(CASE WHEN actual_amount>0 THEN actual_amount ELSE amount END),0) as amount
		FROM canteen_other_expenses WHERE user_id=? AND substr(expense_date,1,7)=?
		GROUP BY category ORDER BY amount DESC`, userID, m).Scan(&others)

	items := make([]map[string]any, len(others))
	for i, r := range others {
		items[i] = map[string]any{"category": r.Category, "amount": r.Amount}
	}
	return map[string]any{"food": foodAmt, "others": items}, nil
}

// ========== 食材分类占比 ==========

// FoodCategoryShare 食材采购分类占比
func (s *CanteenAnalyticsService) FoodCategoryShare(userID uint, month string) ([]map[string]any, error) {
	m := defaultMonth(month)
	type Row struct {
		Category string
		Amount   float64
	}
	var rows []Row
	s.db.Raw(`SELECT COALESCE(c.name,'其他') as category, COALESCE(SUM(pi.subtotal),0) as amount
		FROM canteen_purchase_items pi
		LEFT JOIN canteen_supplies s2 ON pi.supply_id=s2.id
		LEFT JOIN canteen_categories c ON s2.category_id=c.id
		LEFT JOIN canteen_purchases p ON pi.purchase_id=p.id
		WHERE pi.user_id=? AND substr(p.purchase_date,1,7)=?
		GROUP BY c.name ORDER BY amount DESC`, userID, m).Scan(&rows)

	result := make([]map[string]any, len(rows))
	for i, r := range rows {
		result[i] = map[string]any{"category": r.Category, "amount": r.Amount}
	}
	return result, nil
}

// ========== TOP 食材 ==========

// TopSupplies 采购金额最高的 N 个食材
func (s *CanteenAnalyticsService) TopSupplies(userID uint, month string, limit int) ([]map[string]any, error) {
	m := defaultMonth(month)
	if limit <= 0 {
		limit = 10
	}
	type Row struct {
		Name     string
		Unit     string
		Quantity float64
		Amount   float64
	}
	var rows []Row
	s.db.Raw(`SELECT s2.name, s2.unit, COALESCE(SUM(pi.quantity),0) as quantity, COALESCE(SUM(pi.subtotal),0) as amount
		FROM canteen_purchase_items pi
		LEFT JOIN canteen_supplies s2 ON pi.supply_id=s2.id
		LEFT JOIN canteen_purchases p ON pi.purchase_id=p.id
		WHERE pi.user_id=? AND substr(p.purchase_date,1,7)=?
		GROUP BY s2.id ORDER BY amount DESC LIMIT ?`, userID, m, limit).Scan(&rows)

	result := make([]map[string]any, len(rows))
	for i, r := range rows {
		result[i] = map[string]any{"name": r.Name, "unit": r.Unit, "quantity": r.Quantity, "amount": r.Amount}
	}
	return result, nil
}

// ========== 月度对比 ==========

// MonthlyCompare 月度对比（支持半年度/年度）
// 人均成本复用 DailyTrend 保证口径一致
func (s *CanteenAnalyticsService) MonthlyCompare(userID uint, from, to, year string) ([]map[string]any, error) {
	months := buildMonthRange(from, to, year)

	incomeMap := s.aggByMonth(userID, "canteen_daily_income", "income_date", "total_amount", "lunch_count + dinner_count", from, to, year)
	foodMap := s.aggByMonth(userID, "canteen_purchases", "purchase_date", "total_amount", "", from, to, year)
	otherMap := s.aggByMonth(userID, "canteen_other_expenses", "expense_date", "CASE WHEN actual_amount>0 THEN actual_amount ELSE amount END", "", from, to, year)
	resMap := s.aggByMonth(userID, "canteen_resource_fees", "fee_date", "amount", "", from, to, year)

	result := make([]map[string]any, 0, len(months))
	for _, m := range months {
		entry := map[string]any{
			"month":    m,
			"income":   getOrDefault(incomeMap, m, "amount", 0.0),
			"food":     getOrDefault(foodMap, m, "amount", 0.0),
			"other":    getOrDefault(otherMap, m, "amount", 0.0),
			"resource": getOrDefault(resMap, m, "amount", 0.0),
			"count":    getOrDefault(incomeMap, m, "count", 0),
		}
		entry["perCapita"] = s.calcPerCapita(userID, m)
		result = append(result, entry)
	}
	return result, nil
}

// aggByMonth 按月聚合辅助函数
func (s *CanteenAnalyticsService) aggByMonth(userID uint, table, dateCol, amountExpr, countExpr, from, to, year string) map[string]map[string]float64 {
	whereParts := []string{fmt.Sprintf("user_id=%d", userID)}
	if year != "" {
		whereParts = append(whereParts, fmt.Sprintf("substr(%s,1,4)='%s'", dateCol, year))
	} else if from != "" && to != "" {
		whereParts = append(whereParts, fmt.Sprintf("substr(%s,1,7)>='%s' AND substr(%s,1,7)<='%s'", dateCol, from, dateCol, to))
	}
	whereClause := strings.Join(whereParts, " AND ")

	countSelect := ""
	if countExpr != "" {
		countSelect = ", SUM(" + countExpr + ") as cnt"
	}
	sql := fmt.Sprintf("SELECT substr(%s,1,7) as month, SUM(%s) as amount%s FROM %s WHERE %s GROUP BY substr(%s,1,7)",
		dateCol, amountExpr, countSelect, table, whereClause, dateCol)

	type Row struct {
		Month  string
		Amount float64
		Cnt    float64
	}
	var rows []Row
	s.db.Raw(sql).Scan(&rows)
	result := make(map[string]map[string]float64)
	for _, r := range rows {
		result[r.Month] = map[string]float64{"amount": r.Amount, "count": r.Cnt}
	}
	return result
}

// calcPerCapita 计算人均成本（与 DailyTrend 口径一致）
func (s *CanteenAnalyticsService) calcPerCapita(userID uint, month string) float64 {
	trend, err := s.DailyTrend(userID, month)
	if err != nil || len(trend) == 0 {
		return 0
	}
	var perDaySum float64
	var dayCount int
	for _, d := range trend {
		count := float64Of(d["count"])
		if count <= 0 {
			continue
		}
		expense := float64Of(d["expense"])
		share := float64Of(d["share_expense"])
		breakfast := float64Of(d["breakfast"])
		resource := float64Of(d["resource"])
		perDay := (expense + share - breakfast - resource) / count
		perDaySum += perDay
		dayCount++
	}
	if dayCount == 0 {
		return 0
	}
	return roundTo2(perDaySum / float64(dayCount))
}

// ========== 智能建议 ==========

// Suggestions 自动优化建议（规则引擎）
func (s *CanteenAnalyticsService) Suggestions(userID uint, month string) ([]string, error) {
	m := defaultMonth(month)
	suggestions := []string{}
	cur, err := s.MonthlySummary(userID, m)
	if err != nil {
		return suggestions, err
	}
	// 上月
	y, mo := parseYearMonth(m)
	prevM := formatYearMonth(y, mo-1)
	prev, prevErr := s.MonthlySummary(userID, prevM)

	curExp := mapAt(cur, "expense", "total").(float64)
	curInc := mapAt(cur, "income", "total").(float64)
	curProfit := cur["profit"].(float64)

	if prevErr == nil {
		prevExp := mapAt(prev, "expense", "total").(float64)
		prevInc := mapAt(prev, "income", "total").(float64)
		// 成本异常预警（超过15%）
		if prevExp > 0 && curExp > 0 {
			diff := (curExp - prevExp) / prevExp * 100
			if diff > 15 {
				suggestions = append(suggestions, fmt.Sprintf("本月食材及费用支出较上月上涨 %.1f%%，建议排查肉类/蔬菜价格波动原因", diff))
			}
		}
		// 收入异常预警（下降超过10%）
		if prevInc > 0 && curInc > 0 {
			diff := (curInc - prevInc) / prevInc * 100
			if diff < -10 {
				suggestions = append(suggestions, fmt.Sprintf("本月收入较上月下降 %.1f%%，建议关注就餐人数变化", -diff))
			}
		}
	}
	// 盈亏健康度
	if curProfit < 0 {
		suggestions = append(suggestions, fmt.Sprintf("本月亏损 %.2f 元，建议优化采购成本或调整餐费标准", -curProfit))
	}
	// 高成本食材识别
	shares, _ := s.FoodCategoryShare(userID, m)
	if len(shares) > 0 {
		topCat := shares[0]["category"].(string)
		topAmt := shares[0]["amount"].(float64)
		foodTotal := mapAt(cur, "expense", "food").(float64)
		if foodTotal > 0 && topAmt/foodTotal > 0.2 {
			pct := topAmt / foodTotal * 100
			suggestions = append(suggestions, fmt.Sprintf("「%s」本月采购占比达 %.1f%%，可考虑寻找替代供应商", topCat, pct))
		}
	}
	// 人均消费参考
	count := mapAt(cur, "income", "count").(int)
	mealAmt := mapAt(cur, "income", "meal").(float64)
	if count > 0 {
		perCapita := mealAmt / float64(count)
		days := float64(newDate(y, mo, 1).AddDate(0, 1, -1).Day())
		avgDaily := float64(count) / maxF(1, days)
		suggestions = append(suggestions, fmt.Sprintf("本月日均就餐 %.0f 人次，人均消费 ¥%.2f", avgDaily, perCapita))
	}
	return suggestions, nil
}

// ========== 月度费用汇总（CostSummary） ==========

// CostSummaryResult 月度费用汇总结果
type CostSummaryResult struct {
	Month     string           `json:"month"`
	Meat      float64          `json:"meat"`
	Vegetable float64          `json:"vegetable"`
	Dry       float64          `json:"dry"`
	Grain     float64          `json:"grain"`
	Condiment float64          `json:"condiment"`
	OtherFood float64          `json:"otherFood"`
	Recharge  float64          `json:"recharge"`
	Consume   float64          `json:"consume"`
	Refund    float64          `json:"refund"`
	Income    float64          `json:"income"`
	Expense   float64          `json:"expense"`
	Profit    float64          `json:"profit"`
	PerCapita float64          `json:"perCapita"`
	Daily     []map[string]any `json:"daily"`
}

// CostSummary 单月度费用汇总（肉类/蔬菜/干杂/粮油/调味品/其他分类金额+充值/消费/退费+盈亏+人均成本+按日明细）
func (s *CanteenAnalyticsService) CostSummary(userID uint, month string) (*CostSummaryResult, error) {
	m := defaultMonth(month)
	prefix := m

	// 食材采购分类
	foodCat := s.queryFoodCategory(userID, prefix)

	// 充值、退费、消费
	var recharge, refund, consume float64
	s.db.Raw(`SELECT COALESCE(SUM(amount),0) FROM canteen_card_recharges WHERE user_id=? AND substr(recharge_date,1,7)=?`, userID, prefix).Scan(&recharge)
	s.db.Raw(`SELECT COALESCE(SUM(amount),0) FROM canteen_card_refunds WHERE user_id=? AND substr(refund_date,1,7)=?`, userID, prefix).Scan(&refund)
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) FROM canteen_daily_income WHERE user_id=? AND substr(income_date,1,7)=?`, userID, prefix).Scan(&consume)

	// 收入与支出
	var incomeMeal, resource, foodTotal, otherExp float64
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) FROM canteen_daily_income WHERE user_id=? AND substr(income_date,1,7)=?`, userID, prefix).Scan(&incomeMeal)
	s.db.Raw(`SELECT COALESCE(SUM(amount),0) FROM canteen_resource_fees WHERE user_id=? AND substr(fee_date,1,7)=?`, userID, prefix).Scan(&resource)
	s.db.Raw(`SELECT COALESCE(SUM(total_amount),0) FROM canteen_purchases WHERE user_id=? AND substr(purchase_date,1,7)=?`, userID, prefix).Scan(&foodTotal)
	s.db.Raw(`SELECT COALESCE(SUM(CASE WHEN actual_amount>0 THEN actual_amount ELSE amount END),0) FROM canteen_other_expenses WHERE user_id=? AND substr(expense_date,1,7)=?`, userID, prefix).Scan(&otherExp)

	incomeTotal := incomeMeal + resource
	expenseTotal := foodTotal + otherExp

	// 按日明细
	daily := s.buildDailyDetail(userID, prefix, otherExp)

	perCapita := s.calcPerCapita(userID, prefix)

	return &CostSummaryResult{
		Month:     prefix,
		Meat:      foodCat["肉类"],
		Vegetable: foodCat["蔬菜"],
		Dry:       foodCat["干杂"],
		Grain:     foodCat["粮油"],
		Condiment: foodCat["调味品"],
		OtherFood: foodCat["其他"],
		Recharge:  recharge,
		Consume:   consume,
		Refund:    refund,
		Income:    incomeTotal,
		Expense:   expenseTotal,
		Profit:    incomeTotal - expenseTotal,
		PerCapita: perCapita,
		Daily:     daily,
	}, nil
}

// CostSummaryRange 多个月度费用汇总（用于年度/自定义范围）
func (s *CanteenAnalyticsService) CostSummaryRange(userID uint, from, to, year string) ([]*CostSummaryResult, error) {
	months := buildMonthRange(from, to, year)
	var items []*CostSummaryResult
	for _, m := range months {
		r, err := s.CostSummary(userID, m)
		if err != nil {
			continue
		}
		// 只保留有数据的月份
		if r.Meat > 0 || r.Vegetable > 0 || r.Dry > 0 || r.Recharge > 0 || r.Consume > 0 || r.Refund > 0 || r.Income > 0 || r.Expense > 0 {
			items = append(items, r)
		}
	}
	return items, nil
}

// queryFoodCategory 食材采购分类汇总
func (s *CanteenAnalyticsService) queryFoodCategory(userID uint, prefix string) map[string]float64 {
	type Row struct {
		Category string
		Amount   float64
	}
	var rows []Row
	s.db.Raw(`SELECT COALESCE(c.name,'其他') as category, COALESCE(SUM(pi.subtotal),0) as amount
		FROM canteen_purchase_items pi
		LEFT JOIN canteen_purchases p ON pi.purchase_id=p.id
		LEFT JOIN canteen_supplies s2 ON pi.supply_id=s2.id
		LEFT JOIN canteen_categories c ON s2.category_id=c.id
		WHERE pi.user_id=? AND substr(p.purchase_date,1,7)=?
		GROUP BY c.name`, userID, prefix).Scan(&rows)
	cat := map[string]float64{"肉类": 0, "干杂": 0, "蔬菜": 0, "粮油": 0, "调味品": 0, "其他": 0}
	for _, r := range rows {
		k := r.Category
		if _, ok := cat[k]; !ok {
			k = "其他"
		}
		cat[k] += r.Amount
	}
	return cat
}

// buildDailyDetail 构建按日明细（当月每天）
func (s *CanteenAnalyticsService) buildDailyDetail(userID uint, prefix string, otherExp float64) []map[string]any {
	type DailyFood struct {
		Date     string
		Category string
		Amount   float64
	}
	var dailyFood []DailyFood
	s.db.Raw(`SELECT p.purchase_date as date, COALESCE(c.name,'其他') as category, COALESCE(SUM(pi.subtotal),0) as amount
		FROM canteen_purchase_items pi
		LEFT JOIN canteen_purchases p ON pi.purchase_id=p.id
		LEFT JOIN canteen_supplies s2 ON pi.supply_id=s2.id
		LEFT JOIN canteen_categories c ON s2.category_id=c.id
		WHERE pi.user_id=? AND substr(p.purchase_date,1,7)=?
		GROUP BY p.purchase_date, c.name`, userID, prefix).Scan(&dailyFood)

	type DailyAmt struct {
		Date   string
		Amount float64
	}
	var dailyRecharge []DailyAmt
	s.db.Raw(`SELECT substr(recharge_date,1,10) as date, COALESCE(SUM(amount),0) as amount
		FROM canteen_card_recharges WHERE user_id=? AND substr(recharge_date,1,7)=?
		GROUP BY substr(recharge_date,1,10)`, userID, prefix).Scan(&dailyRecharge)

	var dailyRefund []DailyAmt
	s.db.Raw(`SELECT substr(refund_date,1,10) as date, COALESCE(SUM(amount),0) as amount
		FROM canteen_card_refunds WHERE user_id=? AND substr(refund_date,1,7)=?
		GROUP BY substr(refund_date,1,10)`, userID, prefix).Scan(&dailyRefund)

	var dailyIncome []DailyAmt
	s.db.Raw(`SELECT income_date as date, total_amount as amount
		FROM canteen_daily_income WHERE user_id=? AND substr(income_date,1,7)=?`, userID, prefix).Scan(&dailyIncome)

	var dailyResource []DailyAmt
	s.db.Raw(`SELECT fee_date as date, COALESCE(SUM(amount),0) as amount
		FROM canteen_resource_fees WHERE user_id=? AND substr(fee_date,1,7)=?
		GROUP BY fee_date`, userID, prefix).Scan(&dailyResource)

	// 构建每日 map
	dailyMap := make(map[string]map[string]float64)
	ensure := func(date string) map[string]float64 {
		if d, ok := dailyMap[date]; ok {
			return d
		}
		d := map[string]float64{"meat": 0, "vegetable": 0, "dry": 0, "grain": 0, "condiment": 0, "otherFood": 0, "recharge": 0, "consume": 0, "refund": 0}
		dailyMap[date] = d
		return d
	}
	catMap := map[string]string{"肉类": "meat", "干杂": "dry", "蔬菜": "vegetable", "粮油": "grain", "调味品": "condiment"}
	for _, r := range dailyFood {
		d := ensure(r.Date)
		if col, ok := catMap[r.Category]; ok {
			d[col] += r.Amount
		} else {
			d["otherFood"] += r.Amount
		}
	}
	for _, r := range dailyRecharge {
		ensure(r.Date)["recharge"] += r.Amount
	}
	for _, r := range dailyRefund {
		ensure(r.Date)["refund"] += r.Amount
	}
	for _, r := range dailyIncome {
		ensure(r.Date)["consume"] += r.Amount
	}
	// 资源费
	resMap := make(map[string]float64)
	for _, r := range dailyResource {
		resMap[r.Date] += r.Amount
	}

	y, mo := parseYearMonth(prefix)
	daysInMonth := newDate(y, mo, 1).AddDate(0, 1, -1).Day()
	dailyShare := 0.0
	if daysInMonth > 0 {
		dailyShare = otherExp / float64(daysInMonth)
	}

	// 构建结果
	var daily []map[string]any
	for dd := 1; dd <= daysInMonth; dd++ {
		dateStr := fmt.Sprintf("%s-%02d", prefix, dd)
		d := dailyMap[dateStr]
		if d == nil {
			continue
		}
		hasData := d["meat"]+d["vegetable"]+d["dry"]+d["grain"]+d["condiment"]+d["otherFood"]+d["recharge"]+d["consume"]+d["refund"] > 0
		if !hasData {
			continue
		}
		purchase := d["meat"] + d["vegetable"] + d["dry"] + d["grain"] + d["condiment"] + d["otherFood"]
		var incDayAmt, resDay float64
		for _, ri := range dailyIncome {
			if ri.Date == dateStr {
				incDayAmt = ri.Amount
				break
			}
		}
		resDay = resMap[dateStr]
		inc := incDayAmt + resDay
		profit := inc - purchase - dailyShare
		daily = append(daily, map[string]any{
			"date":      dateStr,
			"meat":      d["meat"],
			"vegetable": d["vegetable"],
			"dry":       d["dry"],
			"grain":     d["grain"],
			"condiment": d["condiment"],
			"otherFood": d["otherFood"],
			"recharge":  d["recharge"],
			"consume":   d["consume"],
			"refund":    d["refund"],
			"profit":    profit,
		})
	}
	return daily
}

// ========== 工具函数 ==========

func defaultMonth(month string) string {
	if month == "" {
		return time.Now().Format("2006-01")
	}
	return month
}

func daysInMonthFromPrefix(prefix string) int {
	y, mo := parseYearMonth(prefix)
	return newDate(y, mo, 1).AddDate(0, 1, -1).Day()
}

func parseYearMonth(s string) (int, int) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		now := time.Now()
		return now.Year(), int(now.Month())
	}
	return t.Year(), int(t.Month())
}

func formatYearMonth(y, mo int) string {
	if mo < 1 {
		mo += 12
		y--
	}
	return time.Date(y, time.Month(mo), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

func newDate(y, mo, day int) time.Time {
	return time.Date(y, time.Month(mo), day, 0, 0, 0, 0, time.UTC)
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

func float64Of(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func getOrDefault(m map[string]map[string]float64, key, field string, defaultVal float64) float64 {
	if row, ok := m[key]; ok {
		if v, ok := row[field]; ok {
			return v
		}
	}
	return defaultVal
}

func mapAt(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		if sub, ok := cur.(map[string]any); ok {
			cur = sub[k]
		} else {
			return nil
		}
	}
	return cur
}

func sortByDate(items []map[string]any, key string) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			di, _ := items[i][key].(string)
			dj, _ := items[j][key].(string)
			if di > dj {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func buildMonthRange(from, to, year string) []string {
	if year != "" {
		y := 0
		fmt.Sscanf(year, "%d", &y)
		if y < 2000 {
			return nil
		}
		months := make([]string, 12)
		for m := 1; m <= 12; m++ {
			months[m-1] = formatYearMonth(y, m)
		}
		return months
	}
	if from == "" && to == "" {
		now := time.Now()
		return []string{formatYearMonth(now.Year(), int(now.Month())-1)}
	}
	if from == "" {
		from = to
	}
	if to == "" {
		to = from
	}
	y1, m1 := parseYearMonth(from)
	y2, m2 := parseYearMonth(to)
	var months []string
	y, mo := y1, m1
	for y < y2 || (y == y2 && mo <= m2) {
		months = append(months, formatYearMonth(y, mo))
		mo++
		if mo > 12 {
			mo = 1
			y++
		}
	}
	return months
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
