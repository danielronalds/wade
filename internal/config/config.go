package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"wade/internal/terminal"
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
	Shell                              string
	Agents                             []Agent
	CopyIgnoredFilesOnWorktreeCreation bool
	WorktreeCopyExcludes               []string
}

// Load resolves runtime configuration from settings and environment variables.
func Load() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("getting home directory: %w", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		return Config{}, err
	}

	projectDirs, err := resolveProjectDirectories(homeDir, settings.ProjectDirectories)
	if err != nil {
		return Config{}, err
	}

	agents := trimAgents(settings.Agents)
	if err := ValidateAgents(agents); err != nil {
		return Config{}, err
	}

	worktreeCopyExcludes := trimStrings(settings.WorktreeCopyExcludes)
	if err := ValidateWorktreeCopyExcludes(worktreeCopyExcludes); err != nil {
		return Config{}, err
	}

	return Config{
		Address:                            envOrDefault(addressEnv, defaultAddress(os.Getenv(devModeEnv))),
		ProjectDirs:                        projectDirs,
		Shell:                              terminal.ResolveShell(os.Getenv("SHELL")),
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

// envOrDefault returns an environment value when present, otherwise fallback.
func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
