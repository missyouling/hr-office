package supabase

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SupabaseJWTClaims represents Supabase JWT claims
type SupabaseJWTClaims struct {
	Sub   string `json:"sub"`   // User ID (UUID)
	Email string `json:"email"` // User email
	Role  string `json:"role"`  // User role
	jwt.RegisteredClaims
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// Context keys for storing user information
type contextKey string

const (
	UserIDKey    contextKey = "supabase_user_id"
	UserEmailKey contextKey = "supabase_user_email"
	UserRoleKey  contextKey = "supabase_user_role"
)

var (
	jwksCache     *JWKS
	jwksCacheTime time.Time
	cacheDuration = 1 * time.Hour
)

// FetchJWKS fetches the JWKS from Supabase
func FetchJWKS() (*JWKS, error) {
	// Check cache
	if jwksCache != nil && time.Since(jwksCacheTime) < cacheDuration {
		return jwksCache, nil
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, errors.New("SUPABASE_URL environment variable is required")
	}

	jwksURL := fmt.Sprintf("%s/auth/v1/jwks", supabaseURL)

	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch JWKS: unexpected status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	// Update cache
	jwksCache = &jwks
	jwksCacheTime = time.Now()

	return &jwks, nil
}

// GetPublicKey retrieves the RSA public key from JWK
func (jwk *JWK) GetPublicKey() (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// ValidateSupabaseToken validates a Supabase JWT token
func ValidateSupabaseToken(tokenString string) (*SupabaseJWTClaims, error) {
	// Parse token without validation first to get kid
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &SupabaseJWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	// Get kid from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("kid not found in token header")
	}

	// Fetch JWKS
	jwks, err := FetchJWKS()
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}

	// Find matching key
	var matchingKey *JWK
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			matchingKey = &key
			break
		}
	}

	if matchingKey == nil {
		return nil, errors.New("no matching key found in JWKS")
	}

	// Get public key
	publicKey, err := matchingKey.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("get public key: %w", err)
	}

	// Parse and validate token with public key
	validatedToken, err := jwt.ParseWithClaims(tokenString, &SupabaseJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	if claims, ok := validatedToken.Claims.(*SupabaseJWTClaims); ok && validatedToken.Valid {
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

// SupabaseJWTMiddleware creates a middleware for Supabase JWT authentication
func SupabaseJWTMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			token, err := ExtractTokenFromHeader(authHeader)
			if err != nil {
				http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			claims, err := ValidateSupabaseToken(token)
			if err != nil {
				http.Error(w, `{"error":"Unauthorized: Invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add user information to request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.Sub)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext extracts Supabase user ID from request context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// GetUserEmailFromContext extracts user email from request context
func GetUserEmailFromContext(ctx context.Context) (string, error) {
	email, ok := ctx.Value(UserEmailKey).(string)
	if !ok {
		return "", errors.New("user email not found in context")
	}
	return email, nil
}

// GetUserRoleFromContext extracts user role from request context
func GetUserRoleFromContext(ctx context.Context) (string, error) {
	role, ok := ctx.Value(UserRoleKey).(string)
	if !ok {
		return "", errors.New("user role not found in context")
	}
	return role, nil
}
