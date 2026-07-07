package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"wade/internal/config"
)

// ConfigHandler reads and writes editable WADE settings.
type ConfigHandler struct{}

type configPayload struct {
	ProjectDirectories                 []string `json:"projectDirectories"`
	AgentPaneCommand                   string   `json:"agentPaneCommand"`
	CopyIgnoredFilesOnWorktreeCreation bool     `json:"copyIgnoredFilesOnWorktreeCreation"`
	WorktreeCopyExcludes               []string `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string   `json:"themeAccentColor"`
}

// NewConfig creates a handler for reading and writing WADE settings.
func NewConfig() ConfigHandler {
	return ConfigHandler{}
}

func (h ConfigHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	settings, err := config.LoadSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to load config")
		return
	}

	writeJSON(w, http.StatusOK, configPayload{
		ProjectDirectories:                 settings.ProjectDirectories,
		AgentPaneCommand:                   settings.AgentPaneCommand,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               settings.WorktreeCopyExcludes,
		ThemeAccentColor:                   settings.ThemeAccentColor,
	})
}

func (h ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var request configPayload
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config request")
		return
	}

	projectDirectories := trimProjectDirectories(request.ProjectDirectories)
	if err := config.ValidateProjectDirectories(projectDirectories); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	agentPaneCommand := strings.TrimSpace(request.AgentPaneCommand)
	if err := config.ValidateAgentPaneCommand(agentPaneCommand); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	worktreeCopyExcludes := trimWorktreeCopyExcludes(request.WorktreeCopyExcludes)
	if err := config.ValidateWorktreeCopyExcludes(worktreeCopyExcludes); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	themeAccentColor := config.NormaliseThemeAccentColor(request.ThemeAccentColor)
	if err := config.ValidateThemeAccentColor(themeAccentColor); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	settings, err := config.LoadSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to load config")
		return
	}

	settings.ProjectDirectories = projectDirectories
	settings.AgentPaneCommand = agentPaneCommand
	settings.CopyIgnoredFilesOnWorktreeCreation = request.CopyIgnoredFilesOnWorktreeCreation
	settings.WorktreeCopyExcludes = worktreeCopyExcludes
	settings.ThemeAccentColor = themeAccentColor
	if err := settings.Save(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to save config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func trimProjectDirectories(projectDirectories []string) []string {
	trimmed := make([]string, 0, len(projectDirectories))
	for _, projectDirectory := range projectDirectories {
		trimmed = append(trimmed, strings.TrimSpace(projectDirectory))
	}

	return trimmed
}

func trimWorktreeCopyExcludes(excludes []string) []string {
	trimmed := make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}
		trimmed = append(trimmed, exclude)
	}

	return trimmed
}
