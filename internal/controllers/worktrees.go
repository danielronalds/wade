package controllers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"wade/internal/services/gitrepositories"
	"wade/internal/services/worktrees"
)

type worktreeService interface {
	List(ctx context.Context, repository gitrepositories.Context) ([]worktrees.Worktree, error)
	Create(ctx context.Context, repository gitrepositories.Context, branchRef string) (worktrees.Worktree, error)
	Remove(ctx context.Context, repository gitrepositories.Context, worktreeID string) (worktrees.Worktree, error)
	Branches(ctx context.Context, repository gitrepositories.Context, kind worktrees.BranchKind) ([]worktrees.Branch, error)
}

type Worktrees struct {
	repositories localRepositoryService
	worktrees    worktreeService
}

type WorktreeList struct {
	Items []worktrees.Worktree `json:"items"`
} // @name WorktreeList

type BranchList struct {
	Items []worktrees.Branch `json:"items"`
} // @name BranchList

type CreateWorktreeRequest struct {
	BranchRef string `json:"branchRef"`
} // @name CreateWorktreeRequest

func NewWorktrees(repositoryService localRepositoryService, worktreeService worktreeService) Worktrees {
	return Worktrees{repositories: repositoryService, worktrees: worktreeService}
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
	repository, ok := h.resolveRepository(w, r)
	if !ok {
		return
	}

	availableWorktrees, err := h.worktrees.List(r.Context(), repository)
	if err != nil {
		writeServiceError(w, err, "Unable to list repository worktrees.")
		return
	}

	writeJSON(w, http.StatusOK, WorktreeList{Items: availableWorktrees})
}

// @Summary Create a repository worktree
// @ID createRepositoryWorktree
// @Tags Worktrees
// @Accept json
// @Produce json
// @Param repositoryId path string true "Repository ID"
// @Param request body CreateWorktreeRequest true "Worktree creation request"
// @Success 201 {object} worktrees.Worktree
// @Header 201 {string} Location "Created worktree URL"
// @Failure 400 {object} Problem
// @Failure 404 {object} Problem
// @Failure 409 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/repositories/{repositoryId}/worktrees [post]
func (h Worktrees) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateWorktreeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain a valid worktree creation request.")
		return
	}
	if strings.TrimSpace(request.BranchRef) == "" {
		writeProblem(w, http.StatusUnprocessableEntity, "branch_ref_required", "Branch reference is required", "branchRef must identify a local or remote branch.")
		return
	}

	repository, ok := h.resolveRepository(w, r)
	if !ok {
		return
	}

	created, err := h.worktrees.Create(r.Context(), repository, request.BranchRef)
	if err != nil {
		writeServiceError(w, err, "Unable to create the worktree.")
		return
	}

	w.Header().Set("Location", "/api/v1/repositories/"+url.PathEscape(repository.Repository.ID)+"/worktrees/"+url.PathEscape(created.ID))
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
	repository, ok := h.resolveRepository(w, r)
	if !ok {
		return
	}

	if _, err := h.worktrees.Remove(r.Context(), repository, r.PathValue("worktreeId")); err != nil {
		writeServiceError(w, err, "Unable to remove the worktree.")
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
	kind := worktrees.BranchKind(r.URL.Query().Get("kind"))
	if kind != "" && kind != worktrees.BranchKindLocal && kind != worktrees.BranchKindRemote {
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_branch_kind", "Invalid branch kind", "kind must be local or remote when provided")
		return
	}

	repository, ok := h.resolveRepository(w, r)
	if !ok {
		return
	}

	var branches []worktrees.Branch
	if kind == "" {
		localBranches, err := h.worktrees.Branches(r.Context(), repository, worktrees.BranchKindLocal)
		if err != nil {
			writeServiceError(w, err, "Unable to list repository branches.")
			return
		}
		remoteBranches, err := h.worktrees.Branches(r.Context(), repository, worktrees.BranchKindRemote)
		if err != nil {
			writeServiceError(w, err, "Unable to list repository branches.")
			return
		}
		branches = append(localBranches, remoteBranches...)
	} else {
		var err error
		branches, err = h.worktrees.Branches(r.Context(), repository, kind)
		if err != nil {
			writeServiceError(w, err, "Unable to list repository branches.")
			return
		}
	}

	writeJSON(w, http.StatusOK, BranchList{Items: branches})
}

func (h Worktrees) resolveRepository(w http.ResponseWriter, r *http.Request) (gitrepositories.Context, bool) {
	repository, err := h.repositories.Resolve(r.Context(), r.PathValue("repositoryId"))
	if err != nil {
		writeServiceError(w, err, "Unable to resolve the repository.")
		return gitrepositories.Context{}, false
	}

	return repository, true
}
