package settings

import (
	"encoding/json"
	"fmt"
	"strings"
)

var defaultAgents = []Agent{
	{Name: "Pi", Command: "pi -c", Default: true},
	{Name: "Claude", Command: "claude"},
}

type persistedSettings struct {
	settings Settings
	raw      map[string]json.RawMessage
}

type settingsFile struct {
	WorkspaceDirectories               *[]string       `json:"workspaceDirectories"`
	LegacyProjectDirectories           *[]string       `json:"projectDirectories"`
	Shell                              *string         `json:"shell"`
	Agents                             *[]Agent        `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation *bool           `json:"copyIgnoredFilesOnWorktreeCreation"`
	OpenWorktreesInNewTabs             *bool           `json:"openWorktreesInNewTabs"`
	WorktreeCopyExcludes               *[]string       `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   *string         `json:"themeAccentColor"`
	Linear                             *LinearSettings `json:"linear"`
}

func defaultSettings() Settings {
	return Settings{
		WorkspaceDirectories:               []string{"~/Personal", "~/Work"},
		Shell:                              "",
		Agents:                             cloneAgents(defaultAgents),
		CopyIgnoredFilesOnWorktreeCreation: false,
		OpenWorktreesInNewTabs:             false,
		WorktreeCopyExcludes:               []string{},
		ThemeAccentColor:                   ThemeAccentColorWhite,
		Linear:                             LinearSettings{Enabled: false, Workspace: ""},
	}
}

func parseSettings(contents []byte) (persistedSettings, error) {
	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(contents, &raw); err != nil {
		return persistedSettings{}, fmt.Errorf("reading config file: %w", err)
	}

	file := settingsFile{}
	if err := json.Unmarshal(contents, &file); err != nil {
		return persistedSettings{}, fmt.Errorf("reading config file: %w", err)
	}

	settings := defaultSettings()
	if file.WorkspaceDirectories != nil {
		settings.WorkspaceDirectories = cloneStrings(*file.WorkspaceDirectories)
	} else if file.LegacyProjectDirectories != nil {
		settings.WorkspaceDirectories = cloneStrings(*file.LegacyProjectDirectories)
	}
	if file.Shell != nil {
		settings.Shell = strings.TrimSpace(*file.Shell)
	}
	if file.Agents != nil {
		settings.Agents = cloneAgents(*file.Agents)
	}
	if file.CopyIgnoredFilesOnWorktreeCreation != nil {
		settings.CopyIgnoredFilesOnWorktreeCreation = *file.CopyIgnoredFilesOnWorktreeCreation
	}
	if file.OpenWorktreesInNewTabs != nil {
		settings.OpenWorktreesInNewTabs = *file.OpenWorktreesInNewTabs
	}
	if file.WorktreeCopyExcludes != nil {
		settings.WorktreeCopyExcludes = cloneStrings(*file.WorktreeCopyExcludes)
	}
	if file.ThemeAccentColor != nil {
		themeAccentColor := normaliseThemeAccentColor(*file.ThemeAccentColor)
		if validateThemeAccentColor(themeAccentColor) == nil {
			settings.ThemeAccentColor = themeAccentColor
		}
	}
	if file.Linear != nil {
		settings.Linear = LinearSettings{
			Enabled:   file.Linear.Enabled,
			Workspace: strings.TrimSpace(file.Linear.Workspace),
		}
	}

	return persistedSettings{settings: settings, raw: cloneRawSettings(raw)}, nil
}

func encodeSettings(settings Settings, existing map[string]json.RawMessage) ([]byte, error) {
	raw := cloneRawSettings(existing)
	delete(raw, "agentCommand")
	delete(raw, "agentPaneCommand")
	delete(raw, "projectDirectories")

	knownSettings := []struct {
		name  string
		value any
	}{
		{name: "workspaceDirectories", value: settings.WorkspaceDirectories},
		{name: "shell", value: strings.TrimSpace(settings.Shell)},
		{name: "agents", value: settings.Agents},
		{name: "copyIgnoredFilesOnWorktreeCreation", value: settings.CopyIgnoredFilesOnWorktreeCreation},
		{name: "openWorktreesInNewTabs", value: settings.OpenWorktreesInNewTabs},
		{name: "worktreeCopyExcludes", value: settings.WorktreeCopyExcludes},
		{name: "themeAccentColor", value: normaliseThemeAccentColor(settings.ThemeAccentColor)},
		{name: "linear", value: LinearSettings{
			Enabled:   settings.Linear.Enabled,
			Workspace: strings.TrimSpace(settings.Linear.Workspace),
		}},
	}
	for _, setting := range knownSettings {
		value, err := json.Marshal(setting.value)
		if err != nil {
			return nil, fmt.Errorf("encoding %s setting: %w", setting.name, err)
		}
		raw[setting.name] = value
	}

	contents, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding config file: %w", err)
	}
	return append(contents, '\n'), nil
}

func cloneRawSettings(raw map[string]json.RawMessage) map[string]json.RawMessage {
	cloned := make(map[string]json.RawMessage, len(raw)+1)
	for key, value := range raw {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}
