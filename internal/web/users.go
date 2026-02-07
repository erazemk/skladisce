package web

import (
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// UsersPage handles GET /users (admin only).
func (s *Server) UsersPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	users, _ := store.ListUsers(r.Context(), s.DB)

	s.Templates.Render(w, "users.html", &struct {
		PageData
		Users []model.User
	}{
		PageData: PageData{Title: "Uporabniki", User: claims, Token: GetWebToken(r.Context())},
		Users:    users,
	})
}

// UserCreateSubmit handles POST /users (admin only).
func (s *Server) UserCreateSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	role := r.FormValue("role")

	if username == "" || password == "" || role == "" {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	store.CreateUser(r.Context(), s.DB, username, string(hash), role)
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// UserResetPasswordSubmit handles POST /users/{id}/password (admin only).
func (s *Server) UserResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	newPassword := r.FormValue("new_password")
	if newPassword == "" {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	store.UpdateUserPassword(r.Context(), s.DB, id, string(hash))
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// SettingsPage handles GET /settings.
func (s *Server) SettingsPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	s.Templates.Render(w, "settings.html", &PageData{
		Title: "Nastavitve",
		User:  claims,
		Token: GetWebToken(r.Context()),
	})
}

// SettingsSubmit handles POST /settings (change own password).
func (s *Server) SettingsSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")

	if currentPassword == "" || newPassword == "" {
		s.Templates.Render(w, "settings.html", &PageData{
			Title: "Nastavitve",
			User:  claims,
			Token: GetWebToken(r.Context()),
			Error: "Vnesite trenutno in novo geslo.",
		})
		return
	}

	user, err := store.GetUser(r.Context(), s.DB, claims.UserID)
	if err != nil || user == nil {
		s.Templates.Render(w, "settings.html", &PageData{
			Title: "Nastavitve",
			User:  claims,
			Token: GetWebToken(r.Context()),
			Error: "Napaka pri pridobivanju uporabnika.",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		s.Templates.Render(w, "settings.html", &PageData{
			Title: "Nastavitve",
			User:  claims,
			Token: GetWebToken(r.Context()),
			Error: "Trenutno geslo ni pravilno.",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.Templates.Render(w, "settings.html", &PageData{
			Title: "Nastavitve",
			User:  claims,
			Token: GetWebToken(r.Context()),
			Error: "Napaka pri shranjevanju gesla.",
		})
		return
	}

	if err := store.UpdateUserPassword(r.Context(), s.DB, claims.UserID, string(hash)); err != nil {
		s.Templates.Render(w, "settings.html", &PageData{
			Title: "Nastavitve",
			User:  claims,
			Token: GetWebToken(r.Context()),
			Error: "Napaka pri posodabljanju gesla.",
		})
		return
	}

	s.Templates.Render(w, "settings.html", &PageData{
		Title:   "Nastavitve",
		User:    claims,
		Token:   GetWebToken(r.Context()),
		Success: "Geslo uspešno spremenjeno.",
	})
}
