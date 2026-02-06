package web

import (
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// TransfersPage handles GET /transfers.
func (s *Server) TransfersPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	transfers, _ := store.ListTransfers(r.Context(), s.DB, 0, 0)

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
	items, _ := store.ListItems(r.Context(), s.DB, "")
	owners, _ := store.ListOwners(r.Context(), s.DB, "")

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
	_, err := store.CreateTransfer(r.Context(), s.DB, itemID, fromOwnerID, toOwnerID, quantity, notes, &userID)

	if err != nil {
		items, _ := store.ListItems(r.Context(), s.DB, "")
		owners, _ := store.ListOwners(r.Context(), s.DB, "")

		s.Templates.Render(w, "transfer_new.html", &struct {
			PageData
			Items  []model.Item
			Owners []model.Owner
		}{
			PageData: PageData{Title: "Nov prenos", User: claims, Token: GetWebToken(r.Context()), Error: err.Error()},
			Items:    items,
			Owners:   owners,
		})
		return
	}

	http.Redirect(w, r, "/transfers", http.StatusSeeOther)
}
