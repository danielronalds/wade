package settings

func cloneSettings(settings Settings) Settings {
	cloned := settings
	cloned.WorkspaceDirectories = cloneStrings(settings.WorkspaceDirectories)
	cloned.Agents = cloneAgents(settings.Agents)
	cloned.WorktreeCopyExcludes = cloneStrings(settings.WorktreeCopyExcludes)
	cloned.Linear = settings.Linear
	return cloned
}

func cloneRuntimeConfiguration(configuration RuntimeConfiguration) RuntimeConfiguration {
	cloned := configuration
	cloned.WorkspaceDirectoryPaths = cloneStrings(configuration.WorkspaceDirectoryPaths)
	cloned.WorkspaceDirectorySettings = cloneStrings(configuration.WorkspaceDirectorySettings)
	cloned.Agents = cloneAgents(configuration.Agents)
	cloned.WorktreeCopyExcludes = cloneStrings(configuration.WorktreeCopyExcludes)
	cloned.Linear = configuration.Linear
	return cloned
}

func cloneUpdateResult(result UpdateResult) UpdateResult {
	return UpdateResult{
		Settings:             cloneSettings(result.Settings),
		RuntimeConfiguration: cloneRuntimeConfiguration(result.RuntimeConfiguration),
	}
}

func cloneAgents(agents []Agent) []Agent {
	if agents == nil {
		return nil
	}
	return append(make([]Agent, 0, len(agents)), agents...)
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append(make([]string, 0, len(values)), values...)
}
