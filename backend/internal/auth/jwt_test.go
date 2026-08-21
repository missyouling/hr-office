package auth

import (
	"sync"
	"testing"
)

// newTestJWTManager 构造测试用 JWTManager，避免依赖外部环境变量
func newTestJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	t.Setenv("JWT_SECRET_KEY", "test-secret-key")
	return NewJWTManager()
}

// TestGenerateAccessTokenUnique 同一用户连续生成 access token 必须互不相同（jti 唯一）
func TestGenerateAccessTokenUnique(t *testing.T) {
	jm := newTestJWTManager(t)

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := jm.GenerateAccessToken(1, "alice")
		if err != nil {
			t.Fatalf("GenerateAccessToken 失败: %v", err)
		}
		if seen[token] {
			t.Fatalf("连续生成的 access token 重复: %s", token)
		}
		seen[token] = true
	}
}

// TestGenerateRefreshTokenUnique 同一用户连续生成 refresh token 必须互不相同（jti 唯一）
func TestGenerateRefreshTokenUnique(t *testing.T) {
	jm := newTestJWTManager(t)

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := jm.GenerateRefreshToken(1)
		if err != nil {
			t.Fatalf("GenerateRefreshToken 失败: %v", err)
		}
		if seen[token] {
			t.Fatalf("连续生成的 refresh token 重复: %s", token)
		}
		seen[token] = true
	}
}

// TestGenerateTokenConcurrentUnique 同一用户并发签发 access/refresh token，全部必须互不相同
func TestGenerateTokenConcurrentUnique(t *testing.T) {
	jm := newTestJWTManager(t)

	const count = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[string]bool)

	generate := func(fn func() (string, error)) {
		defer wg.Done()
		token, err := fn()
		if err != nil {
			t.Errorf("签发 token 失败: %v", err)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if seen[token] {
			t.Errorf("并发签发的 token 重复: %s", token)
			return
		}
		seen[token] = true
	}

	wg.Add(count * 2)
	for i := 0; i < count; i++ {
		go generate(func() (string, error) { return jm.GenerateAccessToken(1, "alice") })
		go generate(func() (string, error) { return jm.GenerateRefreshToken(1) })
	}
	wg.Wait()
}

// TestTokenTypeClaim 签发后解析 claims：token_type 必须正确区分 access/refresh，jti 必须为唯一 UUID
func TestTokenTypeClaim(t *testing.T) {
	jm := newTestJWTManager(t)

	accessToken, err := jm.GenerateAccessToken(1, "alice")
	if err != nil {
		t.Fatalf("GenerateAccessToken 失败: %v", err)
	}
	refreshToken, err := jm.GenerateRefreshToken(1)
	if err != nil {
		t.Fatalf("GenerateRefreshToken 失败: %v", err)
	}

	accessClaims, err := jm.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("验证 access token 失败: %v", err)
	}
	if accessClaims.TokenType != "access" {
		t.Errorf("access token 的 token_type 应为 access，实际为 %q", accessClaims.TokenType)
	}
	if accessClaims.ID == "" || accessClaims.ID == "access" {
		t.Errorf("access token 的 jti 应为唯一 UUID，实际为 %q", accessClaims.ID)
	}

	refreshClaims, err := jm.ValidateToken(refreshToken)
	if err != nil {
		t.Fatalf("验证 refresh token 失败: %v", err)
	}
	if refreshClaims.TokenType != "refresh" {
		t.Errorf("refresh token 的 token_type 应为 refresh，实际为 %q", refreshClaims.TokenType)
	}
	if refreshClaims.ID == "" || refreshClaims.ID == "refresh" {
		t.Errorf("refresh token 的 jti 应为唯一 UUID，实际为 %q", refreshClaims.ID)
	}

	// 模拟 api.Refresh 的类型校验语义：仅 refresh 类型可通过
	if refreshClaims.TokenType != "refresh" {
		t.Error("Refresh 类型判断失败：refresh token 应通过校验")
	}
	if accessClaims.TokenType == "refresh" {
		t.Error("Refresh 类型判断失败：access token 不应被当作 refresh")
	}
}
