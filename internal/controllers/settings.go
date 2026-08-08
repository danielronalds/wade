package controllers

import (
	"net/http"
	"sync"

	"wade/internal/models/repositories"
	"wade/internal/models/settings"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
)

// Settings coordinates persisted settings and cross-Model runtime configuration.
type Settings struct {
	settings     SettingsModel
	workspaces   WorkspacesModel
	repositories RepositoriesModel
	terminals    TerminalsModel

	orchestrationMu sync.Mutex
}

// NewSettings constructs the Settings controller.
func NewSettings(settingsModel SettingsModel, workspaceModel WorkspacesModel, repositoryModel RepositoriesModel, terminalModel TerminalsModel) *Settings {
	return &Settings{
		settings:     settingsModel,
		workspaces:   workspaceModel,
		repositories: repositoryModel,
		terminals:    terminalModel,
	}
}

// @Summary Get settings
// @ID getSettings
// @Tags Settings
// @Produce json
// @Success 200 {object} settings.Settings
// @Failure 500 {object} Problem
// @Router /api/v1/settings [get]
func (controller *Settings) Get(response http.ResponseWriter, _ *http.Request) {
	currentSettings, err := controller.settings.Get()
	if err != nil {
		writeModelError(response, err, "Unable to load settings.")
		return
	}
	writeJSON(response, http.StatusOK, currentSettings)
}

// @Summary Update settings
// @ID updateSettings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body settings.Settings true "Complete settings"
// @Success 200 {object} settings.Settings
// @Failure 400 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/settings [put]
func (controller *Settings) Update(response http.ResponseWriter, request *http.Request) {
	var requestedSettings settings.Settings
	if err := decodeJSONBody(request, &requestedSettings); err != nil {
		writeProblem(response, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a complete settings representation.")
		return
	}

	controller.orchestrationMu.Lock()
	defer controller.orchestrationMu.Unlock()

	result, err := controller.settings.Update(requestedSettings)
	if err != nil {
		writeModelError(response, err, "Unable to update settings.")
		return
	}
	controller.applyRuntimeConfiguration(result.RuntimeConfiguration)
	writeJSON(response, http.StatusOK, result.Settings)
}

// @Summary Reload settings from disk
// @ID reloadSettings
// @Tags Settings
// @Produce json
// @Success 200 {object} settings.Settings
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/settings/reload [post]
func (controller *Settings) Reload(response http.ResponseWriter, _ *http.Request) {
	controller.orchestrationMu.Lock()
	defer controller.orchestrationMu.Unlock()

	result, err := controller.settings.Reload()
	if err != nil {
		writeModelError(response, err, "Unable to reload settings.")
		return
	}
	controller.applyRuntimeConfiguration(result.RuntimeConfiguration)
	writeJSON(response, http.StatusOK, result.Settings)
}

func (controller *Settings) applyRuntimeConfiguration(configuration settings.RuntimeConfiguration) {
	workspaceDirectories := make([]workspaces.WorkspaceDirectory, 0, len(configuration.WorkspaceDirectoryPaths))
	for index, path := range configuration.WorkspaceDirectoryPaths {
		setting := path
		if index < len(configuration.WorkspaceDirectorySettings) {
			setting = configuration.WorkspaceDirectorySettings[index]
		}
		workspaceDirectories = append(workspaceDirectories, workspaces.WorkspaceDirectory{Setting: setting, Path: path})
	}
	controller.workspaces.Configure(workspaces.Configuration{WorkspaceDirectories: workspaceDirectories})
	controller.repositories.Configure(repositories.Configuration{
		CopyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
	})

	agents := make([]terminals.Agent, 0, len(configuration.Agents))
	for _, agent := range configuration.Agents {
		agents = append(agents, terminals.Agent{Name: agent.Name, Command: agent.Command, Default: agent.Default})
	}
	controller.terminals.Configure(terminals.Configuration{Shell: configuration.Shell, Agents: agents})
}
