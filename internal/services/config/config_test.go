package config

// TODO: Review properly

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"wade/internal/repositories"
)

func TestResolveRuntimeShellUsesConfiguredShellOverEnvironment(t *testing.T) {
	directory := t.TempDir()
	shell := writeExecutable(t, directory, "custom-shell")
	t.Setenv("PATH", directory)

	resolvedShell, err := resolveRuntimeShell("custom-shell", "/bin/zsh")
	if err != nil {
		t.Fatalf("resolveRuntimeShell() error = %v, want nil", err)
	}

	if resolvedShell != shell {
		t.Fatalf("resolveRuntimeShell() shell = %q, want %q", resolvedShell, shell)
	}
}

func TestLoadPreservesConfiguredWorkspaceDirectoryStrings(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/sh")

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		WorkspaceDirectories: []string{"~/Code"},
		Agents:               []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
	}
	writeSettings(t, path, settings)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if len(configuration.WorkspaceDirectorySettings) != 1 || configuration.WorkspaceDirectorySettings[0] != "~/Code" {
		t.Fatalf("WorkspaceDirectorySettings = %#v, want [~/Code]", configuration.WorkspaceDirectorySettings)
	}
	wantResolvedPath := filepath.Join(homeDir, "Code")
	if len(configuration.WorkspaceDirs) != 1 || configuration.WorkspaceDirs[0] != wantResolvedPath {
		t.Fatalf("WorkspaceDirs = %#v, want [%s]", configuration.WorkspaceDirs, wantResolvedPath)
	}
}

func TestLoadUsesConfiguredShellOverEnvironment(t *testing.T) {
	homeDir := t.TempDir()
	shell := writeExecutable(t, homeDir, "custom-shell")
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/zsh")

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		WorkspaceDirectories: []string{},
		Shell:                shell,
		Agents:               []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
	}
	writeSettings(t, path, settings)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if configuration.Shell != shell {
		t.Fatalf("Shell = %q, want %q", configuration.Shell, shell)
	}
}

func TestLoadUsesEnvironmentShellWhenShellSettingIsEmpty(t *testing.T) {
	homeDir := t.TempDir()
	environmentShell := "/bin/custom-shell"
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", environmentShell)

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		WorkspaceDirectories: []string{},
		Shell:                "",
		Agents:               []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
	}
	writeSettings(t, path, settings)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if configuration.Shell != environmentShell {
		t.Fatalf("Shell = %q, want %q", configuration.Shell, environmentShell)
	}
}

func TestResolveAddressGivesDevelopmentModePrecedence(t *testing.T) {
	tests := map[string]struct {
		devMode string
		address string
		want    string
	}{
		"development ignores inherited address": {
			devMode: "1",
			address: "editor.localhost:8765",
			want:    "editor-dev.localhost:8090",
		},
		"runtime uses configured address": {
			address: "custom.localhost:9000",
			want:    "custom.localhost:9000",
		},
		"runtime uses default address": {
			want: "editor.localhost:8765",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := resolveAddress(test.devMode, test.address); got != test.want {
				t.Fatalf("resolveAddress(%q, %q) = %q, want %q", test.devMode, test.address, got, test.want)
			}
		})
	}
}

func TestLoadUsesAddressEnvironmentOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv(addressEnv, "custom.localhost:9000")
	t.Setenv(devModeEnv, "")

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		WorkspaceDirectories: []string{},
		Agents:               []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
	}
	writeSettings(t, path, settings)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if configuration.Address != "custom.localhost:9000" {
		t.Fatalf("Address = %q, want %q", configuration.Address, "custom.localhost:9000")
	}
}

func writeExecutable(t *testing.T, directory string, name string) string {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	return path
}

func writeSettings(t *testing.T, path string, settings repositories.Settings) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	contents, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v, want nil", err)
	}

	contents = append(contents, '\n')
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
}
