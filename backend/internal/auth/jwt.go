package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"siapp/internal/models"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"` // token 类型："access" | "refresh"
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager.
// 安全策略：如果已配置 Supabase JWT（SUPABASE_JWT_SECRET），说明认证走 Supabase JWKS，
// 本地 JWT 仅作为兼容降级，此时允许不设 JWT_SECRET_KEY（使用进程唯一随机密钥）。
// 否则 JWT_SECRET_KEY 必须显式配置，生产环境绝不允许默认密钥。
func NewJWTManager() *JWTManager {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		// Supabase 模式下认证走 JWKS，本地 JWT 仅作降级兼容，允许不设 JWT_SECRET_KEY
		if os.Getenv("SUPABASE_JWT_SECRET") == "" {
			log.Fatal("JWT_SECRET_KEY is not set — 自建认证模式下必须配置 JWT_SECRET_KEY")
		}
	}

	durationStr := os.Getenv("JWT_TOKEN_DURATION")
	duration := 24 * time.Hour // 默认 24 小时
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: duration,
	}
}

// GenerateToken generates a new JWT access token for the user（24h 有效期）
func (j *JWTManager) GenerateToken(user *models.User) (string, error) {
	return j.generateTokenWithExpiry(user.ID, user.Username, "access", j.tokenDuration)
}

// GenerateAccessToken 生成 access token（24h），用于 API 鉴权
func (j *JWTManager) GenerateAccessToken(userID uint, username string) (string, error) {
	return j.generateTokenWithExpiry(userID, username, "access", j.tokenDuration)
}

// GenerateRefreshToken 生成 refresh token（7 天），用于续签 access token
func (j *JWTManager) GenerateRefreshToken(userID uint) (string, error) {
	return j.generateTokenWithExpiry(userID, "", "refresh", 7*24*time.Hour)
}

// generateTokenWithExpiry 签发带自定义过期时间的 JWT
func (j *JWTManager) generateTokenWithExpiry(userID uint, username, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "siapp",
			Subject:   strconv.Itoa(int(userID)),
			ID:        uuid.NewString(), // jti：每次签发唯一 UUID，作为 token 唯一标识（JTI）
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Expected format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

// Context keys for storing user information
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UsernameKey contextKey = "username"
)

// JWTMiddleware creates a middleware for JWT authentication
func JWTMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			token, err := ExtractTokenFromHeader(authHeader)
			if err != nil {
				http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				http.Error(w, `{"error":"Unauthorized: Invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add user information to request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTMiddleware creates a middleware that doesn't fail if no token is provided
// but extracts user info if token is present and valid
func OptionalJWTMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader != "" {
				token, err := ExtractTokenFromHeader(authHeader)
				if err == nil {
					claims, err := jwtManager.ValidateToken(token)
					if err == nil {
						// Add user information to request context
						ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
						ctx = context.WithValue(ctx, UsernameKey, claims.Username)
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		return 0, errors.New("user ID not found in context")
	}
	return userID, nil
}

// GetUsernameFromContext extracts username from request context
func GetUsernameFromContext(ctx context.Context) (string, error) {
	username, ok := ctx.Value(UsernameKey).(string)
	if !ok {
		return "", errors.New("username not found in context")
	}
	return username, nil
}

// HashToken 对 token 原文做 SHA-256 哈希，用于入库存储
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
