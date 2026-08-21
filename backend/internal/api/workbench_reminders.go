package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// ============ 工作台提醒（P5.1-5：档案/宿舍/发票/请款 四类统一提醒；P12.3.5 新增行政合同到期） ============

// 提醒类型枚举（reminder_type 取值）
const (
	reminderTypeDocumentExpiration    = "document_expiration"     // 档案到期（documents.expiration_date）
	reminderTypeDormBillDue           = "dorm_bill_due"           // 宿舍账单到期（dorm_bills.due_date）
	reminderTypeInvoicePending        = "invoice_pending"         // 发票待处理（按 Invoice 状态聚合，不伪造日期）
	reminderTypePaymentPending        = "payment_request_pending" // 请款待处理（按 OfficePaymentRequest 状态聚合，不伪造日期）
	reminderTypeAdminContractExpiring = "admin_contract_expiring" // 行政合同到期（admin_contracts.end_date，P12.3.5）
)

// 校验与状态常量
const (
	workbenchReminderDefaultDays = 30  // days 参数缺省值（天）
	workbenchRemindersMaxDays    = 365 // days 参数合法范围上限（天）

	documentStatusActive = "active" // 档案有效状态
	dormBillStatusPaid   = "paid"   // 宿舍账单"已付清"（未完成 = 非该状态）

	paymentRequestStatusSubmitted = "submitted" // 请款单"已提交待处理"状态
)

// workbenchReminder 工作台提醒统一响应项。
// 仅返回标识、类型、标题、状态与到期时间；敏感字段（身份证号/金额/银行账号等）一律不暴露。
type workbenchReminder struct {
	ID           uint       `json:"id"`
	ReminderType string     `json:"reminder_type"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	DueAt        *time.Time `json:"due_at"` // 发票/请款无到期日期，返回 null
}

// workbenchRemindersResponse 工作台提醒统一响应。
// items 始终为非空数组，无数据时返回空数组而非 null。
type workbenchRemindersResponse struct {
	Days  int                 `json:"days"`
	Items []workbenchReminder `json:"items"`
}

// getWorkbenchReminders GET /api/user/workbench-reminders?days=30
// 汇总当前用户的四类待办提醒：档案到期、宿舍账单到期、发票待处理、请款待处理。
// 必须登录；始终按当前登录用户隔离，不接受任何 user_id 参数。
func (h *Handler) getWorkbenchReminders(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	days, err := parseWorkbenchReminderDays(r.URL.Query().Get("days"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	items, err := h.loadWorkbenchReminders(userID, days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load workbench reminders", err)
		return
	}
	respondJSON(w, http.StatusOK, workbenchRemindersResponse{Days: days, Items: items})
}

// parseWorkbenchReminderDays 解析 days 查询参数：缺省 30，非法（非数字/越界）报错。
func parseWorkbenchReminderDays(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return workbenchReminderDefaultDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > workbenchRemindersMaxDays {
		return 0, fmt.Errorf("invalid days: must be an integer between 1 and %d", workbenchRemindersMaxDays)
	}
	return days, nil
}

// loadWorkbenchReminders 聚合四类提醒，完成去重与稳定排序后返回。
func (h *Handler) loadWorkbenchReminders(userID uint, days int) ([]workbenchReminder, error) {
	now := time.Now()
	end := now.AddDate(0, 0, days)

	items := make([]workbenchReminder, 0, 8)
	seen := make(map[string]struct{})
	// 防御性去重：同一 (reminder_type, id) 只保留一条（四类来源互斥，天然不重复）。
	appendUnique := func(item workbenchReminder) {
		key := fmt.Sprintf("%s:%d", item.ReminderType, item.ID)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}

	docs, err := h.loadExpiringDocuments(userID, now, end)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		appendUnique(workbenchReminder{
			ID:           docs[i].ID,
			ReminderType: reminderTypeDocumentExpiration,
			Title:        reminderTitle(docs[i].DocumentCode, docs[i].FileName),
			Status:       docs[i].Status,
			DueAt:        docs[i].ExpirationDate,
		})
	}

	bills, err := h.loadDormBillsDue(userID, now, end)
	if err != nil {
		return nil, err
	}
	for i := range bills {
		appendUnique(workbenchReminder{
			ID:           bills[i].ID,
			ReminderType: reminderTypeDormBillDue,
			Title:        reminderTitle(bills[i].BillCode, bills[i].PeriodLabel),
			Status:       bills[i].Status,
			DueAt:        &bills[i].DueDate,
		})
	}

	invoices, err := h.loadPendingInvoices(userID)
	if err != nil {
		return nil, err
	}
	for i := range invoices {
		appendUnique(workbenchReminder{
			ID:           invoices[i].ID,
			ReminderType: reminderTypeInvoicePending,
			Title:        reminderTitle(invoices[i].InvoiceNo, invoices[i].Seller),
			Status:       invoices[i].Status,
			DueAt:        nil,
		})
	}

	requests, err := h.loadPendingPaymentRequests(userID)
	if err != nil {
		return nil, err
	}
	for i := range requests {
		appendUnique(workbenchReminder{
			ID:           requests[i].ID,
			ReminderType: reminderTypePaymentPending,
			Title:        requests[i].RequestNo,
			Status:       requests[i].Status,
			DueAt:        nil,
		})
	}

	adminContracts, err := h.loadAdminContractsExpiring(userID, now, end)
	if err != nil {
		return nil, err
	}
	for i := range adminContracts {
		appendUnique(workbenchReminder{
			ID:           adminContracts[i].ID,
			ReminderType: reminderTypeAdminContractExpiring,
			Title:        reminderTitle(adminContracts[i].ContractNo, adminContracts[i].Name),
			Status:       adminContracts[i].Status,
			DueAt:        parseAdminContractEndDate(adminContracts[i].EndDate),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return workbenchReminderLess(items[i], items[j])
	})
	return items, nil
}

// reminderTitle 优先取编码/单号，缺省回退到名称字段。
func reminderTitle(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// loadExpiringDocuments 档案到期：未来 days 天内到期且状态有效（active）的记录。
func (h *Handler) loadExpiringDocuments(userID uint, from, to time.Time) ([]models.Document, error) {
	var docs []models.Document
	err := h.db.
		Where("user_id = ? AND expiration_date IS NOT NULL AND expiration_date >= ? AND expiration_date <= ? AND status = ?",
			userID, from, to, documentStatusActive).
		Order("expiration_date ASC").
		Find(&docs).Error
	return docs, err
}

// loadDormBillsDue 宿舍账单到期：未来 days 天内到期且未完成（非已付清）的账单。
func (h *Handler) loadDormBillsDue(userID uint, from, to time.Time) ([]models.DormBill, error) {
	var bills []models.DormBill
	err := h.db.
		Where("user_id = ? AND due_date >= ? AND due_date <= ? AND (status IS NULL OR status <> ?)",
			userID, from, to, dormBillStatusPaid).
		Order("due_date ASC").
		Find(&bills).Error
	return bills, err
}

// loadPendingInvoices 发票待处理：仅明确待处理状态（已提交待审批 / 已审批待报销）。
// 不伪造日期，due_at 由调用方置空。
func (h *Handler) loadPendingInvoices(userID uint) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := h.db.
		Where("user_id = ? AND status IN ?",
			userID, []string{models.InvoiceStatusSubmitted, models.InvoiceStatusApproved}).
		Order("id ASC").
		Find(&invoices).Error
	return invoices, err
}

// loadPendingPaymentRequests 请款待处理：仅已提交状态（前端两态：草稿/已提交）。
// 不伪造日期，due_at 由调用方置空。
func (h *Handler) loadPendingPaymentRequests(userID uint) ([]models.OfficePaymentRequest, error) {
	var requests []models.OfficePaymentRequest
	err := h.db.
		Where("user_id = ? AND status = ?", userID, paymentRequestStatusSubmitted).
		Order("id ASC").
		Find(&requests).Error
	return requests, err
}

// loadAdminContractsExpiring 行政合同到期（P12.3.5）：生效中（active）且未来 days 天内到期。
// end_date 为 YYYY-MM-DD 字符串，与边界日期做字典序（等价时间序）比较，含今日。
func (h *Handler) loadAdminContractsExpiring(userID uint, from, to time.Time) ([]models.AdminContract, error) {
	var contracts []models.AdminContract
	err := h.db.
		Where("user_id = ? AND status = ? AND end_date >= ? AND end_date <= ?",
			userID, models.AdminContractStatusActive, from.Format("2006-01-02"), to.Format("2006-01-02")).
		Order("end_date ASC").
		Find(&contracts).Error
	return contracts, err
}

// parseAdminContractEndDate 将行政合同到期日字符串解析为时间指针（格式非法返回 nil，不中断提醒聚合）。
func parseAdminContractEndDate(raw string) *time.Time {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil
	}
	return &t
}

// workbenchReminderLess 稳定排序规则：
// due_at 升序（null 排最后）→ reminder_type 升序 → id 升序，保证结果顺序可复现。
func workbenchReminderLess(a, b workbenchReminder) bool {
	switch {
	case a.DueAt == nil && b.DueAt == nil:
		// 继续比较后续字段
	case a.DueAt == nil:
		return false
	case b.DueAt == nil:
		return true
	case !a.DueAt.Equal(*b.DueAt):
		return a.DueAt.Before(*b.DueAt)
	}
	if a.ReminderType != b.ReminderType {
		return a.ReminderType < b.ReminderType
	}
	return a.ID < b.ID
}
