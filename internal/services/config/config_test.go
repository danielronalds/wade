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

func TestLoadUsesConfiguredShellOverEnvironment(t *testing.T) {
	homeDir := t.TempDir()
	shell := writeExecutable(t, homeDir, "custom-shell")
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/zsh")

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		ProjectDirectories: []string{},
		Shell:              shell,
		Agents:             []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
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
		ProjectDirectories: []string{},
		Shell:              "",
		Agents:             []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
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

func TestLoadUsesAddressEnvironmentOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv(addressEnv, "custom.localhost:9000")

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		ProjectDirectories: []string{},
		Agents:             []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
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

func TestLoadDisablesSwaggerByDefault(t *testing.T) {
	configuration := loadConfigurationWithSwaggerEnvironment(t, "", "")

	if configuration.SwaggerEnabled {
		t.Fatal("SwaggerEnabled = true, want false")
	}
}

func TestLoadEnablesSwaggerInDevMode(t *testing.T) {
	configuration := loadConfigurationWithSwaggerEnvironment(t, "1", "")

	if !configuration.SwaggerEnabled {
		t.Fatal("SwaggerEnabled = false, want true")
	}
}

func TestLoadEnablesSwaggerWhenEnvironmentIsTrue(t *testing.T) {
	configuration := loadConfigurationWithSwaggerEnvironment(t, "", "true")

	if !configuration.SwaggerEnabled {
		t.Fatal("SwaggerEnabled = false, want true")
	}
}

func loadConfigurationWithSwaggerEnvironment(t *testing.T, devMode string, swaggerEnabled string) Config {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv(addressEnv, "")
	t.Setenv(devModeEnv, devMode)
	t.Setenv(swaggerEnabledEnv, swaggerEnabled)

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	settings := repositories.Settings{
		ProjectDirectories: []string{},
		Shell:              "",
		Agents:             []repositories.Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
	}
	writeSettings(t, path, settings)

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	return configuration
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
