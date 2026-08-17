package models

// 工作台配置偏好常量：天气/新闻配置存储于 UserPreference 表
// （PrefKey = WorkbenchConfigPrefKey），复用现有偏好表，不新增表。
// 与 avatar.go 头像偏好保持同一套持久化模式。
const (
	// WorkbenchConfigPrefKey UserPreference 表中工作台配置的键名
	WorkbenchConfigPrefKey = "workbench_config"
)

// WorkbenchWeatherConfig 天气模块用户配置（本阶段不接入真实第三方 API，
// 仅保存用户配置并返回配置状态/空状态）。
type WorkbenchWeatherConfig struct {
	Enabled bool   `json:"enabled"` // 是否在工作台展示天气卡片
	City    string `json:"city"`    // 用户所在城市
}

// WorkbenchNewsConfig 新闻模块用户配置（本阶段不接入真实第三方 API，
// 仅保存用户配置并返回配置状态/空状态）。
type WorkbenchNewsConfig struct {
	Enabled    bool     `json:"enabled"`    // 是否在工作台展示新闻卡片
	Categories []string `json:"categories"` // 关注的新闻分类
}

// WorkbenchConfig 工作台配置（UserPreference.Value 的 JSON 结构）。
// Weather/News 均可为 null，表示未配置该模块（GET 返回空状态）。
// 允许字段白名单由 api 层 DisallowUnknownFields + 校验函数保证。
type WorkbenchConfig struct {
	Weather *WorkbenchWeatherConfig `json:"weather"`
	News    *WorkbenchNewsConfig    `json:"news"`
}

// DockPreferenceKey UserPreference 表中桌面 Dock 偏好的键名
const DockPreferenceKey = "dock_preferences"

// DockDesktopPosition 桌面端 Dock 锚点位置（像素坐标）。
// 指针字段用于区分"未提供"与"0 值"；NaN/Inf 由 JSON 解码层天然拒绝。
type DockDesktopPosition struct {
	Left *float64 `json:"left"` // 距左侧像素（必填，≥0）
	Top  *float64 `json:"top"`  // 距顶部像素（必填，≥0）
}

// DockPreference 桌面 Dock 偏好（UserPreference.Value 的 JSON 结构）。
// DesktopPosition 为 null 表示恢复默认位置；MobileExpanded 必填布尔。
// 允许字段白名单由 api 层 DisallowUnknownFields + 校验函数保证。
type DockPreference struct {
	DesktopPosition *DockDesktopPosition `json:"desktop_position"`
	MobileExpanded  bool                 `json:"mobile_expanded"`
}
