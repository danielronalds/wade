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
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)

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
