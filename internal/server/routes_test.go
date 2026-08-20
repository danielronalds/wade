package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"wade/internal/controllers"
)

func TestSwaggerRoutesAreAlwaysServed(t *testing.T) {
	httpServer := New(testControllers())

	openAPIResponse := serveRequest(httpServer, "/api/openapi.json")
	if openAPIResponse.Code != http.StatusOK {
		t.Fatalf("expected OpenAPI spec status %d, got %d", http.StatusOK, openAPIResponse.Code)
	}

	if !strings.Contains(openAPIResponse.Body.String(), `"swagger": "2.0"`) {
		t.Fatal("expected OpenAPI spec response")
	}

	docsRootResponse := serveRequest(httpServer, "/api/docs")
	if docsRootResponse.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected docs root status %d, got %d", http.StatusTemporaryRedirect, docsRootResponse.Code)
	}

	if location := docsRootResponse.Header().Get("Location"); location != "/api/docs/" {
		t.Fatalf("expected docs root to redirect to /api/docs/, got %q", location)
	}

	docsResponse := serveRequest(httpServer, "/api/docs/index.html")
	if docsResponse.Code != http.StatusOK {
		t.Fatalf("expected Swagger UI status %d, got %d", http.StatusOK, docsResponse.Code)
	}
}

func TestAPIV1RoutesAreRegistered(t *testing.T) {
	httpServer := New(testControllers())
	paths := []string{
		"/api/v1/workspaces",
		"/api/v1/workspaces/wade",
		"/api/v1/workspaces/wade/start",
		"/api/v1/remote-repositories",
		"/api/v1/repositories/wade",
		"/api/v1/repositories/wade/worktrees",
		"/api/v1/repositories/wade/worktrees/wade-feature",
		"/api/v1/repositories/wade/branches",
		"/api/v1/workspaces/wade/terminals",
		"/api/v1/workspaces/wade/terminals/agent:pi",
		"/api/v1/workspaces/wade/terminals/agent:pi/input",
		"/api/v1/workspaces/wade/terminals/agent:pi/socket",
		"/api/v1/workspaces/wade/review-snapshots",
		"/api/v1/review-snapshots/review_snapshot_01",
		"/api/v1/review-snapshots/review_snapshot_01/files/file_01/contents",
		"/api/v1/settings",
		"/api/v1/settings/reload",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			response := serveMethodRequest(httpServer, http.MethodPatch, path)
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("PATCH %s status = %d, want %d", path, response.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestLegacyAPIRoutesAreNotRegistered(t *testing.T) {
	httpServer := New(testControllers())
	paths := []string{
		"/api/projects",
		"/api/project?project=wade",
		"/api/remote-projects",
		"/api/sessions",
		"/api/terminal/reload",
		"/api/worktrees",
		"/api/review?project=wade",
		"/api/config",
		"/ws?project=wade",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			response := serveRequest(httpServer, path)
			if response.Code != http.StatusNotFound {
				t.Fatalf("GET %s status = %d, want %d", path, response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestServiceWorkerIsServedAtApplicationRoot(t *testing.T) {
	httpServer := New(testControllers())

	response := serveRequest(httpServer, "/service-worker.js")
	if response.Code != http.StatusOK {
		t.Fatalf("expected service worker status %d, got %d", http.StatusOK, response.Code)
	}

	if response.Body.String() != "service worker" {
		t.Fatalf("expected service worker response, got %q", response.Body.String())
	}

	if response.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected service worker to bypass the HTTP cache")
	}

	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/javascript") {
		t.Fatalf("expected JavaScript content type, got %q", contentType)
	}
}

func serveRequest(httpServer *Server, path string) *httptest.ResponseRecorder {
	return serveMethodRequest(httpServer, http.MethodGet, path)
}

func serveMethodRequest(httpServer *Server, method string, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)

	httpServer.Mux.ServeHTTP(response, request)

	return response
}

func testControllers() controllers.Controllers {
	return controllers.Controllers{
		Docs: controllers.NewDocs(),
		Page: controllers.NewPage(fstest.MapFS{
			"index.html":        {Data: []byte("application page")},
			"service-worker.js": {Data: []byte("service worker")},
		}),
	}
}
