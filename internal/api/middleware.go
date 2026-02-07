package api

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

type contextKey string

const claimsKey contextKey = "claims"
const tokenKey contextKey = "rawtoken"

// AuthMiddleware validates JWT from Authorization header, checks token
// revocation, and adds claims + raw token to context.
func AuthMiddleware(secret string, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				jsonError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(secret, tokenStr)
			if err != nil {
				jsonError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Check if the token has been revoked.
			if claims.ID != "" {
				revoked, err := store.IsTokenRevoked(r.Context(), db, claims.ID)
				if err != nil {
					slog.Error("failed to check token revocation", "error", err)
					jsonError(w, http.StatusInternalServerError, "internal error")
					return
				}
				if revoked {
					jsonError(w, http.StatusUnauthorized, "token has been revoked")
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, tokenKey, tokenStr)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks if the user has at least the given role.
func RequireRole(minimum string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				jsonError(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !model.RoleAtLeast(claims.Role, minimum) {
				jsonError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims retrieves the JWT claims from the context.
func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}

// GetRawToken retrieves the raw JWT token from the context.
func GetRawToken(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey).(string)
	return token
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs HTTP requests that result in client or server errors (4xx/5xx).
// Successful requests are not logged here — business-level actions are logged by handlers.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		if rec.status < 400 {
			return
		}

		duration := time.Since(start)
		attrs := []any{
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"status", rec.status,
			"duration", duration.Round(time.Millisecond).String(),
			"remote", r.RemoteAddr,
		}

		// Add user info if authenticated.
		if claims := GetClaims(r.Context()); claims != nil {
			attrs = append(attrs, "user", claims.Username)
		}

		if rec.status >= 500 {
			slog.Error("request", attrs...)
		} else {
			slog.Warn("request", attrs...)
		}
	})
}
