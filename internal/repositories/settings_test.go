package repositories

// TODO: Review properly

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultSettingsIncludesAgents(t *testing.T) {
	settings := defaultSettings(filepath.Join(t.TempDir(), "config.json"))

	if !reflect.DeepEqual(settings.Agents, defaultAgents) {
		t.Fatalf("Agents = %#v, want %#v", settings.Agents, defaultAgents)
	}
}

func TestDefaultSettingsUsesEnvironmentShell(t *testing.T) {
	settings := defaultSettings(filepath.Join(t.TempDir(), "config.json"))

	if settings.Shell != "" {
		t.Fatalf("Shell = %q, want empty", settings.Shell)
	}
}

func TestParseSettingsDefaultsAgentsWhenMissing(t *testing.T) {
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"]}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(settings.Agents, defaultAgents) {
		t.Fatalf("Agents = %#v, want %#v", settings.Agents, defaultAgents)
	}
}

func TestParseSettingsUsesConfiguredAgents(t *testing.T) {
	configuredAgents := []Agent{{Name: "Custom", Command: "custom-agent", Default: true}}
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"],"agents":[{"name":"Custom","command":"custom-agent","default":true}]}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(settings.Agents, configuredAgents) {
		t.Fatalf("Agents = %#v, want %#v", settings.Agents, configuredAgents)
	}
}

func TestParseSettingsUsesConfiguredShell(t *testing.T) {
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"],"shell":" /bin/zsh "}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if settings.Shell != "/bin/zsh" {
		t.Fatalf("Shell = %q, want %q", settings.Shell, "/bin/zsh")
	}
}

func TestSettingsSaveWritesAgentsAndPreservesUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	settings := Settings{
		ProjectDirectories: []string{"~/Code"},
		Shell:              " /bin/zsh ",
		Agents:             []Agent{{Name: "Custom", Command: "custom-agent", Default: true}},
		path:               path,
		raw: map[string]json.RawMessage{
			"agentCommand":     json.RawMessage(`"legacy-agent"`),
			"agentPaneCommand": json.RawMessage(`"pane-agent"`),
			"theme":            json.RawMessage(`"dark"`),
		},
	}

	if err := settings.Save(); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}

	var saved map[string]any
	if err := json.Unmarshal(contents, &saved); err != nil {
		t.Fatalf("Unmarshal() error = %v, want nil", err)
	}

	agents, ok := saved["agents"].([]any)
	if !ok || len(agents) != 1 {
		t.Fatalf("agents = %#v, want one agent", saved["agents"])
	}

	agent, ok := agents[0].(map[string]any)
	if !ok {
		t.Fatalf("agents[0] = %#v, want object", agents[0])
	}
	if agent["name"] != "Custom" || agent["command"] != "custom-agent" || agent["default"] != true {
		t.Fatalf("agents[0] = %#v, want Custom custom-agent default", agent)
	}

	if saved["shell"] != "/bin/zsh" {
		t.Fatalf("shell = %#v, want %q", saved["shell"], "/bin/zsh")
	}

	if _, ok := saved["agentCommand"]; ok {
		t.Fatalf("agentCommand was preserved, want it removed")
	}
	if _, ok := saved["agentPaneCommand"]; ok {
		t.Fatalf("agentPaneCommand was preserved, want it removed")
	}

	if saved["theme"] != "dark" {
		t.Fatalf("theme = %#v, want %q", saved["theme"], "dark")
	}
}

func TestValidateShellAllowsEmptyShell(t *testing.T) {
	if err := ValidateShell(""); err != nil {
		t.Fatalf("ValidateShell() error = %v, want nil", err)
	}
}

func TestValidateShellAllowsAbsoluteExecutableShell(t *testing.T) {
	shell := writeExecutable(t, t.TempDir(), "custom-shell")

	if err := ValidateShell(shell); err != nil {
		t.Fatalf("ValidateShell() error = %v, want nil", err)
	}
}

func TestValidateShellAllowsCommandOnPath(t *testing.T) {
	directory := t.TempDir()
	writeExecutable(t, directory, "custom-shell")
	t.Setenv("PATH", directory)

	if err := ValidateShell("custom-shell"); err != nil {
		t.Fatalf("ValidateShell() error = %v, want nil", err)
	}
}

func TestValidateShellRejectsShellWithArguments(t *testing.T) {
	if err := ValidateShell("/bin/zsh -l"); err == nil {
		t.Fatal("ValidateShell() error = nil, want error")
	}
}

func TestValidateShellRejectsMissingShell(t *testing.T) {
	if err := ValidateShell(filepath.Join(t.TempDir(), "missing-shell")); err == nil {
		t.Fatal("ValidateShell() error = nil, want error")
	}
}

func TestValidateAgentsRejectsMissingAgents(t *testing.T) {
	if err := ValidateAgents(nil); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestValidateAgentsRejectsEmptyNames(t *testing.T) {
	if err := ValidateAgents([]Agent{{Name: "   ", Command: "pi -c", Default: true}}); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestValidateAgentsRejectsEmptyCommands(t *testing.T) {
	if err := ValidateAgents([]Agent{{Name: "Pi", Command: "   ", Default: true}}); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestValidateAgentsRejectsCaseInsensitiveDuplicateNames(t *testing.T) {
	agents := []Agent{
		{Name: "Pi", Command: "pi -c", Default: true},
		{Name: "pi", Command: "pi -c"},
	}

	if err := ValidateAgents(agents); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestValidateAgentsRejectsMissingDefaultAgent(t *testing.T) {
	agents := []Agent{{Name: "Pi", Command: "pi -c"}}

	if err := ValidateAgents(agents); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestValidateAgentsRejectsMultipleDefaultAgents(t *testing.T) {
	agents := []Agent{
		{Name: "Pi", Command: "pi -c", Default: true},
		{Name: "Claude", Command: "claude", Default: true},
	}

	if err := ValidateAgents(agents); err == nil {
		t.Fatal("ValidateAgents() error = nil, want error")
	}
}

func TestEnsureFileCreatesDefaultConfigWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	path, err := EnsureFile()
	if err != nil {
		t.Fatalf("EnsureFile() error = %v, want nil", err)
	}

	expectedPath := filepath.Join(homeDir, ".config", "wade", "config.json")
	if path != expectedPath {
		t.Fatalf("EnsureFile() path = %q, want %q", path, expectedPath)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(settings.Agents, defaultAgents) {
		t.Fatalf("Agents = %#v, want %#v", settings.Agents, defaultAgents)
	}
}

func TestEnsureFileDoesNotParseExistingConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	path := filepath.Join(homeDir, ".config", "wade", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	actualPath, err := EnsureFile()
	if err != nil {
		t.Fatalf("EnsureFile() error = %v, want nil", err)
	}

	if actualPath != path {
		t.Fatalf("EnsureFile() path = %q, want %q", actualPath, path)
	}
}

func writeExecutable(t *testing.T, directory string, name string) string {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	return path
}
