package server

import (
	"net/http"

	"wade/internal/controllers"
)

// Server owns HTTP routing.
type Server struct {
	Mux *http.ServeMux
}

type Options struct {
	SwaggerEnabled bool
}

// New registers routes for the supplied controllers.
func New(controllers controllers.Controllers, options Options) *Server {
	server := &Server{Mux: http.NewServeMux()}
	server.registerRoutes(controllers, options)
	return server
}
