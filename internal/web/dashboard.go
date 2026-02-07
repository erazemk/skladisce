package web

import (
	"log/slog"
	"net/http"

	"github.com/erazemk/skladisce/internal/store"
)

// Dashboard handles GET /.
func (s *Server) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())

	inventory, err := store.ListInventory(r.Context(), s.DB)
	if err != nil {
		slog.Error("failed to list inventory for dashboard", "error", err)
	}
	transfers, err := store.ListTransfers(r.Context(), s.DB, 0, 0)
	if err != nil {
		slog.Error("failed to list transfers for dashboard", "error", err)
	}

	// Limit recent transfers to 10.
	if len(transfers) > 10 {
		transfers = transfers[:10]
	}

	s.Templates.Render(w, "dashboard.html", &struct {
		PageData
		Inventory       any
		RecentTransfers any
	}{
		PageData:        PageData{Title: "Nadzorna plošča", User: claims, Token: GetWebToken(r.Context())},
		Inventory:       inventory,
		RecentTransfers: transfers,
	})
}
