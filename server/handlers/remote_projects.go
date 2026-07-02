package handlers

import (
	"context"
	"net/http"

	"wade/project"
	"wade/remote"
)

type remoteProjectService interface {
	List(ctx context.Context, localProjectNames []string) ([]remote.Project, error)
	Clone(ctx context.Context, request remote.CloneRequest) (remote.ClonedProject, error)
}

type RemoteProjects struct {
	projects project.Store
	remote   remoteProjectService
}

type remoteProjectsResponse struct {
	Projects []remote.Project `json:"projects"`
}

type cloneRemoteProjectRequest struct {
	NameWithOwner  string `json:"nameWithOwner"`
	DirectoryIndex int    `json:"directoryIndex"`
}

type cloneRemoteProjectResponse struct {
	Project remote.ClonedProject `json:"project"`
}

func NewRemoteProjects(projects project.Store, remote remoteProjectService) RemoteProjects {
	return RemoteProjects{projects: projects, remote: remote}
}

func (h RemoteProjects) List(w http.ResponseWriter, r *http.Request) {
	localProjectNames, err := h.projects.Names()
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

func (h RemoteProjects) Clone(w http.ResponseWriter, r *http.Request) {
	var request cloneRemoteProjectRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid remote project request")
		return
	}

	localProjectNames, err := h.projects.Names()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to list local projects")
		return
	}

	project, err := h.remote.Clone(r.Context(), remote.CloneRequest{
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
