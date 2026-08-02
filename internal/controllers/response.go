package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"wade/internal/services/config"
	"wade/internal/services/gitrepositories"
	"wade/internal/services/remoterepositories"
	"wade/internal/services/review"
	"wade/internal/services/terminals"
	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
} // @name Problem

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

func writeServiceError(w http.ResponseWriter, err error, fallbackDetail string) {
	var invalidSettings config.InvalidSettingsError
	var invalidWorkspaceID workspaces.InvalidWorkspaceIDError
	var workspaceNotFound workspaces.WorkspaceNotFoundError
	var gitWorkspaceNotFound gitrepositories.WorkspaceNotFoundError
	var invalidRepositoryID gitrepositories.InvalidRepositoryIDError
	var repositoryNotFound gitrepositories.RepositoryNotFoundError
	var repositoryIDConflict gitrepositories.RepositoryIDConflictError
	var invalidRemoteRepositoryID remoterepositories.InvalidRemoteRepositoryIDError
	var workspaceDirectoryNotConfigured remoterepositories.WorkspaceDirectoryNotConfiguredError
	var workspaceAlreadyExists remoterepositories.WorkspaceAlreadyExistsError
	var invalidWorktreeID worktrees.InvalidWorktreeIDError
	var worktreeNotFound worktrees.WorktreeNotFoundError
	var worktreeNotRemovable worktrees.WorktreeNotRemovableError
	var invalidTerminalID terminals.InvalidTerminalIDError
	var agentNotConfigured terminals.AgentNotConfiguredError
	var terminalNotFound terminals.TerminalNotFoundError
	var terminalInputRequired terminals.TerminalInputRequiredError
	var invalidInputMode terminals.InvalidInputModeError
	var workspaceNotGitRepository review.WorkspaceNotGitRepositoryError
	var snapshotNotFound review.SnapshotNotFoundError
	var snapshotFileNotFound review.SnapshotFileNotFoundError
	var invalidReviewScope review.InvalidScopeError

	switch {
	case errors.As(err, &workspaceNotFound), errors.As(err, &gitWorkspaceNotFound):
		writeProblem(w, http.StatusNotFound, "workspace_not_found", "Workspace not found", err.Error())
	case errors.As(err, &repositoryNotFound):
		writeProblem(w, http.StatusNotFound, "repository_not_found", "Repository not found", err.Error())
	case errors.As(err, &worktreeNotFound):
		writeProblem(w, http.StatusNotFound, "worktree_not_found", "Worktree not found", err.Error())
	case errors.As(err, &terminalNotFound):
		writeProblem(w, http.StatusNotFound, "terminal_not_found", "Terminal not found", err.Error())
	case errors.As(err, &snapshotNotFound):
		writeProblem(w, http.StatusNotFound, "review_snapshot_not_found", "Review snapshot not found", err.Error())
	case errors.As(err, &snapshotFileNotFound):
		writeProblem(w, http.StatusNotFound, "review_snapshot_file_not_found", "Review snapshot file not found", err.Error())
	case errors.As(err, &repositoryIDConflict):
		writeProblem(w, http.StatusConflict, "repository_id_conflict", "Repository ID conflict", err.Error())
	case errors.As(err, &workspaceAlreadyExists):
		writeProblem(w, http.StatusConflict, "workspace_already_exists", "Workspace already exists", err.Error())
	case errors.As(err, &worktreeNotRemovable):
		writeProblem(w, http.StatusConflict, "worktree_not_removable", "Worktree cannot be removed", err.Error())
	case errors.As(err, &invalidSettings):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_settings", "Invalid settings", err.Error())
	case errors.As(err, &invalidWorkspaceID):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_workspace_id", "Invalid workspace ID", err.Error())
	case errors.As(err, &invalidRepositoryID):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_repository_id", "Invalid repository ID", err.Error())
	case errors.As(err, &invalidRemoteRepositoryID):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_remote_repository_id", "Invalid remote repository ID", err.Error())
	case errors.As(err, &workspaceDirectoryNotConfigured):
		writeProblem(w, http.StatusUnprocessableEntity, "workspace_directory_not_configured", "Workspace directory is not configured", err.Error())
	case errors.As(err, &invalidWorktreeID):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_worktree_id", "Invalid worktree ID", err.Error())
	case errors.As(err, &invalidTerminalID):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_terminal_id", "Invalid terminal ID", err.Error())
	case errors.As(err, &agentNotConfigured):
		writeProblem(w, http.StatusUnprocessableEntity, "agent_not_configured", "Agent is not configured", err.Error())
	case errors.As(err, &terminalInputRequired):
		writeProblem(w, http.StatusUnprocessableEntity, "terminal_input_required", "Terminal input is required", err.Error())
	case errors.As(err, &invalidInputMode):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_terminal_input_mode", "Invalid terminal input mode", err.Error())
	case errors.As(err, &workspaceNotGitRepository):
		writeProblem(w, http.StatusUnprocessableEntity, "workspace_not_git_repository", "Workspace is not a Git repository", err.Error())
	case errors.As(err, &invalidReviewScope):
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_review_scope", "Invalid review scope", err.Error())
	default:
		log.Printf("request failed: %v", err)
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error", fallbackDetail)
	}
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
