package api

import (
	"net/http"
	"strconv"
)

// ======== 分析 API ========

func (h *OfficeSupplyHandler) getAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	summary, err := h.analyticsSvc.GetAnalyticsSummary(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询分析摘要失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{
		"totalAmount": summary.TotalAmount, "totalPurchases": summary.TotalPurchases,
		"avgOrderAmount": summary.AvgOrderAmount, "yoyChange": summary.YoYChange,
		"prevTotal": summary.PrevTotal, "currentTotal": summary.CurrentTotal,
		"changePercent": summary.ChangePercent,
	})
}

func (h *OfficeSupplyHandler) getCategoryTrend(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	stats, err := h.analyticsSvc.GetCategoryTrend(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询分类趋势失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"categoryStats": stats})
}

func (h *OfficeSupplyHandler) getFrequency(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	data, err := h.analyticsSvc.GetFrequency(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询频次数据失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"frequencyData": data})
}

func (h *OfficeSupplyHandler) getTopItems(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	items, err := h.analyticsSvc.GetTopItems(periodType, date, limit, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询高频用品失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"topSupplies": items})
}

func (h *OfficeSupplyHandler) getPriceAnomaly(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	anomalies, err := h.analyticsSvc.GetPriceAnomaly(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询价格异常失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"priceAnomalies": anomalies})
}

func (h *OfficeSupplyHandler) getSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	suggestions, err := h.analyticsSvc.GetSuggestions(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询建议失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"suggestions": suggestions})
}

func (h *OfficeSupplyHandler) getTrend(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	date := r.URL.Query().Get("date")
	months, _ := strconv.Atoi(r.URL.Query().Get("months"))
	if months <= 0 {
		months = 12
	}
	trend, err := h.analyticsSvc.GetMonthlyTrend(date, months, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "查询趋势失败")
		return
	}
	respondOfficeOK(w, map[string]interface{}{"trend": trend})
}

func (h *OfficeSupplyHandler) getReportPDF(w http.ResponseWriter, r *http.Request) {
	userID, err := fetchOfficeUser(r)
	if err != nil {
		respondOfficeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	periodType := r.URL.Query().Get("type")
	if periodType == "" {
		periodType = "monthly"
	}
	date := r.URL.Query().Get("date")
	html, err := h.analyticsSvc.GenerateReportHTML(periodType, date, userID)
	if err != nil {
		respondOfficeError(w, http.StatusInternalServerError, "生成报告失败")
		return
	}
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.Write([]byte(html))
}
