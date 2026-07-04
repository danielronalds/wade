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
	Mux         *http.ServeMux
}

// New wires handlers to shared project and terminal state.
func New(configuration config.Config, staticFiles fs.FS) *Server {
	server := &Server{
		projects:    project.NewStore(configuration.ProjectDirs),
		staticFiles: staticFiles,
		terminals:   terminalmanager.New(configuration.Shell, configuration.AgentPaneCommand),
		Mux:         http.NewServeMux(),
	}

	configHandler := handlers.NewConfig()

	projectsHandler := handlers.NewProjects(server.projects)
	sessionsHandler := handlers.NewSessions(server.projects, server.terminals)

	remoteService := remote.NewService(remote.RunCommand)
	remoteHandler := handlers.NewRemoteProjects(server.projects, remoteService)

	worktreeService := worktree.NewService(configuration)
	worktreesHandler := handlers.NewWorktrees(server.projects, worktreeService, server.terminals)

	reviewHandler := handlers.NewReview(server.projects)

	pageHandler := handlers.NewPage(server.staticFiles)

	server.Mux.HandleFunc("GET /ws", server.handleTerminal)
	server.Mux.HandleFunc("POST /api/terminal/reload", server.handleTerminalReload)
	server.Mux.HandleFunc("GET /api/sessions", sessionsHandler.ListSessions)
	server.Mux.HandleFunc("DELETE /api/session/{sessionName}", sessionsHandler.CloseSession)

	server.Mux.HandleFunc("GET /api/config", configHandler.GetConfig)
	server.Mux.HandleFunc("POST /api/config", configHandler.UpdateConfig)
	server.Mux.HandleFunc("POST /api/config/reload", server.handleConfigReload)

	server.Mux.HandleFunc("GET /api/project", projectsHandler.GetProjectDetails)
	server.Mux.HandleFunc("GET /api/projects", projectsHandler.ListProjects)
	server.Mux.HandleFunc("GET /api/remote-projects", remoteHandler.List)
	server.Mux.HandleFunc("POST /api/remote-projects/clone", remoteHandler.Clone)

	server.Mux.HandleFunc("GET /api/worktrees", worktreesHandler.ListWorktrees)
	server.Mux.HandleFunc("POST /api/worktrees", worktreesHandler.CreateWorktree)
	server.Mux.HandleFunc("DELETE /api/worktrees", worktreesHandler.RemoveWorktree)
	server.Mux.HandleFunc("GET /api/worktrees/remote-branches", worktreesHandler.ListRemoteBranches)

	server.Mux.HandleFunc("GET /api/review", reviewHandler.GetReviewWindowData)
	server.Mux.HandleFunc("POST /api/review/file", reviewHandler.GetReviewFileContents)

	server.Mux.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))
	server.Mux.HandleFunc("GET /", pageHandler.GetApplicationPage)

	return server
}

// Close stops terminal sessions.
func (s *Server) Close() {
	s.terminals.Close()
}
