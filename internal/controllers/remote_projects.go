package controllers

// TODO: Review properly

import (
	"context"
	"net/http"
	"path/filepath"

	"wade/internal/services/remoterepositories"
	"wade/internal/services/workspaces"
)

type remoteRepositoryService interface {
	List(ctx context.Context) ([]remoterepositories.RemoteRepository, error)
	Clone(ctx context.Context, request remoterepositories.CloneRequest) (workspaces.Workspace, error)
	WorkspaceDirectories() []remoterepositories.WorkspaceDirectory
}

type RemoteProjects struct {
	remoteRepositories remoteRepositoryService
}

type legacyRemoteProject struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
	IsLocal       bool   `json:"isLocal"`
	LocalName     string `json:"localName"`
} // @name remote.Project

type remoteProjectsResponse struct {
	Projects []legacyRemoteProject `json:"projects"`
} // @name handlers.remoteProjectsResponse

type cloneRemoteProjectRequest struct {
	NameWithOwner  string `json:"nameWithOwner"`
	DirectoryIndex int    `json:"directoryIndex"`
} // @name handlers.cloneRemoteProjectRequest

type legacyClonedProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
} // @name remote.ClonedProject

type cloneRemoteProjectResponse struct {
	Project legacyClonedProject `json:"project"`
} // @name handlers.cloneRemoteProjectResponse

func NewRemoteProjects(remoteRepositories remoteRepositoryService) RemoteProjects {
	return RemoteProjects{remoteRepositories: remoteRepositories}
}

// @Summary List remote projects
// @ID listRemoteProjects
// @Tags Remote projects
// @Produce json
// @Success 200 {object} remoteProjectsResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/remote-projects [get]
func (h RemoteProjects) List(w http.ResponseWriter, r *http.Request) {
	remoteRepositories, err := h.remoteRepositories.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	projects := make([]legacyRemoteProject, 0, len(remoteRepositories))
	for _, repository := range remoteRepositories {
		localName := ""
		if len(repository.LocalWorkspaceIDs) > 0 {
			localName = repository.LocalWorkspaceIDs[0]
		}

		projects = append(projects, legacyRemoteProject{
			Name:          repository.Name,
			NameWithOwner: repository.ID,
			URL:           repository.WebURL,
			SSHURL:        repository.CloneURL,
			IsLocal:       len(repository.LocalWorkspaceIDs) > 0,
			LocalName:     localName,
		})
	}

	writeJSON(w, http.StatusOK, remoteProjectsResponse{Projects: projects})
}

// @Summary Clone remote project
// @ID cloneRemoteProject
// @Tags Remote projects
// @Accept json
// @Produce json
// @Param request body cloneRemoteProjectRequest true "Remote project clone request"
// @Success 200 {object} cloneRemoteProjectResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/remote-projects/clone [post]
func (h RemoteProjects) Clone(w http.ResponseWriter, r *http.Request) {
	var request cloneRemoteProjectRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid remote project request")
		return
	}

	workspaceDirectories := h.remoteRepositories.WorkspaceDirectories()
	if request.DirectoryIndex < 0 || request.DirectoryIndex >= len(workspaceDirectories) {
		writeJSONError(w, http.StatusBadRequest, "invalid project directory")
		return
	}
	workspaceDirectory := workspaceDirectories[request.DirectoryIndex]

	workspace, err := h.remoteRepositories.Clone(r.Context(), remoterepositories.CloneRequest{
		RemoteRepositoryID: request.NameWithOwner,
		WorkspaceDirectory: workspaceDirectory.Setting,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cloneRemoteProjectResponse{Project: legacyClonedProject{
		Name: workspace.Name,
		Path: filepath.Join(workspaceDirectory.Path, workspace.ID),
	}})
}
