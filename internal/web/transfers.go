package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// TransfersPage handles GET /transfers.
func (s *Server) TransfersPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	transfers, err := store.ListTransfers(r.Context(), s.DB, 0, 0)
	if err != nil {
		slog.Error("failed to list transfers", "error", err)
	}

	s.Templates.Render(w, "transfers.html", &struct {
		PageData
		Transfers []model.Transfer
	}{
		PageData:  PageData{Title: "Prenosi", User: claims, Token: GetWebToken(r.Context())},
		Transfers: transfers,
	})
}

// TransferNewPage handles GET /transfers/new.
func (s *Server) TransferNewPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	items, err := store.ListItems(r.Context(), s.DB, "")
	if err != nil {
		slog.Error("failed to list items for transfer form", "error", err)
	}
	owners, err := store.ListOwners(r.Context(), s.DB, "")
	if err != nil {
		slog.Error("failed to list owners for transfer form", "error", err)
	}

	s.Templates.Render(w, "transfer_new.html", &struct {
		PageData
		Items  []model.Item
		Owners []model.Owner
	}{
		PageData: PageData{Title: "Nov prenos", User: claims, Token: GetWebToken(r.Context())},
		Items:    items,
		Owners:   owners,
	})
}

// TransferCreateSubmit handles POST /transfers/new.
func (s *Server) TransferCreateSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())

	itemID, _ := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	fromOwnerID, _ := strconv.ParseInt(r.FormValue("from_owner_id"), 10, 64)
	toOwnerID, _ := strconv.ParseInt(r.FormValue("to_owner_id"), 10, 64)
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	notes := r.FormValue("notes")

	userID := claims.UserID
	transfer, err := store.CreateTransfer(r.Context(), s.DB, itemID, fromOwnerID, toOwnerID, quantity, notes, &userID)

	if err != nil {
		slog.Warn("transfer creation failed", "error", err, "user", claims.Username)
		items, err2 := store.ListItems(r.Context(), s.DB, "")
		if err2 != nil {
			slog.Error("failed to list items for transfer error page", "error", err2)
		}
		owners, err2 := store.ListOwners(r.Context(), s.DB, "")
		if err2 != nil {
			slog.Error("failed to list owners for transfer error page", "error", err2)
		}

		s.Templates.Render(w, "transfer_new.html", &struct {
			PageData
			Items  []model.Item
			Owners []model.Owner
		}{
			PageData: PageData{Title: "Nov prenos", User: claims, Token: GetWebToken(r.Context()), Error: "Prenos ni uspel. Preverite količino in lastnika."},
			Items:    items,
			Owners:   owners,
		})
		return
	}

	slog.Info("transfer created", "user", claims.Username,
		"item", transfer.ItemName, "quantity", transfer.Quantity,
		"from", transfer.FromOwnerName, "to", transfer.ToOwnerName)
	http.Redirect(w, r, "/transfers", http.StatusSeeOther)
}
