package controllers

// TODO: Review properly

import (
	"net/http"

	projectservice "wade/internal/services/projects"
)

type Projects struct {
	projects projectservice.Service
}

type projectResponse struct {
	Name            string `json:"name"`
	GitBranch       string `json:"gitBranch"`
	LinearTicketURL string `json:"linearTicketUrl"`
	PullRequestURL  string `json:"pullRequestUrl"`
	GitHubURL       string `json:"githubUrl"`
} // @name handlers.projectResponse

type projectsResponse struct {
	Projects []string `json:"projects"`
} // @name handlers.projectsResponse

func NewProjects(projects projectservice.Service) Projects {
	return Projects{projects: projects}
}

// @Summary Get project details
// @ID getProjectDetails
// @Tags Projects
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} projectResponse
// @Failure 404 {object} errorResponse
// @Router /api/project [get]
func (h Projects) GetProjectDetails(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")
	metadata, err := h.projects.Details(r.Context(), projectName)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, projectResponse{
		Name:            projectName,
		GitBranch:       metadata.GitBranch,
		LinearTicketURL: metadata.LinearTicketURL,
		PullRequestURL:  metadata.PullRequestURL,
		GitHubURL:       metadata.GitHubURL,
	})
}

// @Summary List projects
// @ID listProjects
// @Tags Projects
// @Produce json
// @Success 200 {object} projectsResponse
// @Failure 500 {object} errorResponse
// @Router /api/projects [get]
func (h Projects) ListProjects(w http.ResponseWriter, r *http.Request) {
	projectNames, err := h.projects.List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list projects")
		return
	}

	writeJSON(w, http.StatusOK, projectsResponse{Projects: projectNames})
}
