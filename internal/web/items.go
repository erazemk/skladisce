package web

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// ItemsPage handles GET /items.
func (s *Server) ItemsPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	items, _ := store.ListItems(r.Context(), s.DB, "")

	s.Templates.Render(w, "items.html", &struct {
		PageData
		Items []model.Item
	}{
		PageData: PageData{Title: "Predmeti", User: claims, Token: GetWebToken(r.Context())},
		Items:    items,
	})
}

// ItemDetailPage handles GET /items/{id}.
func (s *Server) ItemDetailPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	item, err := store.GetItem(r.Context(), s.DB, id)
	if err != nil || item == nil {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	dist, _ := store.GetItemDistribution(r.Context(), s.DB, id)
	history, _ := store.GetItemHistory(r.Context(), s.DB, id)
	owners, _ := store.ListOwners(r.Context(), s.DB, "")

	s.Templates.Render(w, "item_detail.html", &struct {
		PageData
		Item         *model.Item
		Distribution []model.Inventory
		History      []model.Transfer
		Owners       []model.Owner
		CreatedAt    any
	}{
		PageData:     PageData{Title: item.Name, User: claims, Token: GetWebToken(r.Context())},
		Item:         item,
		Distribution: dist,
		History:      history,
		Owners:       owners,
		CreatedAt:    item.CreatedAt,
	})
}

// ItemCreateSubmit handles POST /items.
func (s *Server) ItemCreateSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleManager) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Redirect(w, r, "/items", http.StatusSeeOther)
		return
	}

	store.CreateItem(r.Context(), s.DB, name, description)
	http.Redirect(w, r, "/items", http.StatusSeeOther)
}

// ItemUpdateSubmit handles POST /items/{id}.
func (s *Server) ItemUpdateSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleManager) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	status := r.FormValue("status")

	if err := store.UpdateItem(r.Context(), s.DB, id, name, description, status); err != nil {
		http.Error(w, "failed to update", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}

// ItemStockSubmit handles POST /items/{id}/stock.
func (s *Server) ItemStockSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleManager) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ownerID, _ := strconv.ParseInt(r.FormValue("owner_id"), 10, 64)
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))

	userID := claims.UserID
	if err := store.AddStock(r.Context(), s.DB, id, ownerID, quantity, &userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}

// ItemImageSubmit handles POST /items/{id}/image.
func (s *Server) ItemImageSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleManager) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mime := header.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		http.Error(w, "invalid image type", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read image", http.StatusInternalServerError)
		return
	}

	if err := store.SetItemImage(r.Context(), s.DB, id, data, mime); err != nil {
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}
