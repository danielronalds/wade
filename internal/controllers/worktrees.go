package controllers

// TODO: Review properly

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

type worktreeSessionCloser interface {
	CloseSessionsForDirectory(directory string) int
}

type Worktrees struct {
	workspaces workspaces.Service
	worktrees  worktrees.Service
	terminals  worktreeSessionCloser
}

func NewWorktrees(workspaceService workspaces.Service, worktreeService worktrees.Service, terminals worktreeSessionCloser) Worktrees {
	return Worktrees{workspaces: workspaceService, worktrees: worktreeService, terminals: terminals}
}

type worktreesResponse struct {
	Worktrees []worktrees.Worktree `json:"worktrees"`
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
	projectPath, ok := h.resolveProjectPath(w, r.URL.Query().Get("project"))
	if !ok {
		return
	}

	worktrees, err := h.worktrees.List(r.Context(), projectPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreesResponse{Worktrees: worktrees})
}

type createWorktreeRequest struct {
	Project string `json:"project"`
	Branch  string `json:"branch"`
} // @name handlers.createWorktreeRequest

type worktreeResponse struct {
	Worktree worktrees.Worktree `json:"worktree"`
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

	projectPath, ok := h.resolveProjectPath(w, request.Project)
	if !ok {
		return
	}

	if strings.TrimSpace(request.Branch) == "" {
		writeJSONError(w, http.StatusBadRequest, "branch is required")
		return
	}

	created, err := h.worktrees.Create(r.Context(), projectPath, request.Branch)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreeResponse{Worktree: created})
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

	projectName := firstNonEmpty(request.Project, r.URL.Query().Get("project"))
	worktreeName := firstNonEmpty(request.Worktree, r.URL.Query().Get("worktree"))
	projectPath, ok := h.resolveProjectPath(w, projectName)
	if !ok {
		return
	}

	target, err := h.worktrees.Find(r.Context(), projectPath, worktreeName)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !target.IsRemovable {
		writeJSONError(w, http.StatusBadRequest, "cannot remove base worktree")
		return
	}

	if h.terminals != nil {
		h.terminals.CloseSessionsForDirectory(target.Path)
	}

	if err := h.worktrees.Remove(r.Context(), projectPath, target); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, worktreeResponse{Worktree: target})
}

// @Summary List remote branches
// @ID listRemoteBranches
// @Tags Worktrees
// @Produce json
// @Param project query string true "Project name"
// @Success 200 {object} worktrees.RemoteBranchList
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/worktrees/remote-branches [get]
func (h Worktrees) ListRemoteBranches(w http.ResponseWriter, r *http.Request) {
	projectPath, ok := h.resolveProjectPath(w, r.URL.Query().Get("project"))
	if !ok {
		return
	}

	branches, err := h.worktrees.RemoteBranches(r.Context(), projectPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, branches)
}

func (h Worktrees) resolveProjectPath(w http.ResponseWriter, projectName string) (string, bool) {
	if strings.TrimSpace(projectName) == "" {
		writeJSONError(w, http.StatusBadRequest, "project is required")
		return "", false
	}

	projectPath, err := h.workspaces.Path(projectName)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return "", false
	}

	return projectPath, true
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
