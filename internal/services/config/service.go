package config

// TODO: Review properly

import (
	"fmt"
	"net"
	"os"
	"strings"

	"wade/internal/repositories"
	"wade/internal/services/terminalsessions"
)

const (
	addressEnv  = "WADE_ADDR"
	devModeEnv  = "WADE_DEV"
	defaultPort = "8765"
	devHost     = "editor-dev.localhost"
	runHost     = "editor.localhost"
)

// Config is the resolved runtime configuration used by the server.
type Config struct {
	Address                            string
	ProjectDirs                        []string
	ProjectDirectorySettings           []string
	Shell                              string
	Agents                             []repositories.Agent
	CopyIgnoredFilesOnWorktreeCreation bool
	WorktreeCopyExcludes               []string
}

// Load resolves runtime configuration from settings and environment variables.
func Load() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("getting home directory: %w", err)
	}

	settings, err := repositories.LoadSettings()
	if err != nil {
		return Config{}, err
	}

	projectDirs, err := resolveProjectDirectories(homeDir, settings.ProjectDirectories)
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
		Address:                            envOrDefault(addressEnv, defaultAddress(devMode)),
		ProjectDirs:                        projectDirs,
		ProjectDirectorySettings:           append([]string(nil), settings.ProjectDirectories...),
		Shell:                              shell,
		Agents:                             agents,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               worktreeCopyExcludes,
	}, nil
}

// defaultAddress chooses the local host used for dev or normal runs.
func defaultAddress(devMode string) string {
	if isEnabled(devMode) {
		return net.JoinHostPort(devHost, defaultPort)
	}

	return net.JoinHostPort(runHost, defaultPort)
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
		return terminalsessions.ResolveShell(environmentShell), nil
	}

	return repositories.ResolveConfiguredShell(configuredShell)
}

// envOrDefault returns an environment value when present, otherwise fallback.
func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
