package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wade/internal/server/openapi"
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
		"GET /api/config":                    "getSettings",
		"POST /api/config":                   "updateSettings",
		"POST /api/config/reload":            "reloadConfig",
		"GET /api/openapi.json":              "getOpenAPISpec",
		"GET /api/project":                   "getProjectDetails",
		"GET /api/projects":                  "listProjects",
		"GET /api/remote-projects":           "listRemoteProjects",
		"POST /api/remote-projects/clone":    "cloneRemoteProject",
		"GET /api/review":                    "getReviewWindowData",
		"POST /api/review/file":              "getReviewFileContents",
		"DELETE /api/session/{sessionName}":  "closeProjectSession",
		"GET /api/sessions":                  "listActiveProjectSessions",
		"POST /api/terminal/reload":          "reloadTerminalSession",
		"GET /api/worktrees":                 "listWorktrees",
		"POST /api/worktrees":                "createWorktree",
		"DELETE /api/worktrees":              "removeWorktree",
		"GET /api/worktrees/remote-branches": "listRemoteBranches",
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

func TestOpenAPIDocsHandlerRedirectsDocsRoots(t *testing.T) {
	handler := newOpenAPIDocsHandler()

	for _, path := range []string{"/api/docs", "/api/docs/"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)

		handler(recorder, request)

		if recorder.Code != http.StatusFound {
			t.Fatalf("expected %s to return %d, got %d", path, http.StatusFound, recorder.Code)
		}

		if location := recorder.Header().Get("Location"); location != "/api/docs/index.html" {
			t.Fatalf("expected %s to redirect to docs index, got %q", path, location)
		}
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
