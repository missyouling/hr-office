package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// buyerEntitySettingRequest 购方主体设置白名单 DTO，不直接解码模型。
type buyerEntitySettingRequest struct {
	Name        string `json:"name"`
	TaxNo       string `json:"tax_no"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
}

// matchBuyerEntity 解析后自动匹配购方主体；匹配成功返回 true。
// 仅用于解析写回：购方为空或与主体一致时以主体设置填充（识别环节的权威购方）。
func (h *Handler) matchBuyerEntity(tx *gorm.DB, invoice *models.Invoice) bool {
	var setting models.BuyerEntitySetting
	if err := tx.First(&setting, 1).Error; err != nil || setting.Name == "" {
		return false
	}
	if invoice.Buyer == "" || invoice.Buyer == setting.Name {
		invoice.Buyer = setting.Name
		invoice.BuyerTaxNo = setting.TaxNo
		invoice.BuyerMatched = true
		invoice.BuyerMatchNote = "已匹配购方主体"
		return true
	}
	return false
}

// buyerMatchAtConfirm 确认/更正时评估购方主体匹配（不自动强匹配）：
//   - 税号完全匹配（双方非空且一致）→ 强匹配；
//   - 名称匹配但缺税号 → 仅 warning，不写入税号；
//   - 购方缺失或不匹配 → warning。
//
// 返回 warning 列表，同时更新 invoice 的 BuyerMatched/BuyerMatchNote。
func (h *Handler) buyerMatchAtConfirm(tx *gorm.DB, invoice *models.Invoice) []string {
	var setting models.BuyerEntitySetting
	if err := tx.First(&setting, 1).Error; err != nil || strings.TrimSpace(setting.Name) == "" {
		invoice.BuyerMatched = false
		invoice.BuyerMatchNote = "未配置购方主体设置"
		return nil
	}
	name := strings.TrimSpace(invoice.Buyer)
	taxNo := strings.TrimSpace(invoice.BuyerTaxNo)
	expectedTaxNo := strings.TrimSpace(setting.TaxNo)
	if taxNo != "" && expectedTaxNo != "" && strings.EqualFold(taxNo, expectedTaxNo) {
		invoice.BuyerMatched = true
		invoice.BuyerMatchNote = "已匹配购方主体（税号一致）"
		return nil
	}
	if name != "" && strings.EqualFold(name, strings.TrimSpace(setting.Name)) {
		invoice.BuyerMatched = false
		invoice.BuyerMatchNote = "购方名称匹配但缺少税号，未自动匹配"
		return []string{"购方名称匹配但缺少税号，未自动匹配"}
	}
	if name == "" {
		invoice.BuyerMatched = false
		invoice.BuyerMatchNote = "购方信息缺失，未自动匹配"
		return []string{"购方信息缺失，未自动匹配"}
	}
	invoice.BuyerMatched = false
	invoice.BuyerMatchNote = "购方与主体设置不一致"
	return []string{"购方与主体设置不一致"}
}

// getBuyerEntitySetting 读取购方主体设置（仅 admin 路由）。
func (h *Handler) getBuyerEntitySetting(w http.ResponseWriter, r *http.Request) {
	var setting models.BuyerEntitySetting
	if err := h.db.First(&setting, 1).Error; err != nil {
		respondJSON(w, http.StatusOK, models.BuyerEntitySetting{})
		return
	}
	respondJSON(w, http.StatusOK, setting)
}

// updateBuyerEntitySetting 更新购方主体设置（仅 admin 路由）。
// 白名单 DTO 解码；旧值/新值同事务审计；不回写历史发票。
func (h *Handler) updateBuyerEntitySetting(w http.ResponseWriter, r *http.Request) {
	userID, err := getInvoiceUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "未登录", nil)
		return
	}
	var payload buyerEntitySettingRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "请求格式错误", nil)
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		respondError(w, http.StatusBadRequest, "购方主体名称必填", nil)
		return
	}
	var setting models.BuyerEntitySetting
	exists := h.db.First(&setting, 1).Error == nil
	oldSnapshot := buyerEntityAuditSnapshot(setting)
	setting.Name = strings.TrimSpace(payload.Name)
	setting.TaxNo = strings.TrimSpace(payload.TaxNo)
	setting.Address = strings.TrimSpace(payload.Address)
	setting.Phone = strings.TrimSpace(payload.Phone)
	setting.BankName = strings.TrimSpace(payload.BankName)
	setting.BankAccount = strings.TrimSpace(payload.BankAccount)
	if !exists {
		setting.ID = 1
	}
	// 事务：保存 + 旧值/新值审计，任一步失败整体回滚
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&setting).Error; err != nil {
			return err
		}
		rid := strconv.FormatUint(uint64(setting.ID), 10)
		return models.CreateAuditLogWithDB(tx, models.CreateAuditLogParams{
			UserID: &userID, Action: models.ActionUpdateBuyerEntity, Resource: "buyer_entity_setting", ResourceID: &rid,
			Status: models.StatusSuccess, StatusCode: http.StatusOK,
			Details: &models.LogDetails{Custom: map[string]any{
				"old": oldSnapshot,
				"new": buyerEntityAuditSnapshot(setting),
			}},
		})
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "保存购方主体设置失败", nil)
		return
	}
	respondJSON(w, http.StatusOK, setting)
}

// buyerEntityAuditSnapshot 生成主体设置的审计快照（不含敏感无关字段）。
func buyerEntityAuditSnapshot(setting models.BuyerEntitySetting) map[string]any {
	return map[string]any{
		"name": setting.Name, "tax_no": setting.TaxNo, "address": setting.Address,
		"phone": setting.Phone, "bank_name": setting.BankName, "bank_account": setting.BankAccount,
	}
}
