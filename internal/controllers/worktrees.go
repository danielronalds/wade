package controllers

import (
	"net/http"
	"net/url"
	"strings"

	"wade/internal/models/repositories"
)

// Worktrees serves worktrees and branches owned by Repositories.
type Worktrees struct {
	repositories RepositoriesModel
	terminals    TerminalsModel
}

type WorktreeList struct {
	Items []repositories.Worktree `json:"items"`
} // @name WorktreeList

type BranchList struct {
	Items []repositories.Branch `json:"items"`
} // @name BranchList

// NewWorktrees constructs the Worktrees controller.
func NewWorktrees(repositories RepositoriesModel, terminals TerminalsModel) Worktrees {
	return Worktrees{repositories: repositories, terminals: terminals}
}

// @Summary List repository worktrees
// @ID listRepositoryWorktrees
// @Tags Worktrees
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Success 200 {object} WorktreeList
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId}/worktrees [get]
func (h Worktrees) List(w http.ResponseWriter, r *http.Request) {
	worktrees, err := h.repositories.ListWorktrees(r.Context(), r.PathValue("repositoryId"))
	if err != nil {
		writeModelError(w, err, "Unable to list repository worktrees.")
		return
	}
	writeJSON(w, http.StatusOK, WorktreeList{Items: worktrees})
}

// @Summary Create a repository worktree
// @ID createRepositoryWorktree
// @Tags Worktrees
// @Accept json
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Param request body repositories.CreateWorktreeRequest true "Worktree creation request"
// @Success 201 {object} repositories.Worktree
// @Header 201 {string} Location "Created worktree URL"
// @Failure 400 {object} Problem
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId}/worktrees [post]
func (h Worktrees) Create(w http.ResponseWriter, r *http.Request) {
	var request repositories.CreateWorktreeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a valid worktree creation request.")
		return
	}
	if strings.TrimSpace(request.BranchRef) == "" {
		writeProblem(w, http.StatusUnprocessableEntity, "branch_ref_required", "Branch reference is required", "branchRef must identify a local or remote branch.")
		return
	}

	repositoryID := r.PathValue("repositoryId")
	created, err := h.repositories.CreateWorktree(r.Context(), repositoryID, request)
	if err != nil {
		writeModelError(w, err, "Unable to create the worktree.")
		return
	}
	w.Header().Set("Location", "/api/v1/repositories/"+url.PathEscape(repositoryID)+"/worktrees/"+url.PathEscape(created.ID))
	writeJSON(w, http.StatusCreated, created)
}

// @Summary Remove a repository worktree
// @ID deleteRepositoryWorktree
// @Tags Worktrees
// @Param repositoryId path string true "Repository ID"
// @Param worktreeId path string true "Worktree ID"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId}/worktrees/{worktreeId} [delete]
func (h Worktrees) Delete(w http.ResponseWriter, r *http.Request) {
	repositoryID := r.PathValue("repositoryId")
	worktreeID := r.PathValue("worktreeId")
	target, err := h.repositories.GetWorktree(r.Context(), repositoryID, worktreeID)
	if err != nil {
		writeModelError(w, err, "Unable to inspect the worktree.")
		return
	}
	if target.IsRemovable {
		if _, err := h.terminals.DeleteAll(r.Context(), target.WorkspaceID); err != nil {
			writeModelError(w, err, "Unable to close worktree terminals.")
			return
		}
	}
	if _, err := h.repositories.RemoveWorktree(r.Context(), repositoryID, worktreeID); err != nil {
		writeModelError(w, err, "Unable to remove the worktree.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary List repository branches
// @ID listRepositoryBranches
// @Tags Branches
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Param kind query string false "Branch kind" Enums(local,remote)
// @Success 200 {object} BranchList
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId}/branches [get]
func (h Worktrees) ListBranches(w http.ResponseWriter, r *http.Request) {
	kind := repositories.BranchKind(r.URL.Query().Get("kind"))
	if kind != "" && kind != repositories.BranchKindLocal && kind != repositories.BranchKindRemote {
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_branch_kind", "Invalid branch kind", "kind must be local or remote when provided")
		return
	}

	repositoryID := r.PathValue("repositoryId")
	var branches []repositories.Branch
	if kind == "" {
		localBranches, err := h.repositories.ListBranches(r.Context(), repositoryID, repositories.BranchKindLocal)
		if err != nil {
			writeModelError(w, err, "Unable to list repository branches.")
			return
		}
		remoteBranches, err := h.repositories.ListBranches(r.Context(), repositoryID, repositories.BranchKindRemote)
		if err != nil {
			writeModelError(w, err, "Unable to list repository branches.")
			return
		}
		branches = append(localBranches, remoteBranches...)
	} else {
		var err error
		branches, err = h.repositories.ListBranches(r.Context(), repositoryID, kind)
		if err != nil {
			writeModelError(w, err, "Unable to list repository branches.")
			return
		}
	}
	writeJSON(w, http.StatusOK, BranchList{Items: branches})
}
