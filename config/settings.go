package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultAgentPaneCommand = "pi -c"

// Settings is the editable user configuration stored on disk.
type Settings struct {
	ProjectDirectories []string `json:"projectDirectories"`
	AgentPaneCommand   string   `json:"agentPaneCommand"`
	path               string
	raw                map[string]json.RawMessage
}

type settingsFile struct {
	ProjectDirectories *[]string `json:"projectDirectories"`
	AgentPaneCommand   *string   `json:"agentPaneCommand"`
	LegacyAgentCommand *string   `json:"agentCommand"`
}

// LoadSettings reads the settings file, creating it with defaults when missing.
func LoadSettings() (Settings, error) {
	path, err := FilePath()
	if err != nil {
		return Settings{}, err
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			settings := defaultSettings(path)
			if err := settings.Save(); err != nil {
				return Settings{}, err
			}

			return settings, nil
		}

		return Settings{}, fmt.Errorf("reading config file: %w", err)
	}

	return parseSettings(path, contents)
}

// ValidateAgentPaneCommand checks that the configured agent pane command is usable.
func ValidateAgentPaneCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return errors.New("agent pane command cannot be empty")
	}

	return nil
}

// FilePath returns the current user's WADE settings file path.
func FilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	return configPath(homeDir), nil
}

// Save writes settings while preserving keys WADE does not understand yet.
func (s Settings) Save() error {
	if s.path == "" {
		return errors.New("settings path is not set")
	}

	raw := cloneRawSettings(s.raw)
	projectDirectories, err := json.Marshal(s.ProjectDirectories)
	if err != nil {
		return fmt.Errorf("encoding project directories: %w", err)
	}

	agentPaneCommand, err := json.Marshal(s.AgentPaneCommand)
	if err != nil {
		return fmt.Errorf("encoding agent pane command: %w", err)
	}

	delete(raw, "agentCommand")
	raw["projectDirectories"] = projectDirectories
	raw["agentPaneCommand"] = agentPaneCommand

	return writeJSON(s.path, raw)
}

// defaultSettings creates the built-in settings for a first run.
func defaultSettings(path string) Settings {
	return Settings{
		ProjectDirectories: []string{"~/Personal", "~/Work"},
		AgentPaneCommand:   defaultAgentPaneCommand,
		path:               path,
		raw:                make(map[string]json.RawMessage),
	}
}

// parseSettings decodes known settings and stores raw JSON for future writes.
func parseSettings(path string, contents []byte) (Settings, error) {
	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(contents, &raw); err != nil {
		return Settings{}, fmt.Errorf("reading config file: %w", err)
	}

	file := settingsFile{}
	if err := json.Unmarshal(contents, &file); err != nil {
		return Settings{}, fmt.Errorf("reading config file: %w", err)
	}

	settings := defaultSettings(path)
	settings.raw = raw
	if file.ProjectDirectories != nil {
		settings.ProjectDirectories = *file.ProjectDirectories
	}
	if file.AgentPaneCommand != nil {
		settings.AgentPaneCommand = *file.AgentPaneCommand
	} else if file.LegacyAgentCommand != nil {
		settings.AgentPaneCommand = *file.LegacyAgentCommand
	}

	return settings, nil
}

// cloneRawSettings copies raw JSON values so saves cannot mutate loaded state.
func cloneRawSettings(raw map[string]json.RawMessage) map[string]json.RawMessage {
	cloned := make(map[string]json.RawMessage, len(raw)+1)
	for key, value := range raw {
		clonedValue := make([]byte, len(value))
		copy(clonedValue, value)
		cloned[key] = clonedValue
	}

	return cloned
}

// writeJSON writes an indented JSON file, creating parent directories first.
func writeJSON(path string, value any) error {
	contents, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config file: %w", err)
	}

	contents = append(contents, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// configPath builds the settings path for an already-resolved home directory.
func configPath(homeDir string) string {
	return filepath.Join(homeDir, ".config", "wade", "config.json")
}
