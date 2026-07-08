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
	ProjectDirectories                 []string       `json:"projectDirectories"`
	Shell                              string         `json:"shell"`
	Agents                             []config.Agent `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation bool           `json:"copyIgnoredFilesOnWorktreeCreation"`
	WorktreeCopyExcludes               []string       `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string         `json:"themeAccentColor"`
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
		Shell:                              settings.Shell,
		Agents:                             settings.Agents,
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

	shell := strings.TrimSpace(request.Shell)
	if err := config.ValidateShell(shell); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	agents := trimAgents(request.Agents)
	if err := config.ValidateAgents(agents); err != nil {
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
	settings.Shell = shell
	settings.Agents = agents
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

func trimAgents(agents []config.Agent) []config.Agent {
	trimmed := make([]config.Agent, 0, len(agents))
	for _, agent := range agents {
		trimmed = append(trimmed, config.Agent{
			Name:    strings.TrimSpace(agent.Name),
			Command: strings.TrimSpace(agent.Command),
			Default: agent.Default,
		})
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
