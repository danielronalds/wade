package controllers

// TODO: Review properly

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"wade/internal/services/gitrepositories"
	"wade/internal/services/worktrees"
)

type Worktrees struct {
	repositories gitrepositories.Service
	worktrees    worktrees.Service
}

func NewWorktrees(repositoryService gitrepositories.Service, worktreeService worktrees.Service) Worktrees {
	return Worktrees{repositories: repositoryService, worktrees: worktreeService}
}

type legacyWorktree struct {
	Name                    string   `json:"name"`
	ProjectName             string   `json:"projectName"`
	Path                    string   `json:"path"`
	Branch                  string   `json:"branch"`
	IsBase                  bool     `json:"isBase"`
	IsCurrent               bool     `json:"isCurrent"`
	IsRemovable             bool     `json:"isRemovable"`
	IgnoredFileCopyWarnings []string `json:"ignoredFileCopyWarnings,omitempty"`
} // @name worktree.Worktree

type worktreesResponse struct {
	Worktrees []legacyWorktree `json:"worktrees"`
} // @name handlers.worktreesResponse

// @Summary List worktrees
// @ID listWorktrees
// @Tags Worktrees
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} worktreesResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/worktrees [get]
func (h Worktrees) ListWorktrees(w http.ResponseWriter, r *http.Request) {
	repository, workspaceID, ok := h.resolveRepository(w, r, r.URL.Query().Get("project"))
	if !ok {
		return
	}

	availableWorktrees, err := h.worktrees.List(r.Context(), repository)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreesResponse{Worktrees: legacyWorktrees(availableWorktrees, workspaceID)})
}

type createWorktreeRequest struct {
	Project string `json:"project"`
	Branch  string `json:"branch"`
} // @name handlers.createWorktreeRequest

type worktreeResponse struct {
	Worktree legacyWorktree `json:"worktree"`
} // @name handlers.worktreeResponse

// @Summary Create worktree
// @ID createWorktree
// @Tags Worktrees
// @Accept json
// @Produce json
// @Param request body createWorktreeRequest true "Worktree creation request"
// @Success 200 {object} worktreeResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/worktrees [post]
func (h Worktrees) CreateWorktree(w http.ResponseWriter, r *http.Request) {
	var request createWorktreeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid worktree request")
		return
	}

	repository, _, ok := h.resolveRepository(w, r, request.Project)
	if !ok {
		return
	}
	if strings.TrimSpace(request.Branch) == "" {
		writeJSONError(w, http.StatusBadRequest, "branch is required")
		return
	}

	created, err := h.worktrees.Create(r.Context(), repository, request.Branch)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreeResponse{Worktree: legacyWorktreeResponse(created, created.WorkspaceID)})
}

type removeWorktreeRequest struct {
	Project  string `json:"project"`
	Worktree string `json:"worktree"`
} // @name handlers.removeWorktreeRequest

// @Summary Remove worktree
// @ID removeWorktree
// @Tags Worktrees
// @Accept json
// @Produce json
// @Param request body removeWorktreeRequest false "Worktree removal request"
// @Param project query string false "Project name"
// @Param worktree query string false "Worktree name"
// @Success 200 {object} worktreeResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/worktrees [delete]
func (h Worktrees) RemoveWorktree(w http.ResponseWriter, r *http.Request) {
	var request removeWorktreeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid worktree request")
		return
	}

	workspaceID := firstNonEmpty(request.Project, r.URL.Query().Get("project"))
	worktreeID := firstNonEmpty(request.Worktree, r.URL.Query().Get("worktree"))
	repository, _, ok := h.resolveRepository(w, r, workspaceID)
	if !ok {
		return
	}

	removed, err := h.worktrees.Remove(r.Context(), repository, worktreeID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreeResponse{Worktree: legacyWorktreeResponse(removed, workspaceID)})
}

type legacyRemoteBranchList struct {
	Remote   string               `json:"remote"`
	Branches []legacyRemoteBranch `json:"branches"`
} // @name worktree.RemoteBranchList

type legacyRemoteBranch struct {
	Name                string `json:"name"`
	Branch              string `json:"branch"`
	HasLocalBranch      bool   `json:"hasLocalBranch"`
	IsCheckedOut        bool   `json:"isCheckedOut"`
	WorktreeName        string `json:"worktreeName"`
	WorktreeProjectName string `json:"worktreeProjectName"`
} // @name worktree.RemoteBranch

// @Summary List remote branches
// @ID listRemoteBranches
// @Tags Worktrees
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} legacyRemoteBranchList
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/worktrees/remote-branches [get]
func (h Worktrees) ListRemoteBranches(w http.ResponseWriter, r *http.Request) {
	repository, _, ok := h.resolveRepository(w, r, r.URL.Query().Get("project"))
	if !ok {
		return
	}

	branches, err := h.worktrees.Branches(r.Context(), repository, worktrees.BranchKindRemote)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := legacyRemoteBranchList{Branches: make([]legacyRemoteBranch, 0, len(branches))}
	for _, branch := range branches {
		remote := referencedString(branch.Remote)
		if response.Remote == "" {
			response.Remote = remote
		}

		worktreeID := referencedString(branch.CheckedOutWorkspaceID)
		worktreeName := strings.TrimPrefix(worktreeID, repository.Repository.ID+"-")
		response.Branches = append(response.Branches, legacyRemoteBranch{
			Name:                remote + "/" + branch.Name,
			Branch:              branch.Name,
			HasLocalBranch:      branch.HasLocalBranch,
			IsCheckedOut:        branch.CheckedOutWorkspaceID != nil,
			WorktreeName:        worktreeName,
			WorktreeProjectName: worktreeID,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h Worktrees) resolveRepository(
	w http.ResponseWriter,
	r *http.Request,
	workspaceID string,
) (gitrepositories.Context, string, bool) {
	if strings.TrimSpace(workspaceID) == "" {
		writeJSONError(w, http.StatusBadRequest, "project is required")
		return gitrepositories.Context{}, "", false
	}

	workspaceContext, isGit, err := h.repositories.ResolveWorkspace(r.Context(), workspaceID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return gitrepositories.Context{}, "", false
	}
	if !isGit {
		writeJSONError(w, http.StatusBadRequest, "project is not a Git repository")
		return gitrepositories.Context{}, "", false
	}

	return workspaceContext.RepositoryContext, workspaceID, true
}

func legacyWorktrees(availableWorktrees []worktrees.Worktree, currentWorkspaceID string) []legacyWorktree {
	responses := make([]legacyWorktree, 0, len(availableWorktrees))
	for _, worktree := range availableWorktrees {
		responses = append(responses, legacyWorktreeResponse(worktree, currentWorkspaceID))
	}
	return responses
}

func legacyWorktreeResponse(worktree worktrees.Worktree, currentWorkspaceID string) legacyWorktree {
	branchName := ""
	if worktree.Branch != nil {
		branchName = worktree.Branch.Name
	}

	return legacyWorktree{
		Name:                    worktree.Name,
		ProjectName:             worktree.WorkspaceID,
		Path:                    worktree.Path(),
		Branch:                  branchName,
		IsBase:                  worktree.IsMain,
		IsCurrent:               worktree.WorkspaceID == currentWorkspaceID,
		IsRemovable:             worktree.IsRemovable,
		IgnoredFileCopyWarnings: worktree.IgnoredFileCopyWarnings,
	}
}

func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
