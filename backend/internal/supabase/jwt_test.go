package supabase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

const testUUID = "11111111-2222-3333-4444-555555555555"

// setupTestDB 创建内存 SQLite 数据库并迁移 User 表
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// 内存 SQLite 单连接，避免多连接各自独立库
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

// createUser 创建测试本地用户
func createUser(t *testing.T, db *gorm.DB, username, email, supabaseUID string) models.User {
	t.Helper()
	user := models.User{
		Username:    username,
		Email:       email,
		SupabaseUID: supabaseUID,
		Active:      true,
	}
	require.NoError(t, user.SetPassword("test123"))
	require.NoError(t, db.Create(&user).Error)
	return user
}

// fakeValidator 返回固定 claims 的验证器，用于绕开远程 Supabase JWKS
func fakeValidator(claims *SupabaseJWTClaims, err error) SupabaseTokenValidator {
	return func(tokenString string) (*SupabaseJWTClaims, error) {
		return claims, err
	}
}

// runMiddleware 执行中间件，返回响应与经过中间件处理后的请求 context
func runMiddleware(t *testing.T, db *gorm.DB, validate SupabaseTokenValidator, token string) (*httptest.ResponseRecorder, context.Context) {
	t.Helper()
	var gotCtx context.Context
	handler := SupabaseJWTMiddlewareWithValidator(db, validate)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, gotCtx
}

func TestSupabaseJWTMiddleware_ValidUUIDMapping(t *testing.T) {
	db := setupTestDB(t)
	user := createUser(t, db, "u1", "u1@test.com", testUUID)

	rec, ctx := runMiddleware(t, db, fakeValidator(&SupabaseJWTClaims{Sub: testUUID, Email: "u1@test.com"}, nil), "dummy-token")

	assert.Equal(t, http.StatusOK, rec.Code)
	// 本地 uint ID 写入 auth context，权限/GetProfile 中间件可直接使用
	localID, err := auth.GetUserIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, user.ID, localID)
	// Supabase 原始标识仍保留
	supabaseID, err := GetUserIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, testUUID, supabaseID)
}

func TestSupabaseJWTMiddleware_NoMappingRejected(t *testing.T) {
	db := setupTestDB(t) // 空库，无任何映射

	rec, _ := runMiddleware(t, db, fakeValidator(&SupabaseJWTClaims{Sub: testUUID, Email: "nobody@test.com"}, nil), "dummy-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSupabaseJWTMiddleware_MappingConflictRejected(t *testing.T) {
	db := setupTestDB(t)
	createUser(t, db, "u1", "u1@test.com", testUUID)
	// 删除唯一索引后插入第二个同 UID 用户，构造脏数据冲突
	require.NoError(t, db.Exec("DROP INDEX IF EXISTS idx_users_supabase_uid").Error)
	createUser(t, db, "u2", "u2@test.com", testUUID)

	rec, _ := runMiddleware(t, db, fakeValidator(&SupabaseJWTClaims{Sub: testUUID}, nil), "dummy-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSupabaseJWTMiddleware_EmptySubRejected(t *testing.T) {
	db := setupTestDB(t)
	// 存在 SupabaseUID 为空的本地用户，空 sub 也不得匹配（防误映射）
	createUser(t, db, "u1", "u1@test.com", "")

	rec, _ := runMiddleware(t, db, fakeValidator(&SupabaseJWTClaims{Sub: ""}, nil), "dummy-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSupabaseJWTMiddleware_NumericSubCompat(t *testing.T) {
	db := setupTestDB(t) // 空库：数字 sub 不查库，保持现有兼容

	rec, ctx := runMiddleware(t, db, fakeValidator(&SupabaseJWTClaims{Sub: "42"}, nil), "dummy-token")

	assert.Equal(t, http.StatusOK, rec.Code)
	localID, err := auth.GetUserIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint(42), localID)
}

func TestSupabaseJWTMiddleware_LocalJWTRegression(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret-for-local-jwt")
	db := setupTestDB(t)

	jwtManager := auth.NewJWTManager()
	token, err := jwtManager.GenerateAccessToken(7, "tester")
	require.NoError(t, err)

	// Supabase 验证失败时回退本地 JWT（本地登录兼容性回归）
	rec, ctx := runMiddleware(t, db, fakeValidator(nil, assert.AnError), token)

	assert.Equal(t, http.StatusOK, rec.Code)
	localID, err := auth.GetUserIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint(7), localID)
}

func TestResolveLocalUserBySupabaseUID(t *testing.T) {
	t.Run("找到映射", func(t *testing.T) {
		db := setupTestDB(t)
		user := createUser(t, db, "u1", "u1@test.com", testUUID)

		got, err := ResolveLocalUserBySupabaseUID(db, testUUID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
	})

	t.Run("未找到映射", func(t *testing.T) {
		db := setupTestDB(t)

		_, err := ResolveLocalUserBySupabaseUID(db, testUUID)
		require.Error(t, err)
	})

	t.Run("映射冲突", func(t *testing.T) {
		db := setupTestDB(t)
		createUser(t, db, "u1", "u1@test.com", testUUID)
		require.NoError(t, db.Exec("DROP INDEX IF EXISTS idx_users_supabase_uid").Error)
		createUser(t, db, "u2", "u2@test.com", testUUID)

		_, err := ResolveLocalUserBySupabaseUID(db, testUUID)
		require.Error(t, err)
	})

	t.Run("数据库不可用", func(t *testing.T) {
		_, err := ResolveLocalUserBySupabaseUID(nil, testUUID)
		require.Error(t, err)
	})
}
