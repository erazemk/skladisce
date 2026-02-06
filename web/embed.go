package web

import (
	"embed"
	"io/fs"
)

//go:embed static templates
var content embed.FS

// StaticFS returns the static file system.
func StaticFS() fs.FS {
	sub, _ := fs.Sub(content, "static")
	return sub
}

// TemplatesFS returns the templates file system.
func TemplatesFS() fs.FS {
	sub, _ := fs.Sub(content, "templates")
	return sub
}
