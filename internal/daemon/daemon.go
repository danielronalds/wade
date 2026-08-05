package daemon

import (
	"errors"
	"os"
	"syscall"
	"time"
)

const (
	defaultControlTimeout  = 2 * time.Second
	defaultShutdownTimeout = 5 * time.Second
	defaultStartupTimeout  = 10 * time.Second
)

// Manager coordinates the lifecycle of the single managed background daemon.
type Manager struct {
	executablePath  func() (string, error)
	controlTimeout  time.Duration
	shutdownTimeout time.Duration
	startupTimeout  time.Duration
	pollInterval    time.Duration
}

// NewManager creates a daemon manager with the standard lifecycle timeouts.
func NewManager() *Manager {
	return &Manager{
		executablePath:  os.Executable,
		controlTimeout:  defaultControlTimeout,
		shutdownTimeout: defaultShutdownTimeout,
		startupTimeout:  defaultStartupTimeout,
		pollInterval:    25 * time.Millisecond,
	}
}

// Status queries the managed daemon through its control socket.
func (m *Manager) Status() (Status, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return Status{}, err
	}

	return m.request(paths.SocketPath, controlCommandStatus)
}

// Stop requests graceful shutdown and waits for the managed daemon to exit.
func (m *Manager) Stop() error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	status, err := m.request(paths.SocketPath, controlCommandStop)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(m.shutdownTimeout)
	for {
		_, statusError := m.Status()
		var notRunningError NotRunningError
		socketStopped := errors.As(statusError, &notRunningError)

		processIsRunning := false
		process, findError := os.FindProcess(status.PID)
		if findError == nil {
			signalError := process.Signal(syscall.Signal(0))
			processIsRunning = signalError == nil || errors.Is(signalError, syscall.EPERM)
		}

		if socketStopped && !processIsRunning {
			return nil
		}
		if statusError != nil && !socketStopped {
			return statusError
		}
		if !time.Now().Before(deadline) {
			return ShutdownTimeoutError{}
		}
		time.Sleep(m.pollInterval)
	}
}

func validateStatus(status Status) error {
	if status.PID <= 0 {
		return errors.New("PID must be positive")
	}
	if status.Address == "" {
		return errors.New("address is required")
	}
	if status.LogPath == "" {
		return errors.New("log path is required")
	}
	return nil
}
