package config

// TODO: Review properly

import (
	"fmt"
	"net"
	"os"
	"strings"

	"wade/internal/repositories"
	"wade/internal/services/terminals"
)

const (
	addressEnv     = "WADE_ADDR"
	devModeEnv     = "WADE_DEV"
	defaultDevPort = "8090"
	defaultRunPort = "8765"
	devHost        = "editor-dev.localhost"
	runHost        = "editor.localhost"
)

// Config is the resolved runtime configuration used by the server.
type Config struct {
	Address                            string
	WorkspaceDirs                      []string
	WorkspaceDirectorySettings         []string
	Shell                              string
	Agents                             []repositories.Agent
	CopyIgnoredFilesOnWorktreeCreation bool
	WorktreeCopyExcludes               []string
}

// Load resolves runtime configuration from settings and environment variables.
func Load() (Config, error) {
	settings, err := repositories.LoadSettings()
	if err != nil {
		return Config{}, err
	}

	return resolveRuntimeConfig(settings)
}

func resolveRuntimeConfig(settings Settings) (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("getting home directory: %w", err)
	}

	workspaceDirs, err := resolveWorkspaceDirectories(homeDir, settings.WorkspaceDirectories)
	if err != nil {
		return Config{}, err
	}

	agents := trimAgents(settings.Agents)
	if err := repositories.ValidateAgents(agents); err != nil {
		return Config{}, err
	}

	worktreeCopyExcludes := trimStrings(settings.WorktreeCopyExcludes)
	if err := repositories.ValidateWorktreeCopyExcludes(worktreeCopyExcludes); err != nil {
		return Config{}, err
	}

	shell, err := resolveRuntimeShell(settings.Shell, os.Getenv("SHELL"))
	if err != nil {
		return Config{}, err
	}

	devMode := os.Getenv(devModeEnv)

	return Config{
		Address:                            resolveAddress(devMode, os.Getenv(addressEnv)),
		WorkspaceDirs:                      workspaceDirs,
		WorkspaceDirectorySettings:         append([]string(nil), settings.WorkspaceDirectories...),
		Shell:                              shell,
		Agents:                             agents,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               worktreeCopyExcludes,
	}, nil
}

// resolveAddress gives development mode precedence over an inherited runtime address.
func resolveAddress(devMode string, address string) string {
	if isEnabled(devMode) {
		return net.JoinHostPort(devHost, defaultDevPort)
	}
	if address != "" {
		return address
	}

	return net.JoinHostPort(runHost, defaultRunPort)
}

// isEnabled treats common disabled strings as false and anything else as true.
func isEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}

func resolveRuntimeShell(configuredShell string, environmentShell string) (string, error) {
	configuredShell = strings.TrimSpace(configuredShell)
	if configuredShell == "" {
		return terminals.ResolveShell(environmentShell), nil
	}

	return repositories.ResolveConfiguredShell(configuredShell)
}
