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
	"sync"
	"syscall"
	"time"

	"wade/internal/app"
	"wade/internal/daemon"
	"wade/internal/services/config"
	"wade/internal/web"
)

const (
	ServerCommand = "server"
	StatusCommand = "status"
	StopCommand   = "stop"

	foregroundFlag = "--foreground"
)

type daemonLifecycle interface {
	Acquire(address string) (*daemon.ControlServer, error)
	Start(foregroundCommand ...string) (daemon.Status, error)
	Status() (daemon.Status, error)
	Stop() error
}

type Controller struct {
	stdout io.Writer
	daemon daemonLifecycle
}

func NewController(stdout io.Writer) Controller {
	return Controller{stdout: stdout, daemon: daemon.NewManager()}
}

func (c Controller) HandleArgs(args []string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("usage: wade server [%s]", foregroundFlag)
	}

	switch args[0] {
	case ServerCommand:
		return c.handleServer(args)
	case StatusCommand:
		return c.handleStatus(args)
	case StopCommand:
		return c.handleStop(args)
	default:
		return 0, fmt.Errorf("unsupported server lifecycle command: %s", args[0])
	}
}

func (c Controller) handleServer(args []string) (int, error) {
	if len(args) == 2 && args[1] == foregroundFlag {
		return 0, c.runForeground()
	}
	if len(args) != 1 {
		return 0, fmt.Errorf("usage: wade server [%s]", foregroundFlag)
	}

	status, err := c.daemon.Start(ServerCommand, foregroundFlag)
	var alreadyRunningError daemon.AlreadyRunningError
	if errors.As(err, &alreadyRunningError) {
		_, writeError := fmt.Fprintf(
			c.stdout,
			"WADE is already running\nPID: %d\nAddress: %s\n",
			alreadyRunningError.Status.PID,
			alreadyRunningError.Status.Address,
		)
		return 0, writeError
	}
	if err != nil {
		return 0, err
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"WADE server listening on %s\nPID: %d\nLog: %s\n",
		status.Address,
		status.PID,
		status.LogPath,
	)
	return 0, err
}

func (c Controller) handleStatus(args []string) (int, error) {
	if len(args) != 1 {
		return 0, errors.New("usage: wade status")
	}

	status, err := c.daemon.Status()
	var notRunningError daemon.NotRunningError
	if errors.As(err, &notRunningError) {
		_, writeError := fmt.Fprintln(c.stdout, "WADE is not running")
		return 1, writeError
	}
	if err != nil {
		return 0, err
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"WADE is running\nPID: %d\nAddress: %s\nLog: %s\n",
		status.PID,
		status.Address,
		status.LogPath,
	)
	return 0, err
}

func (c Controller) handleStop(args []string) (int, error) {
	if len(args) != 1 {
		return 0, errors.New("usage: wade stop")
	}

	err := c.daemon.Stop()
	var notRunningError daemon.NotRunningError
	if errors.As(err, &notRunningError) {
		_, writeError := fmt.Fprintln(c.stdout, "WADE is not running")
		return 0, writeError
	}
	if err != nil {
		return 0, err
	}

	_, err = fmt.Fprintln(c.stdout, "WADE stopped")
	return 0, err
}

func (c Controller) runForeground() (runError error) {
	reporter, err := daemon.ConsumeStartupReporter()
	if err != nil {
		return err
	}
	if reporter != nil {
		defer func() {
			reporter.Close(runError)
		}()
	}

	return c.runServer(reporter)
}

func (c Controller) runServer(reporter *daemon.StartupReporter) error {
	configuration, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	staticFiles, err := web.Files()
	if err != nil {
		return fmt.Errorf("failed to load web assets: %w", err)
	}

	var controlServer *daemon.ControlServer
	if reporter != nil {
		controlServer, err = c.daemon.Acquire(configuration.Address)
		var alreadyRunningError daemon.AlreadyRunningError
		if errors.As(err, &alreadyRunningError) {
			return reporter.ReportAlreadyRunning(alreadyRunningError.Status)
		}
		if err != nil {
			return err
		}
		defer func() {
			if err := controlServer.Close(); err != nil {
				log.Printf("daemon control shutdown failed: %v", err)
			}
		}()
	}

	listener, err := net.Listen("tcp", configuration.Address)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", configuration.Address, err)
	}
	defer listener.Close()

	application := app.New(configuration, staticFiles)
	if controlServer != nil {
		controlServer.MarkReady()
		if err := reporter.ReportReady(controlServer.Status()); err != nil {
			application.Close()
			return fmt.Errorf("reporting server readiness: %w", err)
		}
	}

	return serveServer(configuration.Address, listener, application, controlServer)
}

func serveServer(
	address string,
	listener net.Listener,
	application *app.Application,
	controlServer *daemon.ControlServer,
) error {
	httpServer := &http.Server{Handler: application.Mux}
	closeApplication := sync.OnceFunc(application.Close)
	defer closeApplication()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.Serve(listener)
	}()

	var stopRequests <-chan struct{}
	if controlServer != nil {
		stopRequests = controlServer.StopRequests()
	}

	log.Printf("listening on %s", address)
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignals:
	case <-stopRequests:
	}

	closeApplication()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}

	serveError := <-serverErrors
	if serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
		return serveError
	}
	return nil
}
