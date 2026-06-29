package server

import (
	"io/fs"
	"net/http"

	"web-terminal/config"
)

type Server struct {
	configuration config.Config
}

func New(configuration config.Config, staticFiles fs.FS) http.Handler {
	server := &Server{configuration: configuration}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleTerminal)
	mux.Handle("/", http.FileServer(http.FS(staticFiles)))

	return mux
}
