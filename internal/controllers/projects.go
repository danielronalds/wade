package controllers

// TODO: Review properly

import (
	"net/http"

	"wade/internal/services/workspaces"
)

type Projects struct {
	workspaces workspaces.Service
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

func NewProjects(workspaceService workspaces.Service) Projects {
	return Projects{workspaces: workspaceService}
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
	workspaceID := r.URL.Query().Get("project")
	workspace, err := h.workspaces.Get(r.Context(), workspaceID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	}

	var branchName string
	if workspace.Branch != nil {
		branchName = workspace.Branch.Name
	}

	var issueURL string
	if workspace.Links.Issue != nil {
		issueURL = workspace.Links.Issue.URL
	}

	writeJSON(w, http.StatusOK, projectResponse{
		Name:            workspace.Name,
		GitBranch:       branchName,
		LinearTicketURL: issueURL,
		PullRequestURL:  referencedString(workspace.Links.PullRequest),
		GitHubURL:       referencedString(workspace.Links.Repository),
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
	workspaceSummaries, err := h.workspaces.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list projects")
		return
	}

	writeJSON(w, http.StatusOK, projectsResponse{Projects: workspaceIDs(workspaceSummaries)})
}

func workspaceIDs(workspaceSummaries []workspaces.WorkspaceSummary) []string {
	ids := make([]string, 0, len(workspaceSummaries))
	for _, workspace := range workspaceSummaries {
		ids = append(ids, workspace.ID)
	}

	return ids
}

func referencedString(reference *string) string {
	if reference == nil {
		return ""
	}

	return *reference
}
