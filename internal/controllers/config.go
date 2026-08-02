package controllers

// TODO: Review properly

import (
	"net/http"

	"wade/internal/services/config"
)

type settingsService interface {
	Get() (config.Settings, error)
	Update(request config.Settings) (config.Settings, error)
	Reload() (config.Settings, error)
}

// ConfigHandler reads and writes editable WADE settings.
type ConfigHandler struct {
	settings settingsService
}

type configPayload struct {
	WorkspaceDirectories               []string       `json:"workspaceDirectories"`
	Shell                              string         `json:"shell"`
	Agents                             []config.Agent `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation bool           `json:"copyIgnoredFilesOnWorktreeCreation"`
	OpenWorktreesInNewTabs             bool           `json:"openWorktreesInNewTabs"`
	WorktreeCopyExcludes               []string       `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string         `json:"themeAccentColor"`
} // @name handlers.configPayload

// NewConfig creates a handler for reading and writing WADE settings.
func NewConfig(settings settingsService) ConfigHandler {
	return ConfigHandler{settings: settings}
}

// @Summary Get settings
// @ID getSettings
// @Tags Config
// @Produce json
// @Success 200 {object} configPayload
// @Failure 500 {object} errorResponse
// @Router /api/config [get]
func (h ConfigHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settings.Get()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to load config")
		return
	}

	writeJSON(w, http.StatusOK, configResponse(settings))
}

// @Summary Reload runtime config
// @ID reloadConfig
// @Tags Config
// @Produce json
// @Success 200 {object} configPayload
// @Failure 400 {object} errorResponse
// @Router /api/config/reload [post]
func (h ConfigHandler) ReloadConfig(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settings.Reload()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "unable to reload config")
		return
	}

	writeJSON(w, http.StatusOK, configResponse(settings))
}

// @Summary Update settings
// @ID updateSettings
// @Tags Config
// @Accept json
// @Produce json
// @Param request body configPayload true "Settings"
// @Success 200 {object} configPayload
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/config [post]
func (h ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var request configPayload
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config request")
		return
	}

	settings, err := h.settings.Update(config.Settings{
		WorkspaceDirectories:               request.WorkspaceDirectories,
		Shell:                              request.Shell,
		Agents:                             request.Agents,
		CopyIgnoredFilesOnWorktreeCreation: request.CopyIgnoredFilesOnWorktreeCreation,
		OpenWorktreesInNewTabs:             request.OpenWorktreesInNewTabs,
		WorktreeCopyExcludes:               request.WorktreeCopyExcludes,
		ThemeAccentColor:                   request.ThemeAccentColor,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, configResponse(settings))
}

func configResponse(settings config.Settings) configPayload {
	return configPayload{
		WorkspaceDirectories:               settings.WorkspaceDirectories,
		Shell:                              settings.Shell,
		Agents:                             settings.Agents,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		OpenWorktreesInNewTabs:             settings.OpenWorktreesInNewTabs,
		WorktreeCopyExcludes:               settings.WorktreeCopyExcludes,
		ThemeAccentColor:                   settings.ThemeAccentColor,
	}
}
