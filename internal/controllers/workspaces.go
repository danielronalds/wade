package controllers

import (
	"log"
	"net/http"
	"net/url"

	"wade/internal/models/repositories"
	"wade/internal/models/workspaces"
)

// Workspaces coordinates Workspace responses across aggregate Models.
type Workspaces struct {
	workspaces   WorkspacesModel
	repositories RepositoriesModel
	terminals    TerminalsModel
}

// WorkspaceList is the collection response for workspace summaries.
type WorkspaceList struct {
	Items []workspaces.WorkspaceSummary `json:"items"`
} // @name WorkspaceList

// NewWorkspaces constructs the Workspace controller.
func NewWorkspaces(workspaceModel WorkspacesModel, repositoryModel RepositoriesModel, terminalModel TerminalsModel) Workspaces {
	return Workspaces{workspaces: workspaceModel, repositories: repositoryModel, terminals: terminalModel}
}

// List returns enriched workspace summaries using targeted loading for active workspaces.
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

	var summaries []workspaces.WorkspaceSummary
	var contexts map[string]repositories.WorkspaceContext
	var err error
	if activity == "active" {
		activeWorkspaceIDs := h.terminals.ActiveWorkspaceIDs()
		if len(activeWorkspaceIDs) == 0 {
			writeJSON(w, http.StatusOK, WorkspaceList{Items: []workspaces.WorkspaceSummary{}})
			return
		}
		summaries, err = h.workspaces.ListByIDs(r.Context(), activeWorkspaceIDs)
		if err == nil {
			discoveredActiveWorkspaceIDs := make([]string, 0, len(summaries))
			for _, summary := range summaries {
				discoveredActiveWorkspaceIDs = append(discoveredActiveWorkspaceIDs, summary.ID)
			}
			contexts, err = h.repositories.ListWorkspaceContextsByIDs(r.Context(), discoveredActiveWorkspaceIDs)
		}
	} else {
		summaries, err = h.workspaces.List(r.Context())
		if err == nil {
			workspaceContexts, contextErr := h.repositories.ListWorkspaceContexts(r.Context())
			err = contextErr
			contexts = make(map[string]repositories.WorkspaceContext, len(workspaceContexts))
			for _, workspaceContext := range workspaceContexts {
				contexts[workspaceContext.WorkspaceID] = workspaceContext
			}
		}
	}
	if err != nil {
		writeModelError(w, err, "Unable to list workspaces.")
		return
	}

	repositoryID := r.URL.Query().Get("repositoryId")
	remoteRepositoryID := r.URL.Query().Get("remoteRepositoryId")
	items := make([]workspaces.WorkspaceSummary, 0, len(summaries))
	for _, summary := range summaries {
		workspace := workspaces.Workspace(summary)
		h.enrichWorkspace(r, &workspace, contexts[summary.ID], false)
		if activity == "active" && workspace.Activity.ActiveTerminalCount == 0 {
			continue
		}
		if repositoryID != "" && !matchesReference(workspace.RepositoryID, repositoryID) {
			continue
		}
		if remoteRepositoryID != "" && !matchesReference(workspace.RemoteRepositoryID, remoteRepositoryID) {
			continue
		}
		items = append(items, workspaces.WorkspaceSummary(workspace))
	}

	writeJSON(w, http.StatusOK, WorkspaceList{Items: items})
}

// Get returns one workspace enriched across aggregate Models.
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
		writeModelError(w, err, "Unable to load the workspace.")
		return
	}
	workspaceContext, err := h.repositories.GetWorkspaceContext(r.Context(), workspace.ID)
	if err != nil {
		writeModelError(w, err, "Unable to load the workspace.")
		return
	}
	if workspaceContext != nil {
		h.enrichWorkspace(r, &workspace, *workspaceContext, true)
	} else {
		workspace.Activity.ActiveTerminalCount = h.terminals.ActiveTerminalCount(workspace.ID)
	}

	writeJSON(w, http.StatusOK, workspace)
}

// Materialise creates and enriches a workspace from a remote repository.
// @Summary Materialise a remote repository as a workspace
// @ID materialiseWorkspace
// @Tags Workspaces
// @Accept json
// @Produce json
// @Param request body workspaces.MaterialiseRequest true "Workspace materialisation request"
// @Success 201 {object} workspaces.Workspace
// @Header 201 {string} Location "Created workspace URL"
// @Failure 400 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces [post]
func (h Workspaces) Materialise(w http.ResponseWriter, r *http.Request) {
	var request workspaces.MaterialiseRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a valid workspace materialisation request.")
		return
	}

	workspace, err := h.workspaces.Materialise(r.Context(), request)
	if err != nil {
		writeModelError(w, err, "Unable to materialise the remote repository.")
		return
	}
	workspaceContext, err := h.repositories.GetWorkspaceContext(r.Context(), workspace.ID)
	if err != nil {
		writeModelError(w, err, "Unable to load the materialised workspace.")
		return
	}
	if workspaceContext != nil {
		h.enrichWorkspace(r, &workspace, *workspaceContext, true)
	}

	w.Header().Set("Location", "/api/v1/workspaces/"+url.PathEscape(workspace.ID))
	writeJSON(w, http.StatusCreated, workspace)
}

func (h Workspaces) enrichWorkspace(r *http.Request, workspace *workspaces.Workspace, context repositories.WorkspaceContext, resolvePullRequest bool) {
	workspace.Activity.ActiveTerminalCount = h.terminals.ActiveTerminalCount(workspace.ID)
	if context.WorkspaceID == "" {
		return
	}

	workspace.RepositoryID = stringReference(context.Repository.ID)
	workspace.RemoteRepositoryID = cloneStringReference(context.Repository.RemoteRepositoryID)
	workspace.Worktree = &workspaces.WorktreeReference{
		ID:          workspace.ID,
		IsMain:      context.IsMain,
		IsRemovable: context.IsRemovable,
	}
	workspace.Branch = &workspaces.Branch{
		Ref:        context.Branch.Ref,
		Name:       context.Branch.Name,
		IsDetached: context.Branch.IsDetached,
		Commit:     context.Branch.Commit,
	}
	links, err := h.workspaces.ResolveLinks(r.Context(), workspaces.LinkContext{
		RemoteRepositoryID: context.Repository.RemoteRepositoryID,
		BranchName:         context.Branch.Name,
		ResolvePullRequest: resolvePullRequest,
	})
	workspace.Links = links
	if err != nil {
		log.Printf("workspace %q provider link enrichment failed: %v", workspace.ID, err)
	}
}

func matchesReference(reference *string, expected string) bool {
	return reference != nil && *reference == expected
}

func stringReference(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func cloneStringReference(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
