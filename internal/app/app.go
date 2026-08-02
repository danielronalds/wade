package app

// TODO: Review properly

import (
	"io/fs"
	"net/http"

	"wade/internal/controllers"
	"wade/internal/repositories"
	"wade/internal/server"
	"wade/internal/services/config"
	"wade/internal/services/gitrepositories"
	"wade/internal/services/remoterepositories"
	"wade/internal/services/review"
	"wade/internal/services/sessions"
	"wade/internal/services/terminalsessions"
	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

type Application struct {
	Mux       *http.ServeMux
	terminals *terminalsessions.Service
}

func New(configuration config.Config, staticFiles fs.FS) *Application {
	workspaceRepository := repositories.NewWorkspaceStore(configuration.ProjectDirs)
	gitRepository := repositories.NewGitRepository()
	gitHubRepository := repositories.NewGitHubRepository(repositories.RunCommand)
	fileRepository := repositories.NewFileRepository()

	localRepositoryService := gitrepositories.NewService(workspaceRepository, gitRepository)
	workspaceService := workspaces.NewService(workspaceRepository, localRepositoryService, gitHubRepository)
	reviewService := review.NewService(gitRepository, gitHubRepository, fileRepository)
	terminalSessionService := terminalsessions.NewService(configuration.Shell, configuration.Address, terminalAgents(configuration.Agents))
	sessionService := sessions.NewService(workspaceService, terminalSessionService)
	worktreeService := worktrees.NewService(configuration, gitRepository, fileRepository, workspaceRepository, terminalSessionService)
	remoteRepositoryService := remoterepositories.NewService(
		gitHubRepository,
		fileRepository,
		localRepositoryService,
		workspaceRepository,
		workspaceService,
		remoteWorkspaceDirectories(configuration),
	)

	runtimeConfigReloader := configReloader{
		workspaces:         workspaceService,
		remoteRepositories: remoteRepositoryService,
		terminals:          terminalSessionService,
	}

	controllerSet := controllers.Controllers{
		Config:         controllers.NewConfig(runtimeConfigReloader),
		Projects:       controllers.NewProjects(workspaceService),
		RemoteProjects: controllers.NewRemoteProjects(remoteRepositoryService),
		Sessions:       controllers.NewSessions(sessionService),
		Terminals:      controllers.NewTerminals(workspaceService, terminalSessionService, server.AllowSameOrigin),
		Worktrees:      controllers.NewWorktrees(localRepositoryService, worktreeService),
		Review:         controllers.NewReview(workspaceService, reviewService),
		Docs:           controllers.NewDocs(),
		Page:           controllers.NewPage(staticFiles),
	}

	httpServer := server.New(controllerSet)
	return &Application{Mux: httpServer.Mux, terminals: terminalSessionService}
}

func (a *Application) Close() {
	a.terminals.Close()
}

func terminalAgents(agents []config.Agent) []terminalsessions.Agent {
	terminalAgents := make([]terminalsessions.Agent, 0, len(agents))
	for _, agent := range agents {
		terminalAgents = append(terminalAgents, terminalsessions.Agent{
			Name:    agent.Name,
			Command: agent.Command,
			Default: agent.Default,
		})
	}
	return terminalAgents
}
