package controllers

// TODO: Review properly

import (
	"context"
	"net/http"

	projectservice "wade/internal/services/projects"
	"wade/internal/services/remoteprojects"
)

type remoteProjectService interface {
	List(ctx context.Context, localProjectNames []string) ([]remoteprojects.Project, error)
	Clone(ctx context.Context, request remoteprojects.CloneRequest) (remoteprojects.ClonedProject, error)
}

type RemoteProjects struct {
	projects projectservice.Service
	remote   remoteProjectService
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

func NewRemoteProjects(projects projectservice.Service, remote remoteProjectService) RemoteProjects {
	return RemoteProjects{projects: projects, remote: remote}
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
	localProjectNames, err := h.projects.List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list local projects")
		return
	}

	projects, err := h.remote.List(r.Context(), localProjectNames)
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

	localProjectNames, err := h.projects.List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list local projects")
		return
	}

	project, err := h.remote.Clone(r.Context(), remoteprojects.CloneRequest{
		NameWithOwner:      request.NameWithOwner,
		ProjectDirectories: h.projects.Directories(),
		DirectoryIndex:     request.DirectoryIndex,
		LocalProjectNames:  localProjectNames,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cloneRemoteProjectResponse{Project: project})
}
