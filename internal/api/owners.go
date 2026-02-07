package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// OwnersHandler handles owner CRUD endpoints.
type OwnersHandler struct {
	DB *sql.DB
}

type createOwnerRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type updateOwnerRequest struct {
	Name string `json:"name"`
}

// List handles GET /api/owners.
func (h *OwnersHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerType := r.URL.Query().Get("type")
	owners, err := store.ListOwners(r.Context(), h.DB, ownerType)
	if err != nil {
		slog.Error("failed to list owners", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to list owners")
		return
	}
	if owners == nil {
		owners = []model.Owner{}
	}
	jsonResponse(w, http.StatusOK, owners)
}

// Create handles POST /api/owners.
func (h *OwnersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOwnerRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Type == "" {
		jsonError(w, http.StatusBadRequest, "name and type required")
		return
	}

	if req.Type != model.OwnerTypePerson && req.Type != model.OwnerTypeLocation {
		jsonError(w, http.StatusBadRequest, "type must be 'person' or 'location'")
		return
	}

	owner, err := store.CreateOwner(r.Context(), h.DB, req.Name, req.Type)
	if err != nil {
		slog.Error("failed to create owner", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create owner")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("owner created", "user", claims.Username, "owner", req.Name, "type", req.Type)
	jsonResponse(w, http.StatusCreated, owner)
}

// Get handles GET /api/owners/{id}.
func (h *OwnersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid owner id")
		return
	}

	owner, err := store.GetOwner(r.Context(), h.DB, id)
	if err != nil {
		slog.Error("failed to get owner", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to get owner")
		return
	}
	if owner == nil || owner.DeletedAt != nil {
		jsonError(w, http.StatusNotFound, "owner not found")
		return
	}

	jsonResponse(w, http.StatusOK, owner)
}

// Update handles PUT /api/owners/{id}.
func (h *OwnersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid owner id")
		return
	}

	var req updateOwnerRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name required")
		return
	}

	if err := store.UpdateOwner(r.Context(), h.DB, id, req.Name); err != nil {
		slog.Error("failed to update owner", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to update owner")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("owner updated", "user", claims.Username, "owner", req.Name)
	owner, _ := store.GetOwner(r.Context(), h.DB, id)
	jsonResponse(w, http.StatusOK, owner)
}

// Delete handles DELETE /api/owners/{id}.
func (h *OwnersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid owner id")
		return
	}

	owner, _ := store.GetOwner(r.Context(), h.DB, id)
	ownerName := fmt.Sprintf("id:%d", id)
	if owner != nil {
		ownerName = owner.Name
	}

	if err := store.DeleteOwner(r.Context(), h.DB, id); err != nil {
		// Check if it's a business rule error (holding inventory) vs internal error.
		slog.Warn("failed to delete owner", "owner", ownerName, "error", err)
		jsonError(w, http.StatusBadRequest, "cannot delete owner: still holds inventory or not found")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("owner deleted", "user", claims.Username, "owner", ownerName)
	jsonResponse(w, http.StatusOK, map[string]string{"message": "owner deleted"})
}

// GetInventory handles GET /api/owners/{id}/inventory.
func (h *OwnersHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid owner id")
		return
	}

	inventory, err := store.GetOwnerInventory(r.Context(), h.DB, id)
	if err != nil {
		slog.Error("failed to get owner inventory", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to get owner inventory")
		return
	}
	if inventory == nil {
		inventory = []model.Inventory{}
	}
	jsonResponse(w, http.StatusOK, inventory)
}
