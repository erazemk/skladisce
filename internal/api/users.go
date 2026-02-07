package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// UsersHandler handles user management endpoints (admin only).
type UsersHandler struct {
	DB *sql.DB
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Role string `json:"role"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

// List handles GET /api/users.
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := store.ListUsers(r.Context(), h.DB)
	if err != nil {
		slog.Error("failed to list users", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if users == nil {
		users = []model.User{}
	}
	jsonResponse(w, http.StatusOK, users)
}

// Create handles POST /api/users.
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" || req.Role == "" {
		jsonError(w, http.StatusBadRequest, "username, password, and role required")
		return
	}

	if req.Role != model.RoleAdmin && req.Role != model.RoleManager && req.Role != model.RoleUser {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := model.ValidatePassword(req.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := store.CreateUser(r.Context(), h.DB, req.Username, string(hash), req.Role)
	if err != nil {
		jsonError(w, http.StatusConflict, "username already exists")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("user created", "user", claims.Username, "new_user", req.Username, "role", req.Role)
	jsonResponse(w, http.StatusCreated, user)
}

// Get handles GET /api/users/{id}.
func (h *UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := store.GetUser(r.Context(), h.DB, id)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if user == nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}

// Update handles PUT /api/users/{id}.
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != model.RoleAdmin && req.Role != model.RoleManager && req.Role != model.RoleUser {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := store.UpdateUser(r.Context(), h.DB, id, req.Role); err != nil {
		slog.Error("failed to update user", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	user, _ := store.GetUser(r.Context(), h.DB, id)
	claims := GetClaims(r.Context())
	if user != nil {
		slog.Info("user role updated", "user", claims.Username, "target_user", user.Username, "new_role", req.Role)
	}
	jsonResponse(w, http.StatusOK, user)
}

// ResetPassword handles PUT /api/users/{id}/password.
func (h *UsersHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		jsonError(w, http.StatusBadRequest, "password required")
		return
	}

	if err := model.ValidatePassword(req.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := store.UpdateUserPassword(r.Context(), h.DB, id, string(hash)); err != nil {
		slog.Error("failed to reset password", "error", err)
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	claims := GetClaims(r.Context())
	target, _ := store.GetUser(r.Context(), h.DB, id)
	targetName := fmt.Sprintf("id:%d", id)
	if target != nil {
		targetName = target.Username
	}
	slog.Info("user password reset", "user", claims.Username, "target_user", targetName)
	jsonResponse(w, http.StatusOK, map[string]string{"message": "password reset"})
}

// Delete handles DELETE /api/users/{id}.
func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Prevent self-deletion.
	claims := GetClaims(r.Context())
	if claims != nil && claims.UserID == id {
		jsonError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	// Look up target name before deleting.
	target, _ := store.GetUser(r.Context(), h.DB, id)
	targetName := fmt.Sprintf("id:%d", id)
	if target != nil {
		targetName = target.Username
	}

	if err := store.DeleteUser(r.Context(), h.DB, id); err != nil {
		slog.Error("failed to delete user", "error", err)
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	slog.Info("user deleted", "user", claims.Username, "deleted_user", targetName)
	jsonResponse(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
