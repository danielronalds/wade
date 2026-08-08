package controllers

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Page serves embedded frontend assets and application routes.
type Page struct {
	staticFiles fs.FS
}

// NewPage constructs the frontend page controller.
func NewPage(staticFiles fs.FS) Page {
	return Page{staticFiles: staticFiles}
}

// StaticFiles returns the embedded filesystem used by the static file server.
func (h Page) StaticFiles() fs.FS {
	return h.staticFiles
}

// GetServiceWorker serves the service worker without browser caching.
func (h Page) GetServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, h.staticFiles, "service-worker.js")
}

// GetApplicationPage serves the application shell for recognised frontend routes.
func (h Page) GetApplicationPage(w http.ResponseWriter, r *http.Request) {
	requestedPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestedPath == "" || isApplicationPagePath(requestedPath) {
		http.ServeFileFS(w, r, h.staticFiles, "index.html")
		return
	}

	http.NotFound(w, r)
}

func isApplicationPagePath(requestedPath string) bool {
	if requestedPath == "settings" {
		return true
	}

	parts := strings.Split(requestedPath, "/")
	return len(parts) == 2 && parts[0] == "workspaces" && parts[1] != ""
}
