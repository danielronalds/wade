package handlers

import (
	"encoding/json"
	"net/http"

	"wade/project"
)

type Project struct {
	projects project.Store
}

type projectResponse struct {
	Name            string `json:"name"`
	GitBranch       string `json:"gitBranch"`
	LinearTicketURL string `json:"linearTicketUrl"`
	PullRequestURL  string `json:"pullRequestUrl"`
	GitHubURL       string `json:"githubUrl"`
}

func NewProject(projects project.Store) Project {
	return Project{projects: projects}
}

func (h Project) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Query().Get("project")
	projectPath, err := h.projects.Path(projectName)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	metadata := project.Details(projectPath)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projectResponse{
		Name:            projectName,
		GitBranch:       metadata.GitBranch,
		LinearTicketURL: metadata.LinearTicketURL,
		PullRequestURL:  metadata.PullRequestURL,
		GitHubURL:       metadata.GitHubURL,
	})
}
