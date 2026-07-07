package handlers

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
