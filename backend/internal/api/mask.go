package api

import (
	"encoding/json"
	"strings"

	"siapp/internal/models"
)

// sensitiveConfigKeys 存储/SMTP/通知配置中需要脱敏的键名集合
// 匹配规则：键名小写化后包含这些子串即视为敏感字段，不将原文回传给前端
var sensitiveConfigKeys = []string{
	"secret",
	"password",
	"access_key",
	"api_key",
	"apikey",
	"token",
	"app_secret",
	"webhook_secret",
	"bot_token",
	"authorization",
}

// maskSecretValue 对敏感字符串做脱敏：保留前 4 位 + ****
func maskSecretValue(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return v[:4] + "****"
}

// maskJSONConfig 深拷贝 JSON 配置并脱敏其中的敏感键值（不污染数据库原始对象）
func maskJSONConfig(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return raw
	}
	for key, value := range data {
		if isSensitiveKey(key) {
			if str, ok := value.(string); ok {
				data[key] = maskSecretValue(str)
			}
		}
	}
	out, err := json.Marshal(data)
	if err != nil {
		return raw
	}
	return out
}

// isSensitiveKey 判断键名是否为敏感字段
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, substr := range sensitiveConfigKeys {
		if strings.Contains(lower, substr) {
			return true
		}
	}
	return false
}

// maskStorageConfig 对存储配置响应做脱敏（Config JSON 中的密钥不返回原文）
func maskStorageConfig(c *models.StorageConfig) {
	if c == nil || len(c.Config) == 0 {
		return
	}
	c.Config = maskJSONConfig(c.Config)
}

// maskNotificationConfig 对通知配置响应做脱敏（Config JSON 中的 token/secret 不返回原文）
func maskNotificationConfig(c *models.NotificationConfig) {
	if c == nil || len(c.Config) == 0 {
		return
	}
	c.Config = maskJSONConfig(c.Config)
}
