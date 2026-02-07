package web

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed static templates
var content embed.FS

// StaticFS returns the static file system.
func StaticFS() fs.FS {
	sub, err := fs.Sub(content, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}
	return sub
}

// TemplatesFS returns the templates file system.
func TemplatesFS() fs.FS {
	sub, err := fs.Sub(content, "templates")
	if err != nil {
		log.Fatalf("failed to create templates sub-filesystem: %v", err)
	}
	return sub
}
