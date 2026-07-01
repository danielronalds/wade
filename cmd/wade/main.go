package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wade/config"
	"wade/server"
	"wade/web"
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

	application := server.New(configuration, staticFiles)
	httpServer := &http.Server{
		Addr:    configuration.Address,
		Handler: application,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	go func() {
		<-shutdownSignals
		application.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()

	log.Printf("open http://%s", configuration.Address)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		application.Close()
		log.Fatal(err)
	}

	application.Close()
}
