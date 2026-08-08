package app

import (
	"wade/internal/models/repositories"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
	"wade/internal/services/config"
)

type runtimeConfigApplier struct {
	workspaces   *workspaces.Model
	repositories *repositories.Model
	terminals    *terminals.Model
}

func (applier runtimeConfigApplier) ApplyConfig(configuration config.Config) {
	applier.workspaces.Configure(workspaceConfiguration(configuration))
	applier.repositories.Configure(repositoryConfiguration(configuration))
	applier.terminals.Configure(terminals.Configuration{
		Shell:  configuration.Shell,
		Agents: terminalAgents(configuration.Agents),
	})
}

func workspaceConfiguration(configuration config.Config) workspaces.Configuration {
	workspaceDirectories := make([]workspaces.WorkspaceDirectory, 0, len(configuration.WorkspaceDirs))
	for index, path := range configuration.WorkspaceDirs {
		setting := path
		if index < len(configuration.WorkspaceDirectorySettings) {
			setting = configuration.WorkspaceDirectorySettings[index]
		}
		workspaceDirectories = append(workspaceDirectories, workspaces.WorkspaceDirectory{Setting: setting, Path: path})
	}
	return workspaces.Configuration{WorkspaceDirectories: workspaceDirectories}
}

func repositoryConfiguration(configuration config.Config) repositories.Configuration {
	return repositories.Configuration{
		CopyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
	}
}
