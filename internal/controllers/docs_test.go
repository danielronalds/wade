package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wade/internal/openapi"
)

func TestOpenAPISpecIncludesOperationIDs(t *testing.T) {
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(openapi.JSON(), &spec); err != nil {
		t.Fatalf("expected generated OpenAPI spec to be valid JSON: %v", err)
	}

	expectedOperationIDs := map[string]string{
		"GET /api/openapi.json":                                              "getOpenAPISpec",
		"GET /api/v1/remote-repositories":                                    "listRemoteRepositories",
		"GET /api/v1/repositories/{repositoryId}":                            "getRepository",
		"GET /api/v1/repositories/{repositoryId}/branches":                   "listRepositoryBranches",
		"GET /api/v1/repositories/{repositoryId}/worktrees":                  "listRepositoryWorktrees",
		"POST /api/v1/repositories/{repositoryId}/worktrees":                 "createRepositoryWorktree",
		"DELETE /api/v1/repositories/{repositoryId}/worktrees/{worktreeId}":  "deleteRepositoryWorktree",
		"GET /api/v1/review-snapshots/{snapshotId}":                          "getReviewSnapshot",
		"DELETE /api/v1/review-snapshots/{snapshotId}":                       "deleteReviewSnapshot",
		"GET /api/v1/review-snapshots/{snapshotId}/files/{fileId}/contents":  "getReviewSnapshotFileContents",
		"GET /api/v1/settings":                                               "getSettings",
		"PUT /api/v1/settings":                                               "updateSettings",
		"POST /api/v1/settings/reload":                                       "reloadSettings",
		"GET /api/v1/workspaces":                                             "listWorkspaces",
		"POST /api/v1/workspaces":                                            "materialiseWorkspace",
		"GET /api/v1/workspaces/{workspaceId}":                               "getWorkspace",
		"POST /api/v1/workspaces/{workspaceId}/start":                        "startWorkspace",
		"POST /api/v1/workspaces/{workspaceId}/review-snapshots":             "createReviewSnapshot",
		"GET /api/v1/workspaces/{workspaceId}/terminals":                     "listWorkspaceTerminals",
		"DELETE /api/v1/workspaces/{workspaceId}/terminals":                  "deleteWorkspaceTerminals",
		"GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}":        "getWorkspaceTerminal",
		"PUT /api/v1/workspaces/{workspaceId}/terminals/{terminalId}":        "putWorkspaceTerminal",
		"DELETE /api/v1/workspaces/{workspaceId}/terminals/{terminalId}":     "deleteWorkspaceTerminal",
		"POST /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input": "sendWorkspaceTerminalInput",
		"GET /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket": "connectWorkspaceTerminal",
	}

	for endpoint, expectedOperationID := range expectedOperationIDs {
		method, path, found := strings.Cut(endpoint, " ")
		if !found {
			t.Fatalf("invalid endpoint key %q", endpoint)
		}

		operation, ok := spec.Paths[path][strings.ToLower(method)]
		if !ok {
			t.Fatalf("expected operation %s", endpoint)
		}

		if operation.OperationID != expectedOperationID {
			t.Fatalf("expected operation %s to have ID %q, got %q", endpoint, expectedOperationID, operation.OperationID)
		}
	}
}

func TestOpenAPIDocsHandlerRedirectsDocsRoot(t *testing.T) {
	handler := newOpenAPIDocsHandler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/docs/", nil)

	handler(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, recorder.Code)
	}

	if location := recorder.Header().Get("Location"); location != "/api/docs/index.html" {
		t.Fatalf("expected redirect to docs index, got %q", location)
	}
}

func TestOpenAPIDocsHandlerServesSwaggerUI(t *testing.T) {
	handler := newOpenAPIDocsHandler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/docs/index.html", nil)

	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `\/api\/openapi.json`) {
		t.Fatal("expected docs page to load generated OpenAPI spec")
	}

	if !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatal("expected docs page to render Swagger UI")
	}
}
