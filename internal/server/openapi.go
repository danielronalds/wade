package server

import (
	"net/http"

	"wade/internal/server/openapi"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @Summary Get OpenAPI spec
// @Tags OpenAPI
// @Produce json
// @Success 200 {object} object
// @Router /api/openapi.json [get]
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openapi.JSON())
}

func newOpenAPIDocsHandler() http.HandlerFunc {
	docsHandler := httpSwagger.Handler(
		httpSwagger.URL("/api/openapi.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
	)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/docs" || r.URL.Path == "/api/docs/" {
			http.Redirect(w, r, "/api/docs/index.html", http.StatusFound)
			return
		}

		docsHandler(w, r)
	}
}
