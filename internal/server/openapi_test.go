package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
