package settings

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var linearWorkspacePattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

func normaliseAndValidateSettings(settings Settings, homeDirectory string, files FileSystem, environment Environment) (Settings, error) {
	settings = cloneSettings(settings)
	settings.WorkspaceDirectories = trimWorkspaceDirectories(settings.WorkspaceDirectories)
	if _, err := resolveWorkspaceDirectories(homeDirectory, settings.WorkspaceDirectories); err != nil {
		return Settings{}, err
	}

	settings.Shell = strings.TrimSpace(settings.Shell)
	if settings.Shell != "" {
		if _, err := resolveConfiguredShell(settings.Shell, homeDirectory, files, environment); err != nil {
			return Settings{}, err
		}
	}

	settings.Agents = trimAgents(settings.Agents)
	if err := validateAgents(settings.Agents); err != nil {
		return Settings{}, err
	}

	settings.WorktreeCopyExcludes = trimStrings(settings.WorktreeCopyExcludes)
	if err := validateWorktreeCopyExcludes(settings.WorktreeCopyExcludes); err != nil {
		return Settings{}, err
	}

	settings.ThemeAccentColor = normaliseThemeAccentColor(settings.ThemeAccentColor)
	if err := validateThemeAccentColor(settings.ThemeAccentColor); err != nil {
		return Settings{}, err
	}

	settings.Linear.Workspace = strings.TrimSpace(settings.Linear.Workspace)
	if err := validateLinearSettings(settings.Linear); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func resolveWorkspaceDirectories(homeDirectory string, directories []string) ([]string, error) {
	resolved := make([]string, 0, len(directories))
	for _, directory := range directories {
		path, err := resolveWorkspaceDirectory(homeDirectory, directory)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, path)
	}
	return resolved, nil
}

func resolveWorkspaceDirectory(homeDirectory string, directory string) (string, error) {
	if directory == "" {
		return "", errors.New("workspace directory cannot be empty")
	}
	if directory == "~" {
		return homeDirectory, nil
	}
	if strings.HasPrefix(directory, "~/") {
		return filepath.Join(homeDirectory, strings.TrimPrefix(directory, "~/")), nil
	}
	if filepath.IsAbs(directory) {
		return filepath.Clean(directory), nil
	}
	return "", fmt.Errorf("workspace directory %q must use ~ or an absolute path", directory)
}

func resolveConfiguredShell(shell string, homeDirectory string, files FileSystem, environment Environment) (string, error) {
	shell = strings.TrimSpace(shell)
	if len(strings.Fields(shell)) != 1 {
		return "", errors.New("shell must be a program path or command without arguments")
	}

	expandedShell := expandHomePath(shell, homeDirectory)
	if filepath.IsAbs(expandedShell) {
		executable, err := files.IsExecutableFile(expandedShell)
		if err != nil || !executable {
			return "", fmt.Errorf("shell %q is not executable", expandedShell)
		}
		return filepath.Clean(expandedShell), nil
	}
	if strings.ContainsRune(expandedShell, filepath.Separator) {
		return "", fmt.Errorf("shell %q must be an absolute path or command on PATH", expandedShell)
	}

	path, err := environment.LookPath(expandedShell)
	if err != nil {
		return "", fmt.Errorf("shell %q was not found on PATH", expandedShell)
	}
	return path, nil
}

func expandHomePath(path string, homeDirectory string) string {
	if path == "~" {
		return homeDirectory
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDirectory, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func validateAgents(agents []Agent) error {
	if len(agents) == 0 {
		return errors.New("at least one agent is required")
	}

	defaultAgentCount := 0
	seen := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		if strings.TrimSpace(agent.Name) == "" {
			return errors.New("agent name cannot be empty")
		}
		if strings.TrimSpace(agent.Command) == "" {
			return errors.New("agent command cannot be empty")
		}
		if agent.Default {
			defaultAgentCount++
		}

		key := strings.ToLower(strings.TrimSpace(agent.Name))
		if _, found := seen[key]; found {
			return fmt.Errorf("agent name %q is configured more than once", strings.TrimSpace(agent.Name))
		}
		seen[key] = struct{}{}
	}
	if defaultAgentCount != 1 {
		return errors.New("exactly one default agent is required")
	}
	return nil
}

func validateWorktreeCopyExcludes(excludes []string) error {
	for _, pattern := range excludes {
		if !doublestar.ValidatePattern(pattern) {
			return fmt.Errorf("invalid worktree copy exclude pattern %q", pattern)
		}
	}
	return nil
}

func validateLinearSettings(linear LinearSettings) error {
	if !linear.Enabled {
		return nil
	}
	if linear.Workspace == "" {
		return errors.New("linear workspace is required when the integration is enabled")
	}
	if linear.Workspace == "." || linear.Workspace == ".." {
		return fmt.Errorf("linear workspace %q is not a valid workspace slug", linear.Workspace)
	}
	if !linearWorkspacePattern.MatchString(linear.Workspace) {
		return fmt.Errorf("linear workspace %q contains unsupported characters", linear.Workspace)
	}
	return nil
}

func validateThemeAccentColor(color string) error {
	switch color {
	case ThemeAccentColorWhite, ThemeAccentColorOrange, ThemeAccentColorPurple:
		return nil
	default:
		return fmt.Errorf("invalid theme accent color %q", color)
	}
}

func normaliseThemeAccentColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return ThemeAccentColorWhite
	}
	return color
}

func trimWorkspaceDirectories(directories []string) []string {
	trimmed := make([]string, 0, len(directories))
	for _, directory := range directories {
		trimmed = append(trimmed, strings.TrimSpace(directory))
	}
	return trimmed
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
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	return trimmed
}
