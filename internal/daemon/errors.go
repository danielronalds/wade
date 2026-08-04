package daemon

import "fmt"

type NotRunningError struct{}

func (NotRunningError) Error() string {
	return "WADE is not running"
}

type AlreadyRunningError struct {
	Status Status
}

func (e AlreadyRunningError) Error() string {
	return fmt.Sprintf("WADE is already running with PID %d", e.Status.PID)
}

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

type InvalidControlResponseError struct {
	Message string
}

func (e InvalidControlResponseError) Error() string {
	return fmt.Sprintf("invalid daemon control response: %s", e.Message)
}

type ConnectionTimeoutError struct{}

func (ConnectionTimeoutError) Error() string {
	return "timed out connecting to the WADE daemon"
}

type ShutdownTimeoutError struct{}

func (ShutdownTimeoutError) Error() string {
	return "timed out waiting for the WADE daemon to stop"
}
