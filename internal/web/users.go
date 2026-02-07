package web

import (
	"fmt"
	"log/slog"
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
	slog.Info("user created", "user", claims.Username, "new_user", username, "role", role)
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

	target, _ := store.GetUser(r.Context(), s.DB, id)
	targetName := fmt.Sprintf("id:%d", id)
	if target != nil {
		targetName = target.Username
	}
	slog.Info("user password reset", "user", claims.Username, "target_user", targetName)
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// UserUpdateRoleSubmit handles POST /users/{id}/role (admin only).
func (s *Server) UserUpdateRoleSubmit(w http.ResponseWriter, r *http.Request) {
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

	role := r.FormValue("role")
	if role != model.RoleAdmin && role != model.RoleManager && role != model.RoleUser {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	if claims.UserID == id {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	if err := store.UpdateUser(r.Context(), s.DB, id, role); err != nil {
		slog.Error("failed to update user role", "user", claims.Username, "target_id", id, "error", err)
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	target, _ := store.GetUser(r.Context(), s.DB, id)
	targetName := fmt.Sprintf("id:%d", id)
	if target != nil {
		targetName = target.Username
	}
	slog.Info("user role updated", "user", claims.Username, "target_user", targetName, "new_role", role)
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

	slog.Info("user changed own password", "user", claims.Username)
	s.Templates.Render(w, "settings.html", &PageData{
		Title:   "Nastavitve",
		User:    claims,
		Token:   GetWebToken(r.Context()),
		Success: "Geslo uspešno spremenjeno.",
	})
}
