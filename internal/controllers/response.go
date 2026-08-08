package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"wade/internal/models/remoterepositories"
	"wade/internal/models/repositories"
	"wade/internal/models/reviewsnapshots"
	"wade/internal/models/settings"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
)

// Problem is the stable application/problem+json response shape.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
} // @name Problem

// WriteAPINotFound writes the response for an unregistered API route.
func WriteAPINotFound(w http.ResponseWriter, _ *http.Request) {
	writeProblem(w, http.StatusNotFound, "endpoint_not_found", "Endpoint not found", "The requested API endpoint does not exist.")
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func writeProblem(w http.ResponseWriter, statusCode int, code string, title string, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(Problem{
		Type:   "https://wade.local/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:  title,
		Status: statusCode,
		Detail: detail,
		Code:   code,
	})
}

func writeModelError(w http.ResponseWriter, err error, fallbackDetail string) {
	switch {
	case matchesError[workspaces.WorkspaceNotFoundError](err),
		matchesError[repositories.WorkspaceNotFoundError](err),
		matchesError[terminals.WorkspaceNotFoundError](err),
		matchesError[reviewsnapshots.WorkspaceNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "workspace_not_found", "Workspace not found", err.Error())
	case matchesError[repositories.RepositoryNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "repository_not_found", "Repository not found", err.Error())
	case matchesError[repositories.WorktreeNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "worktree_not_found", "Worktree not found", err.Error())
	case matchesError[terminals.TerminalNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "terminal_not_found", "Terminal not found", err.Error())
	case matchesError[reviewsnapshots.SnapshotNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "review_snapshot_not_found", "Review snapshot not found", err.Error())
	case matchesError[reviewsnapshots.SnapshotFileNotFoundError](err):
		writeProblem(w, http.StatusNotFound, "review_snapshot_file_not_found", "Review snapshot file not found", err.Error())
	case matchesError[repositories.RepositoryIDConflictError](err):
		writeProblem(w, http.StatusConflict, "repository_id_conflict", "Repository ID conflict", err.Error())
	case matchesError[workspaces.WorkspaceAlreadyExistsError](err):
		writeProblem(w, http.StatusConflict, "workspace_already_exists", "Workspace already exists", err.Error())
	case matchesError[repositories.WorktreeNotRemovableError](err):
		writeProblem(w, http.StatusConflict, "worktree_not_removable", "Worktree cannot be removed", err.Error())
	case matchesError[settings.InvalidSettingsError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_settings", "Invalid settings", err.Error())
	case matchesError[workspaces.InvalidWorkspaceIDError](err), matchesError[repositories.InvalidWorkspaceIDError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_workspace_id", "Invalid workspace ID", err.Error())
	case matchesError[repositories.InvalidRepositoryIDError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_repository_id", "Invalid repository ID", err.Error())
	case matchesError[workspaces.InvalidRemoteRepositoryIDError](err), matchesError[remoterepositories.InvalidRemoteRepositoryIDError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_remote_repository_id", "Invalid remote repository ID", err.Error())
	case matchesError[workspaces.WorkspaceDirectoryNotConfiguredError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "workspace_directory_not_configured", "Workspace directory is not configured", err.Error())
	case matchesError[repositories.BranchReferenceRequiredError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "branch_ref_required", "Branch reference is required", "branchRef must identify a local or remote branch.")
	case matchesError[repositories.InvalidBranchReferenceError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_branch_reference", "Invalid branch reference", err.Error())
	case matchesError[repositories.InvalidWorktreeIDError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_worktree_id", "Invalid worktree ID", err.Error())
	case matchesError[terminals.InvalidTerminalIDError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_terminal_id", "Invalid terminal ID", err.Error())
	case matchesError[terminals.AgentNotConfiguredError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "agent_not_configured", "Agent is not configured", err.Error())
	case matchesError[terminals.TerminalInputRequiredError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "terminal_input_required", "Terminal input is required", err.Error())
	case matchesError[terminals.InvalidInputModeError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_terminal_input_mode", "Invalid terminal input mode", err.Error())
	case matchesError[reviewsnapshots.WorkspaceNotGitRepositoryError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "workspace_not_git_repository", "Workspace is not a Git repository", err.Error())
	case matchesError[reviewsnapshots.InvalidScopeError](err):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_review_scope", "Invalid review scope", err.Error())
	default:
		log.Printf("request failed: %v", err)
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error", fallbackDetail)
	}
}

func matchesError[T error](err error) bool {
	return errors.As(err, new(T))
}

func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}

	return nil
}
