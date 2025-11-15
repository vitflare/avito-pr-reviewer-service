package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"pr-reviewer-service/internal/dto"
)

type AuthService interface {
	ValidateToken(ctx context.Context, token string) (string, error)
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

type contextKey string

const (
	UserIDKey  contextKey = "user_id"
	IsAdminKey contextKey = "is_admin"
)

// AuthMiddleware check JWT token in Authorization
func AuthMiddleware(authService AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:    dto.ErrCodeNotFound,
						Message: "missing authorization header",
					},
				})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondError(w, http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:    dto.ErrCodeNotFound,
						Message: "invalid authorization header format",
					},
				})
				return
			}

			token := parts[1]
			userID, err := authService.ValidateToken(r.Context(), token)
			if err != nil {
				respondError(w, http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:    dto.ErrCodeNotFound,
						Message: "invalid or expired token",
					},
				})
				return
			}

			// check user is admin
			isAdmin, err := authService.IsAdmin(r.Context(), userID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:    dto.ErrCodeNotFound,
						Message: "failed to check admin status",
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, IsAdminKey, isAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminMiddleware check, that user is admin
func AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAdmin, ok := r.Context().Value(IsAdminKey).(bool)
			if !ok || !isAdmin {
				respondError(w, http.StatusForbidden, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:    dto.ErrCodeNotFound,
						Message: "forbidden: admin access required",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func respondError(w http.ResponseWriter, status int, errResp dto.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Warn("failed to encode JSON response", "error", err)
	}
}
