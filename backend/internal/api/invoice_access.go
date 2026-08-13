package api

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"siapp/internal/models"
)

// 采购关联校验错误（可识别，不泄露内部信息）。
var (
	errInvoiceSourceInvalid   = errors.New("无效的采购关联类型")
	errInvoiceSourceMissing   = errors.New("关联的采购记录不存在")
	errInvoiceSourceForbidden = errors.New("无权关联该采购记录")
)

// invoiceAccessScope 发票资源访问范围。
type invoiceAccessScope int

const (
	invoiceScopeNone       invoiceAccessScope = iota // 无权访问
	invoiceScopeSelf                                 // 仅本人（普通用户）
	invoiceScopeDepartment                           // 本部门（manager）
	invoiceScopeAll                                  // 全量（admin）
)

// resolveInvoiceAccessScope 解析当前用户对发票资源的访问范围。
func (h *Handler) resolveInvoiceAccessScope(userID uint) invoiceAccessScope {
	var roleNames []string
	h.db.Model(&models.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Pluck("roles.name", &roleNames)
	for _, name := range roleNames {
		switch models.NormalizeRole(name) {
		case models.RoleAdmin:
			return invoiceScopeAll
		case models.RoleManager:
			return invoiceScopeDepartment
		}
	}
	return invoiceScopeSelf
}

// canAccessInvoice 判断当前用户是否有权访问指定发票。
// 软删除记录一律不可见；UserID 为 nil 的发票仅 admin 可访问（普通用户安全拒绝）。
func (h *Handler) canAccessInvoice(userID uint, invoice *models.Invoice) bool {
	if invoice.DeletedAt.Valid {
		return false
	}
	switch h.resolveInvoiceAccessScope(userID) {
	case invoiceScopeAll:
		return true
	case invoiceScopeDepartment:
		if invoice.UserID == nil {
			return false
		}
		return h.sameDepartment(userID, *invoice.UserID)
	default:
		return invoice.UserID != nil && *invoice.UserID == userID
	}
}

// sameDepartment 判断两个用户是否属于同一部门（任一用户无部门视为不同部门）。
func (h *Handler) sameDepartment(userA, userB uint) bool {
	var a, b models.User
	if err := h.db.First(&a, userA).Error; err != nil || a.Department == "" {
		return false
	}
	if err := h.db.First(&b, userB).Error; err != nil || b.Department == "" {
		return false
	}
	return a.Department == b.Department
}

// applyInvoiceScope 将资源访问范围应用到列表/统计/导出查询。
// 软删除由 GORM 默认过滤；UserID=nil 的孤儿记录仅 admin（scopeAll）可见。
func (h *Handler) applyInvoiceScope(query *gorm.DB, scope invoiceAccessScope, userID uint) *gorm.DB {
	switch scope {
	case invoiceScopeAll:
		return query
	case invoiceScopeDepartment:
		// 仅匹配与当前用户同部门且非空部门的用户；无部门经理看不到任何发票
		return query.Where("user_id IN (?)",
			h.db.Model(&models.User{}).Select("id").
				Where("department = (SELECT department FROM users WHERE id = ?) AND department != ''", userID))
	default:
		return query.Where("user_id = ?", userID)
	}
}

// canAccessSource 判断当前用户是否有权访问采购关联记录（归属规则与发票一致）。
func (h *Handler) canAccessSource(userID uint, ownerID *uint) bool {
	if ownerID == nil {
		return false
	}
	switch h.resolveInvoiceAccessScope(userID) {
	case invoiceScopeAll:
		return true
	case invoiceScopeDepartment:
		return h.sameDepartment(userID, *ownerID)
	default:
		return *ownerID == userID
	}
}

// validateInvoiceSource 校验发票采购关联：类型合法、记录存在、未软删、当前用户有权访问。
// 普通用户不能通过发票字段关联他人采购单。可复用，供创建/更新/上传入口调用。
// 说明：当前采购表无软删字段，GORM 默认查询即排除软删；未来增加软删字段后自动生效。
func (h *Handler) validateInvoiceSource(db *gorm.DB, userID uint, sourceType string, sourceID *uint) error {
	sourceType = strings.TrimSpace(sourceType)
	if sourceType == "" || sourceType == models.InvoiceSourceIndependent {
		if sourceID != nil {
			return errInvoiceSourceInvalid
		}
		return nil
	}
	if sourceID == nil {
		return errInvoiceSourceInvalid
	}
	switch sourceType {
	case models.InvoiceSourceOffice:
		var purchase models.OfficePurchase
		if err := db.First(&purchase, *sourceID).Error; err != nil {
			return errInvoiceSourceMissing
		}
		if !h.canAccessSource(userID, purchase.UserID) {
			return errInvoiceSourceForbidden
		}
	case models.InvoiceSourceCanteen:
		var purchase models.CanteenPurchase
		if err := db.First(&purchase, *sourceID).Error; err != nil {
			return errInvoiceSourceMissing
		}
		if !h.canAccessSource(userID, purchase.UserID) {
			return errInvoiceSourceForbidden
		}
	case models.InvoiceSourcePaymentRequest:
		var request models.OfficePaymentRequest
		if err := db.First(&request, *sourceID).Error; err != nil {
			return errInvoiceSourceMissing
		}
		if !h.canAccessSource(userID, request.UserID) {
			return errInvoiceSourceForbidden
		}
	default:
		return errInvoiceSourceInvalid
	}
	return nil
}
