package controllers

import (
	"context"
	"net/http"
	"net/url"

	"wade/internal/services/remoterepositories"
	"wade/internal/services/workspaces"
)

type workspaceService interface {
	List(ctx context.Context) ([]workspaces.WorkspaceSummary, error)
	Get(ctx context.Context, workspaceID string) (workspaces.Workspace, error)
}

type workspaceMaterialiser interface {
	Clone(ctx context.Context, request remoterepositories.CloneRequest) (workspaces.Workspace, error)
}

type Workspaces struct {
	workspaces   workspaceService
	materialiser workspaceMaterialiser
}

type WorkspaceList struct {
	Items []workspaces.WorkspaceSummary `json:"items"`
} // @name WorkspaceList

type MaterialiseWorkspaceRequest struct {
	RemoteRepositoryID string `json:"remoteRepositoryId"`
	WorkspaceDirectory string `json:"workspaceDirectory"`
} // @name MaterialiseWorkspaceRequest

func NewWorkspaces(workspaceService workspaceService, materialiser workspaceMaterialiser) Workspaces {
	return Workspaces{workspaces: workspaceService, materialiser: materialiser}
}

// @Summary List workspaces
// @ID listWorkspaces
// @Tags Workspaces
// @Produce json
// @Param activity query string false "Filter by activity" Enums(active)
// @Param repositoryId query string false "Local repository ID"
// @Param remoteRepositoryId query string false "Remote repository ID"
// @Success 200 {object} WorkspaceList
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces [get]
func (h Workspaces) List(w http.ResponseWriter, r *http.Request) {
	activity := r.URL.Query().Get("activity")
	if activity != "" && activity != "active" {
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_workspace_activity", "Invalid workspace activity", "activity must be active when provided")
		return
	}

	workspaceSummaries, err := h.workspaces.List(r.Context())
	if err != nil {
		writeServiceError(w, err, "Unable to list workspaces.")
		return
	}

	repositoryID := r.URL.Query().Get("repositoryId")
	remoteRepositoryID := r.URL.Query().Get("remoteRepositoryId")
	items := make([]workspaces.WorkspaceSummary, 0, len(workspaceSummaries))
	for _, workspace := range workspaceSummaries {
		if activity == "active" && workspace.Activity.ActiveTerminalCount == 0 {
			continue
		}
		if repositoryID != "" && !matchesReference(workspace.RepositoryID, repositoryID) {
			continue
		}
		if remoteRepositoryID != "" && !matchesReference(workspace.RemoteRepositoryID, remoteRepositoryID) {
			continue
		}

		items = append(items, workspace)
	}

	writeJSON(w, http.StatusOK, WorkspaceList{Items: items})
}

// @Summary Get a workspace
// @ID getWorkspace
// @Tags Workspaces
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} workspaces.Workspace
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId} [get]
func (h Workspaces) Get(w http.ResponseWriter, r *http.Request) {
	workspace, err := h.workspaces.Get(r.Context(), r.PathValue("workspaceId"))
	if err != nil {
		writeServiceError(w, err, "Unable to load the workspace.")
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

// @Summary Materialise a remote repository as a workspace
// @ID materialiseWorkspace
// @Tags Workspaces
// @Accept json
// @Produce json
// @Param request body MaterialiseWorkspaceRequest true "Workspace materialisation request"
// @Success 201 {object} workspaces.Workspace
// @Header 201 {string} Location "Created workspace URL"
// @Failure 400 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces [post]
func (h Workspaces) Materialise(w http.ResponseWriter, r *http.Request) {
	var request MaterialiseWorkspaceRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a valid workspace materialisation request.")
		return
	}

	workspace, err := h.materialiser.Clone(r.Context(), remoterepositories.CloneRequest{
		RemoteRepositoryID: request.RemoteRepositoryID,
		WorkspaceDirectory: request.WorkspaceDirectory,
	})
	if err != nil {
		writeServiceError(w, err, "Unable to materialise the remote repository.")
		return
	}

	w.Header().Set("Location", "/api/v1/workspaces/"+url.PathEscape(workspace.ID))
	writeJSON(w, http.StatusCreated, workspace)
}

func matchesReference(reference *string, expected string) bool {
	return reference != nil && *reference == expected
}
