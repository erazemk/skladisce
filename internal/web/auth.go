package web

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/store"
)

// LoginPage handles GET /login.
func (s *Server) LoginPage(w http.ResponseWriter, r *http.Request) {
	s.Templates.Render(w, "login.html", &PageData{Title: "Prijava"})
}

// LoginSubmit handles POST /login.
func (s *Server) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		s.Templates.Render(w, "login.html", &PageData{
			Title: "Prijava",
			Error: "Vnesite uporabniško ime in geslo.",
		})
		return
	}

	user, err := store.GetUserByUsername(r.Context(), s.DB, username)
	if err != nil || user == nil || user.DeletedAt != nil {
		s.Templates.Render(w, "login.html", &PageData{
			Title: "Prijava",
			Error: "Napačno uporabniško ime ali geslo.",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.Templates.Render(w, "login.html", &PageData{
			Title: "Prijava",
			Error: "Napačno uporabniško ime ali geslo.",
		})
		return
	}

	token, err := auth.GenerateToken(s.JWTSecret, user.ID, user.Username, user.Role)
	if err != nil {
		s.Templates.Render(w, "login.html", &PageData{
			Title: "Prijava",
			Error: "Napaka pri prijavi.",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles POST /logout.
func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
