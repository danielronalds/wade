package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wade/internal/services/gitrepositories"
	"wade/internal/services/remoterepositories"
	"wade/internal/services/review"
	"wade/internal/services/terminals"
	"wade/internal/services/workspaces"
	"wade/internal/services/worktrees"
)

func TestWriteServiceErrorProblems(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"workspace not found": {
			err:        workspaces.WorkspaceNotFoundError{WorkspaceID: "missing"},
			wantStatus: http.StatusNotFound,
			wantCode:   "workspace_not_found",
		},
		"repository conflict": {
			err:        gitrepositories.RepositoryIDConflictError{RepositoryID: "wade"},
			wantStatus: http.StatusConflict,
			wantCode:   "repository_id_conflict",
		},
		"workspace exists": {
			err:        remoterepositories.WorkspaceAlreadyExistsError{WorkspaceID: "wade"},
			wantStatus: http.StatusConflict,
			wantCode:   "workspace_already_exists",
		},
		"worktree not removable": {
			err:        worktrees.WorktreeNotRemovableError{WorktreeID: "wade"},
			wantStatus: http.StatusConflict,
			wantCode:   "worktree_not_removable",
		},
		"invalid terminal": {
			err:        terminals.InvalidTerminalIDError{TerminalID: "unknown"},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "invalid_terminal_id",
		},
		"snapshot not found": {
			err:        review.SnapshotNotFoundError{SnapshotID: "missing"},
			wantStatus: http.StatusNotFound,
			wantCode:   "review_snapshot_not_found",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeServiceError(response, test.err, "Unexpected failure.")

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", contentType)
			}

			var problem Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decoding problem response: %v", err)
			}
			if problem.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", problem.Code, test.wantCode)
			}
			if problem.Status != test.wantStatus {
				t.Fatalf("problem status = %d, want %d", problem.Status, test.wantStatus)
			}
		})
	}
}
