package settings

import (
	"reflect"
	"testing"
)

func TestParseSettingsAppliesMissingFieldDefaults(t *testing.T) {
	persisted, err := parseSettings([]byte(`{"workspaceDirectories":["~/Code"]}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v", err)
	}
	if !reflect.DeepEqual(persisted.settings.Agents, defaultAgents) {
		t.Fatalf("Agents = %#v, want defaults", persisted.settings.Agents)
	}
	if persisted.settings.Shell != "" || persisted.settings.OpenWorktreesInNewTabs {
		t.Fatalf("default settings = %#v", persisted.settings)
	}
}

func TestParseSettingsAppliesConfiguredValues(t *testing.T) {
	persisted, err := parseSettings([]byte(`{
		"workspaceDirectories":["~/Code"],
		"shell":" /bin/zsh ",
		"agents":[{"name":"Custom","command":"custom-agent","default":true}],
		"copyIgnoredFilesOnWorktreeCreation":true,
		"openWorktreesInNewTabs":true,
		"worktreeCopyExcludes":["node_modules"],
		"themeAccentColor":"purple"
	}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v", err)
	}

	settings := persisted.settings
	if settings.Shell != "/bin/zsh" || !settings.CopyIgnoredFilesOnWorktreeCreation || !settings.OpenWorktreesInNewTabs {
		t.Fatalf("settings = %#v", settings)
	}
	if len(settings.Agents) != 1 || settings.Agents[0].Name != "Custom" {
		t.Fatalf("Agents = %#v", settings.Agents)
	}
	if !reflect.DeepEqual(settings.WorktreeCopyExcludes, []string{"node_modules"}) || settings.ThemeAccentColor != ThemeAccentColorPurple {
		t.Fatalf("settings = %#v", settings)
	}
}

func TestParseSettingsPreservesEmptyArrays(t *testing.T) {
	persisted, err := parseSettings([]byte(`{"workspaceDirectories":[],"agents":[],"worktreeCopyExcludes":[]}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v", err)
	}
	settings := cloneSettings(persisted.settings)
	if settings.WorkspaceDirectories == nil || settings.Agents == nil || settings.WorktreeCopyExcludes == nil {
		t.Fatalf("empty arrays became nil: %#v", settings)
	}
}

func TestParseSettingsFallsBackForInvalidThemeAccent(t *testing.T) {
	persisted, err := parseSettings([]byte(`{"themeAccentColor":"blue"}`))
	if err != nil {
		t.Fatalf("parseSettings() error = %v", err)
	}
	if persisted.settings.ThemeAccentColor != ThemeAccentColorWhite {
		t.Fatalf("ThemeAccentColor = %q", persisted.settings.ThemeAccentColor)
	}
}
