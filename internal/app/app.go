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
	"wade/internal/services/terminals"
	"wade/internal/services/workspacequeries"
	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

type Application struct {
	Mux       *http.ServeMux
	terminals *terminals.Service
}

func New(configuration config.Config, staticFiles fs.FS) *Application {
	workspaceRepository := repositories.NewWorkspaceStore(configuration.WorkspaceDirs)
	gitRepository := repositories.NewGitRepository()
	gitHubRepository := repositories.NewGitHubRepository(repositories.RunCommand)
	fileRepository := repositories.NewFileRepository()

	localRepositoryService := gitrepositories.NewService(workspaceRepository, gitRepository)
	workspaceService := workspaces.NewService(workspaceRepository, localRepositoryService, gitHubRepository)
	reviewService := review.NewService(workspaceService, gitRepository, gitHubRepository, fileRepository)
	terminalService := terminals.NewService(workspaceService, configuration.Shell, configuration.Address, terminalAgents(configuration.Agents))
	workspaceQueryService := workspacequeries.NewService(workspaceService, terminalService)
	worktreeService := worktrees.NewService(configuration, gitRepository, fileRepository, workspaceRepository, terminalService)
	remoteRepositoryService := remoterepositories.NewService(
		gitHubRepository,
		fileRepository,
		localRepositoryService,
		workspaceRepository,
		workspaceService,
		remoteWorkspaceDirectories(configuration),
	)

	runtimeApplier := runtimeConfigApplier{
		workspaces:         workspaceService,
		remoteRepositories: remoteRepositoryService,
		terminals:          terminalService,
		worktrees:          worktreeService,
	}
	settingsService := config.NewService(repositories.NewSettingsRepository(), runtimeApplier)

	controllerSet := controllers.Controllers{
		Workspaces:         controllers.NewWorkspaces(workspaceQueryService, remoteRepositoryService),
		Repositories:       controllers.NewRepositories(localRepositoryService),
		RemoteRepositories: controllers.NewRemoteRepositories(remoteRepositoryService),
		Worktrees:          controllers.NewWorktrees(localRepositoryService, worktreeService),
		Terminals:          controllers.NewTerminals(terminalService, server.AllowSameOrigin),
		ReviewSnapshots:    controllers.NewReviewSnapshots(reviewService),
		Settings:           controllers.NewSettings(settingsService),
		Docs:               controllers.NewDocs(),
		Page:               controllers.NewPage(staticFiles),
	}

	httpServer := server.New(controllerSet)
	return &Application{Mux: httpServer.Mux, terminals: terminalService}
}

func (a *Application) Close() {
	a.terminals.Close()
}

func terminalAgents(agents []config.Agent) []terminals.Agent {
	terminalAgents := make([]terminals.Agent, 0, len(agents))
	for _, agent := range agents {
		terminalAgents = append(terminalAgents, terminals.Agent{
			Name:    agent.Name,
			Command: agent.Command,
			Default: agent.Default,
		})
	}
	return terminalAgents
}
