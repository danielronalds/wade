package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Settings is the editable user configuration stored on disk.
type Settings struct {
	ProjectDirectories []string `json:"projectDirectories"`
	path               string
	raw                map[string]json.RawMessage
}

type settingsFile struct {
	ProjectDirectories *[]string `json:"projectDirectories"`
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

	raw["projectDirectories"] = projectDirectories

	return writeJSON(s.path, raw)
}

// defaultSettings creates the built-in settings for a first run.
func defaultSettings(path string) Settings {
	return Settings{
		ProjectDirectories: []string{"~/Personal", "~/Work"},
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
