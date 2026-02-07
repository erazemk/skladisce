package api

import (
	"database/sql"
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/store"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, http.StatusBadRequest, "username and password required")
		return
	}

	user, err := store.GetUserByUsername(r.Context(), h.DB, req.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil || user.DeletedAt != nil {
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("login failed", "username", req.Username, "remote", r.RemoteAddr)
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(h.JWTSecret, user.ID, user.Username, user.Role)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	slog.Info("user logged in", "user", user.Username, "role", user.Role)
	jsonResponse(w, http.StatusOK, loginResponse{Token: token})
}

// ChangePassword handles PUT /api/auth/password.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		jsonError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		jsonError(w, http.StatusBadRequest, "current and new password required")
		return
	}

	user, err := store.GetUser(r.Context(), h.DB, claims.UserID)
	if err != nil || user == nil {
		jsonError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		jsonError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := store.UpdateUserPassword(r.Context(), h.DB, claims.UserID, string(hash)); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	slog.Info("user changed own password", "user", claims.Username)
	jsonResponse(w, http.StatusOK, map[string]string{"message": "password updated"})
}
