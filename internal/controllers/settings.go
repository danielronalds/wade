package controllers

import (
	"net/http"

	"wade/internal/services/config"
)

type settingsService interface {
	Get() (config.Settings, error)
	Update(request config.Settings) (config.Settings, error)
	Reload() (config.Settings, error)
}

type Settings struct {
	settings settingsService
}

type SettingsAgent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Default bool   `json:"default"`
} // @name Agent

type SettingsPayload struct {
	WorkspaceDirectories               []string        `json:"workspaceDirectories"`
	Shell                              string          `json:"shell"`
	Agents                             []SettingsAgent `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation bool            `json:"copyIgnoredFilesOnWorktreeCreation"`
	OpenWorktreesInNewTabs             bool            `json:"openWorktreesInNewTabs"`
	WorktreeCopyExcludes               []string        `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string          `json:"themeAccentColor" enums:"white,orange,purple"`
} // @name Settings

func NewSettings(settings settingsService) Settings {
	return Settings{settings: settings}
}

// @Summary Get settings
// @ID getSettings
// @Tags Settings
// @Produce json
// @Success 200 {object} SettingsPayload
// @Failure 500 {object} Problem
// @Router /api/v1/settings [get]
func (h Settings) Get(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settings.Get()
	if err != nil {
		writeServiceError(w, err, "Unable to load settings.")
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse(settings))
}

// @Summary Update settings
// @ID updateSettings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body SettingsPayload true "Complete settings"
// @Success 200 {object} SettingsPayload
// @Failure 400 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/settings [put]
func (h Settings) Update(w http.ResponseWriter, r *http.Request) {
	var request SettingsPayload
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a complete settings representation.")
		return
	}

	agents := make([]config.Agent, 0, len(request.Agents))
	for _, agent := range request.Agents {
		agents = append(agents, config.Agent{
			Name:    agent.Name,
			Command: agent.Command,
			Default: agent.Default,
		})
	}

	settings, err := h.settings.Update(config.Settings{
		WorkspaceDirectories:               request.WorkspaceDirectories,
		Shell:                              request.Shell,
		Agents:                             agents,
		CopyIgnoredFilesOnWorktreeCreation: request.CopyIgnoredFilesOnWorktreeCreation,
		OpenWorktreesInNewTabs:             request.OpenWorktreesInNewTabs,
		WorktreeCopyExcludes:               request.WorktreeCopyExcludes,
		ThemeAccentColor:                   request.ThemeAccentColor,
	})
	if err != nil {
		writeServiceError(w, err, "Unable to update settings.")
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse(settings))
}

// @Summary Reload settings from disk
// @ID reloadSettings
// @Tags Settings
// @Produce json
// @Success 200 {object} SettingsPayload
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/settings/reload [post]
func (h Settings) Reload(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settings.Reload()
	if err != nil {
		writeServiceError(w, err, "Unable to reload settings.")
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse(settings))
}

func settingsResponse(settings config.Settings) SettingsPayload {
	agents := make([]SettingsAgent, 0, len(settings.Agents))
	for _, agent := range settings.Agents {
		agents = append(agents, SettingsAgent{
			Name:    agent.Name,
			Command: agent.Command,
			Default: agent.Default,
		})
	}

	return SettingsPayload{
		WorkspaceDirectories:               settings.WorkspaceDirectories,
		Shell:                              settings.Shell,
		Agents:                             agents,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		OpenWorktreesInNewTabs:             settings.OpenWorktreesInNewTabs,
		WorktreeCopyExcludes:               settings.WorktreeCopyExcludes,
		ThemeAccentColor:                   settings.ThemeAccentColor,
	}
}
