package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"web-terminal/config"
	"web-terminal/project"
	terminalmanager "web-terminal/terminal/manager"
)

type Server struct {
	configuration config.Config
	projects      project.Store
	staticFiles   fs.FS
	terminals     *terminalmanager.Manager
	mux           *http.ServeMux
}

func New(configuration config.Config, staticFiles fs.FS) *Server {
	server := &Server{
		configuration: configuration,
		projects:      project.NewStore(configuration.ProjectDirs),
		staticFiles:   staticFiles,
		terminals:     terminalmanager.New(configuration.Shell),
		mux:           http.NewServeMux(),
	}

	server.mux.HandleFunc("GET /ws", server.handleTerminal)
	server.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	server.mux.HandleFunc("GET /", server.handlePage)

	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Close() {
	s.terminals.Close()
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
