package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettingsIncludesAgentPaneCommand(t *testing.T) {
	settings := defaultSettings(filepath.Join(t.TempDir(), "config.json"))

	if settings.AgentPaneCommand != defaultAgentPaneCommand {
		t.Fatalf("AgentPaneCommand = %q, want %q", settings.AgentPaneCommand, defaultAgentPaneCommand)
	}
}

func TestParseSettingsDefaultsAgentPaneCommandWhenMissing(t *testing.T) {
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"]}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if settings.AgentPaneCommand != defaultAgentPaneCommand {
		t.Fatalf("AgentPaneCommand = %q, want %q", settings.AgentPaneCommand, defaultAgentPaneCommand)
	}
}

func TestParseSettingsUsesConfiguredAgentPaneCommand(t *testing.T) {
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"],"agentPaneCommand":"custom-agent"}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if settings.AgentPaneCommand != "custom-agent" {
		t.Fatalf("AgentPaneCommand = %q, want %q", settings.AgentPaneCommand, "custom-agent")
	}
}

func TestParseSettingsUsesLegacyAgentCommand(t *testing.T) {
	settings, err := parseSettings("config.json", []byte(`{"projectDirectories":["~/Code"],"agentCommand":"legacy-agent"}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v, want nil", err)
	}

	if settings.AgentPaneCommand != "legacy-agent" {
		t.Fatalf("AgentPaneCommand = %q, want %q", settings.AgentPaneCommand, "legacy-agent")
	}
}

func TestSettingsSaveWritesAgentPaneCommandAndPreservesUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	settings := Settings{
		ProjectDirectories: []string{"~/Code"},
		AgentPaneCommand:   "custom-agent",
		path:               path,
		raw: map[string]json.RawMessage{
			"agentCommand": json.RawMessage(`"legacy-agent"`),
			"theme":        json.RawMessage(`"dark"`),
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

	if saved["agentPaneCommand"] != "custom-agent" {
		t.Fatalf("agentPaneCommand = %#v, want %q", saved["agentPaneCommand"], "custom-agent")
	}

	if _, ok := saved["agentCommand"]; ok {
		t.Fatalf("agentCommand was preserved, want it removed")
	}

	if saved["theme"] != "dark" {
		t.Fatalf("theme = %#v, want %q", saved["theme"], "dark")
	}
}

func TestValidateAgentPaneCommandRejectsEmptyCommands(t *testing.T) {
	if err := ValidateAgentPaneCommand("   "); err == nil {
		t.Fatal("ValidateAgentPaneCommand() error = nil, want error")
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

	if settings.AgentPaneCommand != defaultAgentPaneCommand {
		t.Fatalf("AgentPaneCommand = %q, want %q", settings.AgentPaneCommand, defaultAgentPaneCommand)
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
