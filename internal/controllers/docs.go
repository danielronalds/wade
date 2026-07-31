package controllers

// TODO: Review properly

import (
	"net/http"

	"wade/internal/openapi"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Docs struct {
	docsHandler http.HandlerFunc
}

func NewDocs() Docs {
	return Docs{docsHandler: newOpenAPIDocsHandler()}
}

// @Summary Get OpenAPI spec
// @ID getOpenAPISpec
// @Tags OpenAPI
// @Produce json
// @Success 200 {object} object
// @Router /api/openapi.json [get]
func (h Docs) OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openapi.JSON())
}

func (h Docs) OpenAPIDocs(w http.ResponseWriter, r *http.Request) {
	h.docsHandler(w, r)
}

func newOpenAPIDocsHandler() http.HandlerFunc {
	docsHandler := httpSwagger.Handler(
		httpSwagger.URL("/api/openapi.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
	)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/docs/" {
			http.Redirect(w, r, "/api/docs/index.html", http.StatusFound)
			return
		}

		docsHandler(w, r)
	}
}
