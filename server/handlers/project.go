package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"web-terminal/project"
)

type Project struct {
	projects project.Store
}

type projectResponse struct {
	Name      string `json:"name"`
	GitBranch string `json:"gitBranch"`
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(projectResponse{
		Name:      projectName,
		GitBranch: currentGitBranch(projectPath),
	})
}

func currentGitBranch(projectPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "git", "-C", projectPath, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
