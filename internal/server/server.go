package server

import (
	"net/http"

	"wade/internal/controllers"
)

// Server owns HTTP routing.
type Server struct {
	Mux *http.ServeMux
}

// New registers routes for the supplied controllers.
func New(controllers controllers.Controllers) *Server {
	server := &Server{Mux: http.NewServeMux()}
	server.registerRoutes(controllers)
	return server
}
