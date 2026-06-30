package server

import (
	"io/fs"
	"net/http"

	"web-terminal/config"
	"web-terminal/project"
	"web-terminal/server/handlers"
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
	server.mux.Handle("GET /api/project", handlers.NewProject(server.projects))
	server.mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))
	server.mux.Handle("GET /", handlers.NewPage(server.staticFiles))

	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Close() {
	s.terminals.Close()
}
