package settings

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// Model owns persisted settings, validation, and runtime configuration resolution.
type Model struct {
	files       FileSystem
	environment Environment

	mu sync.Mutex
}

// New constructs an application-scoped Settings Model.
func New(files FileSystem, environment Environment) *Model {
	return &Model{files: files, environment: environment}
}

// EnsureFile creates the settings file with defaults when missing and returns its location.
func (model *Model) EnsureFile() (string, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	homeDirectory, err := model.homeDirectory()
	if err != nil {
		return "", err
	}
	if err := model.ensureFile(homeDirectory); err != nil {
		return "", err
	}
	return model.files.SettingsFilePath(homeDirectory), nil
}

// Get returns detached settings loaded fresh from disk.
func (model *Model) Get() (Settings, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	homeDirectory, err := model.homeDirectory()
	if err != nil {
		return Settings{}, err
	}
	persisted, err := model.loadPersisted(homeDirectory)
	if err != nil {
		return Settings{}, err
	}
	return cloneSettings(persisted.settings), nil
}

// LoadRuntimeConfiguration resolves fresh persisted settings for server startup.
func (model *Model) LoadRuntimeConfiguration() (RuntimeConfiguration, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	homeDirectory, err := model.homeDirectory()
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	persisted, err := model.loadPersisted(homeDirectory)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	configuration, err := resolveRuntimeConfiguration(persisted.settings, homeDirectory, model.files, model.environment)
	if err != nil {
		return RuntimeConfiguration{}, InvalidSettingsError{Err: err}
	}
	return cloneRuntimeConfiguration(configuration), nil
}

// Update validates, persists, and resolves a complete settings representation.
func (model *Model) Update(request Settings) (UpdateResult, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	homeDirectory, err := model.homeDirectory()
	if err != nil {
		return UpdateResult{}, err
	}
	normalised, err := normaliseAndValidateSettings(request, homeDirectory, model.files, model.environment)
	if err != nil {
		return UpdateResult{}, InvalidSettingsError{Err: err}
	}
	configuration, err := resolveRuntimeConfiguration(normalised, homeDirectory, model.files, model.environment)
	if err != nil {
		return UpdateResult{}, InvalidSettingsError{Err: err}
	}

	persisted, err := model.loadPersisted(homeDirectory)
	if err != nil {
		return UpdateResult{}, err
	}
	persisted.settings = cloneSettings(normalised)
	if err := model.writePersisted(homeDirectory, persisted); err != nil {
		return UpdateResult{}, err
	}

	return cloneUpdateResult(UpdateResult{Settings: normalised, RuntimeConfiguration: configuration}), nil
}

// Reload validates fresh out-of-band settings and resolves their runtime configuration.
func (model *Model) Reload() (UpdateResult, error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	homeDirectory, err := model.homeDirectory()
	if err != nil {
		return UpdateResult{}, err
	}
	persisted, err := model.loadPersisted(homeDirectory)
	if err != nil {
		return UpdateResult{}, err
	}
	normalised, err := normaliseAndValidateSettings(persisted.settings, homeDirectory, model.files, model.environment)
	if err != nil {
		return UpdateResult{}, InvalidSettingsError{Err: err}
	}
	configuration, err := resolveRuntimeConfiguration(normalised, homeDirectory, model.files, model.environment)
	if err != nil {
		return UpdateResult{}, InvalidSettingsError{Err: err}
	}
	return cloneUpdateResult(UpdateResult{Settings: normalised, RuntimeConfiguration: configuration}), nil
}

func (model *Model) homeDirectory() (string, error) {
	homeDirectory, err := model.environment.HomeDirectory()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return homeDirectory, nil
}

func (model *Model) ensureFile(homeDirectory string) error {
	exists, err := model.files.SettingsFileExists(homeDirectory)
	if err != nil {
		return fmt.Errorf("checking config file: %w", err)
	}
	if exists {
		return nil
	}
	return model.writePersisted(homeDirectory, persistedSettings{settings: defaultSettings()})
}

func (model *Model) loadPersisted(homeDirectory string) (persistedSettings, error) {
	if err := model.ensureFile(homeDirectory); err != nil {
		return persistedSettings{}, err
	}

	contents, err := model.files.ReadSettingsFile(homeDirectory)
	if errors.Is(err, os.ErrNotExist) {
		defaults := persistedSettings{settings: defaultSettings()}
		if err := model.writePersisted(homeDirectory, defaults); err != nil {
			return persistedSettings{}, err
		}
		return defaults, nil
	}
	if err != nil {
		return persistedSettings{}, fmt.Errorf("reading config file: %w", err)
	}
	return parseSettings(contents)
}

func (model *Model) writePersisted(homeDirectory string, persisted persistedSettings) error {
	contents, err := encodeSettings(persisted.settings, persisted.raw)
	if err != nil {
		return err
	}
	if err := model.files.WriteSettingsFile(homeDirectory, contents); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
