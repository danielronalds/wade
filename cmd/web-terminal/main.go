package main

import (
	"log"
	"net/http"

	"web-terminal/config"
	"web-terminal/server"
	"web-terminal/web"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	staticFiles, err := web.Files()
	if err != nil {
		log.Fatalf("failed to load web assets: %v", err)
	}

	handler := server.New(configuration, staticFiles)

	log.Printf("open http://%s", configuration.Address)
	log.Fatal(http.ListenAndServe(configuration.Address, handler))
}
