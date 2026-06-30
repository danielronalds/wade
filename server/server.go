package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"web-terminal/config"
	"web-terminal/project"
)

type Server struct {
	configuration config.Config
	projects      project.Store
	staticFiles   fs.FS
}

func New(configuration config.Config, staticFiles fs.FS) http.Handler {
	server := &Server{
		configuration: configuration,
		projects:      project.NewStore(configuration.ProjectDirs),
		staticFiles:   staticFiles,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", server.handleTerminal)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /", server.handlePage)

	return mux
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	requestedPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestedPath == "" || isProjectPagePath(requestedPath) {
		http.ServeFileFS(w, r, s.staticFiles, "index.html")
		return
	}

	http.NotFound(w, r)
}

func isProjectPagePath(requestedPath string) bool {
	return requestedPath != "" && !strings.Contains(requestedPath, "/")
}
