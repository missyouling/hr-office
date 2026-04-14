package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
)

// extractOCRRequest 同步 OCR 提取请求
type extractOCRRequest struct {
	FileType int `json:"file_type"` // 0=PDF, 1=image
}

// extractOCRAsyncRequest 异步 OCR 提取请求
type extractOCRAsyncRequest struct {
	FileType   int  `json:"file_type"`   // 0=PDF, 1=image
	DocumentID *uint `json:"document_id"` // 可选，关联文档
}

// extractOCR 同步 OCR 提取
// POST /api/ocr/extract
// 接收 multipart/form-data: file + file_type(0=PDF,1=image)
// 返回 OCRResult JSON
func (h *Handler) extractOCR(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required", err)
		return
	}
	defer file.Close()

	// 获取文件类型
	fileTypeStr := strings.TrimSpace(r.FormValue("file_type"))
	fileType := 0
	if fileTypeStr != "" {
		if parsed, err := strconv.Atoi(fileTypeStr); err == nil {
			fileType = parsed
		}
	}

	// 保存临时文件
	targetDir, err := h.ensureUploadDir("ocr", fmt.Sprintf("%d", userID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".pdf"
	}
	filename := fmt.Sprintf("ocr-%d-%d%s", userID, time.Now().UnixNano(), ext)
	storedPath := filepath.Join(targetDir, filename)

	out, err := os.Create(storedPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create file", err)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		_ = out.Close()
		respondError(w, http.StatusInternalServerError, "failed to save file", err)
		return
	}
	if err := out.Close(); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to finalize file", err)
		return
	}

	// 调用 OCR 服务
	result, err := h.ocrService.ExtractSync(userID, storedPath, fileType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "OCR extraction failed", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// extractOCRAsync 异步 OCR 提取
// POST /api/ocr/extract-async
// 接收 multipart/form-data: file + file_type + document_id(optional)
// 返回 { "job_id": uint }
func (h *Handler) extractOCRAsync(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form", err)
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required", err)
		return
	}
	defer file.Close()

	// 获取可选的 document_id
	var documentID *uint
	if docIDStr := strings.TrimSpace(r.FormValue("document_id")); docIDStr != "" {
		if parsed, err := strconv.ParseUint(docIDStr, 10, 64); err == nil && parsed > 0 {
			id := uint(parsed)
			documentID = &id
		}
	}

	// 保存临时文件
	targetDir, err := h.ensureUploadDir("ocr", fmt.Sprintf("%d", userID))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory", err)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".pdf"
	}
	filename := fmt.Sprintf("ocr-async-%d-%d%s", userID, time.Now().UnixNano(), ext)
	storedPath := filepath.Join(targetDir, filename)

	out, err := os.Create(storedPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create file", err)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		_ = out.Close()
		respondError(w, http.StatusInternalServerError, "failed to save file", err)
		return
	}
	if err := out.Close(); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to finalize file", err)
		return
	}

	// 调用 OCR 服务创建异步任务
	jobID, err := h.ocrService.ExtractAsync(userID, storedPath, documentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create OCR job", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id": jobID,
	})
}

// getOCRJobStatus 查询 OCR 任务状态
// GET /api/ocr/jobs/{jobId}
// 返回 OCRJob JSON
func (h *Handler) getOCRJobStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	jobIDStr := strings.TrimSpace(chi.URLParam(r, "jobId"))
	if jobIDStr == "" {
		respondError(w, http.StatusBadRequest, "job id is required", nil)
		return
	}

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id", err)
		return
	}

	// 查询任务
	var job models.OCRJob
	if err := h.db.Where("id = ? AND user_id = ?", uint(jobID), userID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "job not found", err)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to query job", err)
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// getOCRJobResult 获取 OCR 任务结果
// GET /api/ocr/jobs/{jobId}/result
// 返回 OCRResult JSON
func (h *Handler) getOCRJobResult(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	jobIDStr := strings.TrimSpace(chi.URLParam(r, "jobId"))
	if jobIDStr == "" {
		respondError(w, http.StatusBadRequest, "job id is required", nil)
		return
	}

	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id", err)
		return
	}

	// 查询任务
	job, err := h.ocrService.CheckJobStatus(uint(jobID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, "job not found", err)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to query job", err)
		return
	}

	// 验证权限
	if job.UserID != userID {
		respondError(w, http.StatusForbidden, "access denied", nil)
		return
	}

	// 构建结果
	result := &service.OCRResult{
		Success: job.Status == "completed",
		Error:   job.ErrorMessage,
	}

	if job.Result != nil {
		result.RawResult = string(job.Result)
		// 尝试解析结果
		var resultMap map[string]interface{}
		if err := json.Unmarshal(job.Result, &resultMap); err == nil {
			if text, ok := resultMap["text"].(string); ok {
				result.Text = text
			}
			if markdown, ok := resultMap["markdown"].(string); ok {
				result.Markdown = markdown
			}
		}
	}

	result.Provider = job.Provider
	result.Model = "async-job"

	respondJSON(w, http.StatusOK, result)
}
