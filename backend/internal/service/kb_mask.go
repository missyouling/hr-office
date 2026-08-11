package service

import (
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ApplyFieldMask 根据用户的角色和知识库的字段脱敏规则对值进行脱敏
// - admin 角色豁免（如规则的 ExemptRole=admin）
// - 普通用户按 MaskPattern 应用脱敏：front3back4 / all_star
func ApplyFieldMask(db *gorm.DB, user *models.User, kbID uint, fieldName string, value string) string {
	if value == "" {
		return value
	}

	// 查询该知识库指定字段的脱敏规则
	var mask models.KBFieldMask
	if err := db.Where("knowledge_base_id = ? AND field_name = ?", kbID, fieldName).First(&mask).Error; err != nil {
		// 无脱敏规则，返回原值
		return value
	}

	// 检查用户角色是否在豁免列表中
	if mask.ExemptRole != nil {
		normalizedRole := models.NormalizeRole(user.Role)
		if normalizedRole == *mask.ExemptRole {
			return value
		}
	}

	// 根据脱敏模式处理
	switch mask.MaskPattern {
	case "all_star":
		return "***"
	case "front3back4":
		return applyFront3Back4(value)
	default:
		return applyFront3Back4(value)
	}
}

// applyFront3Back4 对字符串进行"保留前3后4字符，中间替换为星号"的脱敏
// 以 rune 为粒度处理，天然兼容中英文等多字节字符
func applyFront3Back4(value string) string {
	runes := []rune(value)
	total := len(runes)

	// 总字符数 ≤ 7 时，直接返回全星号（如短姓名"张三"）
	if total <= 7 {
		return "***"
	}

	// 中间需要替换的星号数量
	middleStars := strings.Repeat("*", total-7)

	// 前3字符 + 星号 + 后4字符
	return string(runes[:3]) + middleStars + string(runes[total-4:])
}

// MaskKBResult 对单条检索结果的指定字段应用脱敏
func MaskKBResult(db *gorm.DB, user *models.User, kbID uint, result map[string]interface{}) map[string]interface{} {
	if len(result) == 0 {
		return result
	}

	// 查询该知识库的所有字段脱敏规则
	var masks []models.KBFieldMask
	db.Where("knowledge_base_id = ?", kbID).Find(&masks)
	if len(masks) == 0 {
		return result
	}

	// 构建字段名到脱敏模式的映射
	maskMap := make(map[string]*models.KBFieldMask, len(masks))
	for i := range masks {
		maskMap[masks[i].FieldName] = &masks[i]
	}

	// 对结果中的每个字段检查是否需要脱敏
	for key, rawValue := range result {
		mask := maskMap[key]
		if mask == nil {
			continue
		}

		// 检查豁免
		if mask.ExemptRole != nil {
			normalizedRole := models.NormalizeRole(user.Role)
			if normalizedRole == *mask.ExemptRole {
				continue
			}
		}

		// 对匹配字段应用脱敏
		if strValue, ok := rawValue.(string); ok {
			result[key] = ApplyFieldMask(db, user, kbID, key, strValue)
		}
	}

	return result
}
