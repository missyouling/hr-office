package api

import (
	"context"
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

// contractPayload 创建/更新劳动合同请求体。
// 关联员工时快照从员工主表拷贝；未关联员工时由请求体提供快照字段。
type contractPayload struct {
	EmployeeID *uint  `json:"employee_id"`
	ContractNo string `json:"contract_no"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	TermMonths int    `json:"term_months"`
	DocumentID *uint  `json:"document_id"`
	Remarks    string `json:"remarks"`
	// 未关联员工时的手动快照字段
	Name       string `json:"name"`
	Department string `json:"department"`
	Position   string `json:"position"`
	IDNumber   string `json:"id_number"`
}

// contractCancelPayload 作废劳动合同请求体（原因必填）。
type contractCancelPayload struct {
	Reason string `json:"reason"`
}

// registerContractRoutes 注册劳动合同路由（P12.3.2 劳动合同批次）。
// 权限：列表/详情/提醒 contract.view；创建/批量 contract.create；
// 编辑/生效/到期 contract.edit；删除/作废 contract.delete。
func (h *Handler) registerContractRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "contract", "view")).Get("/contracts", h.listContracts)
	r.With(middleware.RequirePermission(h.db, "contract", "create")).Post("/contracts", h.createContract)
	r.With(middleware.RequirePermission(h.db, "contract", "create")).Post("/contracts/batch", h.createContractsBatch)
	r.With(middleware.RequirePermission(h.db, "contract", "view")).Get("/contracts/expiring", h.listExpiringContracts)
	r.With(middleware.RequirePermission(h.db, "contract", "edit")).Post("/contracts/expire", h.expireContracts)
	r.With(middleware.RequirePermission(h.db, "contract", "view")).Get("/contracts/{id}", h.getContract)
	r.With(middleware.RequirePermission(h.db, "contract", "edit")).Put("/contracts/{id}", h.updateContract)
	r.With(middleware.RequirePermission(h.db, "contract", "delete")).Delete("/contracts/{id}", h.deleteContract)
	r.With(middleware.RequirePermission(h.db, "contract", "edit")).Post("/contracts/{id}/activate", h.activateContract)
	r.With(middleware.RequirePermission(h.db, "contract", "delete")).Post("/contracts/{id}/cancel", h.cancelContract)
}

// applyContractDepartmentFilter 根据当前用户所属部门过滤劳动合同（按创建时部门快照）。
func applyContractDepartmentFilter(ctx context.Context, db *gorm.DB) *gorm.DB {
	if dept, ok := middleware.GetUserDepartmentFromContext(ctx); ok && dept != "" {
		return db.Where("snapshot_department = ?", dept)
	}
	return db
}

// loadContract 按用户/部门隔离加载单条劳动合同。
func (h *Handler) loadContract(w http.ResponseWriter, r *http.Request, userID uint) (*models.LaborContract, bool) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "contract id is required", nil)
		return nil, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contract id", err)
		return nil, false
	}
	query := h.db.Where("id = ? AND user_id = ?", id, userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("snapshot_department = ?", dept)
	}
	var contract models.LaborContract
	if err := query.First(&contract).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load contract", err)
		return nil, false
	}
	return &contract, true
}

// validateContractPayload 校验创建/更新劳动合同必填字段。
func validateContractPayload(p *contractPayload) error {
	if strings.TrimSpace(p.StartDate) == "" || strings.TrimSpace(p.EndDate) == "" {
		return errors.New("合同起始日与到期日必填")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.StartDate)); err != nil {
		return errors.New("合同起始日格式必须为 YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(p.EndDate)); err != nil {
		return errors.New("合同到期日格式必须为 YYYY-MM-DD")
	}
	if p.TermMonths <= 0 {
		return errors.New("合同期限月数必须为正整数")
	}
	return nil
}

// checkContractDocument 校验关联档案文档存在且属于当前租户。
func (h *Handler) checkContractDocument(userID uint, documentID *uint) error {
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

// buildContractFromPayload 组装劳动合同记录：
// 关联员工时从员工主表拷贝快照；未关联时使用请求体快照字段。
func (h *Handler) buildContractFromPayload(userID uint, p *contractPayload) (*models.LaborContract, error) {
	contract := &models.LaborContract{
		UserID:       userID,
		EmployeeID:   p.EmployeeID,
		ContractNo:   strings.TrimSpace(p.ContractNo),
		ContractType: models.ContractTypeFixedTerm,
		StartDate:    strings.TrimSpace(p.StartDate),
		EndDate:      strings.TrimSpace(p.EndDate),
		TermMonths:   p.TermMonths,
		DocumentID:   p.DocumentID,
		Remarks:      strings.TrimSpace(p.Remarks),
		Status:       models.ContractStatusDraft,
	}
	if p.EmployeeID != nil {
		var employee models.Employee
		if err := h.db.Where("id = ? AND user_id = ?", *p.EmployeeID, userID).First(&employee).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("关联的员工不存在或不属于当前租户")
			}
			return nil, err
		}
		contract.SnapshotName = employee.Name
		contract.SnapshotDepartment = employee.Department
		contract.SnapshotPosition = employee.Position
		contract.SnapshotIDNumber = employee.IDNumber
	} else {
		contract.SnapshotName = strings.TrimSpace(p.Name)
		contract.SnapshotDepartment = strings.TrimSpace(p.Department)
		contract.SnapshotPosition = strings.TrimSpace(p.Position)
		contract.SnapshotIDNumber = strings.TrimSpace(p.IDNumber)
		if contract.SnapshotDepartment == "" {
			return nil, errors.New("未关联员工时部门必填")
		}
	}
	return contract, nil
}

func (h *Handler) listContracts(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	// 查询前惰性标记已到期合同，保证列表状态准确
	if err := h.markExpiredContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired contracts", err)
		return
	}
	query := applyContractDepartmentFilter(r.Context(), h.db.Where("user_id = ?", userID))
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidContractStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的合同状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	var contracts []models.LaborContract
	if err := query.Order("created_at DESC").Find(&contracts).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list contracts", err)
		return
	}
	if contracts == nil {
		contracts = []models.LaborContract{}
	}
	respondJSON(w, http.StatusOK, contracts)
}

func (h *Handler) getContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	contract, ok := h.loadContract(w, r, userID)
	if !ok {
		return
	}
	// 惰性标记已到期合同，保证详情状态准确
	if err := h.markExpiredContracts(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark expired contracts", err)
		return
	}
	if err := h.db.First(contract, contract.ID).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload contract", err)
		return
	}
	respondJSON(w, http.StatusOK, contract)
}

func (h *Handler) createContract(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload contractPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateContractPayload(&payload); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.checkContractDocument(userID, payload.DocumentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	contract, err := h.buildContractFromPayload(userID, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err := h.db.Create(contract).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create contract", err)
		return
	}
	respondJSON(w, http.StatusCreated, contract)
}

// createContractsBatch 批量创建劳动合同草稿（批次语义）。
// 逐条独立校验，单条失败不影响其他条目；返回成功列表与失败明细。
func (h *Handler) createContractsBatch(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payloads []contractPayload
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(payloads) == 0 {
		respondError(w, http.StatusBadRequest, "contracts cannot be empty", nil)
		return
	}
	created := make([]models.LaborContract, 0, len(payloads))
	failed := make([]map[string]any, 0)
	for i := range payloads {
		if err := validateContractPayload(&payloads[i]); err != nil {
			failed = append(failed, map[string]any{"index": i, "error": err.Error()})
			continue
		}
		if err := h.checkContractDocument(userID, payloads[i].DocumentID); err != nil {
			failed = append(failed, map[string]any{"index": i, "error": err.Error()})
			continue
		}
		contract, err := h.buildContractFromPayload(userID, &payloads[i])
		if err != nil {
			failed = append(failed, map[string]any{"index": i, "error": err.Error()})
			continue
		}
		if err := h.db.Create(contract).Error; err != nil {
			failed = append(failed, map[string]any{"index": i, "error": "创建失败"})
			continue
		}
		created = append(created, *contract)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"created": created,
		"failed":  failed,
	})
}
