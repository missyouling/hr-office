package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"siapp/internal/auth"
	"siapp/internal/models"
	"siapp/internal/service"
	"siapp/internal/supabase"
)

// AuditMiddleware creates middleware for audit logging
func AuditMiddleware(auditService *service.AuditService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Extract user ID from context if available
			// Try Supabase context first, then fall back to legacy auth
			var userID *uint
			var username string

			// Try Supabase JWT context
			if supabaseUserID, err := supabase.GetUserIDFromContext(r.Context()); err == nil {
				// Supabase uses UUID strings, we'll log it in details
				username = supabaseUserID
				// For audit compatibility, we'll use 0 as placeholder for Supabase users
				zeroID := uint(0)
				userID = &zeroID
			} else if id, err := auth.GetUserIDFromContext(r.Context()); err == nil {
				// Legacy JWT
				userID = &id
				if user, err := auth.GetUsernameFromContext(r.Context()); err == nil {
					username = user
				}
			}

			// Process the request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(startTime)

			// Determine action and resource from path
			action, resource, resourceID := determineActionFromPath(r.Method, r.URL.Path)

			// Skip logging for certain paths
			if shouldSkipAuditLog(r.URL.Path) {
				return
			}

			// Extract additional details
			details := extractRequestDetails(r)

			// Add Supabase user info to details if available
			if username != "" {
				if details.Custom == nil {
					details.Custom = make(map[string]interface{})
				}
				details.Custom["supabase_user_id"] = username
				if email, err := supabase.GetUserEmailFromContext(r.Context()); err == nil {
					details.Custom["supabase_email"] = email
				}
			}

			// Log the request
			auditService.LogHTTPRequest(
				r,
				wrapped.statusCode,
				duration,
				userID,
				action,
				resource,
				resourceID,
				"", // Error message would be set by the handler if needed
				details,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// determineActionFromPath determines the audit action based on HTTP method and path
func determineActionFromPath(method, path string) (models.ActionType, string, *string) {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Remove "api" prefix if present
	if len(pathParts) > 0 && pathParts[0] == "api" {
		pathParts = pathParts[1:]
	}

	if len(pathParts) == 0 {
		return models.ActionSystemStart, "system", nil
	}

	resource := pathParts[0]
	var resourceID *string
	var action models.ActionType

	switch resource {
	case "auth":
		if len(pathParts) > 1 {
			switch pathParts[1] {
			case "login":
				action = models.ActionLogin
			case "register":
				action = models.ActionRegister
			case "logout":
				action = models.ActionLogout
			case "change-password":
				action = models.ActionChangePassword
			case "request-password-reset":
				action = models.ActionRequestPasswordReset
			case "reset-password":
				action = models.ActionResetPassword
			case "validate-reset-token":
				action = models.ActionValidateResetToken
			case "verify-email":
				action = models.ActionVerifyEmail
			case "resend-verification":
				action = models.ActionResendVerification
			case "check-email-verification":
				action = models.ActionSendVerificationEmail
			case "profile":
				if method == "GET" {
					action = models.ActionTokenRefresh
				}
			default:
				action = models.ActionTokenRefresh
			}
		}
		resource = "auth"

	case "periods":
		switch method {
		case "GET":
			action = models.ActionSystemStart // List periods
		case "POST":
			if len(pathParts) > 2 && pathParts[2] == "process" {
				action = models.ActionProcessPeriod
				id := pathParts[1]
				resourceID = &id
			} else if len(pathParts) > 2 && pathParts[2] == "reset" {
				action = models.ActionResetPeriod
				id := pathParts[1]
				resourceID = &id
			} else {
				action = models.ActionCreatePeriod
			}
		case "DELETE":
			action = models.ActionDeletePeriod
			if len(pathParts) > 1 {
				id := pathParts[1]
				resourceID = &id
			}
		}

		// Handle sub-resources
		if len(pathParts) > 2 {
			subResource := pathParts[2]
			id := pathParts[1]
			resourceID = &id

			switch subResource {
			case "files":
				if method == "POST" {
					if len(pathParts) > 3 && pathParts[3] == "batch" {
						action = models.ActionUploadBatch
					} else if len(pathParts) > 3 && pathParts[3] == "clear" {
						action = models.ActionClearFiles
					} else {
						action = models.ActionUploadFile
					}
				}
				resource = "files"

			case "roster":
				if method == "POST" {
					action = models.ActionUploadRoster
				}
				resource = "roster"

			case "charges":
				if len(pathParts) > 3 && pathParts[3] == "export" {
					action = models.ActionExportCharges
				} else if len(pathParts) > 4 && pathParts[3] == "scheme" && pathParts[4] == "export" {
					action = models.ActionExportScheme
				}
				resource = "exports"

			case "adjustments":
				if method == "POST" {
					if len(pathParts) > 3 && pathParts[3] == "batch" {
						action = models.ActionUploadAdjustment
					} else if len(pathParts) > 3 && pathParts[3] == "process" {
						action = models.ActionProcessPeriod
					} else if len(pathParts) > 3 && pathParts[3] == "clear" {
						action = models.ActionClearAdjustments
					}
				}
				resource = "adjustments"
			}
		}

	case "roster-template":
		action = models.ActionDownloadTemplate
		resource = "templates"

	case "health":
		action = models.ActionHealthCheck
		resource = "system"

	case "monitoring":
		if len(pathParts) > 1 {
			switch pathParts[1] {
			case "metrics":
				action = models.ActionSystemMetrics
			case "database":
				action = models.ActionDatabaseStatus
			case "info":
				action = models.ActionSystemInfo
			case "maintenance":
				action = models.ActionMaintenance
			default:
				action = models.ActionSystemMetrics
			}
		} else {
			action = models.ActionSystemMetrics
		}
		resource = "system"

	case "version":
		action = models.ActionSystemInfo
		resource = "system"

	default:
		// Check for health sub-paths
		if len(pathParts) >= 2 && pathParts[0] == "health" {
			switch pathParts[1] {
			case "readiness":
				action = models.ActionReadinessCheck
			case "liveness":
				action = models.ActionLivenessCheck
			default:
				action = models.ActionHealthCheck
			}
			resource = "system"
		} else {
			action = models.ActionSystemStart
			resource = "unknown"
		}
	}

	return action, resource, resourceID
}

// shouldSkipAuditLog determines if certain paths should skip audit logging
func shouldSkipAuditLog(path string) bool {
	skipPaths := []string{
		"/favicon.ico",
		"/robots.txt",
		"/health",
		"/metrics",
	}

	for _, skipPath := range skipPaths {
		if strings.HasSuffix(path, skipPath) {
			return true
		}
	}

	return false
}

// extractRequestDetails extracts additional details from the request
func extractRequestDetails(r *http.Request) *models.LogDetails {
	details := &models.LogDetails{
		Custom: make(map[string]interface{}),
	}

	// Extract query parameters
	query := r.URL.Query()
	if len(query) > 0 {
		queryMap := make(map[string]string)
		for key, values := range query {
			if len(values) > 0 {
				queryMap[key] = values[0]
			}
		}
		details.Custom["query"] = queryMap
	}

	// Extract form data for file uploads
	if r.Method == "POST" && strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err == nil { // 32MB max
			if r.MultipartForm != nil {
				// Extract file information
				for fieldName, files := range r.MultipartForm.File {
					if len(files) > 0 {
						file := files[0]
						details.FileName = file.Filename
						details.FileSize = file.Size
						details.Custom["field_name"] = fieldName
						break // Only log the first file
					}
				}

				// Extract form values
				formValues := make(map[string]string)
				for key, values := range r.MultipartForm.Value {
					if len(values) > 0 {
						value := values[0]
						switch key {
						case "scheme":
							details.Scheme = value
						case "part":
							details.Part = value
						default:
							formValues[key] = value
						}
					}
				}
				if len(formValues) > 0 {
					details.Custom["form"] = formValues
				}
			}
		}
	}

	// Extract period ID from path if present
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range pathParts {
		if part == "periods" && i+1 < len(pathParts) {
			if periodID, err := strconv.ParseUint(pathParts[i+1], 10, 32); err == nil {
				details.PeriodID = uint(periodID)
			}
			break
		}
	}

	return details
}

// AuditContext adds audit service to request context
func AuditContext(auditService *service.AuditService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "auditService", auditService)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuditServiceFromContext extracts audit service from request context
func GetAuditServiceFromContext(ctx context.Context) *service.AuditService {
	if auditService, ok := ctx.Value("auditService").(*service.AuditService); ok {
		return auditService
	}
	return nil
}
