package models

// 头像偏好常量：存储于 UserPreference 表（PrefKey = AvatarPrefKey），
// 不新增 User.avatar 字段，保持与现有 UserPreference API/模型兼容。
const (
	// AvatarPrefKey UserPreference 表中头像偏好的键名
	AvatarPrefKey = "avatar"
	// AvatarTypeDefault 默认 SVG 头像
	AvatarTypeDefault = "default"
	// AvatarTypeCustom 用户上传的自定义头像
	AvatarTypeCustom = "custom"
)

// AvatarPreference 用户头像偏好（UserPreference.Value 的 JSON 结构）。
// Seed 用于稳定生成默认 SVG 配色；CustomFileID 关联 SysFile 元数据。
type AvatarPreference struct {
	Seed              string `json:"seed"`                          // 默认 SVG 配色种子（首次生成后持久化，保持稳定）
	Type              string `json:"type"`                          // default / custom
	CustomFileID      *uint  `json:"custom_file_id,omitempty"`      // 自定义头像的 SysFile ID（type=custom 时有效）
	CustomContentType string `json:"custom_content_type,omitempty"` // 自定义头像的 MIME 类型
}
