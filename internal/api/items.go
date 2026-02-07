package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/erazemk/skladisce/internal/imaging"
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
		slog.Error("failed to list items", "error", err)
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
		slog.Error("failed to create item", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to create item")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("item created", "user", claims.Username, "item", req.Name)
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
		slog.Error("failed to get item", "error", err)
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
		slog.Error("failed to get item distribution", "error", err)
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
	if req.Status != model.ItemStatusActive && req.Status != model.ItemStatusDamaged && req.Status != model.ItemStatusLost && req.Status != model.ItemStatusRemoved {
		jsonError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := store.UpdateItem(r.Context(), h.DB, id, req.Name, req.Description, req.Status); err != nil {
		slog.Error("failed to update item", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to update item")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("item updated", "user", claims.Username, "item", req.Name, "status", req.Status)
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

	item, _ := store.GetItem(r.Context(), h.DB, id)
	itemName := fmt.Sprintf("id:%d", id)
	if item != nil {
		itemName = item.Name
	}

	if err := store.DeleteItem(r.Context(), h.DB, id); err != nil {
		slog.Error("failed to delete item", "error", err)
		jsonError(w, http.StatusNotFound, "item not found")
		return
	}

	claims := GetClaims(r.Context())
	slog.Info("item deleted", "user", claims.Username, "item", itemName)
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

	file, _, err := r.FormFile("image")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "image file required")
		return
	}
	defer file.Close()

	// Process the image: validate format by sniffing bytes, downscale, compress.
	result, err := imaging.Process(file)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := store.SetItemImage(r.Context(), h.DB, id, result.Data, result.MIME); err != nil {
		slog.Error("failed to save image", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to save image")
		return
	}

	claims := GetClaims(r.Context())
	item, _ := store.GetItem(r.Context(), h.DB, id)
	itemName := fmt.Sprintf("id:%d", id)
	if item != nil {
		itemName = item.Name
	}
	slog.Info("item image uploaded", "user", claims.Username, "item", itemName)
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
		slog.Error("failed to get image", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to get image")
		return
	}
	if data == nil {
		jsonError(w, http.StatusNotFound, "no image")
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write image response", "error", err)
	}
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
		slog.Error("failed to get item history", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to get item history")
		return
	}
	if history == nil {
		history = []model.Transfer{}
	}
	jsonResponse(w, http.StatusOK, history)
}
