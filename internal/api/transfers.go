package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// TransfersHandler handles transfer endpoints.
type TransfersHandler struct {
	DB *sql.DB
}

type createTransferRequest struct {
	ItemID      int64  `json:"item_id"`
	FromOwnerID int64  `json:"from_owner_id"`
	ToOwnerID   int64  `json:"to_owner_id"`
	Quantity    int    `json:"quantity"`
	Notes       string `json:"notes"`
}

// Create handles POST /api/transfers.
func (h *TransfersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ItemID <= 0 || req.FromOwnerID <= 0 || req.ToOwnerID <= 0 || req.Quantity <= 0 {
		jsonError(w, http.StatusBadRequest, "item_id, from_owner_id, to_owner_id, and quantity are required and must be positive")
		return
	}

	claims := GetClaims(r.Context())
	var userID *int64
	if claims != nil {
		userID = &claims.UserID
	}

	transfer, err := store.CreateTransfer(r.Context(), h.DB, req.ItemID, req.FromOwnerID, req.ToOwnerID, req.Quantity, req.Notes, userID)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("transfer created", "user", claims.Username,
		"item", transfer.ItemName, "quantity", transfer.Quantity,
		"from", transfer.FromOwnerName, "to", transfer.ToOwnerName)
	jsonResponse(w, http.StatusCreated, transfer)
}

// List handles GET /api/transfers.
func (h *TransfersHandler) List(w http.ResponseWriter, r *http.Request) {
	var itemID, ownerID int64

	if v := r.URL.Query().Get("item_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid item_id")
			return
		}
		itemID = id
	}

	if v := r.URL.Query().Get("owner_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid owner_id")
			return
		}
		ownerID = id
	}

	transfers, err := store.ListTransfers(r.Context(), h.DB, itemID, ownerID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list transfers")
		return
	}
	if transfers == nil {
		transfers = []model.Transfer{}
	}
	jsonResponse(w, http.StatusOK, transfers)
}
