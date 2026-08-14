package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// newPasswordTestHandler 构造完整认证 handler（改密依赖 passwordResetService + emailService）
func newPasswordTestHandler(db *gorm.DB) *AuthHandler {
	return NewAuthHandler(
		db, nil,
		service.NewPasswordResetService(db),
		nil,
		service.NewEmailService(),
		nil,
	)
}

// changePasswordRequest 构造带认证上下文的 POST /auth/change-password 请求
func changePasswordRequest(t *testing.T, userID uint, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID))
	return req
}

// seedRefreshToken 插入一条未过期 refresh token 记录
func seedRefreshToken(t *testing.T, db *gorm.DB, userID uint, raw string) {
	t.Helper()
	token := models.AuthToken{
		UserID:    userID,
		TokenHash: auth.HashToken(raw),
		Type:      "refresh",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := db.Create(&token).Error; err != nil {
		t.Fatalf("插入 refresh token 失败: %v", err)
	}
}

// ========== 密码修改：正常流程 ==========

func TestChangePassword_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "chgpwd", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := newPasswordTestHandler(db)
	req := changePasswordRequest(t, user.ID, `{"current_password":"pwd123","new_password":"newpass123"}`)
	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("修改密码返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if !stored.CheckPassword("newpass123") {
		t.Error("新密码应生效")
	}
	if stored.CheckPassword("pwd123") {
		t.Error("旧密码应失效")
	}
}

// ========== 密码修改：错误当前密码 ==========

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "chgwrong", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := newPasswordTestHandler(db)
	req := changePasswordRequest(t, user.ID, `{"current_password":"wrongpass","new_password":"newpass123"}`)
	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("错误当前密码返回 %d，期望 400。body: %s", rec.Code, rec.Body.String())
	}

	// 密码未被修改
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if !stored.CheckPassword("pwd123") {
		t.Error("当前密码错误时原密码应保持不变")
	}
}

// ========== 密码修改：新密码规则 ==========

func TestChangePassword_InvalidNewPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "chgrule", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := newPasswordTestHandler(db)
	bodies := []string{
		`{"current_password":"pwd123","new_password":"ab1"}`,    // 长度不足 6
		`{"current_password":"pwd123","new_password":"abcdef"}`, // 无数字
		`{"current_password":"pwd123","new_password":"123456"}`, // 无字母
	}
	for _, body := range bodies {
		req := changePasswordRequest(t, user.ID, body)
		rec := httptest.NewRecorder()
		handler.ChangePassword(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("请求体 %s 返回 %d，期望 400", body, rec.Code)
		}
	}

	// 密码未被修改
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if !stored.CheckPassword("pwd123") {
		t.Error("新密码不符合规则时原密码应保持不变")
	}
}

// ========== 密码修改：refresh token 吊销 ==========

func TestChangePassword_RevokesRefreshTokens(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "chgrevoke", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	seedRefreshToken(t, db, user.ID, "rt-1")
	seedRefreshToken(t, db, user.ID, "rt-2")

	// 其他用户的 refresh token 不应被误伤
	other, err := createAuthTestUser(db, "chgother", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建其他用户失败: %v", err)
	}
	seedRefreshToken(t, db, other.ID, "rt-other")

	handler := newPasswordTestHandler(db)
	req := changePasswordRequest(t, user.ID, `{"current_password":"pwd123","new_password":"newpass123"}`)
	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("修改密码返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	// 该用户所有未过期 refresh token 应被吊销
	var activeCount int64
	db.Model(&models.AuthToken{}).
		Where("user_id = ? AND type = ? AND is_revoked = ?", user.ID, "refresh", false).
		Count(&activeCount)
	if activeCount != 0 {
		t.Errorf("该用户所有 refresh token 应被吊销，剩余未吊销 %d", activeCount)
	}

	// 其他用户 token 不受影响
	var otherToken models.AuthToken
	if err := db.Where("user_id = ? AND type = ?", other.ID, "refresh").First(&otherToken).Error; err != nil {
		t.Fatalf("查询其他用户 token 失败: %v", err)
	}
	if otherToken.IsRevoked {
		t.Error("其他用户的 refresh token 不应被吊销")
	}
}

// ========== 密码修改：SupabaseUID 非空用户明确拒绝 ==========

func TestChangePassword_SupabaseUserRejected(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "supauser", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	user.SupabaseUID = "00000000-0000-0000-0000-000000000001"
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("设置 SupabaseUID 失败: %v", err)
	}
	seedRefreshToken(t, db, user.ID, "rt-supa")

	handler := newPasswordTestHandler(db)
	req := changePasswordRequest(t, user.ID, `{"current_password":"pwd123","new_password":"newpass123"}`)
	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Supabase 用户改密返回 %d，期望 400（明确拒绝）。body: %s", rec.Code, rec.Body.String())
	}

	// 密码未被悄悄修改
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if !stored.CheckPassword("pwd123") {
		t.Error("Supabase 用户密码不应被本地悄悄修改")
	}

	// refresh token 未被吊销（拒绝发生在改密之前）
	var token models.AuthToken
	if err := db.Where("user_id = ? AND type = ?", user.ID, "refresh").First(&token).Error; err != nil {
		t.Fatalf("查询 refresh token 失败: %v", err)
	}
	if token.IsRevoked {
		t.Error("Supabase 用户被拒绝后 refresh token 不应被吊销")
	}
}

// ========== 密码修改：未认证 ==========

func TestChangePassword_Unauthenticated(t *testing.T) {
	db := setupAuthTestDB(t)
	handler := newPasswordTestHandler(db)

	// 不设置认证上下文
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password",
		strings.NewReader(`{"current_password":"pwd123","new_password":"newpass123"}`))
	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("未认证请求返回 %d，期望 401", rec.Code)
	}
}

// ========== 密码修改：审计不泄露密码明文 ==========

func TestChangePassword_AuditNoPasswordLeak(t *testing.T) {
	db := setupAuthAuditTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "auditpwd", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	auditService := service.NewAuditService(db)
	authHandler := newPasswordTestHandler(db)
	handler := middleware.AuditMiddleware(auditService)(http.HandlerFunc(authHandler.ChangePassword))

	req := changePasswordRequest(t, user.ID, `{"current_password":"pwd123","new_password":"secretpass9"}`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("修改密码返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	var logs []models.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("查询审计日志失败: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("应产生审计日志")
	}
	for _, l := range logs {
		if strings.Contains(l.Details, "pwd123") || strings.Contains(l.Details, "secretpass9") {
			t.Error("审计日志 details 泄露密码明文")
		}
		if strings.Contains(l.ErrorMsg, "pwd123") || strings.Contains(l.ErrorMsg, "secretpass9") {
			t.Error("审计日志 error_msg 泄露密码明文")
		}
	}
}
