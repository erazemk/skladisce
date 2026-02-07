package web

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/store"
)

type webContextKey string

const webClaimsKey webContextKey = "webclaims"
const webTokenKey webContextKey = "webtoken"

// CookieAuthMiddleware validates JWT from cookie, checks token revocation,
// and adds claims to context.
func CookieAuthMiddleware(secret string, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			claims, err := auth.ValidateToken(secret, cookie.Value)
			if err != nil {
				clearAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Check if the token has been revoked.
			if claims.ID != "" {
				revoked, err := store.IsTokenRevoked(r.Context(), db, claims.ID)
				if err != nil {
					slog.Error("failed to check token revocation", "error", err)
					clearAuthCookie(w)
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				if revoked {
					clearAuthCookie(w)
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
			}

			ctx := context.WithValue(r.Context(), webClaimsKey, claims)
			ctx = context.WithValue(ctx, webTokenKey, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// clearAuthCookie clears the authentication cookie with consistent attributes.
func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetWebClaims retrieves the JWT claims from web context.
func GetWebClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(webClaimsKey).(*auth.Claims)
	return claims
}

// GetWebToken retrieves the raw JWT token from web context.
func GetWebToken(ctx context.Context) string {
	token, _ := ctx.Value(webTokenKey).(string)
	return token
}
