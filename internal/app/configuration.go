package app

import (
	"wade/internal/models/repositories"
	"wade/internal/models/settings"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
)

func workspaceConfiguration(configuration settings.RuntimeConfiguration) workspaces.Configuration {
	workspaceDirectories := make([]workspaces.WorkspaceDirectory, 0, len(configuration.WorkspaceDirectoryPaths))
	for index, path := range configuration.WorkspaceDirectoryPaths {
		setting := path
		if index < len(configuration.WorkspaceDirectorySettings) {
			setting = configuration.WorkspaceDirectorySettings[index]
		}
		workspaceDirectories = append(workspaceDirectories, workspaces.WorkspaceDirectory{Setting: setting, Path: path})
	}
	return workspaces.Configuration{WorkspaceDirectories: workspaceDirectories}
}

func repositoryConfiguration(configuration settings.RuntimeConfiguration) repositories.Configuration {
	return repositories.Configuration{
		CopyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
	}
}

func terminalConfiguration(configuration settings.RuntimeConfiguration) terminals.Configuration {
	agents := make([]terminals.Agent, 0, len(configuration.Agents))
	for _, agent := range configuration.Agents {
		agents = append(agents, terminals.Agent{Name: agent.Name, Command: agent.Command, Default: agent.Default})
	}
	return terminals.Configuration{
		Shell:         configuration.Shell,
		ServerAddress: configuration.Address,
		Agents:        agents,
	}
}
