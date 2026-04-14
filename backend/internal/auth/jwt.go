package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"siapp/internal/models"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager() *JWTManager {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		secretKey = "your-secret-key-change-this-in-production"
	}

	durationStr := os.Getenv("JWT_TOKEN_DURATION")
	duration := 24 * time.Hour // Default to 24 hours
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

// GenerateToken generates a new JWT token for the user
func (j *JWTManager) GenerateToken(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "siapp",
			Subject:   strconv.Itoa(int(user.ID)),
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