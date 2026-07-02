package server

import (
	"io/fs"
	"net/http"

	"wade/config"
	"wade/project"
	"wade/remote"
	"wade/server/handlers"
	terminalmanager "wade/terminal/manager"
	"wade/worktree"
)

// Server owns HTTP routing and long-lived runtime state.
type Server struct {
	projects    project.Store
	staticFiles fs.FS
	terminals   *terminalmanager.Manager
	mux         *http.ServeMux
}

// New wires handlers to shared project and terminal state.
func New(configuration config.Config, staticFiles fs.FS) *Server {
	server := &Server{
		projects:    project.NewStore(configuration.ProjectDirs),
		staticFiles: staticFiles,
		terminals:   terminalmanager.New(configuration.Shell, configuration.AgentPaneCommand),
		mux:         http.NewServeMux(),
	}

	configHandler := handlers.NewConfig()
	remoteHandler := handlers.NewRemoteProjects(server.projects, remote.NewService(remote.RunCommand))
	worktreeService := worktree.NewService(configuration)
	worktreesHandler := handlers.NewWorktrees(server.projects, worktreeService, server.terminals)

	server.mux.HandleFunc("GET /ws", server.handleTerminal)
	server.mux.HandleFunc("POST /api/terminal/reload", server.handleTerminalReload)

	server.mux.Handle("GET /api/config", configHandler)
	server.mux.Handle("POST /api/config", configHandler)
	server.mux.HandleFunc("POST /api/config/reload", server.handleConfigReload)

	server.mux.Handle("GET /api/project", handlers.NewProject(server.projects))
	server.mux.Handle("GET /api/projects", handlers.NewProjects(server.projects))
	server.mux.HandleFunc("GET /api/remote-projects", remoteHandler.List)
	server.mux.HandleFunc("POST /api/remote-projects/clone", remoteHandler.Clone)

	server.mux.HandleFunc("GET /api/worktrees", worktreesHandler.ListWorktrees)
	server.mux.HandleFunc("POST /api/worktrees", worktreesHandler.CreateWorktree)
	server.mux.HandleFunc("DELETE /api/worktrees", worktreesHandler.RemoveWorktree)
	server.mux.HandleFunc("GET /api/worktrees/remote-branches", worktreesHandler.ListRemoteBranches)

	server.mux.Handle("GET /api/review", handlers.NewReview(server.projects))
	server.mux.Handle("POST /api/review/file", handlers.NewReviewFile(server.projects))

	server.mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))
	server.mux.Handle("GET /", handlers.NewPage(server.staticFiles))

	return server
}

// ServeHTTP delegates requests to the server mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Close stops terminal sessions.
func (s *Server) Close() {
	s.terminals.Close()
}
