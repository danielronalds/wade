package app

import (
	"io/fs"
	"net/http"

	"wade/internal/controllers"
	"wade/internal/infrastructure/filesystem"
	"wade/internal/infrastructure/git"
	"wade/internal/infrastructure/github"
	"wade/internal/infrastructure/linear"
	"wade/internal/infrastructure/pty"
	"wade/internal/models/remoterepositories"
	"wade/internal/models/repositories"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
	settingsrepositories "wade/internal/repositories"
	"wade/internal/server"
	"wade/internal/services/config"
	"wade/internal/services/review"
)

// Application owns the HTTP handler and application-scoped runtime resources.
type Application struct {
	Mux       *http.ServeMux
	terminals *terminals.Model
}

// New constructs the HTTP application from resolved runtime configuration.
func New(configuration config.Config, staticFiles fs.FS) *Application {
	files := filesystem.NewFileSystem()
	discovery := filesystem.NewWorkspaceDiscovery(configuration.WorkspaceDirs)
	gitClient := git.NewClient()
	githubClient := github.NewClient(github.RunCommand)
	linearClient := linear.NewClient("signinsolutions")
	ptyClient := pty.NewClient()

	workspaceModel := workspaces.New(files, discovery, githubClient, linearClient, workspaceConfiguration(configuration))
	repositoryModel := repositories.New(discovery, gitClient, files, repositoryConfiguration(configuration))
	remoteRepositoryModel := remoterepositories.New(githubClient)
	terminalModel := terminals.New(discovery, ptyClient, terminals.Configuration{
		Shell:         configuration.Shell,
		ServerAddress: configuration.Address,
		Agents:        terminalAgents(configuration.Agents),
	})
	reviewService := review.NewService(discovery, gitClient, githubClient, files)

	runtimeApplier := runtimeConfigApplier{
		workspaces:   workspaceModel,
		repositories: repositoryModel,
		terminals:    terminalModel,
	}
	settingsService := config.NewService(settingsrepositories.NewSettingsRepository(), runtimeApplier)

	controllerSet := controllers.Controllers{
		Workspaces:         controllers.NewWorkspaces(workspaceModel, repositoryModel, terminalModel),
		Repositories:       controllers.NewRepositories(repositoryModel),
		RemoteRepositories: controllers.NewRemoteRepositories(remoteRepositoryModel, repositoryModel),
		Worktrees:          controllers.NewWorktrees(repositoryModel, terminalModel),
		Terminals:          controllers.NewTerminals(terminalModel, server.AllowSameOrigin),
		ReviewSnapshots:    controllers.NewReviewSnapshots(reviewService),
		Settings:           controllers.NewSettings(settingsService),
		Docs:               controllers.NewDocs(),
		Page:               controllers.NewPage(staticFiles),
	}

	httpServer := server.New(controllerSet)
	return &Application{Mux: httpServer.Mux, terminals: terminalModel}
}

// Close releases all application-scoped runtime resources.
func (application *Application) Close() {
	application.terminals.Close()
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
