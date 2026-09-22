package middleware

import (
	"context"
	"net/http"
	"strings"
	"url-shortener/internal/service"
	"url-shortener/internal/utils"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDContextKey   contextKey = "user_id"
	UsernameContextKey contextKey = "username"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func GetUserID(ctx context.Context) *uuid.UUID {
	if val, ok := ctx.Value(UserIDContextKey).(uuid.UUID); ok {
		return &val
	}
	return nil
}

func GetUsername(ctx context.Context) string {
	if val, ok := ctx.Value(UsernameContextKey).(string); ok {
		return val
	}
	return ""
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "authorization token required")
			return
		}

		claims, err := m.authService.ValidateToken(tokenStr)
		if err != nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)

		ctx = context.WithValue(ctx, UsernameContextKey, claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth parses the token if provided, but allows anonymous requests to proceed.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r)
		if tokenStr != "" {
			if claims, err := m.authService.ValidateToken(tokenStr); err == nil {
				ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
				ctx = context.WithValue(ctx, UsernameContextKey, claims.Username)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
