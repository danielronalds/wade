package app

// TODO: Review properly

import (
	"io/fs"
	"net/http"

	"wade/internal/controllers"
	"wade/internal/repositories"
	"wade/internal/server"
	"wade/internal/services/config"
	projectservice "wade/internal/services/projects"
	"wade/internal/services/remoteprojects"
	"wade/internal/services/review"
	"wade/internal/services/sessions"
	"wade/internal/services/terminalsessions"
	"wade/internal/services/worktrees"
)

type Application struct {
	Mux       *http.ServeMux
	terminals *terminalsessions.Service
}

func New(configuration config.Config, staticFiles fs.FS) *Application {
	projectRepository := repositories.NewStore(configuration.ProjectDirs)
	gitRepository := repositories.NewGitRepository()
	gitHubRepository := repositories.NewGitHubRepository(repositories.RunCommand)
	fileRepository := repositories.NewFileRepository()

	projectService := projectservice.NewService(projectRepository, gitRepository, gitHubRepository)
	remoteProjectService := remoteprojects.NewService(gitHubRepository, fileRepository)
	reviewService := review.NewService(gitRepository, gitHubRepository, fileRepository)
	terminalSessionService := terminalsessions.NewService(configuration.Shell, configuration.Address, terminalAgents(configuration.Agents))
	sessionService := sessions.NewService(projectService, terminalSessionService)
	worktreeService := worktrees.NewService(configuration, gitRepository, fileRepository)

	runtimeConfigReloader := configReloader{
		projects:  projectService,
		terminals: terminalSessionService,
	}

	controllerSet := controllers.Controllers{
		Config:         controllers.NewConfig(runtimeConfigReloader),
		Projects:       controllers.NewProjects(projectService),
		RemoteProjects: controllers.NewRemoteProjects(projectService, remoteProjectService),
		Sessions:       controllers.NewSessions(sessionService),
		Terminals:      controllers.NewTerminals(projectService, terminalSessionService, server.AllowSameOrigin),
		Worktrees:      controllers.NewWorktrees(projectService, worktreeService, terminalSessionService),
		Review:         controllers.NewReview(projectService, reviewService),
		Docs:           controllers.NewDocs(),
		Page:           controllers.NewPage(staticFiles),
	}

	httpServer := server.New(controllerSet, server.Options{SwaggerEnabled: configuration.SwaggerEnabled})
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
