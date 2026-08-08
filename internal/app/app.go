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
	"wade/internal/models/reviewsnapshots"
	"wade/internal/models/settings"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
	"wade/internal/server"
)

// Application owns the HTTP handler and application-scoped runtime resources.
type Application struct {
	Mux       *http.ServeMux
	terminals *terminals.Model
}

// New constructs the HTTP application from resolved runtime configuration and the shared Settings Model.
func New(configuration settings.RuntimeConfiguration, settingsModel controllers.SettingsModel, staticFiles fs.FS) *Application {
	files := filesystem.NewFileSystem()
	discovery := filesystem.NewWorkspaceDiscovery(configuration.WorkspaceDirectoryPaths)
	gitClient := git.NewClient()
	githubClient := github.NewClient(github.RunCommand)
	linearClient := linear.NewClient("signinsolutions")
	ptyClient := pty.NewClient()

	workspaceModel := workspaces.New(files, discovery, githubClient, linearClient, workspaceConfiguration(configuration))
	repositoryModel := repositories.New(discovery, gitClient, files, repositoryConfiguration(configuration))
	remoteRepositoryModel := remoterepositories.New(githubClient)
	terminalModel := terminals.New(discovery, ptyClient, terminalConfiguration(configuration))
	reviewSnapshotModel := reviewsnapshots.New(discovery, gitClient, githubClient, files)

	controllerSet := controllers.Controllers{
		Workspaces:         controllers.NewWorkspaces(workspaceModel, repositoryModel, terminalModel),
		Repositories:       controllers.NewRepositories(repositoryModel),
		RemoteRepositories: controllers.NewRemoteRepositories(remoteRepositoryModel, repositoryModel),
		Worktrees:          controllers.NewWorktrees(repositoryModel, terminalModel),
		Terminals:          controllers.NewTerminals(terminalModel, server.AllowSameOrigin),
		ReviewSnapshots:    controllers.NewReviewSnapshots(reviewSnapshotModel),
		Settings:           controllers.NewSettings(settingsModel, workspaceModel, repositoryModel, terminalModel),
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
