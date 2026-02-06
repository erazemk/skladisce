package api

import (
	"database/sql"
	"net/http"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// InventoryHandler handles inventory endpoints.
type InventoryHandler struct {
	DB *sql.DB
}

type addStockRequest struct {
	ItemID   int64 `json:"item_id"`
	OwnerID  int64 `json:"owner_id"`
	Quantity int   `json:"quantity"`
}

type adjustRequest struct {
	ItemID  int64  `json:"item_id"`
	OwnerID int64  `json:"owner_id"`
	Delta   int    `json:"delta"`
	Notes   string `json:"notes"`
}

// List handles GET /api/inventory.
func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	inventory, err := store.ListInventory(r.Context(), h.DB)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list inventory")
		return
	}
	if inventory == nil {
		inventory = []model.Inventory{}
	}
	jsonResponse(w, http.StatusOK, inventory)
}

// AddStock handles POST /api/inventory/stock.
func (h *InventoryHandler) AddStock(w http.ResponseWriter, r *http.Request) {
	var req addStockRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ItemID <= 0 || req.OwnerID <= 0 || req.Quantity <= 0 {
		jsonError(w, http.StatusBadRequest, "item_id, owner_id, and quantity are required and must be positive")
		return
	}

	claims := GetClaims(r.Context())
	var userID *int64
	if claims != nil {
		userID = &claims.UserID
	}

	if err := store.AddStock(r.Context(), h.DB, req.ItemID, req.OwnerID, req.Quantity, userID); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "stock added"})
}

// Adjust handles POST /api/inventory/adjust.
func (h *InventoryHandler) Adjust(w http.ResponseWriter, r *http.Request) {
	var req adjustRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ItemID <= 0 || req.OwnerID <= 0 || req.Delta == 0 {
		jsonError(w, http.StatusBadRequest, "item_id, owner_id, and non-zero delta required")
		return
	}

	claims := GetClaims(r.Context())
	var userID *int64
	if claims != nil {
		userID = &claims.UserID
	}

	if err := store.AdjustInventory(r.Context(), h.DB, req.ItemID, req.OwnerID, req.Delta, req.Notes, userID); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "inventory adjusted"})
}
