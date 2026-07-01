package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"wade/terminal"
)

const (
	addressEnv  = "WADE_ADDR"
	devModeEnv  = "WADE_DEV"
	defaultPort = "8765"
	devHost     = "editor-dev.localhost"
	runHost     = "editor.localhost"
)

type Config struct {
	Address     string
	ProjectDirs []string
	Shell       string
}

func Load() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("getting home directory: %w", err)
	}

	return Config{
		Address:     envOrDefault(addressEnv, defaultAddress(os.Getenv(devModeEnv))),
		ProjectDirs: defaultProjectDirs(homeDir),
		Shell:       terminal.ResolveShell(os.Getenv("SHELL")),
	}, nil
}

func defaultAddress(devMode string) string {
	if isEnabled(devMode) {
		return net.JoinHostPort(devHost, defaultPort)
	}

	return net.JoinHostPort(runHost, defaultPort)
}

func defaultProjectDirs(homeDir string) []string {
	// TODO: Discover these from ~/.config/projman/config.json.
	return []string{
		filepath.Join(homeDir, "Personal"),
		filepath.Join(homeDir, "Work"),
		filepath.Join(homeDir, ".config"),
		filepath.Join(homeDir, "signinsolutions"),
	}
}

func isEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
