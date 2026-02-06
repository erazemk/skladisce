package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// OwnersPage handles GET /owners.
func (s *Server) OwnersPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	owners, _ := store.ListOwners(r.Context(), s.DB, "")

	s.Templates.Render(w, "owners.html", &struct {
		PageData
		Owners []model.Owner
	}{
		PageData: PageData{Title: "Lastniki", User: claims, Token: GetWebToken(r.Context())},
		Owners:   owners,
	})
}

// OwnerDetailPage handles GET /owners/{id}.
func (s *Server) OwnerDetailPage(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	owner, err := store.GetOwner(r.Context(), s.DB, id)
	if err != nil || owner == nil {
		http.Error(w, "owner not found", http.StatusNotFound)
		return
	}

	inventory, _ := store.GetOwnerInventory(r.Context(), s.DB, id)

	s.Templates.Render(w, "owner_detail.html", &struct {
		PageData
		Owner     *model.Owner
		Inventory []model.Inventory
	}{
		PageData:  PageData{Title: owner.Name, User: claims, Token: GetWebToken(r.Context())},
		Owner:     owner,
		Inventory: inventory,
	})
}

// OwnerCreateSubmit handles POST /owners.
func (s *Server) OwnerCreateSubmit(w http.ResponseWriter, r *http.Request) {
	claims := GetWebClaims(r.Context())
	if !model.RoleAtLeast(claims.Role, model.RoleManager) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	name := r.FormValue("name")
	ownerType := r.FormValue("type")

	if name == "" || ownerType == "" {
		http.Redirect(w, r, "/owners", http.StatusSeeOther)
		return
	}

	store.CreateOwner(r.Context(), s.DB, name, ownerType)
	http.Redirect(w, r, "/owners", http.StatusSeeOther)
}

// OwnerUpdateSubmit handles POST /owners/{id}.
func (s *Server) OwnerUpdateSubmit(w http.ResponseWriter, r *http.Request) {
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
	if name == "" {
		http.Redirect(w, r, fmt.Sprintf("/owners/%d", id), http.StatusSeeOther)
		return
	}

	store.UpdateOwner(r.Context(), s.DB, id, name)
	http.Redirect(w, r, fmt.Sprintf("/owners/%d", id), http.StatusSeeOther)
}
