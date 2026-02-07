package api

import (
	"database/sql"
	"net/http"

	"github.com/erazemk/skladisce/internal/model"
)

// NewRouter creates the API router with all endpoints registered.
func NewRouter(db *sql.DB, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	authHandler := &AuthHandler{DB: db, JWTSecret: jwtSecret}
	usersHandler := &UsersHandler{DB: db}
	ownersHandler := &OwnersHandler{DB: db}
	itemsHandler := &ItemsHandler{DB: db}
	transfersHandler := &TransfersHandler{DB: db}
	inventoryHandler := &InventoryHandler{DB: db}

	authMW := AuthMiddleware(jwtSecret, db)
	requireAdmin := RequireRole(model.RoleAdmin)
	requireManager := RequireRole(model.RoleManager)

	// Public: login.
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Authenticated routes.
	mux.Handle("PUT /api/auth/password", authMW(http.HandlerFunc(authHandler.ChangePassword)))
	mux.Handle("POST /api/auth/logout", authMW(http.HandlerFunc(authHandler.Logout)))

	// Users (admin only).
	mux.Handle("GET /api/users", authMW(requireAdmin(http.HandlerFunc(usersHandler.List))))
	mux.Handle("POST /api/users", authMW(requireAdmin(http.HandlerFunc(usersHandler.Create))))
	mux.Handle("GET /api/users/{id}", authMW(requireAdmin(http.HandlerFunc(usersHandler.Get))))
	mux.Handle("PUT /api/users/{id}", authMW(requireAdmin(http.HandlerFunc(usersHandler.Update))))
	mux.Handle("PUT /api/users/{id}/password", authMW(requireAdmin(http.HandlerFunc(usersHandler.ResetPassword))))
	mux.Handle("DELETE /api/users/{id}", authMW(requireAdmin(http.HandlerFunc(usersHandler.Delete))))

	// Owners: read (all roles), write (manager+).
	mux.Handle("GET /api/owners", authMW(http.HandlerFunc(ownersHandler.List)))
	mux.Handle("POST /api/owners", authMW(requireManager(http.HandlerFunc(ownersHandler.Create))))
	mux.Handle("GET /api/owners/{id}", authMW(http.HandlerFunc(ownersHandler.Get)))
	mux.Handle("PUT /api/owners/{id}", authMW(requireManager(http.HandlerFunc(ownersHandler.Update))))
	mux.Handle("DELETE /api/owners/{id}", authMW(requireManager(http.HandlerFunc(ownersHandler.Delete))))
	mux.Handle("GET /api/owners/{id}/inventory", authMW(http.HandlerFunc(ownersHandler.GetInventory)))

	// Items: read (all roles), write (manager+).
	mux.Handle("GET /api/items", authMW(http.HandlerFunc(itemsHandler.List)))
	mux.Handle("POST /api/items", authMW(requireManager(http.HandlerFunc(itemsHandler.Create))))
	mux.Handle("GET /api/items/{id}", authMW(http.HandlerFunc(itemsHandler.Get)))
	mux.Handle("PUT /api/items/{id}", authMW(requireManager(http.HandlerFunc(itemsHandler.Update))))
	mux.Handle("DELETE /api/items/{id}", authMW(requireManager(http.HandlerFunc(itemsHandler.Delete))))
	mux.Handle("PUT /api/items/{id}/image", authMW(requireManager(http.HandlerFunc(itemsHandler.UploadImage))))
	mux.Handle("GET /api/items/{id}/image", authMW(http.HandlerFunc(itemsHandler.GetImage)))
	mux.Handle("GET /api/items/{id}/history", authMW(http.HandlerFunc(itemsHandler.GetHistory)))

	// Transfers (all roles).
	mux.Handle("POST /api/transfers", authMW(http.HandlerFunc(transfersHandler.Create)))
	mux.Handle("GET /api/transfers", authMW(http.HandlerFunc(transfersHandler.List)))

	// Inventory: read (all), write (manager+).
	mux.Handle("GET /api/inventory", authMW(http.HandlerFunc(inventoryHandler.List)))
	mux.Handle("POST /api/inventory/stock", authMW(requireManager(http.HandlerFunc(inventoryHandler.AddStock))))
	mux.Handle("POST /api/inventory/adjust", authMW(requireManager(http.HandlerFunc(inventoryHandler.Adjust))))

	return mux
}
