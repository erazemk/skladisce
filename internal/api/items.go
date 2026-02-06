package api

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
)

// ItemsHandler handles item CRUD endpoints.
type ItemsHandler struct {
	DB *sql.DB
}

type createItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// List handles GET /api/items.
func (h *ItemsHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	items, err := store.ListItems(r.Context(), h.DB, status)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list items")
		return
	}
	if items == nil {
		items = []model.Item{}
	}
	jsonResponse(w, http.StatusOK, items)
}

// Create handles POST /api/items.
func (h *ItemsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createItemRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name required")
		return
	}

	item, err := store.CreateItem(r.Context(), h.DB, req.Name, req.Description)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create item")
		return
	}

	jsonResponse(w, http.StatusCreated, item)
}

// Get handles GET /api/items/{id}.
func (h *ItemsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	item, err := store.GetItem(r.Context(), h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get item")
		return
	}
	if item == nil {
		jsonError(w, http.StatusNotFound, "item not found")
		return
	}

	// Get distribution as well.
	dist, err := store.GetItemDistribution(r.Context(), h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get item distribution")
		return
	}
	if dist == nil {
		dist = []model.Inventory{}
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"item":         item,
		"distribution": dist,
	})
}

// Update handles PUT /api/items/{id}.
func (h *ItemsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req updateItemRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name required")
		return
	}

	if req.Status == "" {
		req.Status = model.ItemStatusActive
	}
	if req.Status != model.ItemStatusActive && req.Status != model.ItemStatusDamaged && req.Status != model.ItemStatusRetired {
		jsonError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := store.UpdateItem(r.Context(), h.DB, id, req.Name, req.Description, req.Status); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update item")
		return
	}

	item, _ := store.GetItem(r.Context(), h.DB, id)
	jsonResponse(w, http.StatusOK, item)
}

// Delete handles DELETE /api/items/{id}.
func (h *ItemsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	if err := store.DeleteItem(r.Context(), h.DB, id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "item deleted"})
}

// UploadImage handles PUT /api/items/{id}/image.
func (h *ItemsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	// Limit to 5 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		jsonError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "image file required")
		return
	}
	defer file.Close()

	// Validate MIME type.
	mime := header.Header.Get("Content-Type")
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		jsonError(w, http.StatusBadRequest, "image must be JPEG, PNG, or WebP")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	if err := store.SetItemImage(r.Context(), h.DB, id, data, mime); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to save image")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "image uploaded"})
}

// GetImage handles GET /api/items/{id}/image.
func (h *ItemsHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	data, mime, err := store.GetItemImage(r.Context(), h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get image")
		return
	}
	if data == nil {
		jsonError(w, http.StatusNotFound, "no image")
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// GetHistory handles GET /api/items/{id}/history.
func (h *ItemsHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	history, err := store.GetItemHistory(r.Context(), h.DB, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get item history")
		return
	}
	if history == nil {
		history = []model.Transfer{}
	}
	jsonResponse(w, http.StatusOK, history)
}
