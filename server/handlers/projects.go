package handlers

import (
	"encoding/json"
	"net/http"

	"wade/project"
)

type Projects struct {
	projects project.Store
}

type projectsResponse struct {
	Projects []string `json:"projects"`
}

func NewProjects(projects project.Store) Projects {
	return Projects{projects: projects}
}

func (h Projects) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectNames, err := h.projects.Names()
	if err != nil {
		http.Error(w, "unable to list projects", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projectsResponse{Projects: projectNames})
}
