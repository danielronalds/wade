package server

import (
	"net/http"

	"wade/internal/controllers"
)

func (s *Server) registerRoutes(controllerSet controllers.Controllers) {
	s.Mux.HandleFunc("GET /api/v1/workspaces", controllerSet.Workspaces.List)
	s.Mux.HandleFunc("POST /api/v1/workspaces", controllerSet.Workspaces.Materialise)
	s.Mux.HandleFunc("GET /api/v1/workspaces/{workspaceId}", controllerSet.Workspaces.Get)
	s.Mux.HandleFunc("POST /api/v1/workspaces/{workspaceId}/start", controllerSet.Workspaces.Start)

	s.Mux.HandleFunc("GET /api/v1/remote-repositories", controllerSet.RemoteRepositories.List)
	s.Mux.HandleFunc("GET /api/v1/repositories/{repositoryId}", controllerSet.Repositories.Get)
	s.Mux.HandleFunc("GET /api/v1/repositories/{repositoryId}/worktrees", controllerSet.Worktrees.List)
	s.Mux.HandleFunc("POST /api/v1/repositories/{repositoryId}/worktrees", controllerSet.Worktrees.Create)
	s.Mux.HandleFunc("DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}", controllerSet.Worktrees.Delete)
	s.Mux.HandleFunc("GET /api/v1/repositories/{repositoryId}/branches", controllerSet.Worktrees.ListBranches)

	s.Mux.HandleFunc("GET /api/v1/workspaces/{workspaceId}/terminals", controllerSet.Terminals.List)
	s.Mux.HandleFunc("DELETE /api/v1/workspaces/{workspaceId}/terminals", controllerSet.Terminals.DeleteAll)
	s.Mux.HandleFunc("PUT /api/v1/workspaces/{workspaceId}/terminals/{terminalId}", controllerSet.Terminals.Put)
	s.Mux.HandleFunc("GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}", controllerSet.Terminals.Get)
	s.Mux.HandleFunc("DELETE /api/v1/workspaces/{workspaceId}/terminals/{terminalId}", controllerSet.Terminals.Delete)
	s.Mux.HandleFunc("POST /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input", controllerSet.Terminals.Input)
	s.Mux.HandleFunc("GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket", controllerSet.Terminals.Connect)

	s.Mux.HandleFunc("POST /api/v1/workspaces/{workspaceId}/review-snapshots", controllerSet.ReviewSnapshots.Create)
	s.Mux.HandleFunc("GET /api/v1/review-snapshots/{snapshotId}", controllerSet.ReviewSnapshots.Get)
	s.Mux.HandleFunc("GET /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents", controllerSet.ReviewSnapshots.GetFileContents)
	s.Mux.HandleFunc("DELETE /api/v1/review-snapshots/{snapshotId}", controllerSet.ReviewSnapshots.Delete)

	s.Mux.HandleFunc("GET /api/v1/settings", controllerSet.Settings.Get)
	s.Mux.HandleFunc("PUT /api/v1/settings", controllerSet.Settings.Update)
	s.Mux.HandleFunc("POST /api/v1/settings/reload", controllerSet.Settings.Reload)

	s.Mux.HandleFunc("GET /api/openapi.json", controllerSet.Docs.OpenAPISpec)
	s.Mux.HandleFunc("GET /api/docs/", controllerSet.Docs.OpenAPIDocs)
	s.Mux.HandleFunc("GET /api/", controllers.WriteAPINotFound)
	s.Mux.HandleFunc("POST /api/", controllers.WriteAPINotFound)
	s.Mux.HandleFunc("PUT /api/", controllers.WriteAPINotFound)
	s.Mux.HandleFunc("DELETE /api/", controllers.WriteAPINotFound)

	s.Mux.Handle("GET /static/", http.FileServer(http.FS(controllerSet.Page.StaticFiles())))
	s.Mux.HandleFunc("GET /service-worker.js", controllerSet.Page.GetServiceWorker)
	s.Mux.HandleFunc("GET /", controllerSet.Page.GetApplicationPage)
}
