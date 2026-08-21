package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/service"
)

// onboardingImportPayload 手动 JSON 批量导入请求体。
type onboardingImportPayload struct {
	Records []service.OnboardingImportRow `json:"records"`
}

// registerOnboardingImportRoutes 注册入职导入路由（P12.3.2.3）。
// 独立端点与独立模板，不改变旧 employees 导入；
// 权限：导入 employee.create；模板下载 employee.view。
func (h *Handler) registerOnboardingImportRoutes(r chi.Router) {
	r.With(middleware.RequirePermission(h.db, "employee", "create")).Post("/onboarding-records/import", h.importOnboardingRecords)
	r.With(middleware.RequirePermission(h.db, "employee", "create")).Post("/onboarding-records/import/excel", h.importOnboardingRecordsExcel)
	r.With(middleware.RequirePermission(h.db, "employee", "view")).Get("/onboarding-records/template", h.downloadOnboardingTemplate)
}

// importOnboardingRecords 手动 JSON 批量导入（全文件预校验 + 单事务落库）。
func (h *Handler) importOnboardingRecords(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}
	var payload onboardingImportPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload", err)
		return
	}
	if len(payload.Records) == 0 {
		respondError(w, http.StatusBadRequest, "records 不能为空", nil)
		return
	}
	// 全文件预校验：任一错误整体拒绝，不落库
	if errs := service.ValidateOnboardingImportRows(h.db, userID, payload.Records); len(errs) > 0 {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "onboarding_import_validation_failed", "errors": errs})
		return
	}
	records, err := service.ImportOnboardingRecords(h.db, userID, payload.Records)
	if err != nil {
		if errors.Is(err, service.ErrOnboardingEmployeeConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "批量导入失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"imported": len(records), "records": records})
}

// importOnboardingRecordsExcel Excel 文件批量导入（全文件预校验 + 单事务落库）。
func (h *Handler) importOnboardingRecordsExcel(w http.ResponseWriter, r *http.Request) {
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
	rows, err := service.ParseOnboardingExcel(content)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// 全文件预校验：任一错误整体拒绝，不落库
	if errs := service.ValidateOnboardingImportRows(h.db, userID, rows); len(errs) > 0 {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "onboarding_import_validation_failed", "errors": errs})
		return
	}
	records, err := service.ImportOnboardingRecords(h.db, userID, rows)
	if err != nil {
		if errors.Is(err, service.ErrOnboardingEmployeeConflict) {
			respondError(w, http.StatusConflict, err.Error(), nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "批量导入失败", err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"imported": len(records), "records": records})
}

// downloadOnboardingTemplate 下载入职导入 Excel 模板（独立模板，与旧员工模板无关）。
func (h *Handler) downloadOnboardingTemplate(w http.ResponseWriter, r *http.Request) {
	templatePath := service.GetOnboardingTemplatePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := service.GenerateOnboardingTemplate(templatePath); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate onboarding template", err)
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
	w.Header().Set("Content-Disposition", "attachment; filename=\"入职导入模板.xlsx\"")
	if _, err := io.Copy(w, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send template", err)
		return
	}
}
