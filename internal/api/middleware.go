package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/model"
)

type contextKey string

const claimsKey contextKey = "claims"

// AuthMiddleware validates JWT from Authorization header and adds claims to context.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), claimsKey, claims)
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

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs HTTP requests with method, path, status, and duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond))
	})
}
