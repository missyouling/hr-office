package api

import (
	"net/http"
	"strconv"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// GetModelUsageStats - GET /api/settings/models/usage
// Query params: config_type, start_date, end_date
// Returns: { total_calls, success_calls, failed_calls, success_rate, total_tokens, input_tokens, output_tokens, total_cost, avg_duration_ms }
func (h *Handler) GetModelUsageStats(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configType := r.URL.Query().Get("config_type")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	query := h.db.Where("user_id = ?", userID)

	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			query = query.Where("created_at >= ?", startDate)
		}
	}

	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = endDate.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endDate)
		}
	}

	type StatsResult struct {
		TotalCalls    int64   `json:"total_calls"`
		SuccessCalls  int64   `json:"success_calls"`
		FailedCalls   int64   `json:"failed_calls"`
		SuccessRate   float64 `json:"success_rate"`
		TotalTokens   int64   `json:"total_tokens"`
		InputTokens   int64   `json:"input_tokens"`
		OutputTokens  int64   `json:"output_tokens"`
		TotalCost     float64 `json:"total_cost"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
	}

	var result StatsResult
	err = query.Model(&models.ModelUsageLog{}).
		Select(
			"COUNT(*) as total_calls",
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
			"SUM(total_tokens) as total_tokens",
			"SUM(input_tokens) as input_tokens",
			"SUM(output_tokens) as output_tokens",
			"SUM(cost_usd) as total_cost",
			"AVG(duration_ms) as avg_duration_ms",
		).
		Scan(&result).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get usage stats", err)
		return
	}

	if result.TotalCalls > 0 {
		result.SuccessRate = float64(result.SuccessCalls) / float64(result.TotalCalls) * 100
	}

	respondJSON(w, http.StatusOK, result)
}

// GetModelUsageTrend - GET /api/settings/models/usage/trend
// Query params: period (day/week/month), config_type
// Returns: array of { date, total_calls, success_calls, failed_calls, total_tokens, total_cost }
func (h *Handler) GetModelUsageTrend(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	configType := r.URL.Query().Get("config_type")

	query := h.db.Where("user_id = ?", userID)
	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}

	type TrendItem struct {
		Date         string  `json:"date"`
		TotalCalls   int64   `json:"total_calls"`
		SuccessCalls int64   `json:"success_calls"`
		FailedCalls  int64   `json:"failed_calls"`
		TotalTokens  int64   `json:"total_tokens"`
		TotalCost    float64 `json:"total_cost"`
	}

	var results []TrendItem

	switch period {
	case "week":
		err = query.Model(&models.ModelUsageLog{}).
			Select(
				"DATE(created_at) as date",
				"COUNT(*) as total_calls",
				"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
				"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
				"SUM(total_tokens) as total_tokens",
				"SUM(cost_usd) as total_cost",
			).
			Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).
			Group("DATE(created_at)").
			Order("date ASC").
			Limit(30).
			Scan(&results).Error
	case "month":
		err = query.Model(&models.ModelUsageLog{}).
			Select(
				"TO_CHAR(created_at, 'YYYY-MM') as date",
				"COUNT(*) as total_calls",
				"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
				"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
				"SUM(total_tokens) as total_tokens",
				"SUM(cost_usd) as total_cost",
			).
			Where("created_at >= ?", time.Now().AddDate(0, -12, 0)).
			Group("TO_CHAR(created_at, 'YYYY-MM')").
			Order("date ASC").
			Limit(30).
			Scan(&results).Error
	default:
		err = query.Model(&models.ModelUsageLog{}).
			Select(
				"DATE(created_at) as date",
				"COUNT(*) as total_calls",
				"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
				"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
				"SUM(total_tokens) as total_tokens",
				"SUM(cost_usd) as total_cost",
			).
			Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
			Group("DATE(created_at)").
			Order("date ASC").
			Limit(30).
			Scan(&results).Error
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get usage trend", err)
		return
	}

	if results == nil {
		results = []TrendItem{}
	}

	respondJSON(w, http.StatusOK, results)
}

// GetModelUsageByModel - GET /api/settings/models/usage/by-model
// Returns: array of { model_name, config_type, provider, total_calls, success_calls, failed_calls, total_tokens, input_tokens, output_tokens, total_cost, avg_duration_ms, success_rate }
func (h *Handler) GetModelUsageByModel(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	type ModelStats struct {
		ModelName     string  `json:"model_name"`
		ConfigType    string  `json:"config_type"`
		Provider      string  `json:"provider"`
		TotalCalls    int64   `json:"total_calls"`
		SuccessCalls  int64   `json:"success_calls"`
		FailedCalls   int64   `json:"failed_calls"`
		TotalTokens   int64   `json:"total_tokens"`
		InputTokens   int64   `json:"input_tokens"`
		OutputTokens  int64   `json:"output_tokens"`
		TotalCost     float64 `json:"total_cost"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		SuccessRate   float64 `json:"success_rate"`
	}

	var results []ModelStats
	err = h.db.Where("user_id = ?", userID).
		Model(&models.ModelUsageLog{}).
		Select(
			"model_name",
			"config_type",
			"provider",
			"COUNT(*) as total_calls",
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
			"SUM(total_tokens) as total_tokens",
			"SUM(input_tokens) as input_tokens",
			"SUM(output_tokens) as output_tokens",
			"SUM(cost_usd) as total_cost",
			"AVG(duration_ms) as avg_duration_ms",
		).
		Group("model_name, config_type, provider").
		Order("total_calls DESC").
		Scan(&results).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get usage by model", err)
		return
	}

	for i := range results {
		if results[i].TotalCalls > 0 {
			results[i].SuccessRate = float64(results[i].SuccessCalls) / float64(results[i].TotalCalls) * 100
		}
	}

	if results == nil {
		results = []ModelStats{}
	}

	respondJSON(w, http.StatusOK, results)
}

// CleanupOldUsageLogs - DELETE /api/settings/models/usage/cleanup
// Query params: days (default 30)
func (h *Handler) CleanupOldUsageLogs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)

	result := h.db.Where("user_id = ? AND created_at < ?", userID, cutoffDate).
		Delete(&models.ModelUsageLog{})

	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "failed to cleanup usage logs", result.Error)
		return
	}

	response := map[string]interface{}{
		"deleted_count": result.RowsAffected,
		"cutoff_date":   cutoffDate.Format("2006-01-02"),
	}

	respondJSON(w, http.StatusOK, response)
}
