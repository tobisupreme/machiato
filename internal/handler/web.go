package handler

import (
	"embed"
	"io/fs"
	"net/http"
)

// NewWebHandler returns an http.Handler that serves the embedded web UI.
func NewWebHandler(assets embed.FS) http.Handler {
	sub, err := fs.Sub(assets, "web")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
