package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// setupAuthAuditTestDB 在共享测试库基础上追加审计日志表（审计不泄露测试用）
func setupAuthAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupAuthTestDB(t)
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("自动建表审计日志失败: %v", err)
	}
	return db
}

// patchProfileRequest 构造带认证上下文的 PATCH /auth/profile 请求
func patchProfileRequest(t *testing.T, userID uint, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/auth/profile", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, userID))
	return req
}

// ========== 资料更新：正常流程 ==========

func TestUpdateProfile_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profileup", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	oldPasswordHash := user.Password

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	req := patchProfileRequest(t, user.ID, `{"full_name":"王小明"}`)
	rec := httptest.NewRecorder()
	handler.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("更新资料返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	// 响应结构：现有安全用户响应（AuthUserPayload）
	var resp AuthUserPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.User.FullName != "王小明" {
		t.Errorf("full_name 期望 王小明，实际 %s", resp.User.FullName)
	}
	if resp.User.Username != user.Username {
		t.Errorf("username 不应被修改，实际 %s", resp.User.Username)
	}
	if resp.User.Email != user.Email {
		t.Errorf("email 不应被修改，实际 %s", resp.User.Email)
	}

	// 数据库校验：仅 full_name 变更，密码哈希等敏感字段未被覆盖
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if stored.FullName != "王小明" {
		t.Errorf("数据库 full_name 期望 王小明，实际 %s", stored.FullName)
	}
	if stored.Password != oldPasswordHash {
		t.Error("密码哈希不应被资料更新覆盖")
	}
	if stored.Username != user.Username || stored.Email != user.Email {
		t.Error("username/email 不应被资料更新修改")
	}
}

func TestUpdateProfile_TrimsWhitespace(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profiletrim", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	req := patchProfileRequest(t, user.ID, `{"full_name":"  王 五  "}`)
	rec := httptest.NewRecorder()
	handler.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("更新资料返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if stored.FullName != "王 五" {
		t.Errorf("full_name 应去除首尾空白，期望 王 五，实际 %q", stored.FullName)
	}
}

// ========== 资料更新：边界 ==========

func TestUpdateProfile_BlankNameRejected(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profileblank", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	for _, body := range []string{`{"full_name":""}`, `{"full_name":"   "}`, `{}`, `null`} {
		req := patchProfileRequest(t, user.ID, body)
		rec := httptest.NewRecorder()
		handler.UpdateProfile(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("请求体 %s 返回 %d，期望 400", body, rec.Code)
		}
	}
}

func TestUpdateProfile_TooLongNameRejected(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profilelong", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	body := fmt.Sprintf(`{"full_name":"%s"}`, strings.Repeat("a", 101))
	req := patchProfileRequest(t, user.ID, body)
	rec := httptest.NewRecorder()
	handler.UpdateProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("101 字符姓名返回 %d，期望 400", rec.Code)
	}
}

func TestUpdateProfile_Exactly100Accepted(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profile100", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	body := fmt.Sprintf(`{"full_name":"%s"}`, strings.Repeat("a", 100))
	req := patchProfileRequest(t, user.ID, body)
	rec := httptest.NewRecorder()
	handler.UpdateProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("恰好 100 字符姓名返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}
}

// ========== 资料更新：字段白名单（越权字段注入） ==========

func TestUpdateProfile_UnknownFieldRejected(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profileinj", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	bodies := []string{
		`{"full_name":"合法姓名","username":"hacker"}`,
		`{"full_name":"合法姓名","email":"hacker@evil.com"}`,
		`{"full_name":"合法姓名","id":999}`,
		`{"full_name":"合法姓名","supabase_uid":"00000000-0000-0000-0000-000000000099"}`,
		`{"username":"hacker"}`,
	}
	for _, body := range bodies {
		req := patchProfileRequest(t, user.ID, body)
		rec := httptest.NewRecorder()
		handler.UpdateProfile(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("请求体 %s 返回 %d，期望 400（未知字段必须拒绝）", body, rec.Code)
		}
	}

	// 数据库确认：注入未生效
	var stored models.User
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if stored.Username != user.Username || stored.Email != user.Email {
		t.Error("越权字段注入不应修改 username/email")
	}
	if stored.SupabaseUID != "" {
		t.Error("越权字段注入不应修改 supabase_uid")
	}
}

// ========== 资料更新：未认证 / 非法 JSON ==========

func TestUpdateProfile_Unauthenticated(t *testing.T) {
	db := setupAuthTestDB(t)
	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)

	// 不设置认证上下文
	req := httptest.NewRequest(http.MethodPatch, "/auth/profile", strings.NewReader(`{"full_name":"x"}`))
	rec := httptest.NewRecorder()
	handler.UpdateProfile(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("未认证请求返回 %d，期望 401", rec.Code)
	}
}

func TestUpdateProfile_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "profilejson", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	for _, body := range []string{`{"full_name":`, `{"full_name":123}`, `[1,2,3]`} {
		req := patchProfileRequest(t, user.ID, body)
		rec := httptest.NewRecorder()
		handler.UpdateProfile(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("请求体 %s 返回 %d，期望 400", body, rec.Code)
		}
	}
}

// ========== 资料更新：审计不泄露敏感字段 ==========

func TestUpdateProfile_AuditNoSensitiveData(t *testing.T) {
	db := setupAuthAuditTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	user, err := createAuthTestUser(db, "auditup", "pwd123", vID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	auditService := service.NewAuditService(db)
	authHandler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	handler := middleware.AuditMiddleware(auditService)(http.HandlerFunc(authHandler.UpdateProfile))

	req := patchProfileRequest(t, user.ID, `{"full_name":"审计敏感姓名"}`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("更新资料返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	var logs []models.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("查询审计日志失败: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("应产生审计日志")
	}
	for _, l := range logs {
		if strings.Contains(l.Details, "审计敏感姓名") {
			t.Error("审计日志泄露 full_name 值")
		}
	}
}
