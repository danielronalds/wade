package config

import (
	"strings"

	"wade/internal/repositories"
)

type SettingsRepository interface {
	Load() (Settings, error)
	Save(settings Settings) error
}

type RuntimeConfigApplier interface {
	ApplyConfig(configuration Config)
}

type Service struct {
	settings SettingsRepository
	runtime  RuntimeConfigApplier
}

func NewService(settings SettingsRepository, runtime RuntimeConfigApplier) Service {
	return Service{settings: settings, runtime: runtime}
}

func (s Service) Get() (Settings, error) {
	return s.settings.Load()
}

func (s Service) Update(request Settings) (Settings, error) {
	normalised, err := normaliseAndValidateSettings(request)
	if err != nil {
		return Settings{}, InvalidSettingsError{Err: err}
	}
	configuration, err := resolveRuntimeConfig(normalised)
	if err != nil {
		return Settings{}, InvalidSettingsError{Err: err}
	}

	persisted, err := s.settings.Load()
	if err != nil {
		return Settings{}, err
	}
	persisted.WorkspaceDirectories = normalised.WorkspaceDirectories
	persisted.Shell = normalised.Shell
	persisted.Agents = normalised.Agents
	persisted.CopyIgnoredFilesOnWorktreeCreation = normalised.CopyIgnoredFilesOnWorktreeCreation
	persisted.OpenWorktreesInNewTabs = normalised.OpenWorktreesInNewTabs
	persisted.WorktreeCopyExcludes = normalised.WorktreeCopyExcludes
	persisted.ThemeAccentColor = normalised.ThemeAccentColor

	if err := s.settings.Save(persisted); err != nil {
		return Settings{}, err
	}
	s.runtime.ApplyConfig(configuration)

	return normalised, nil
}

func (s Service) Reload() (Settings, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return Settings{}, err
	}
	normalised, err := normaliseAndValidateSettings(settings)
	if err != nil {
		return Settings{}, InvalidSettingsError{Err: err}
	}
	configuration, err := resolveRuntimeConfig(normalised)
	if err != nil {
		return Settings{}, InvalidSettingsError{Err: err}
	}

	s.runtime.ApplyConfig(configuration)
	return normalised, nil
}

func normaliseAndValidateSettings(settings Settings) (Settings, error) {
	settings.WorkspaceDirectories = trimWorkspaceDirectories(settings.WorkspaceDirectories)
	if err := ValidateWorkspaceDirectories(settings.WorkspaceDirectories); err != nil {
		return Settings{}, err
	}

	settings.Shell = strings.TrimSpace(settings.Shell)
	if err := repositories.ValidateShell(settings.Shell); err != nil {
		return Settings{}, err
	}

	settings.Agents = trimAgents(settings.Agents)
	if err := repositories.ValidateAgents(settings.Agents); err != nil {
		return Settings{}, err
	}

	settings.WorktreeCopyExcludes = trimStrings(settings.WorktreeCopyExcludes)
	if err := repositories.ValidateWorktreeCopyExcludes(settings.WorktreeCopyExcludes); err != nil {
		return Settings{}, err
	}

	settings.ThemeAccentColor = repositories.NormaliseThemeAccentColor(settings.ThemeAccentColor)
	if err := repositories.ValidateThemeAccentColor(settings.ThemeAccentColor); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func trimWorkspaceDirectories(workspaceDirectories []string) []string {
	trimmed := make([]string, 0, len(workspaceDirectories))
	for _, workspaceDirectory := range workspaceDirectories {
		trimmed = append(trimmed, strings.TrimSpace(workspaceDirectory))
	}

	return trimmed
}
