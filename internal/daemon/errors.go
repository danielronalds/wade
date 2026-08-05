package daemon

import "fmt"

// NotRunningError indicates that no managed daemon is reachable.
type NotRunningError struct{}

func (NotRunningError) Error() string {
	return "WADE is not running"
}

// AlreadyRunningError identifies the daemon that already owns the control socket.
type AlreadyRunningError struct {
	Status Status
}

func (e AlreadyRunningError) Error() string {
	return fmt.Sprintf("WADE is already running with PID %d", e.Status.PID)
}

// StartupError reports why a detached daemon failed to become ready.
type StartupError struct {
	Message string
	LogPath string
}

func (e StartupError) Error() string {
	if e.LogPath == "" {
		return fmt.Sprintf("failed to start WADE server: %s", e.Message)
	}
	return fmt.Sprintf("failed to start WADE server: %s; see %s", e.Message, e.LogPath)
}

// InvalidControlResponseError reports a malformed daemon control response.
type InvalidControlResponseError struct {
	Message string
}

func (e InvalidControlResponseError) Error() string {
	return fmt.Sprintf("invalid daemon control response: %s", e.Message)
}

// ConnectionTimeoutError indicates that the daemon control request timed out.
type ConnectionTimeoutError struct{}

func (ConnectionTimeoutError) Error() string {
	return "timed out connecting to the WADE daemon"
}

// ShutdownTimeoutError indicates that the daemon did not exit before the deadline.
type ShutdownTimeoutError struct{}

func (ShutdownTimeoutError) Error() string {
	return "timed out waiting for the WADE daemon to stop"
}
