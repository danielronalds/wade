package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wade/internal/app"
	"wade/internal/services/config"
	"wade/internal/web"
)

const foregroundFlag = "--foreground"

type Controller struct {
	stdout         io.Writer
	executablePath func() (string, error)
}

func NewController(stdout io.Writer) Controller {
	return Controller{
		stdout:         stdout,
		executablePath: os.Executable,
	}
}

func (c Controller) HandleArgs(args []string) error {
	if len(args) == 2 && args[1] == foregroundFlag {
		return c.runForeground()
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: wade server [%s]", foregroundFlag)
	}

	startup, logPath, err := c.runBackground()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"WADE server listening on %s\nPID: %d\nLog: %s\n",
		startup.Address,
		startup.PID,
		logPath,
	)
	return err
}

func (c Controller) runBackground() (serverStartup, string, error) {
	server, err := c.startBackgroundServer()
	if err != nil {
		return serverStartup{}, "", err
	}

	return server.waitForStartup()
}

func (c Controller) runForeground() (runError error) {
	reporter, err := newServerStartupReporter()
	if err != nil {
		return err
	}
	if reporter == nil {
		return c.runServer(nil)
	}
	defer func() {
		reporter.close(runError)
	}()

	return c.runServer(reporter.report)
}

func (c Controller) runServer(reportReady func(serverStartup) error) error {
	configuration, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	staticFiles, err := web.Files()
	if err != nil {
		return fmt.Errorf("failed to load web assets: %w", err)
	}

	listener, err := net.Listen("tcp", configuration.Address)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", configuration.Address, err)
	}
	defer listener.Close()

	application := app.New(configuration, staticFiles)
	return serveServer(configuration.Address, listener, application, reportReady)
}

func (c Controller) startBackgroundServer() (*backgroundServer, error) {
	executable, err := c.executablePath()
	if err != nil {
		return nil, fmt.Errorf("finding WADE executable: %w", err)
	}

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}

	logPath, logFile, err := openBackgroundServerLog(homeDirectory)
	if err != nil {
		return nil, err
	}

	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("creating server readiness pipe: %w", err)
	}

	command := backgroundServerCommand(executable, homeDirectory, logFile, readyWriter)
	if err := command.Start(); err != nil {
		_ = readyReader.Close()
		_ = readyWriter.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("starting WADE server: %w", err)
	}
	_ = readyWriter.Close()
	_ = logFile.Close()

	return &backgroundServer{
		command:     command,
		logPath:     logPath,
		readyReader: readyReader,
	}, nil
}

func serveServer(
	address string,
	listener net.Listener,
	application *app.Application,
	reportReady func(serverStartup) error,
) error {
	httpServer := &http.Server{Handler: application.Mux}
	defer application.Close()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	serverStopped := make(chan struct{})
	defer close(serverStopped)
	go shutdownServerOnSignal(httpServer, application, shutdownSignals, serverStopped)

	log.Printf("listening on %s", address)
	if reportReady != nil {
		startup := serverStartup{Address: address, PID: os.Getpid()}
		if err := reportReady(startup); err != nil {
			return fmt.Errorf("reporting server readiness: %w", err)
		}
	}

	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdownServerOnSignal(
	httpServer *http.Server,
	application *app.Application,
	shutdownSignals <-chan os.Signal,
	serverStopped <-chan struct{},
) {
	select {
	case <-shutdownSignals:
		application.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	case <-serverStopped:
	}
}
