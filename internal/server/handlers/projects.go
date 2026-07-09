package handlers

import (
	"net/http"

	"wade/internal/project"
)

type Projects struct {
	projects project.Store
}

type projectResponse struct {
	Name            string `json:"name"`
	GitBranch       string `json:"gitBranch"`
	LinearTicketURL string `json:"linearTicketUrl"`
	PullRequestURL  string `json:"pullRequestUrl"`
	GitHubURL       string `json:"githubUrl"`
}

type projectsResponse struct {
	Projects []string `json:"projects"`
}

func NewProjects(projects project.Store) Projects {
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
	projectPath, err := h.projects.Path(projectName)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	}

	metadata := project.Details(projectPath)

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
	projectNames, err := h.projects.Names()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list projects")
		return
	}

	writeJSON(w, http.StatusOK, projectsResponse{Projects: projectNames})
}
