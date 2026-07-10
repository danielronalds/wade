package config

// TODO: Review properly

import (
	"strings"

	"wade/internal/repositories"
)

func LoadSettings() (Settings, error) {
	return repositories.LoadSettings()
}

func EnsureFile() (string, error) {
	return repositories.EnsureFile()
}

func ValidateShell(shell string) error {
	return repositories.ValidateShell(shell)
}

func ValidateAgents(agents []Agent) error {
	return repositories.ValidateAgents(agents)
}

func ValidateThemeAccentColor(color string) error {
	return repositories.ValidateThemeAccentColor(color)
}

func NormaliseThemeAccentColor(color string) string {
	return repositories.NormaliseThemeAccentColor(color)
}

func ValidateWorktreeCopyExcludes(excludes []string) error {
	return repositories.ValidateWorktreeCopyExcludes(excludes)
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
