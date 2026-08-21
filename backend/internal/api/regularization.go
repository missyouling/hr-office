package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// regularizationTimeZone 转正业务日期时区（Asia/Shanghai），不依赖服务器时区。
const regularizationTimeZone = "Asia/Shanghai"

// regularizationNowFunc 返回当前时间（上海时区）；测试可替换以固定业务日期。
var regularizationNowFunc = func() time.Time {
	loc, err := time.LoadLocation(regularizationTimeZone)
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}

// regularizationToday 上海时区当日（YYYY-MM-DD，字典序即时间序）。
func regularizationToday() string {
	return regularizationNowFunc().Format("2006-01-02")
}

// isStrictDate 校验严格 YYYY-MM-DD 日期格式。
func isStrictDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// isRegularizationInFlight 判断状态是否为进行中（存在即拒绝重复发起/不可作废例外见 void）。
func isRegularizationInFlight(status string) bool {
	switch status {
	case models.RegularizationStatusPendingSupervisor,
		models.RegularizationStatusPendingHRReview,
		models.RegularizationStatusScheduled,
		models.RegularizationStatusPostponedScheduled:
		return true
	default:
		return false
	}
}

// registerRegularizationRoutes 注册转正管理路由（P12.3.3-2 只读 + P12.3.3-3 写接口）。
// 权限：employee.edit（与迁移矩阵「转正管理」权限依据一致）。
// 审批/作废/生效均为专用动作端点，不提供通用 Update/Delete。
func (h *Handler) registerRegularizationRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Get("/regularization-records", h.listRegularizationRecords)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Get("/regularization-records/{id}", h.getRegularizationRecord)
	// P12.3.3-3 写接口
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records", h.createRegularizationRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/import", h.importRegularizationRecords)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Get("/regularization-records/template", h.downloadRegularizationTemplate)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/supervisor-approve", h.approveSupervisorRegularization)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/supervisor-reject", h.rejectSupervisorRegularization)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/hr-approve", h.approveHRRegularization)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/hr-reject", h.rejectHRRegularization)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/postpone", h.postponeRegularization)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/void", h.voidRegularizationRecord)
	r.With(middleware.RequirePermission(h.db, "employee", "edit")).Post("/regularization-records/{id}/effect", h.effectRegularizationRecord)
}

// regularizationBaseQuery 构造转正记录查询条件：
//   - 租户隔离：user_id = 当前登录用户；
//   - 部门隔离：复用 DepartmentContext 注入的部门，按快照部门过滤（与员工模块规则一致）。
func (h *Handler) regularizationBaseQuery(r *http.Request, userID uint) *gorm.DB {
	query := h.db.Where("user_id = ?", userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("snapshot_department = ?", dept)
	}
	return query
}

// listRegularizationRecords 转正记录只读列表（支持 status/source 可选过滤）。
func (h *Handler) listRegularizationRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	query := h.regularizationBaseQuery(r, userID)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		if !models.IsValidRegularizationStatus(status) {
			respondError(w, http.StatusBadRequest, "无效的转正状态", nil)
			return
		}
		query = query.Where("status = ?", status)
	}
	if source := strings.TrimSpace(r.URL.Query().Get("source")); source != "" {
		if !models.IsValidRegularizationSource(source) {
			respondError(w, http.StatusBadRequest, "无效的转正来源", nil)
			return
		}
		query = query.Where("source = ?", source)
	}
	var records []models.RegularizationRecord
	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list regularization records", err)
		return
	}
	if records == nil {
		records = []models.RegularizationRecord{}
	}
	respondJSON(w, http.StatusOK, records)
}

// getRegularizationRecord 转正记录只读详情（租户+部门隔离，越权一律 404）。
func (h *Handler) getRegularizationRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, ok := parseRegularizationID(w, r)
	if !ok {
		return
	}
	var record models.RegularizationRecord
	if err := h.regularizationBaseQuery(r, userID).Where("id = ?", id).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "failed to load regularization record", err)
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// ---------- P12.3.3-3 写接口 ----------

// decodeRegularizationPayload 解码请求体（空 body 视为空 payload，字段校验兜底）。
func decodeRegularizationPayload(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// createRegularizationRequest 创建转正申请请求体。
type createRegularizationRequest struct {
	EmployeeID               uint   `json:"employee_id"`                 // 关联员工（必填）
	ContractTermMonths       int    `json:"contract_term_months"`        // 劳动合同期限（月，必填正整数）
	EmployeeSelfReview       string `json:"employee_self_review"`        // 员工自评（可选）
	SupervisorApproverUserID *uint  `json:"supervisor_approver_user_id"` // 直属上级审批人（必填，同租户）
	HRReviewerUserID         *uint  `json:"hr_reviewer_user_id"`         // HR 复核人（必填，同租户）
	PlannedRegularDate       string `json:"planned_regular_date"`        // 计划转正日期（YYYY-MM-DD，允许今天/未来/过去）
	ProbationEndDate         string `json:"probation_end_date"`          // 试用期结束日期（可选；为空时取员工主表快照）
}

// createRegularizationRecord 创建转正申请（创建即提交）：
// 仅 active+trial 员工可发起；快照复制姓名/部门/岗位/用工状态/试用期结束日；
// 发起人=当前用户，直属上级与 HR 复核须同租户且三人两两不同；
// 审批编号服务端生成（租户内唯一），禁止客户端指定。
func (h *Handler) createRegularizationRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var req createRegularizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if err := validateCreateRegularizationRequest(&req, userID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	var initiator models.User
	if err := h.db.First(&initiator, userID).Error; err != nil {
		respondError(w, http.StatusNotFound, "当前用户不存在", err)
		return
	}
	if _, ok := h.loadSameTenantUser(w, &initiator, *req.SupervisorApproverUserID); !ok {
		return
	}
	if _, ok := h.loadSameTenantUser(w, &initiator, *req.HRReviewerUserID); !ok {
		return
	}
	var emp models.Employee
	query := h.db.Where("id = ? AND user_id = ?", req.EmployeeID, userID)
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		query = query.Where("department = ?", dept)
	}
	if err := query.First(&emp).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "员工不存在或不可见", err)
		return
	}
	if emp.Status != models.EmployeeStatusActive || emp.EmploymentStatus != models.EmploymentStatusTrial {
		respondError(w, http.StatusConflict, "仅在职试用期员工可发起转正申请", nil)
		return
	}
	var inFlight int64
	if err := h.db.Model(&models.RegularizationRecord{}).
		Where("user_id = ? AND employee_id = ? AND status IN ?", userID, emp.ID,
			[]string{models.RegularizationStatusPendingSupervisor, models.RegularizationStatusPendingHRReview,
				models.RegularizationStatusScheduled, models.RegularizationStatusPostponedScheduled}).
		Count(&inFlight).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "检查进行中转正申请失败", err)
		return
	}
	if inFlight > 0 {
		respondError(w, http.StatusConflict, "该员工已有进行中的转正申请", nil)
		return
	}
	probationEndDate := strings.TrimSpace(req.ProbationEndDate)
	if probationEndDate == "" {
		probationEndDate = strings.TrimSpace(emp.ProbationEndDate)
	}
	now := time.Now()
	record := models.RegularizationRecord{
		UserID:                   userID,
		EmployeeID:               &emp.ID,
		SnapshotName:             emp.Name,
		SnapshotDepartment:       emp.Department,
		SnapshotPosition:         emp.Position,
		SnapshotEmploymentStatus: emp.EmploymentStatus,
		SnapshotProbationEndDate: probationEndDate,
		ContractTermMonths:       req.ContractTermMonths,
		EmployeeSelfReview:       strings.TrimSpace(req.EmployeeSelfReview),
		InitiatorHRUserID:        &userID,
		SupervisorApproverUserID: req.SupervisorApproverUserID,
		HRReviewerUserID:         req.HRReviewerUserID,
		PlannedRegularDate:       req.PlannedRegularDate,
		Status:                   models.RegularizationStatusPendingSupervisor,
		Source:                   models.RegularizationSourceManual,
		InitiatorSubmittedAt:     &now,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		return createRegularizationWithApprovalNo(tx, &record)
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "创建转正申请失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

// validateCreateRegularizationRequest 创建请求校验：必填、三人不同、日期严格格式。
func validateCreateRegularizationRequest(req *createRegularizationRequest, userID uint) error {
	if req.EmployeeID == 0 {
		return errors.New("employee_id 必填")
	}
	if req.ContractTermMonths <= 0 {
		return errors.New("contract_term_months 必须为正整数")
	}
	if req.SupervisorApproverUserID == nil || *req.SupervisorApproverUserID == 0 {
		return errors.New("直属上级审批人必填")
	}
	if req.HRReviewerUserID == nil || *req.HRReviewerUserID == 0 {
		return errors.New("HR复核人必填")
	}
	if *req.SupervisorApproverUserID == userID ||
		*req.HRReviewerUserID == userID ||
		*req.SupervisorApproverUserID == *req.HRReviewerUserID {
		return errors.New("发起人、直属上级、HR复核人必须为三名不同用户")
	}
	if !isStrictDate(req.PlannedRegularDate) {
		return errors.New("计划转正日期格式必须为 YYYY-MM-DD")
	}
	if req.ProbationEndDate != "" && !isStrictDate(req.ProbationEndDate) {
		return errors.New("试用期结束日期格式必须为 YYYY-MM-DD")
	}
	return nil
}

// loadSameTenantUser 加载审批人并校验与发起人属于同一租户（company_id 相同且非空）。
func (h *Handler) loadSameTenantUser(w http.ResponseWriter, initiator *models.User, userID uint) (*models.User, bool) {
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		respondError(w, http.StatusBadRequest, "审批人不存在", err)
		return nil, false
	}
	if user.CompanyID == "" || user.CompanyID != initiator.CompanyID {
		respondError(w, http.StatusBadRequest, "审批人必须属于同一租户", nil)
		return nil, false
	}
	return &user, true
}

// regularizationApprovalNoMaxAttempts 审批编号唯一冲突最大重试次数。
const regularizationApprovalNoMaxAttempts = 3

// generateRegularizationApprovalNo 服务端生成审批编号（时间+随机，租户内唯一由唯一索引兜底）。
func generateRegularizationApprovalNo() string {
	return fmt.Sprintf("REG-%s-%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}

// createRegularizationWithApprovalNo 创建记录：编号唯一冲突时重新生成并重试。
func createRegularizationWithApprovalNo(tx *gorm.DB, rec *models.RegularizationRecord) error {
	for attempt := 0; attempt < regularizationApprovalNoMaxAttempts; attempt++ {
		rec.ApprovalNo = generateRegularizationApprovalNo()
		if err := tx.Create(rec).Error; err == nil {
			return nil
		} else if !isUniqueViolation(err) {
			return err
		}
	}
	return errors.New("审批编号生成冲突，请重试")
}

// parseRegularizationID 解析路径参数 id。
func parseRegularizationID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "regularization record id is required", nil)
		return 0, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid regularization record id", err)
		return 0, false
	}
	return uint(id), true
}

// loadRegularizationForApproval 加载转正记录用于审批类操作：
//   - 按 id 定位（审批人可能不是记录归属者）；
//   - 部门隔离：当前用户有部门时快照部门须一致，否则 404；
//   - 操作人须为指定审批人（supervisor/hr），否则 409 越级。
func (h *Handler) loadRegularizationForApproval(w http.ResponseWriter, r *http.Request, userID uint, approverField string) (*models.RegularizationRecord, bool) {
	id, ok := parseRegularizationID(w, r)
	if !ok {
		return nil, false
	}
	var record models.RegularizationRecord
	if err := h.db.First(&record, id).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "转正记录不存在", err)
		return nil, false
	}
	if dept, ok := middleware.GetUserDepartmentFromContext(r.Context()); ok && dept != "" {
		if record.SnapshotDepartment != dept {
			respondError(w, http.StatusNotFound, "转正记录不存在", nil)
			return nil, false
		}
	}
	var assigned *uint
	switch approverField {
	case "supervisor":
		assigned = record.SupervisorApproverUserID
	case "hr":
		assigned = record.HRReviewerUserID
	}
	if assigned == nil || *assigned != userID {
		respondError(w, http.StatusConflict, "当前用户不是该步骤指定审批人", nil)
		return nil, false
	}
	return &record, true
}

// reloadRegularizationRecord 重新加载单条记录。
func (h *Handler) reloadRegularizationRecord(w http.ResponseWriter, id uint) (*models.RegularizationRecord, bool) {
	var record models.RegularizationRecord
	if err := h.db.First(&record, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload regularization record", err)
		return nil, false
	}
	return &record, true
}

// ErrRegularizationStateConflict 转正记录状态已变化（重复/并发/越级操作）。
var ErrRegularizationStateConflict = errors.New("转正记录状态已变化，请刷新后重试")

// ErrRegularizationEmployeeState 员工不存在或状态不符合转正生效条件。
var ErrRegularizationEmployeeState = errors.New("员工不存在或状态不符合转正条件")

// updateRegularizationRecord 条件更新转正记录（id+user_id+当前status），0 行视为状态冲突。
func updateRegularizationRecord(tx *gorm.DB, record *models.RegularizationRecord, updates map[string]any) error {
	result := tx.Model(&models.RegularizationRecord{}).
		Where("id = ? AND user_id = ? AND status = ?", record.ID, record.UserID, record.Status).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRegularizationStateConflict
	}
	return nil
}

// supervisorApprovePayload 上级审批通过请求体。
type supervisorApprovePayload struct {
	Comment string `json:"comment"`
}

// approveSupervisorRegularization 上级审批通过：pending_supervisor → pending_hr_review。
func (h *Handler) approveSupervisorRegularization(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadRegularizationForApproval(w, r, userID, "supervisor")
	if !ok {
		return
	}
	var payload supervisorApprovePayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	now := time.Now()
	result := h.db.Model(&models.RegularizationRecord{}).
		Where("id = ? AND user_id = ? AND status = ?", record.ID, record.UserID, models.RegularizationStatusPendingSupervisor).
		Updates(map[string]any{
			"status":                      models.RegularizationStatusPendingHRReview,
			"supervisor_approved_at":      now,
			"supervisor_approval_comment": strings.TrimSpace(payload.Comment),
		})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "上级审批失败", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(w, http.StatusConflict, ErrRegularizationStateConflict.Error(), nil)
		return
	}
	updated, ok := h.reloadRegularizationRecord(w, record.ID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// supervisorRejectPayload 上级拒绝请求体（拒绝原因必填）。
type supervisorRejectPayload struct {
	Reason  string `json:"reason"`
	Comment string `json:"comment"`
}

// rejectSupervisorRegularization 上级拒绝：pending_supervisor → rejected（拒绝原因必填）。
func (h *Handler) rejectSupervisorRegularization(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadRegularizationForApproval(w, r, userID, "supervisor")
	if !ok {
		return
	}
	var payload supervisorRejectPayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "拒绝原因必填", nil)
		return
	}
	now := time.Now()
	result := h.db.Model(&models.RegularizationRecord{}).
		Where("id = ? AND user_id = ? AND status = ?", record.ID, record.UserID, models.RegularizationStatusPendingSupervisor).
		Updates(map[string]any{
			"status":                      models.RegularizationStatusRejected,
			"rejection_reason":            reason,
			"supervisor_rejected_at":      now,
			"supervisor_approval_comment": strings.TrimSpace(payload.Comment),
		})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "上级审批失败", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(w, http.StatusConflict, ErrRegularizationStateConflict.Error(), nil)
		return
	}
	updated, ok := h.reloadRegularizationRecord(w, record.ID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// hrApprovePayload HR 复核通过请求体。
type hrApprovePayload struct {
	Comment string `json:"comment"`
}

// approveHRRegularization HR 复核通过：仅 pending_hr_review；
// 计划日<=上海当日 → 同事务生效员工并转 effective；未来 → 转 scheduled。
func (h *Handler) approveHRRegularization(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadRegularizationForApproval(w, r, userID, "hr")
	if !ok {
		return
	}
	if record.Status != models.RegularizationStatusPendingHRReview {
		respondError(w, http.StatusConflict, "仅待HR复核的转正记录可通过", nil)
		return
	}
	var payload hrApprovePayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	updated, err := h.hrApproveTx(record, strings.TrimSpace(payload.Comment), regularizationToday())
	if err != nil {
		handleRegularizationEffectError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// hrApproveTx HR 复核通过事务：记录更新与员工生效/排期同事务。
func (h *Handler) hrApproveTx(record *models.RegularizationRecord, comment, today string) (*models.RegularizationRecord, error) {
	var updated models.RegularizationRecord
	err := h.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"hr_reviewed_at":    time.Now(),
			"hr_review_comment": comment,
		}
		if record.PlannedRegularDate <= today {
			if err := h.applyRegularizationEffect(tx, record, record.PlannedRegularDate); err != nil {
				return err
			}
			updates["status"] = models.RegularizationStatusEffective
			updates["actual_regular_date"] = record.PlannedRegularDate
		} else {
			updates["status"] = models.RegularizationStatusScheduled
		}
		if err := updateRegularizationRecord(tx, record, updates); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// applyRegularizationEffect 事务内生效员工（trial→formal + actual_regular_date），
// 校验员工 active+trial；失败返回错误由调用方回滚，不得静默改状态。
func (h *Handler) applyRegularizationEffect(tx *gorm.DB, record *models.RegularizationRecord, effectDate string) error {
	if record.EmployeeID == nil {
		return ErrRegularizationEmployeeState
	}
	var emp models.Employee
	if err := tx.Where("id = ? AND user_id = ?", *record.EmployeeID, record.UserID).First(&emp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRegularizationEmployeeState
		}
		return err
	}
	if emp.Status != models.EmployeeStatusActive || emp.EmploymentStatus != models.EmploymentStatusTrial {
		return ErrRegularizationEmployeeState
	}
	return tx.Model(&models.Employee{}).
		Where("id = ? AND user_id = ?", emp.ID, record.UserID).
		Updates(map[string]any{
			"employment_status":   models.EmploymentStatusFormal,
			"actual_regular_date": effectDate,
		}).Error
}

// handleRegularizationEffectError 生效类错误映射：员工状态/状态冲突 → 409，其余 500。
func handleRegularizationEffectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRegularizationEmployeeState), errors.Is(err, ErrRegularizationStateConflict):
		respondError(w, http.StatusConflict, err.Error(), nil)
	default:
		respondError(w, http.StatusInternalServerError, "转正生效失败", err)
	}
}

// hrRejectPayload HR 拒绝请求体（拒绝原因必填）。
type hrRejectPayload struct {
	Reason  string `json:"reason"`
	Comment string `json:"comment"`
}

// rejectHRRegularization HR 复核拒绝：仅 pending_hr_review → rejected，
// 员工保持 active+trial，同事务创建唯一离职办理待办（不自动离职）。
func (h *Handler) rejectHRRegularization(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadRegularizationForApproval(w, r, userID, "hr")
	if !ok {
		return
	}
	if record.Status != models.RegularizationStatusPendingHRReview {
		respondError(w, http.StatusConflict, "仅待HR复核的转正记录可拒绝", nil)
		return
	}
	var payload hrRejectPayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "拒绝原因必填", nil)
		return
	}
	comment := strings.TrimSpace(payload.Comment)
	now := time.Now()
	var updated models.RegularizationRecord
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := updateRegularizationRecord(tx, record, map[string]any{
			"status":            models.RegularizationStatusRejected,
			"rejection_reason":  reason,
			"hr_reviewed_at":    now,
			"hr_review_comment": comment,
		}); err != nil {
			return err
		}
		if err := createRegularizationRejectionTodo(tx, record, reason); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		if errors.Is(err, ErrRegularizationStateConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "HR 复核失败", err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// createRegularizationRejectionTodo 创建唯一离职办理待办（提示 HR 办理离职，不自动离职）。
func createRegularizationRejectionTodo(tx *gorm.DB, record *models.RegularizationRecord, reason string) error {
	todo := models.WorkTodo{
		UserID:       record.UserID,
		BusinessType: "regularization_rejection",
		BusinessID:   record.ID,
		Title:        "转正被驳回，请办理离职手续：" + record.SnapshotName,
		Description:  "员工 " + record.SnapshotName + " 转正申请被驳回，请办理离职手续。驳回原因：" + reason,
		Status:       models.WorkTodoStatusPending,
		AssigneeID:   record.InitiatorHRUserID,
	}
	if err := tx.Create(&todo).Error; err != nil && !isUniqueViolation(err) {
		return err
	}
	return nil
}

// postponeRegularizationPayload 延期请求体（新计划日期与原因必填）。
type postponeRegularizationPayload struct {
	NewPlannedRegularDate string `json:"new_planned_regular_date"`
	Reason                string `json:"reason"`
	Comment               string `json:"comment"`
}

// postponeRegularization HR 复核延期：仅 pending_hr_review、最多一次；
// 新日期<=上海当日 → 同事务立即生效转 effective；未来 → 转 postponed_scheduled；
// 只更新计划日期/延期次数/原计划日期/状态/复核意见，不覆盖快照，不重走上级审批。
func (h *Handler) postponeRegularization(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	record, ok := h.loadRegularizationForApproval(w, r, userID, "hr")
	if !ok {
		return
	}
	if record.Status != models.RegularizationStatusPendingHRReview {
		respondError(w, http.StatusConflict, "仅待HR复核的转正记录可延期", nil)
		return
	}
	if record.ExtensionCount >= 1 {
		respondError(w, http.StatusConflict, "转正申请最多延期一次", nil)
		return
	}
	var payload postponeRegularizationPayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	newDate := strings.TrimSpace(payload.NewPlannedRegularDate)
	reason := strings.TrimSpace(payload.Reason)
	if !isStrictDate(newDate) {
		respondError(w, http.StatusBadRequest, "新计划转正日期格式必须为 YYYY-MM-DD", nil)
		return
	}
	if reason == "" {
		respondError(w, http.StatusBadRequest, "延期原因必填", nil)
		return
	}
	today := regularizationToday()
	now := time.Now()
	updates := map[string]any{
		"planned_regular_date": newDate,
		"extension_count":      record.ExtensionCount + 1,
		"postponed_reason":     reason,
		"hr_reviewed_at":       now,
		"hr_review_comment":    strings.TrimSpace(payload.Comment),
	}
	if record.OriginalPlannedRegularDate == "" {
		updates["original_planned_regular_date"] = record.PlannedRegularDate
	}
	var updated models.RegularizationRecord
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if newDate <= today {
			if err := h.applyRegularizationEffect(tx, record, newDate); err != nil {
				return err
			}
			updates["status"] = models.RegularizationStatusEffective
			updates["actual_regular_date"] = newDate
		} else {
			updates["status"] = models.RegularizationStatusPostponedScheduled
		}
		if err := updateRegularizationRecord(tx, record, updates); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		handleRegularizationEffectError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// voidRegularizationPayload 作废请求体（原因必填）。
type voidRegularizationPayload struct {
	Reason string `json:"reason"`
}

// voidRegularizationRecord 作废转正申请：仅非 effective 终态可作废（错误申请作废后新建），
// 必填原因，状态转 voided；不删除、不覆盖快照。由记录归属者（发起 HR）操作。
func (h *Handler) voidRegularizationRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, ok := parseRegularizationID(w, r)
	if !ok {
		return
	}
	var record models.RegularizationRecord
	if err := h.regularizationBaseQuery(r, userID).Where("id = ?", id).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "转正记录不存在", err)
		return
	}
	if record.Status == models.RegularizationStatusEffective ||
		record.Status == models.RegularizationStatusVoided {
		respondError(w, http.StatusConflict, "已生效或已作废的转正记录不可作废", nil)
		return
	}
	var payload voidRegularizationPayload
	if err := decodeRegularizationPayload(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		respondError(w, http.StatusBadRequest, "作废原因必填", nil)
		return
	}
	now := time.Now()
	result := h.db.Model(&models.RegularizationRecord{}).
		Where("id = ? AND user_id = ? AND status = ?", record.ID, record.UserID, record.Status).
		Updates(map[string]any{
			"status":      models.RegularizationStatusVoided,
			"void_reason": reason,
			"voided_at":   now,
		})
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, "作废失败", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(w, http.StatusConflict, ErrRegularizationStateConflict.Error(), nil)
		return
	}
	updated, ok := h.reloadRegularizationRecord(w, record.ID)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// effectRegularizationRecord 人工生效（scheduled/postponed_scheduled 未来记录）：
// 仅 employee.edit；再次校验员工 active+trial 与计划日期<=上海当日，成功事务性更新；
// 失败返回错误，不得静默改状态。由记录归属者（发起 HR）操作。
func (h *Handler) effectRegularizationRecord(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	id, ok := parseRegularizationID(w, r)
	if !ok {
		return
	}
	var record models.RegularizationRecord
	if err := h.regularizationBaseQuery(r, userID).Where("id = ?", id).First(&record).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		respondError(w, status, "转正记录不存在", err)
		return
	}
	if record.Status != models.RegularizationStatusScheduled &&
		record.Status != models.RegularizationStatusPostponedScheduled {
		respondError(w, http.StatusConflict, "仅排期中的转正记录可人工生效", nil)
		return
	}
	today := regularizationToday()
	if record.PlannedRegularDate > today {
		respondError(w, http.StatusConflict, "计划转正日期未到，不能提前生效", nil)
		return
	}
	var updated models.RegularizationRecord
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.applyRegularizationEffect(tx, &record, record.PlannedRegularDate); err != nil {
			return err
		}
		if err := updateRegularizationRecord(tx, &record, map[string]any{
			"status":              models.RegularizationStatusEffective,
			"actual_regular_date": record.PlannedRegularDate,
		}); err != nil {
			return err
		}
		return tx.First(&updated, record.ID).Error
	})
	if err != nil {
		handleRegularizationEffectError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, updated)
}

// downloadRegularizationTemplate 下载转正导入模板。
func (h *Handler) downloadRegularizationTemplate(w http.ResponseWriter, r *http.Request) {
	templatePath := service.GetRegularizationTemplatePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := service.GenerateRegularizationTemplate(templatePath); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate regularization template", err)
			return
		}
	}
	file, err := os.Open(templatePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to open template", err)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"转正导入模板.xlsx\"")
	if _, err := io.Copy(w, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send template", err)
		return
	}
}

type regularizationImportResponse struct {
	Imported int                              `json:"imported"`
	Records  []models.RegularizationRecord     `json:"records"`
	Warnings []service.RegularizationImportWarning `json:"warnings,omitempty"`
}

// importRegularizationRecords Excel 批量转正导入（employee.edit）。
func (h *Handler) importRegularizationRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required", err)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read file", err)
		return
	}
	rows, err := service.ParseRegularizationExcel(content)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	warnings, errs := service.ValidateRegularizationImportRows(h.db, userID, rows)
	if len(errs) > 0 {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "regularization_import_validation_failed", "warnings": warnings, "errors": errs})
		return
	}
	records, warnings, err := service.BuildRegularizationRecords(h.db, userID, rows)
	if err != nil {
		respondError(w, http.StatusConflict, err.Error(), nil)
		return
	}
	created := make([]models.RegularizationRecord, 0, len(records))
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, rec := range records {
			row := rec
			if err := createRegularizationWithApprovalNo(tx, &row); err != nil {
				return err
			}
			if row.PlannedRegularDate <= regularizationToday() {
				if err := h.applyRegularizationEffect(tx, &row, row.PlannedRegularDate); err != nil {
					return err
				}
				row.Status = models.RegularizationStatusEffective
				row.ActualRegularDate = row.PlannedRegularDate
			} else {
				row.Status = models.RegularizationStatusScheduled
			}
			if err := tx.Model(&models.RegularizationRecord{}).Where("id = ? AND user_id = ?", row.ID, userID).Updates(map[string]any{
				"status":              row.Status,
				"actual_regular_date": row.ActualRegularDate,
			}).Error; err != nil {
				return err
			}
			created = append(created, row)
		}
		return nil
	}); err != nil {
		respondError(w, http.StatusInternalServerError, "批量导入失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, regularizationImportResponse{Imported: len(created), Records: created, Warnings: warnings})
}
