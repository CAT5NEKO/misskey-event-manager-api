package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"miSchedule/internal/auth"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	AdminKey  contextKey = "is_admin"
)

func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"認証情報がありません"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"認証形式が不正です"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.ValidateAccessToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"トークンが無効か期限切れです"}`, http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				http.Error(w, `{"error":"トークン情報が不正です"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, AdminKey, claims.IsAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r) {
			http.Error(w, `{"error":"管理者権限が必要です"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(r *http.Request) (uuid.UUID, error) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, http.ErrNoCookie
	}
	return userID, nil
}

func IsAdmin(r *http.Request) bool {
	isAdmin, ok := r.Context().Value(AdminKey).(bool)
	return ok && isAdmin
}

func GetIPAddress(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
