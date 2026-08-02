package app

// TODO: Review properly

import (
	"wade/internal/services/config"
	"wade/internal/services/remoterepositories"
	"wade/internal/services/terminalsessions"
	"wade/internal/services/workspaces"
)

type configReloader struct {
	workspaces         workspaces.Service
	remoteRepositories *remoterepositories.Service
	terminals          *terminalsessions.Service
}

func (r configReloader) ReloadConfig() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}

	r.workspaces.Reload(configuration.ProjectDirs)
	r.remoteRepositories.Configure(remoteWorkspaceDirectories(configuration))
	r.terminals.Configure(configuration.Shell, terminalAgents(configuration.Agents))
	return nil
}

func remoteWorkspaceDirectories(configuration config.Config) []remoterepositories.WorkspaceDirectory {
	workspaceDirectories := make([]remoterepositories.WorkspaceDirectory, 0, len(configuration.ProjectDirs))
	for index, path := range configuration.ProjectDirs {
		setting := path
		if index < len(configuration.ProjectDirectorySettings) {
			setting = configuration.ProjectDirectorySettings[index]
		}

		workspaceDirectories = append(workspaceDirectories, remoterepositories.WorkspaceDirectory{
			Setting: setting,
			Path:    path,
		})
	}

	return workspaceDirectories
}
