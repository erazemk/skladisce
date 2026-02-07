package web

import (
	"database/sql"
	"net/http"

	webembed "github.com/erazemk/skladisce/web"
)

// NewRouter creates the web page router with all page routes registered.
func NewRouter(db *sql.DB, jwtSecret string) (http.Handler, error) {
	templates, err := LoadTemplates()
	if err != nil {
		return nil, err
	}

	s := &Server{
		DB:        db,
		Templates: templates,
		JWTSecret: jwtSecret,
	}

	mux := http.NewServeMux()
	cookieAuth := CookieAuthMiddleware(jwtSecret)

	// Static assets.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(webembed.StaticFS()))))

	// Public routes.
	mux.HandleFunc("GET /login", s.LoginPage)
	mux.HandleFunc("POST /login", s.LoginSubmit)
	mux.HandleFunc("POST /logout", s.Logout)

	// Authenticated routes.
	mux.Handle("GET /{$}", cookieAuth(http.HandlerFunc(s.Dashboard)))

	mux.Handle("GET /items", cookieAuth(http.HandlerFunc(s.ItemsPage)))
	mux.Handle("POST /items", cookieAuth(http.HandlerFunc(s.ItemCreateSubmit)))
	mux.Handle("GET /items/{id}", cookieAuth(http.HandlerFunc(s.ItemDetailPage)))
	mux.Handle("POST /items/{id}", cookieAuth(http.HandlerFunc(s.ItemUpdateSubmit)))
	mux.Handle("POST /items/{id}/stock", cookieAuth(http.HandlerFunc(s.ItemStockSubmit)))
	mux.Handle("POST /items/{id}/image", cookieAuth(http.HandlerFunc(s.ItemImageSubmit)))

	mux.Handle("GET /owners", cookieAuth(http.HandlerFunc(s.OwnersPage)))
	mux.Handle("POST /owners", cookieAuth(http.HandlerFunc(s.OwnerCreateSubmit)))
	mux.Handle("GET /owners/{id}", cookieAuth(http.HandlerFunc(s.OwnerDetailPage)))
	mux.Handle("POST /owners/{id}", cookieAuth(http.HandlerFunc(s.OwnerUpdateSubmit)))

	mux.Handle("GET /transfers", cookieAuth(http.HandlerFunc(s.TransfersPage)))
	mux.Handle("GET /transfers/new", cookieAuth(http.HandlerFunc(s.TransferNewPage)))
	mux.Handle("POST /transfers/new", cookieAuth(http.HandlerFunc(s.TransferCreateSubmit)))

	mux.Handle("GET /users", cookieAuth(http.HandlerFunc(s.UsersPage)))
	mux.Handle("POST /users", cookieAuth(http.HandlerFunc(s.UserCreateSubmit)))
	mux.Handle("POST /users/{id}/password", cookieAuth(http.HandlerFunc(s.UserResetPasswordSubmit)))
	mux.Handle("POST /users/{id}/role", cookieAuth(http.HandlerFunc(s.UserUpdateRoleSubmit)))

	mux.Handle("GET /settings", cookieAuth(http.HandlerFunc(s.SettingsPage)))
	mux.Handle("POST /settings", cookieAuth(http.HandlerFunc(s.SettingsSubmit)))

	return mux, nil
}
