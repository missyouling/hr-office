package api

import (
	"net/http"
	"strconv"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// GetModelUsageStats - GET /api/settings/models/usage
// Query params: config_type, model_name, start_date, end_date
// Returns: { total_calls, success_calls, failed_calls, success_rate, total_tokens, input_tokens, output_tokens, total_cost, avg_duration_ms, today_calls, today_cost, today_input_tokens, today_output_tokens, rpm, tpm }
func (h *Handler) GetModelUsageStats(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configType := r.URL.Query().Get("config_type")
	modelName := r.URL.Query().Get("model_name")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// 查询用户角色判断是否为管理员（通过 user_roles 联表）
	var isAdmin int64
	h.db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (roles.name = ? OR roles.name = ?)", userID, models.RoleAdmin, "super_admin").
		Count(&isAdmin)

	// 构建查询：管理员看全部数据，普通用户只看自己的
	query := h.db
	if isAdmin == 0 {
		query = query.Where("user_id = ?", userID)
	}

	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	if startDateStr != "" {
		startDate, parseErr := time.Parse("2006-01-02", startDateStr)
		if parseErr == nil {
			query = query.Where("created_at >= ?", startDate)
		}
	}

	if endDateStr != "" {
		endDate, parseErr := time.Parse("2006-01-02", endDateStr)
		if parseErr == nil {
			endDate = endDate.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endDate)
		}
	}

	type StatsResult struct {
		TotalCalls        int64   `json:"total_calls"`
		SuccessCalls      int64   `json:"success_calls"`
		FailedCalls       int64   `json:"failed_calls"`
		SuccessRate       float64 `json:"success_rate"`
		TotalTokens       int64   `json:"total_tokens"`
		InputTokens       int64   `json:"input_tokens"`
		OutputTokens      int64   `json:"output_tokens"`
		TotalCost         float64 `json:"total_cost"`
		AvgDurationMs     float64 `json:"avg_duration_ms"`
		TodayCalls        int64   `json:"today_calls"`
		TodayCost         float64 `json:"today_cost"`
		TodayInputTokens  int64   `json:"today_input_tokens"`
		TodayOutputTokens int64   `json:"today_output_tokens"`
		RPM               float64 `json:"rpm"`
		TPM               float64 `json:"tpm"`
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

	// Calculate today's metrics
	todayStart := time.Now().Truncate(24 * time.Hour)
	var todayStats struct {
		TodayCalls        int64
		TodayCost         float64
		TodayInputTokens  int64
		TodayOutputTokens int64
	}

	err = query.Model(&models.ModelUsageLog{}).
		Where("created_at >= ?", todayStart).
		Select(
			"COUNT(*) as today_calls",
			"SUM(cost_usd) as today_cost",
			"SUM(input_tokens) as today_input_tokens",
			"SUM(output_tokens) as today_output_tokens",
		).
		Scan(&todayStats).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get today stats", err)
		return
	}

	result.TodayCalls = todayStats.TodayCalls
	result.TodayCost = todayStats.TodayCost
	result.TodayInputTokens = todayStats.TodayInputTokens
	result.TodayOutputTokens = todayStats.TodayOutputTokens

	// Calculate RPM (requests per minute) and TPM (tokens per minute) from last 60 seconds
	sixtySecondsAgo := time.Now().Add(-60 * time.Second)
	var recentStats struct {
		RecentCalls  int64
		RecentTokens int64
	}

	err = query.Model(&models.ModelUsageLog{}).
		Where("created_at >= ?", sixtySecondsAgo).
		Select(
			"COUNT(*) as recent_calls",
			"SUM(total_tokens) as recent_tokens",
		).
		Scan(&recentStats).Error

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get recent stats", err)
		return
	}

	result.RPM = float64(recentStats.RecentCalls)
	result.TPM = float64(recentStats.RecentTokens)

	if result.TotalCalls > 0 {
		result.SuccessRate = float64(result.SuccessCalls) / float64(result.TotalCalls) * 100
	}

	respondJSON(w, http.StatusOK, result)
}

// GetModelUsageTrend - GET /api/settings/models/usage/trend
// Query params: period (day/week/month), config_type, model_name, start_date, end_date
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
	modelName := r.URL.Query().Get("model_name")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// 查询用户角色判断是否为管理员（通过 user_roles 联表）
	var isAdmin int64
	h.db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (roles.name = ? OR roles.name = ?)", userID, models.RoleAdmin, "super_admin").
		Count(&isAdmin)

	// 构建查询：管理员看全部数据，普通用户只看自己的
	query := h.db
	if isAdmin == 0 {
		query = query.Where("user_id = ?", userID)
	}
	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	hasExplicitRange := false
	if startDateStr != "" {
		if startDate, parseErr := time.Parse("2006-01-02", startDateStr); parseErr == nil {
			query = query.Where("created_at >= ?", startDate)
			hasExplicitRange = true
		}
	}
	if endDateStr != "" {
		if endDate, parseErr := time.Parse("2006-01-02", endDateStr); parseErr == nil {
			query = query.Where("created_at < ?", endDate.Add(24*time.Hour))
			hasExplicitRange = true
		}
	}

	type TrendItem struct {
		Date         string  `json:"date"`
		TotalCalls   int64   `json:"total_calls"`
		SuccessCalls int64   `json:"success_calls"`
		FailedCalls  int64   `json:"failed_calls"`
		TotalTokens  int64   `json:"total_tokens"`
		InputTokens  int64   `json:"input_tokens"`
		OutputTokens int64   `json:"output_tokens"`
		TotalCost    float64 `json:"total_cost"`
	}

	var results []TrendItem

	if hasExplicitRange {
		err = query.Model(&models.ModelUsageLog{}).
			Select(
				"DATE(created_at) as date",
				"COUNT(*) as total_calls",
				"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
				"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
				"SUM(total_tokens) as total_tokens",
				"SUM(input_tokens) as input_tokens",
				"SUM(output_tokens) as output_tokens",
				"SUM(cost_usd) as total_cost",
			).
			Group("DATE(created_at)").
			Order("date ASC").
			Limit(365).
			Scan(&results).Error
	} else {
		switch period {
		case "week":
			err = query.Model(&models.ModelUsageLog{}).
				Select(
					"DATE(created_at) as date",
					"COUNT(*) as total_calls",
					"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls",
					"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_calls",
					"SUM(total_tokens) as total_tokens",
					"SUM(input_tokens) as input_tokens",
					"SUM(output_tokens) as output_tokens",
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
					"SUM(input_tokens) as input_tokens",
					"SUM(output_tokens) as output_tokens",
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
					"SUM(input_tokens) as input_tokens",
					"SUM(output_tokens) as output_tokens",
					"SUM(cost_usd) as total_cost",
				).
				Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
				Group("DATE(created_at)").
				Order("date ASC").
				Limit(30).
				Scan(&results).Error
		}
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
// Query params: config_type, model_name, start_date, end_date
// Returns: array of { model_name, config_type, provider, total_calls, success_calls, failed_calls, total_tokens, input_tokens, output_tokens, total_cost, avg_duration_ms, success_rate }
func (h *Handler) GetModelUsageByModel(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	configType := r.URL.Query().Get("config_type")
	modelName := r.URL.Query().Get("model_name")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// 查询用户角色判断是否为管理员（通过 user_roles 联表）
	var isAdmin int64
	h.db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (roles.name = ? OR roles.name = ?)", userID, models.RoleAdmin, "super_admin").
		Count(&isAdmin)

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

	// 构建查询：管理员看全部数据，普通用户只看自己的
	query := h.db
	if isAdmin == 0 {
		query = query.Where("user_id = ?", userID)
	}
	if configType != "" {
		query = query.Where("config_type = ?", configType)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if startDateStr != "" {
		if startDate, parseErr := time.Parse("2006-01-02", startDateStr); parseErr == nil {
			query = query.Where("created_at >= ?", startDate)
		}
	}
	if endDateStr != "" {
		if endDate, parseErr := time.Parse("2006-01-02", endDateStr); parseErr == nil {
			query = query.Where("created_at < ?", endDate.Add(24*time.Hour))
		}
	}

	var results []ModelStats
	err = query.
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
