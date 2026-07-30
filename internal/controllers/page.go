package controllers

// TODO: Review properly

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type Page struct {
	staticFiles fs.FS
}

func NewPage(staticFiles fs.FS) Page {
	return Page{staticFiles: staticFiles}
}

func (h Page) StaticFiles() fs.FS {
	return h.staticFiles
}

func (h Page) GetServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, h.staticFiles, "service-worker.js")
}

func (h Page) GetApplicationPage(w http.ResponseWriter, r *http.Request) {
	requestedPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestedPath == "" || isProjectPagePath(requestedPath) {
		http.ServeFileFS(w, r, h.staticFiles, "index.html")
		return
	}

	http.NotFound(w, r)
}

func isProjectPagePath(requestedPath string) bool {
	return requestedPath != "" && !strings.Contains(requestedPath, "/")
}
