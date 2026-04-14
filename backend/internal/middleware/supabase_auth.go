package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

// SupabaseJWTClaims Supabase JWT Claims结构
type SupabaseJWTClaims struct {
	Sub   string `json:"sub"`   // 用户ID (uuid)
	Email string `json:"email"` // 用户邮箱
	jwt.RegisteredClaims
}

// SupabaseAuthMiddleware 验证Supabase JWT的中间件
func SupabaseAuthMiddleware() func(http.Handler) http.Handler {
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		panic("SUPABASE_JWT_SECRET environment variable is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Unauthorized: No token provided"}`, http.StatusUnauthorized)
				return
			}

			// 提取Bearer token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, `{"error":"Unauthorized: Invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			// 解析并验证JWT
			token, err := jwt.ParseWithClaims(tokenString, &SupabaseJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				// 验证签名方法
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				http.Error(w, `{"error":"Unauthorized: Invalid token"}`, http.StatusUnauthorized)
				return
			}

			// 提取claims
			if claims, ok := token.Claims.(*SupabaseJWTClaims); ok && token.Valid {
				// 将用户信息添加到context
				ctx := context.WithValue(r.Context(), UserIDKey, claims.Sub)
				ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

				// 继续处理请求
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, `{"error":"Unauthorized: Invalid token claims"}`, http.StatusUnauthorized)
				return
			}
		})
	}
}

// GetUserIDFromContext 从context中提取用户ID（uuid字符串）
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// GetUserEmailFromContext 从context中提取用户邮箱
func GetUserEmailFromContext(ctx context.Context) (string, error) {
	email, ok := ctx.Value(UserEmailKey).(string)
	if !ok || email == "" {
		return "", fmt.Errorf("user email not found in context")
	}
	return email, nil
}
