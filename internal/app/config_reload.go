package app

// TODO: Review properly

import (
	"wade/internal/services/config"
	"wade/internal/services/remoterepositories"
	"wade/internal/services/terminals"
	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

type runtimeConfigApplier struct {
	workspaces         workspaces.Service
	remoteRepositories *remoterepositories.Service
	terminals          *terminals.Service
	worktrees          worktrees.Service
}

func (a runtimeConfigApplier) ApplyConfig(configuration config.Config) {
	a.workspaces.Reload(configuration.WorkspaceDirs)
	a.remoteRepositories.Configure(remoteWorkspaceDirectories(configuration))
	a.terminals.Configure(configuration.Shell, terminalAgents(configuration.Agents))
	a.worktrees.Configure(configuration)
}

func remoteWorkspaceDirectories(configuration config.Config) []remoterepositories.WorkspaceDirectory {
	workspaceDirectories := make([]remoterepositories.WorkspaceDirectory, 0, len(configuration.WorkspaceDirs))
	for index, path := range configuration.WorkspaceDirs {
		setting := path
		if index < len(configuration.WorkspaceDirectorySettings) {
			setting = configuration.WorkspaceDirectorySettings[index]
		}

		workspaceDirectories = append(workspaceDirectories, remoterepositories.WorkspaceDirectory{
			Setting: setting,
			Path:    path,
		})
	}

	return workspaceDirectories
}
