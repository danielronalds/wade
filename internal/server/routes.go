package server

// TODO: Review properly

import (
	"net/http"

	"wade/internal/controllers"
)

func (s *Server) registerRoutes(controllers controllers.Controllers, options Options) {
	s.Mux.HandleFunc("GET /ws", controllers.Terminals.Connect)
	s.Mux.HandleFunc("POST /api/terminal/reload", controllers.Terminals.Reload)
	s.Mux.HandleFunc("GET /api/sessions", controllers.Sessions.ListSessions)
	s.Mux.HandleFunc("DELETE /api/sessions/{sessionName}", controllers.Sessions.CloseSession)
	s.Mux.HandleFunc("POST /api/sessions/{projectName}/agent", controllers.Sessions.SendToAgent)

	s.Mux.HandleFunc("GET /api/config", controllers.Config.GetConfig)
	s.Mux.HandleFunc("POST /api/config", controllers.Config.UpdateConfig)
	s.Mux.HandleFunc("POST /api/config/reload", controllers.Config.ReloadConfig)

	s.Mux.HandleFunc("GET /api/project", controllers.Projects.GetProjectDetails)
	s.Mux.HandleFunc("GET /api/projects", controllers.Projects.ListProjects)
	s.Mux.HandleFunc("GET /api/remote-projects", controllers.RemoteProjects.List)
	s.Mux.HandleFunc("POST /api/remote-projects/clone", controllers.RemoteProjects.Clone)

	s.Mux.HandleFunc("GET /api/worktrees", controllers.Worktrees.ListWorktrees)
	s.Mux.HandleFunc("POST /api/worktrees", controllers.Worktrees.CreateWorktree)
	s.Mux.HandleFunc("DELETE /api/worktrees", controllers.Worktrees.RemoveWorktree)
	s.Mux.HandleFunc("GET /api/worktrees/remote-branches", controllers.Worktrees.ListRemoteBranches)

	s.Mux.HandleFunc("GET /api/review", controllers.Review.GetReviewWindowData)
	s.Mux.HandleFunc("POST /api/review/file", controllers.Review.GetReviewFileContents)

	registerSwaggerRoutes(s.Mux, controllers.Docs, options.SwaggerEnabled)

	s.Mux.Handle("GET /static/", http.FileServer(http.FS(controllers.Page.StaticFiles())))
	s.Mux.HandleFunc("GET /", controllers.Page.GetApplicationPage)
}

func registerSwaggerRoutes(mux *http.ServeMux, docs controllers.Docs, swaggerEnabled bool) {
	if !swaggerEnabled {
		mux.HandleFunc("GET /api/openapi.json", http.NotFound)
		mux.HandleFunc("GET /api/docs", http.NotFound)
		mux.HandleFunc("GET /api/docs/", http.NotFound)
		return
	}

	mux.HandleFunc("GET /api/openapi.json", docs.OpenAPISpec)
	mux.HandleFunc("GET /api/docs", docs.OpenAPIDocs)
	mux.HandleFunc("GET /api/docs/", docs.OpenAPIDocs)
}
