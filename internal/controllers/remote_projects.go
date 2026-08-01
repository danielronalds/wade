package controllers

// TODO: Review properly

import (
	"context"
	"net/http"

	"wade/internal/services/remoteprojects"
	"wade/internal/services/workspaces"
)

type remoteProjectService interface {
	List(ctx context.Context, localProjectNames []string) ([]remoteprojects.Project, error)
	Clone(ctx context.Context, request remoteprojects.CloneRequest) (remoteprojects.ClonedProject, error)
}

type RemoteProjects struct {
	workspaces workspaces.Service
	remote     remoteProjectService
}

type remoteProjectsResponse struct {
	Projects []remoteprojects.Project `json:"projects"`
} // @name handlers.remoteProjectsResponse

type cloneRemoteProjectRequest struct {
	NameWithOwner  string `json:"nameWithOwner"`
	DirectoryIndex int    `json:"directoryIndex"`
} // @name handlers.cloneRemoteProjectRequest

type cloneRemoteProjectResponse struct {
	Project remoteprojects.ClonedProject `json:"project"`
} // @name handlers.cloneRemoteProjectResponse

func NewRemoteProjects(workspaceService workspaces.Service, remote remoteProjectService) RemoteProjects {
	return RemoteProjects{workspaces: workspaceService, remote: remote}
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
	workspaceSummaries, err := h.workspaces.List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list local projects")
		return
	}

	projects, err := h.remote.List(r.Context(), workspaceIDs(workspaceSummaries))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
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

	workspaceSummaries, err := h.workspaces.List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list local projects")
		return
	}

	project, err := h.remote.Clone(r.Context(), remoteprojects.CloneRequest{
		NameWithOwner:      request.NameWithOwner,
		ProjectDirectories: h.workspaces.Directories(),
		DirectoryIndex:     request.DirectoryIndex,
		LocalProjectNames:  workspaceIDs(workspaceSummaries),
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cloneRemoteProjectResponse{Project: project})
}

func workspaceIDs(workspaceSummaries []workspaces.WorkspaceSummary) []string {
	ids := make([]string, 0, len(workspaceSummaries))
	for _, workspace := range workspaceSummaries {
		ids = append(ids, workspace.ID)
	}

	return ids
}
