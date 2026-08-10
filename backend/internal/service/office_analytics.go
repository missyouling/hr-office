package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PeriodCondition 时间段过滤条件
type PeriodCondition struct {
	StartDate string
	EndDate   string
}

// PeriodPair 当前期和上期的过滤条件对
type PeriodPair struct {
	Current PeriodCondition
	Prev    PeriodCondition
}

// GetPeriodFilter 根据类型(monthly/half-yearly/yearly)和日期 计算当前期和上期的时间范围
func GetPeriodFilter(periodType, date string) PeriodPair {
	pair := PeriodPair{}
	switch strings.ToLower(periodType) {
	case "yearly":
		year := date
		pair.Current = PeriodCondition{
			StartDate: year + "-01-01",
			EndDate:   year + "-12-31",
		}
		y, _ := strconv.Atoi(year)
		pair.Prev = PeriodCondition{
			StartDate: fmt.Sprintf("%d-01-01", y-1),
			EndDate:   fmt.Sprintf("%d-12-31", y-1),
		}
	case "half-yearly":
		parts := strings.Split(date, "-")
		if len(parts) != 2 {
			// 退化为月模式
			return getMonthlyPeriodPair(date)
		}
		y, _ := strconv.Atoi(parts[0])
		h, _ := strconv.Atoi(parts[1])
		if h <= 6 {
			pair.Current = PeriodCondition{
				StartDate: fmt.Sprintf("%d-01-01", y),
				EndDate:   fmt.Sprintf("%d-06-30", y),
			}
			pair.Prev = PeriodCondition{
				StartDate: fmt.Sprintf("%d-07-01", y-1),
				EndDate:   fmt.Sprintf("%d-12-31", y-1),
			}
		} else {
			pair.Current = PeriodCondition{
				StartDate: fmt.Sprintf("%d-07-01", y),
				EndDate:   fmt.Sprintf("%d-12-31", y),
			}
			pair.Prev = PeriodCondition{
				StartDate: fmt.Sprintf("%d-01-01", y),
				EndDate:   fmt.Sprintf("%d-06-30", y),
			}
		}
	default:
		// monthly
		pair = getMonthlyPeriodPair(date)
	}
	return pair
}

// getMonthlyPeriodPair 计算月和上月的起止日期
func getMonthlyPeriodPair(date string) PeriodPair {
	parts := strings.Split(date, "-")
	if len(parts) != 2 {
		// 用当前月兜底
		now := time.Now()
		date = now.Format("2006-01")
		parts = strings.Split(date, "-")
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])

	startOfMonth := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)
	prevMonth := startOfMonth.AddDate(0, -1, 0)
	prevEnd := startOfMonth.AddDate(0, 0, -1)

	return PeriodPair{
		Current: PeriodCondition{
			StartDate: startOfMonth.Format("2006-01-02"),
			EndDate:   endOfMonth.Format("2006-01-02"),
		},
		Prev: PeriodCondition{
			StartDate: prevMonth.Format("2006-01-02"),
			EndDate:   prevEnd.Format("2006-01-02"),
		},
	}
}

// AnalyticsSummary 分析摘要结果
type AnalyticsSummary struct {
	TotalAmount    float64 `json:"totalAmount"`
	TotalPurchases int64   `json:"totalPurchases"`
	AvgOrderAmount float64 `json:"avgOrderAmount"`
	YoYChange      float64 `json:"yoyChange"`
	PrevTotal      float64 `json:"prevTotal"`
	CurrentTotal   float64 `json:"currentTotal"`
	ChangePercent  float64 `json:"changePercent"`
}

// CategoryStat 分类统计数据
type CategoryStat struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Quantity int64   `json:"quantity"`
}

// FrequencyItem 采购频率数据
type FrequencyItem struct {
	Period      string  `json:"period"`
	Count       int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// TopSupplyItem 高频用品
type TopSupplyItem struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Spec        string  `json:"spec"`
	Category    string  `json:"category"`
	TotalQty    int64   `json:"total_qty"`
	TotalAmount float64 `json:"total_amount"`
	AvgPrice    float64 `json:"avg_price"`
}

// PriceAnomaly 价格异常项
type PriceAnomaly struct {
	SupplyID         uint    `json:"supplyId"`
	SupplyName       string  `json:"supplyName"`
	Spec             string  `json:"spec"`
	Category         string  `json:"category"`
	LastUnitPrice    float64 `json:"lastUnitPrice"`
	PrevUnitPrice    float64 `json:"prevUnitPrice"`
	ChangePercent    float64 `json:"changePercent"`
	LastPurchaseDate string  `json:"lastPurchaseDate"`
	PrevPurchaseDate string  `json:"prevPurchaseDate"`
}

// SuggestionItem 建议项
type SuggestionItem struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// TrendItem 月度趋势项
type TrendItem struct {
	Period   string  `json:"period"`
	Amount   float64 `json:"amount"`
	Quantity int64   `json:"quantity"`
	Count    int64   `json:"count"`
}

// OfficeAnalyticsService 办公用品分析服务
type OfficeAnalyticsService struct {
	db *gorm.DB
}

// NewOfficeAnalyticsService 构造函数
func NewOfficeAnalyticsService(db *gorm.DB) *OfficeAnalyticsService {
	return &OfficeAnalyticsService{db: db}
}

// GetAnalyticsSummary 获取分析摘要（总额/单数/客单价/同比）
func (s *OfficeAnalyticsService) GetAnalyticsSummary(periodType, date string, userID uint) (*AnalyticsSummary, error) {
	pf := GetPeriodFilter(periodType, date)

	type resultRow struct {
		Amount float64
		Count  int64
	}
	var curRow, prevRow resultRow

	query := s.db.Table("office_purchases").Where("user_id = ?", userID)
	query.Where("purchase_date >= ? AND purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Select("COALESCE(SUM(total_amount),0) as amount, COUNT(*) as count").
		Scan(&curRow)

	query = s.db.Table("office_purchases").Where("user_id = ?", userID)
	query.Where("purchase_date >= ? AND purchase_date <= ?", pf.Prev.StartDate, pf.Prev.EndDate).
		Select("COALESCE(SUM(total_amount),0) as amount, COUNT(*) as count").
		Scan(&prevRow)

	curAmt := curRow.Amount
	prvAmt := prevRow.Amount
	curCnt := curRow.Count
	yoy := 0.0
	if prvAmt > 0 {
		yoy = math.Round(((curAmt-prvAmt)/prvAmt)*10000) / 100
	}
	avgOrder := 0.0
	if curCnt > 0 {
		avgOrder = math.Round(curAmt/float64(curCnt)*100) / 100
	}

	return &AnalyticsSummary{
		TotalAmount:    curAmt,
		TotalPurchases: curCnt,
		AvgOrderAmount: avgOrder,
		YoYChange:      yoy,
		PrevTotal:      prvAmt,
		CurrentTotal:   curAmt,
		ChangePercent:  yoy,
	}, nil
}

// GetCategoryTrend 分类趋势分析
func (s *OfficeAnalyticsService) GetCategoryTrend(periodType, date string, userID uint) ([]CategoryStat, error) {
	pf := GetPeriodFilter(periodType, date)

	var stats []CategoryStat
	err := s.db.Table("office_purchase_items pi").
		Select("oc.name as category, SUM(pi.subtotal) as amount, SUM(pi.quantity) as quantity").
		Joins("JOIN office_purchases p ON pi.purchase_id = p.id").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Joins("JOIN office_categories oc ON os.category_id = oc.id").
		Where("p.user_id = ?", userID).
		Where("p.purchase_date >= ? AND p.purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Group("oc.name").Order("amount DESC").Scan(&stats).Error
	return stats, err
}

// GetFrequency 采购频次分析
func (s *OfficeAnalyticsService) GetFrequency(periodType, date string, userID uint) ([]FrequencyItem, error) {
	pf := GetPeriodFilter(periodType, date)

	var items []FrequencyItem
	err := s.db.Table("office_purchases").
		Select("purchase_date as period, COUNT(*) as count, SUM(total_amount) as total_amount").
		Where("user_id = ?", userID).
		Where("purchase_date >= ? AND purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Group("purchase_date").Order("purchase_date").Scan(&items).Error
	return items, err
}

// GetTopItems 获取高频用品Top N
func (s *OfficeAnalyticsService) GetTopItems(periodType, date string, limit int, userID uint) ([]TopSupplyItem, error) {
	pf := GetPeriodFilter(periodType, date)
	if limit <= 0 {
		limit = 10
	}

	var items []TopSupplyItem
	err := s.db.Table("office_purchase_items pi").
		Select("os.id, os.name, os.spec, oc.name as category, SUM(pi.quantity) as total_qty, SUM(pi.subtotal) as total_amount, AVG(pi.unit_price) as avg_price").
		Joins("JOIN office_purchases p ON pi.purchase_id = p.id").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Joins("LEFT JOIN office_categories oc ON os.category_id = oc.id").
		Where("p.user_id = ?", userID).
		Where("p.purchase_date >= ? AND p.purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Group("os.id").Order("total_amount DESC").Limit(limit).Scan(&items).Error
	return items, err
}

// GetPriceAnomaly 价格异常检测（>5%波动）
func (s *OfficeAnalyticsService) GetPriceAnomaly(periodType, date string, userID uint) ([]PriceAnomaly, error) {
	pf := GetPeriodFilter(periodType, date)

	type priceRecord struct {
		SupplyID     uint    `json:"supply_id"`
		Name         string  `json:"name"`
		Spec         string  `json:"spec"`
		Category     string  `json:"category"`
		UnitPrice    float64 `json:"unit_price"`
		PurchaseDate string  `json:"purchase_date"`
	}

	var records []priceRecord
	err := s.db.Table("office_purchase_items pi").
		Select("pi.supply_id, os.name, os.spec, oc.name as category, pi.unit_price, p.purchase_date").
		Joins("JOIN office_purchases p ON pi.purchase_id = p.id").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Joins("LEFT JOIN office_categories oc ON os.category_id = oc.id").
		Where("p.user_id = ?", userID).
		Where("p.purchase_date >= ? AND p.purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Order("pi.supply_id, p.purchase_date").Scan(&records).Error
	if err != nil {
		return nil, err
	}

	// 按 supply_id 分组，计算价差
	grouped := make(map[uint][]priceRecord)
	for _, r := range records {
		grouped[r.SupplyID] = append(grouped[r.SupplyID], r)
	}

	var anomalies []PriceAnomaly
	for _, prices := range grouped {
		if len(prices) < 2 {
			continue
		}
		last := prices[len(prices)-1]
		prev := prices[len(prices)-2]
		chg := 0.0
		if prev.UnitPrice > 0 {
			chg = math.Round(((last.UnitPrice-prev.UnitPrice)/prev.UnitPrice)*10000) / 100
		}
		if math.Abs(chg) > 5 {
			anomalies = append(anomalies, PriceAnomaly{
				SupplyID:         last.SupplyID,
				SupplyName:       last.Name,
				Spec:             last.Spec,
				Category:         last.Category,
				LastUnitPrice:    last.UnitPrice,
				PrevUnitPrice:    prev.UnitPrice,
				ChangePercent:    chg,
				LastPurchaseDate: last.PurchaseDate,
				PrevPurchaseDate: prev.PurchaseDate,
			})
		}
	}
	return anomalies, nil
}

// GetSuggestions 获取采购建议（5条规则）
func (s *OfficeAnalyticsService) GetSuggestions(periodType, date string, userID uint) ([]SuggestionItem, error) {
	pf := GetPeriodFilter(periodType, date)
	var suggestions []SuggestionItem

	// 规则1: 分类金额增长 > 5000 则警告
	type catRow struct {
		Name string
		Amt  float64
	}
	var cats []catRow
	s.db.Table("office_purchase_items pi").
		Select("oc.name, SUM(pi.subtotal) as amt").
		Joins("JOIN office_purchases p ON pi.purchase_id = p.id").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Joins("JOIN office_categories oc ON os.category_id = oc.id").
		Where("p.user_id = ?", userID).
		Where("p.purchase_date >= ? AND p.purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Group("oc.id").Order("amt DESC").Scan(&cats)

	for _, cat := range cats {
		if cat.Amt > 5000 {
			suggestions = append(suggestions, SuggestionItem{
				Type:        "warning",
				Title:       cat.Name + " 费用偏高",
				Description: fmt.Sprintf("该分类本期采购金额 ¥%.0f，占总费用比例较高，建议核查实际需求。", cat.Amt),
				Action:      "审查采购计划，考虑批量议价",
			})
		}
	}

	// 规则2: 碎片化采购
	var smallCount int64
	s.db.Table("office_purchases").
		Where("user_id = ?", userID).
		Where("purchase_date >= ? AND purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Where("total_amount < ?", 100).Count(&smallCount)
	if smallCount >= 3 {
		suggestions = append(suggestions, SuggestionItem{
			Type:        "optimize",
			Title:       "小额采购频次偏高",
			Description: fmt.Sprintf("本期有 %d 笔采购单金额低于 ¥100，存在碎片化现象，增加物流和管理成本。", smallCount),
			Action:      "合并小额采购为月度集中采购",
		})
	}

	// 规则3: 价格波动 > 15%
	type priceRec struct {
		SupplyID  uint
		Name      string
		UnitPrice float64
	}
	var priceRecords []priceRec
	s.db.Table("office_purchase_items pi").
		Select("pi.supply_id, os.name, pi.unit_price").
		Joins("JOIN office_purchases p ON pi.purchase_id = p.id").
		Joins("JOIN office_supplies os ON pi.supply_id = os.id").
		Where("p.user_id = ?", userID).
		Where("p.purchase_date >= ? AND p.purchase_date <= ?", pf.Current.StartDate, pf.Current.EndDate).
		Order("pi.supply_id, p.purchase_date").Scan(&priceRecords)

	pmap := make(map[uint][]float64)
	nameMap := make(map[uint]string)
	for _, r := range priceRecords {
		pmap[r.SupplyID] = append(pmap[r.SupplyID], r.UnitPrice)
		nameMap[r.SupplyID] = r.Name
	}
	for sid, prices := range pmap {
		if len(prices) >= 2 {
			last := prices[len(prices)-1]
			prev := prices[len(prices)-2]
			chg := 0.0
			if prev > 0 {
				chg = ((last - prev) / prev) * 100
			}
			if chg > 15 {
				suggestions = append(suggestions, SuggestionItem{
					Type:        "warning",
					Title:       nameMap[sid] + " 价格上涨",
					Description: fmt.Sprintf("单价涨幅 %.0f%%，建议关注价格走势并寻找替代供应商。", chg),
					Action:      "询价对比，锁定长期协议价",
				})
			}
		}
	}

	// 规则4: 闲置用品
	activeSet := make(map[uint]bool)
	for _, r := range priceRecords {
		activeSet[r.SupplyID] = true
	}
	var totalActive int64
	s.db.Table("office_supplies").Where("user_id = ? AND status = ?", userID, "active").Count(&totalActive)
	unused := int(totalActive) - len(activeSet)
	if unused > 0 {
		suggestions = append(suggestions, SuggestionItem{
			Type:        "info",
			Title:       fmt.Sprintf("%d 种用品本期未采购", unused),
			Description: "这些用品可能库存充足或已不再需要，建议评估。",
			Action:      "审查用品字典，移除或停用闲置用品",
		})
	}

	// 规则5: 默认健康
	if len(suggestions) == 0 {
		suggestions = append(suggestions, SuggestionItem{
			Type:        "success",
			Title:       "整体采购状况良好",
			Description: "各项指标正常，建议继续保持当前采购策略。",
			Action:      "定期回顾数据，持续优化",
		})
	}

	return suggestions, nil
}

// GetMonthlyTrend 获取多月度趋势
func (s *OfficeAnalyticsService) GetMonthlyTrend(date string, months int, userID uint) ([]TrendItem, error) {
	if months <= 0 {
		months = 12
	}
	parts := strings.Split(date, "-")
	if len(parts) != 2 {
		now := time.Now()
		date = now.Format("2006-01")
		parts = strings.Split(date, "-")
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])

	var result []TrendItem
	for i := 0; i < months; i++ {
		targetM := m - i
		targetY := y
		for targetM < 1 {
			targetM += 12
			targetY--
		}
		period := fmt.Sprintf("%d-%02d", targetY, targetM)
		nextM := targetM + 1
		nextY := targetY
		if nextM > 12 {
			nextM = 1
			nextY++
		}
		endDate := time.Date(nextY, time.Month(nextM), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

		type trendRow struct {
			Amount   float64
			Quantity int64
			Count    int64
		}
		var row trendRow
		s.db.Table("office_purchases p").
			Select("COALESCE(SUM(p.total_amount),0) as amount, COALESCE(SUM(pi.quantity),0) as quantity, COUNT(DISTINCT p.id) as count").
			Joins("JOIN office_purchase_items pi ON pi.purchase_id = p.id").
			Where("p.user_id = ?", userID).
			Where("p.purchase_date >= ? AND p.purchase_date <= ?",
				fmt.Sprintf("%d-%02d-01", targetY, targetM),
				endDate.Format("2006-01-02")).
			Scan(&row)

		result = append(result, TrendItem{
			Period:   period,
			Amount:   row.Amount,
			Quantity: row.Quantity,
			Count:    row.Count,
		})
	}

	// 反转使其按时间正序
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

// GenerateReportHTML 生成分析报告 HTML（用于打印为 PDF）
func (s *OfficeAnalyticsService) GenerateReportHTML(periodType, date string, userID uint) (string, error) {
	summary, err := s.GetAnalyticsSummary(periodType, date, userID)
	if err != nil {
		return "", err
	}
	topItems, err := s.GetTopItems(periodType, date, 10, userID)
	if err != nil {
		return "", err
	}
	anomalies, err := s.GetPriceAnomaly(periodType, date, userID)
	if err != nil {
		return "", err
	}
	suggestions, err := s.GetSuggestions(periodType, date, userID)
	if err != nil {
		return "", err
	}

	topRows := ""
	for i, item := range topItems {
		topRows += fmt.Sprintf(`<tr><td>%d</td><td>%s</td><td>%s</td><td class="num">%.0f</td><td class="num">¥%.2f</td></tr>`,
			i+1, item.Name, item.Category, float64(item.TotalQty), item.TotalAmount)
	}

	anomalyRows := ""
	for _, a := range anomalies {
		anomalyRows += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td class="num">¥%.2f</td><td class="num">¥%.2f</td><td class="num">%.1f%%</td></tr>`,
			a.SupplyName, a.Spec, a.LastUnitPrice, a.PrevUnitPrice, a.ChangePercent)
	}
	if anomalyRows == "" {
		anomalyRows = `<tr><td colspan="5" style="text-align:center;color:#999">未检测到价格异常</td></tr>`
	}

	sugRows := ""
	for _, sg := range suggestions {
		color := "#10b981"
		switch sg.Type {
		case "warning":
			color = "#f59e0b"
		case "optimize":
			color = "#3b82f6"
		case "info":
			color = "#8b5cf6"
		}
		sugRows += fmt.Sprintf(`<tr><td><span style="color:%s;font-weight:bold">%s</span></td><td>%s</td><td>%s</td></tr>`,
			color, sg.Title, sg.Description, sg.Action)
	}

	return fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>办公用品分析报告</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:"Microsoft YaHei","PingFang SC",sans-serif;padding:30px 40px;color:#333;font-size:14px}
h1{font-size:22px;margin-bottom:8px;color:#1e40af}
.meta{color:#666;font-size:13px;margin-bottom:16px}
.cards{display:flex;gap:16px;margin-bottom:24px;flex-wrap:wrap}
.card{flex:1;min-width:150px;padding:14px 16px;border-radius:8px;background:#f0f9ff;border:1px solid #bae6fd}
.card .label{font-size:12px;color:#64748b;margin-bottom:4px}
.card .value{font-size:22px;font-weight:bold;color:#1e40af}
.card .sub{font-size:12px;color:#64748b;margin-top:2px}
h2{font-size:17px;margin:20px 0 10px;color:#334155}
table{width:100%%;border-collapse:collapse;margin-bottom:20px}
th{background:#1e40af;color:#fff;padding:7px 8px;text-align:left;font-size:13px}
td{padding:6px 8px;border-bottom:1px solid #e2e8f0;font-size:13px}
.num{text-align:right;font-family:"Courier New",monospace}
@media print{body{padding:20px 30px}th{background:#1e40af!important;-webkit-print-color-adjust:exact;print-color-adjust:exact}}
</style></head><body>
<h1>📊 办公用品采购分析报告</h1>
<div class="meta">统计范围：%s · 生成时间：%s</div>
<div class="cards">
<div class="card"><div class="label">采购总额</div><div class="value">¥%.2f</div><div class="sub">同比 %.1f%%</div></div>
<div class="card"><div class="label">采购单数</div><div class="value">%d</div><div class="sub">笔</div></div>
<div class="card"><div class="label">客单价</div><div class="value">¥%.2f</div><div class="sub">元/单</div></div>
</div>
<h2>TOP 10 用品</h2>
<table><thead><tr><th>#</th><th>品名</th><th>分类</th><th class="num">数量</th><th class="num">金额</th></tr></thead><tbody>%s</tbody></table>
<h2>价格异常</h2>
<table><thead><tr><th>品名</th><th>规格</th><th class="num">最新价</th><th class="num">前次价</th><th class="num">波动</th></tr></thead><tbody>%s</tbody></table>
<h2>采购建议</h2>
<table><thead><tr><th>标题</th><th>描述</th><th>建议操作</th></tr></thead><tbody>%s</tbody></table>
</body></html>`,
		pfDesc(periodType, date), time.Now().Format("2006-01-02 15:04"),
		summary.TotalAmount, summary.YoYChange, summary.TotalPurchases, summary.AvgOrderAmount,
		topRows, anomalyRows, sugRows), nil
}

func pfDesc(periodType, date string) string {
	switch periodType {
	case "yearly":
		return date + "年度"
	case "half-yearly":
		return date + "半年度"
	default:
		return date + "月"
	}
}
