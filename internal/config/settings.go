package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var defaultAgents = []Agent{
	{Name: "Pi", Command: "pi -c", Default: true},
	{Name: "Claude", Command: "claude"},
}

const (
	ThemeAccentColorWhite  = "white"
	ThemeAccentColorOrange = "orange"
	ThemeAccentColorPurple = "purple"
)

type Agent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Default bool   `json:"default"`
}

// Settings is the editable user configuration stored on disk.
type Settings struct {
	ProjectDirectories                 []string `json:"projectDirectories"`
	Shell                              string   `json:"shell"`
	Agents                             []Agent  `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation bool     `json:"copyIgnoredFilesOnWorktreeCreation"`
	WorktreeCopyExcludes               []string `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string   `json:"themeAccentColor"`
	path                               string
	raw                                map[string]json.RawMessage
}

type settingsFile struct {
	ProjectDirectories                 *[]string `json:"projectDirectories"`
	Shell                              *string   `json:"shell"`
	Agents                             *[]Agent  `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation *bool     `json:"copyIgnoredFilesOnWorktreeCreation"`
	WorktreeCopyExcludes               *[]string `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   *string   `json:"themeAccentColor"`
}

// LoadSettings reads the settings file, creating it with defaults when missing.
func LoadSettings() (Settings, error) {
	path, err := EnsureFile()
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

func ValidateShell(shell string) error {
	_, err := resolveConfiguredShell(shell)
	return err
}

func resolveConfiguredShell(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return "", nil
	}

	if len(strings.Fields(shell)) != 1 {
		return "", errors.New("shell must be a program path or command without arguments")
	}

	shell = expandHomePath(shell)
	if filepath.IsAbs(shell) {
		return executablePath(shell)
	}

	if strings.ContainsRune(shell, filepath.Separator) {
		return "", fmt.Errorf("shell %q must be an absolute path or command on PATH", shell)
	}

	path, err := exec.LookPath(shell)
	if err != nil {
		return "", fmt.Errorf("shell %q was not found on PATH", shell)
	}

	return path, nil
}

func expandHomePath(path string) string {
	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		return homeDir
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	}

	return path
}

func executablePath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("shell %q is not executable", path)
	}

	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("shell %q is not executable", path)
	}

	return filepath.Clean(path), nil
}

func ValidateAgents(agents []Agent) error {
	if len(agents) == 0 {
		return errors.New("at least one agent is required")
	}

	defaultAgents := 0
	seen := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		name := strings.TrimSpace(agent.Name)
		command := strings.TrimSpace(agent.Command)
		if name == "" {
			return errors.New("agent name cannot be empty")
		}
		if command == "" {
			return errors.New("agent command cannot be empty")
		}
		if agent.Default {
			defaultAgents += 1
		}

		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("agent name %q is configured more than once", name)
		}
		seen[key] = struct{}{}
	}

	if defaultAgents != 1 {
		return errors.New("exactly one default agent is required")
	}

	return nil
}

func ValidateThemeAccentColor(color string) error {
	switch color {
	case ThemeAccentColorWhite, ThemeAccentColorOrange, ThemeAccentColorPurple:
		return nil
	default:
		return fmt.Errorf("invalid theme accent color %q", color)
	}
}

func NormaliseThemeAccentColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return ThemeAccentColorWhite
	}

	return color
}

func ValidateWorktreeCopyExcludes(excludes []string) error {
	for _, pattern := range excludes {
		if !doublestar.ValidatePattern(pattern) {
			return fmt.Errorf("invalid worktree copy exclude pattern %q", pattern)
		}
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

func EnsureFile() (string, error) {
	path, err := FilePath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("checking config file: %w", err)
	}

	if err := defaultSettings(path).Save(); err != nil {
		return "", err
	}

	return path, nil
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

	shell, err := json.Marshal(strings.TrimSpace(s.Shell))
	if err != nil {
		return fmt.Errorf("encoding shell: %w", err)
	}

	agents, err := json.Marshal(s.Agents)
	if err != nil {
		return fmt.Errorf("encoding agents: %w", err)
	}

	copyIgnoredFilesOnWorktreeCreation, err := json.Marshal(s.CopyIgnoredFilesOnWorktreeCreation)
	if err != nil {
		return fmt.Errorf("encoding worktree copy setting: %w", err)
	}

	worktreeCopyExcludes, err := json.Marshal(s.WorktreeCopyExcludes)
	if err != nil {
		return fmt.Errorf("encoding worktree copy excludes: %w", err)
	}

	themeAccentColor, err := json.Marshal(NormaliseThemeAccentColor(s.ThemeAccentColor))
	if err != nil {
		return fmt.Errorf("encoding theme accent color: %w", err)
	}

	delete(raw, "agentCommand")
	delete(raw, "agentPaneCommand")
	raw["projectDirectories"] = projectDirectories
	raw["shell"] = shell
	raw["agents"] = agents
	raw["copyIgnoredFilesOnWorktreeCreation"] = copyIgnoredFilesOnWorktreeCreation
	raw["worktreeCopyExcludes"] = worktreeCopyExcludes
	raw["themeAccentColor"] = themeAccentColor

	return writeJSON(s.path, raw)
}

// defaultSettings creates the built-in settings for a first run.
func defaultSettings(path string) Settings {
	return Settings{
		ProjectDirectories:                 []string{"~/Personal", "~/Work"},
		Shell:                              "",
		Agents:                             cloneAgents(defaultAgents),
		CopyIgnoredFilesOnWorktreeCreation: false,
		WorktreeCopyExcludes:               []string{},
		ThemeAccentColor:                   ThemeAccentColorWhite,
		path:                               path,
		raw:                                make(map[string]json.RawMessage),
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
	if file.Shell != nil {
		settings.Shell = strings.TrimSpace(*file.Shell)
	}
	if file.Agents != nil {
		settings.Agents = cloneAgents(*file.Agents)
	}
	if file.CopyIgnoredFilesOnWorktreeCreation != nil {
		settings.CopyIgnoredFilesOnWorktreeCreation = *file.CopyIgnoredFilesOnWorktreeCreation
	}
	if file.WorktreeCopyExcludes != nil {
		settings.WorktreeCopyExcludes = *file.WorktreeCopyExcludes
	}
	if file.ThemeAccentColor != nil {
		themeAccentColor := NormaliseThemeAccentColor(*file.ThemeAccentColor)
		if ValidateThemeAccentColor(themeAccentColor) == nil {
			settings.ThemeAccentColor = themeAccentColor
		}
	}

	return settings, nil
}

func trimAgents(agents []Agent) []Agent {
	trimmed := make([]Agent, 0, len(agents))
	for _, agent := range agents {
		trimmed = append(trimmed, Agent{
			Name:    strings.TrimSpace(agent.Name),
			Command: strings.TrimSpace(agent.Command),
			Default: agent.Default,
		})
	}
	return trimmed
}

func trimStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func cloneAgents(agents []Agent) []Agent {
	cloned := make([]Agent, len(agents))
	copy(cloned, agents)
	return cloned
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
