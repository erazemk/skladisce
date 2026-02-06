package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/model"
	webembed "github.com/erazemk/skladisce/web"
)

// Templates holds parsed HTML templates.
type Templates struct {
	templates map[string]*template.Template
}

// FuncMap returns the template function map.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"roleAtLeast": model.RoleAtLeast,
		"roleName": func(role string) string {
			switch role {
			case "admin":
				return "Administrator"
			case "manager":
				return "Skladiščar"
			case "user":
				return "Uporabnik"
			default:
				return role
			}
		},
		"statusName": func(status string) string {
			switch status {
			case "active":
				return "Aktiven"
			case "damaged":
				return "Poškodovan"
			case "retired":
				return "Umaknjen"
			default:
				return status
			}
		},
	}
}

// LoadTemplates parses all page templates with the layout.
func LoadTemplates() (*Templates, error) {
	tfs := webembed.TemplatesFS()

	// Read layout.
	layoutBytes, err := fs.ReadFile(tfs, "layout.html")
	if err != nil {
		return nil, fmt.Errorf("reading layout template: %w", err)
	}

	pages := []string{
		"login.html",
		"dashboard.html",
		"items.html",
		"item_detail.html",
		"owners.html",
		"owner_detail.html",
		"transfers.html",
		"transfer_new.html",
		"users.html",
		"settings.html",
	}

	ts := &Templates{templates: make(map[string]*template.Template)}

	for _, page := range pages {
		pageBytes, err := fs.ReadFile(tfs, page)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", page, err)
		}

		tmpl := template.New(page).Funcs(FuncMap())
		tmpl, err = tmpl.Parse(string(layoutBytes))
		if err != nil {
			return nil, fmt.Errorf("parsing layout for %s: %w", page, err)
		}
		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}

		ts.templates[page] = tmpl
	}

	return ts, nil
}

// Render renders a template with the given data.
func (ts *Templates) Render(w http.ResponseWriter, name string, data any) {
	tmpl, ok := ts.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("failed to render template", "template", name, "error", err)
	}
}

// PageData is the base data passed to all templates.
type PageData struct {
	Title string
	User  *auth.Claims
	Token string
	Error string
	Success string
}

// Server holds all dependencies for page handlers.
type Server struct {
	DB        *sql.DB
	Templates *Templates
	JWTSecret string
}
