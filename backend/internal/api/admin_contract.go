package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
)

// adminContractPayload 创建/更新行政合同请求体。
// 必填：合同编号/名称/相对方名称/合同类型/起止日期；选填：含税金额/币种/负责人/备注/关联档案文档。
type adminContractPayload struct {
	ContractNo    string   `json:"contract_no"`
	Name          string   `json:"name"`
	Counterparty  string   `json:"counterparty"`
	ContractType  string   `json:"contract_type"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	AmountInclTax *float64 `json:"amount_incl_tax"`
	Currency      string   `json:"currency"`
	Owner         string   `json:"owner"`
	Remarks       string   `json:"remarks"`
	DocumentID    *uint    `json:"document_id"`
}

// adminContractCancelPayload 作废行政合同请求体（原因必填）。
type adminContractCancelPayload struct {
	Reason string `json:"reason"`
}

// registerAdminContractRoutes 注册行政合同路由（P12.3.5）。
// 权限：列表/详情/到期提醒 admin_contract.view；创建 admin_contract.create；
// 编辑/生效/到期 admin_contract.edit；删除/作废 admin_contract.delete。
func (h *Handler) registerAdminContractRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "admin_contract", "view")).Get("/admin-contracts", h.listAdminContracts)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "create")).Post("/admin-contracts", h.createAdminContract)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "view")).Get("/admin-contracts/expiring", h.listExpiringAdminContracts)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "edit")).Post("/admin-contracts/expire", h.expireAdminContracts)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "view")).Get("/admin-contracts/{id}", h.getAdminContract)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "edit")).Put("/admin-contracts/{id}", h.updateAdminContract)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "delete")).Delete("/admin-contracts/{id}", h.deleteAdminContract)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "edit")).Post("/admin-contracts/{id}/activate", h.activateAdminContract)
	r.With(middleware.RequirePermission(h.db, "admin_contract", "delete")).Post("/admin-contracts/{id}/cancel", h.cancelAdminContract)
}

// loadAdminContract 按租户隔离加载单条行政合同。
func (h *Handler) loadAdminContract(w http.ResponseWriter, r *http.Request, userID uint) (*models.AdminContract, bool) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "admin contract id is required", nil)
		return nil, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid admin contract id", err)
		return nil, false
	}
	var contract models.AdminContract
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&contract).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load admin contract", err)
		return nil, false
	}
	return &contract, true
}

// validateAdminContractPayload 校验创建/更新行政合同必填字段。
func validateAdminContractPayload(p *adminContractPayload) error {
	if strings.TrimSpace(p.ContractNo) == "" {
		return errors.New("合同编号必填")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("合同名称必填")
	}
	if strings.TrimSpace(p.Counterparty) == "" {
		return errors.New("相对方名称必填")
	}
	if strings.TrimSpace(p.ContractType) == "" {
		return errors.New("合同类型必填")
	}
	if strings.TrimSpace(p.StartDate) == "" || strings.TrimSpace(p.EndDate) == "" {
		return errors.New("合同起始日与到期日必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.StartDate)); err != nil {
		return errors.New("合同起始日格式必须为 YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.EndDate)); err != nil {
		return errors.New("合同到期日格式必须为 YYYY-MM-DD")
	}
	return nil
}

// checkAdminContractDocument 校验关联档案文档存在且属于当前租户。
func (h *Handler) checkAdminContractDocument(userID uint, documentID *uint) error {
	if documentID == nil {
		return nil
	}
	var count int64
	if err := h.db.Model(&models.Document{}).Where("id = ? AND user_id = ?", *documentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("关联的档案文档不存在或不属于当前租户")
	}
	return nil
}

// buildAdminContractFromPayload 组装行政合同记录（初始草稿态）。
func (h *Handler) buildAdminContractFromPayload(userID uint, p *adminContractPayload) *models.AdminContract {
	return &models.AdminContract{
		UserID:        userID,
		ContractNo:    strings.TrimSpace(p.ContractNo),
		Name:          strings.TrimSpace(p.Name),
		Counterparty:  strings.TrimSpace(p.Counterparty),
		ContractType:  strings.TrimSpace(p.ContractType),
		StartDate:     strings.TrimSpace(p.StartDate),
		EndDate:       strings.TrimSpace(p.EndDate),
		AmountInclTax: p.AmountInclTax,
		Currency:      strings.TrimSpace(p.Currency),
		Owner:         strings.TrimSpace(p.Owner),
		Remarks:       strings.TrimSpace(p.Remarks),
		DocumentID:    p.DocumentID,
		Status:        models.AdminContractStatusDraft,
	}
}

func (h *Handler) listAdminContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	// 查询前惰性标记已到期合同，保证列表状态准确
	if err := h.markExpiredAdminContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired admin contracts", err)
		return
	}
	query := h.db.Where("user_id = ?", userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidAdminContractStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的合同状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var contracts []models.AdminContract
	if err := query.Order("created_at DESC").Find(&contracts).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list admin contracts", err)
		return
	}
	if contracts == nil {
		contracts = []models.AdminContract{}
	}
	respondJSON(w, http.StatusOK, contracts)
}

func (h *Handler) getAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadAdminContract(w, r, userID)
	if !ok {
		return
	}
	// 惰性标记已到期合同，保证详情状态准确
	if err := h.markExpiredAdminContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired admin contracts", err)
		return
	}
	if err := h.db.First(contract, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload admin contract", err)
		return
	}
	respondJSON(w, http.StatusOK, contract)
}

func (h *Handler) createAdminContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload adminContractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateAdminContractPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.checkAdminContractDocument(userID, payload.DocumentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	contract := h.buildAdminContractFromPayload(userID, &payload)
	if err := h.db.Create(contract).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create admin contract", err)
		return
	}
	respondJSON(w, http.StatusCreated, contract)
}
