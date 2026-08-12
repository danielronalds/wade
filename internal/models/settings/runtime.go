package settings

import (
	"net"
	"strings"
)

const (
	addressEnvironmentVariable = "WADE_ADDR"
	developmentEnvironment     = "WADE_DEV"
	defaultDevelopmentPort     = "8090"
	defaultRuntimePort         = "8765"
	developmentHost            = "editor-dev.localhost"
	runtimeHost                = "editor.localhost"
	fallbackShell              = "/bin/bash"
)

func resolveRuntimeConfiguration(settings Settings, homeDirectory string, files FileSystem, environment Environment) (RuntimeConfiguration, error) {
	workspaceDirectoryPaths, err := resolveWorkspaceDirectories(homeDirectory, settings.WorkspaceDirectories)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	agents := trimAgents(settings.Agents)
	if err := validateAgents(agents); err != nil {
		return RuntimeConfiguration{}, err
	}
	worktreeCopyExcludes := trimStrings(settings.WorktreeCopyExcludes)
	if err := validateWorktreeCopyExcludes(worktreeCopyExcludes); err != nil {
		return RuntimeConfiguration{}, err
	}

	linearSettings := LinearSettings{
		Enabled:   settings.Linear.Enabled,
		Workspace: strings.TrimSpace(settings.Linear.Workspace),
	}
	if err := validateLinearSettings(linearSettings); err != nil {
		return RuntimeConfiguration{}, err
	}

	shell := strings.TrimSpace(settings.Shell)
	if shell == "" {
		shell = environment.InheritedShell()
		if shell == "" {
			shell = fallbackShell
		}
	} else {
		shell, err = resolveConfiguredShell(shell, homeDirectory, files, environment)
		if err != nil {
			return RuntimeConfiguration{}, err
		}
	}

	return RuntimeConfiguration{
		Address:                            resolveAddress(environment.Variable(developmentEnvironment), environment.Variable(addressEnvironmentVariable)),
		WorkspaceDirectoryPaths:            workspaceDirectoryPaths,
		WorkspaceDirectorySettings:         append([]string(nil), settings.WorkspaceDirectories...),
		Shell:                              shell,
		Agents:                             cloneAgents(agents),
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               append([]string(nil), worktreeCopyExcludes...),
		Linear:                             linearSettings,
	}, nil
}

func resolveAddress(developmentMode string, address string) string {
	if isEnabled(developmentMode) {
		return net.JoinHostPort(developmentHost, defaultDevelopmentPort)
	}
	if address != "" {
		return address
	}
	return net.JoinHostPort(runtimeHost, defaultRuntimePort)
}

func isEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}
