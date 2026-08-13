package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// ============================================================
// 知识库字段脱敏（P9.2）
//
// 字段规则与检索结果字段（content/snippet/title）的映射约定：
//   - 规则 field_name = "content"  → 结果.Content 整段脱敏
//   - 规则 field_name = "snippet"  → 结果.Snippet 整段脱敏
//   - 规则 field_name = "title"    → 结果.Title 整段脱敏
//   - 规则 field_name = 其他业务字段（id_card/phone/address/bank_card/amount/自定义）
//     → 由 kb_field_value.go 的 maskFieldValueInText 做字段级精准替换，
//       仅替换文本中"字段标签 + 值"对应的敏感值，不整段破坏内容；
//       提取的敏感值映射用于最终答案/SSE 增量脱敏（全链路同一规则）。
//
// 安全失败约定：规则查询或豁免角色查询失败一律返回错误，
// 调用方必须拒绝返回原文（不得吞错降级），并记录日志/审计。
// ============================================================

// applyMaskValue 按脱敏模式对单个值脱敏（纯函数，无数据库依赖）
func applyMaskValue(pattern, value string) string {
	switch pattern {
	case "all_star":
		return "***"
	default: // front3back4 及未知模式按 front3back4 处理
		return applyFront3Back4(value)
	}
}

// applyMaskRule 对单个值应用脱敏规则（含 ExemptRole 豁免检查）
// wholeText=true 表示 value 是整段字段文本（content/snippet/title 规则），
// 豁免时提取其中全部数字敏感模式值供答案/SSE 层跳过；
// wholeText=false 表示 value 是单个字段值，豁免时无需提取。
// 豁免角色查询失败返回错误（安全失败）
func applyMaskRule(db *gorm.DB, user *models.User, mask *models.KBFieldMask, value string, wholeText bool) (string, []string, error) {
	if value == "" {
		return value, nil, nil
	}
	exempt, err := userExemptFromMask(db, user, mask)
	if err != nil {
		return "", nil, err
	}
	if exempt {
		if wholeText {
			return value, collectWholeTextExemptValues(value), nil
		}
		return value, nil, nil
	}
	return applyMaskValue(mask.MaskPattern, value), nil, nil
}

// ApplyFieldMask 根据用户的角色和知识库的字段脱敏规则对值进行脱敏
// - 无规则（RecordNotFound）返回原值，nil
// - 查询失败返回错误（安全失败：调用方不得返回原文）
func ApplyFieldMask(db *gorm.DB, user *models.User, kbID uint, fieldName string, value string) (string, error) {
	if value == "" {
		return value, nil
	}

	// 查询该知识库指定字段的脱敏规则
	var mask models.KBFieldMask
	err := db.Where("knowledge_base_id = ? AND field_name = ?", kbID, fieldName).First(&mask).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 无脱敏规则，返回原值
			return value, nil
		}
		logMaskFailure(db, user, kbID, fieldName, err)
		return "", fmt.Errorf("查询脱敏规则失败: %w", err)
	}

	masked, _, err := applyMaskRule(db, user, &mask, value, false)
	if err != nil {
		return "", err
	}
	return masked, nil
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

// MaskKBResult 对单条检索结果的指定字段应用脱敏（map 形式）
// 失败时返回错误，调用方不得返回原文
func MaskKBResult(db *gorm.DB, user *models.User, kbID uint, result map[string]interface{}) (map[string]interface{}, error) {
	if len(result) == 0 {
		return result, nil
	}

	// 查询该知识库的所有字段脱敏规则
	var masks []models.KBFieldMask
	if err := db.Where("knowledge_base_id = ?", kbID).Find(&masks).Error; err != nil {
		logMaskFailure(db, user, kbID, "", err)
		return nil, fmt.Errorf("查询脱敏规则失败: %w", err)
	}
	if len(masks) == 0 {
		return result, nil
	}

	// 构建字段名到脱敏规则的映射
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
		if strValue, ok := rawValue.(string); ok {
			masked, _, err := applyMaskRule(db, user, mask, strValue, false)
			if err != nil {
				return nil, err
			}
			result[key] = masked
		}
	}
	return result, nil
}

// MaskSearchResult 对单条 SearchResult 应用脱敏（含业务字段精准替换）
// kbID 为该结果所属知识库 ID（>0 才脱敏；0 表示无 KB 关联，跳过）
// 返回值：脱敏结果、敏感值映射（原始值→脱敏值，供最终答案/SSE 复用）、
// 豁免原始值列表（ExemptRole 命中时提取，答案/SSE 防御层须跳过）、错误
func MaskSearchResult(db *gorm.DB, user *models.User, kbID uint, result SearchResult) (SearchResult, map[string]string, []string, error) {
	sensitive := map[string]string{}
	var exempt []string
	if kbID == 0 {
		return result, sensitive, exempt, nil
	}

	// 查询该知识库的所有字段脱敏规则（失败安全失败）
	var masks []models.KBFieldMask
	if err := db.Where("knowledge_base_id = ?", kbID).Find(&masks).Error; err != nil {
		logMaskFailure(db, user, kbID, "", err)
		return result, nil, nil, fmt.Errorf("查询脱敏规则失败: %w", err)
	}
	if len(masks) == 0 {
		return result, sensitive, exempt, nil
	}

	// 规则分类：content/snippet/title 直接规则；其余为业务字段规则
	var contentMask, snippetMask, titleMask *models.KBFieldMask
	businessMasks := make([]*models.KBFieldMask, 0, len(masks))
	for i := range masks {
		switch masks[i].FieldName {
		case "content":
			contentMask = &masks[i]
		case "snippet":
			snippetMask = &masks[i]
		case "title":
			titleMask = &masks[i]
		default:
			businessMasks = append(businessMasks, &masks[i])
		}
	}

	// 第一遍：收集所有豁免规则的原始值（从原文提取），并标记豁免字段。
	// 必须先收集再替换：非豁免字段（如 phone）的全局正则可能命中豁免字段
	// 原文的内部子串（如身份证 110101199001011234 内的 19900101123），
	// 提前收集后非豁免替换会跳过这些子串，保证豁免字段原文完整保留。
	exemptedFields := make(map[string]bool, len(masks))
	for i := range masks {
		m := &masks[i]
		ex, err := userExemptFromMask(db, user, m)
		if err != nil {
			return result, nil, nil, err
		}
		if !ex {
			continue
		}
		exemptedFields[m.FieldName] = true
		switch m.FieldName {
		case "content":
			exempt = append(exempt, collectWholeTextExemptValues(result.Content)...)
		case "snippet":
			exempt = append(exempt, collectWholeTextExemptValues(result.Snippet)...)
		case "title":
			exempt = append(exempt, collectWholeTextExemptValues(result.Title)...)
		default:
			kind := fieldKindByRule[m.FieldName]
			labels := fieldLabelPattern(m.FieldName)
			exempt = append(exempt, collectExemptValues(result.Content, kind, labels)...)
			exempt = append(exempt, collectExemptValues(result.Snippet, kind, labels)...)
		}
	}
	exemptSet := toExemptSet(exempt)

	// 第二遍：非豁免规则执行替换（豁免字段跳过；替换时跳过豁免值包含的子串）
	if contentMask != nil && !exemptedFields["content"] {
		masked, vals, err := applyMaskRule(db, user, contentMask, result.Content, true)
		if err != nil {
			return result, nil, nil, err
		}
		result.Content = masked
		exempt = append(exempt, vals...)
	}
	if snippetMask != nil && !exemptedFields["snippet"] {
		masked, vals, err := applyMaskRule(db, user, snippetMask, result.Snippet, true)
		if err != nil {
			return result, nil, nil, err
		}
		result.Snippet = masked
		exempt = append(exempt, vals...)
	}
	if titleMask != nil && !exemptedFields["title"] {
		masked, vals, err := applyMaskRule(db, user, titleMask, result.Title, true)
		if err != nil {
			return result, nil, nil, err
		}
		result.Title = masked
		exempt = append(exempt, vals...)
	}

	// 业务字段规则：精准替换（content 与 snippet 均应用；title 不含业务字段值，跳过）
	for _, mask := range businessMasks {
		if exemptedFields[mask.FieldName] {
			continue // 豁免字段原文保留（第一遍已收集其原始值）
		}
		if result.Content != "" {
			masked, vals, _, err := maskFieldValueInText(db, user, result.Content, mask.FieldName, mask, exemptSet)
			if err != nil {
				return result, nil, nil, err
			}
			result.Content = masked
			mergeSensitive(sensitive, vals)
		}
		if result.Snippet != "" {
			masked, vals, _, err := maskFieldValueInText(db, user, result.Snippet, mask.FieldName, mask, exemptSet)
			if err != nil {
				return result, nil, nil, err
			}
			result.Snippet = masked
			mergeSensitive(sensitive, vals)
		}
	}

	return result, sensitive, exempt, nil
}

// ============================================================
// 敏感模式检测脱敏（答案/SSE 增量防御层）见 kb_mask_stream.go
// ============================================================

// logMaskFailure 记录脱敏失败日志与审计（安全失败：不得吞错降级原文）
func logMaskFailure(db *gorm.DB, user *models.User, kbID uint, fieldName string, err error) {
	if user == nil {
		log.Printf("[kb_mask] 脱敏失败 kb=%d field=%s: %v", kbID, fieldName, err)
		return
	}
	log.Printf("[kb_mask] 脱敏失败 kb=%d field=%s user=%d: %v", kbID, fieldName, user.ID, err)
	resourceID := strconv.FormatUint(uint64(kbID), 10)
	_ = models.CreateAuditLogWithDB(db, models.CreateAuditLogParams{
		UserID:     &user.ID,
		Action:     models.ActionSystemError,
		Resource:   "knowledge_base",
		ResourceID: &resourceID,
		Status:     models.StatusError,
		ErrorMsg:   fmt.Sprintf("字段脱敏失败(%s): %v", fieldName, err),
	})
}
