package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// ExtraFiles[0] becomes descriptor 3 after stdin, stdout, and stderr.
	serverReadyFileDescriptor = 3
	serverReadyFileEnv        = "WADE_INTERNAL_SERVER_READY_FD"
	serverStartupTimeout      = 10 * time.Second
)

type backgroundServer struct {
	command     *exec.Cmd
	logPath     string
	readyReader *os.File
}

type serverStartup struct {
	Address string `json:"address,omitempty"`
	Error   string `json:"error,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

type serverStartupResult struct {
	startup serverStartup
	err     error
}

type serverStartupReporter struct {
	file *os.File
}

func (s *backgroundServer) waitForStartup() (serverStartup, string, error) {
	defer s.readyReader.Close()

	startupResults := make(chan serverStartupResult, 1)
	go func() {
		var startup serverStartup
		err := json.NewDecoder(s.readyReader).Decode(&startup)
		startupResults <- serverStartupResult{startup: startup, err: err}
	}()

	timer := time.NewTimer(serverStartupTimeout)
	defer timer.Stop()

	select {
	case result := <-startupResults:
		return s.handleStartupResult(result)
	case <-timer.C:
		s.terminate()
		return serverStartup{}, "", fmt.Errorf("timed out waiting for WADE server to start; see %s", s.logPath)
	}
}

func (s *backgroundServer) handleStartupResult(result serverStartupResult) (serverStartup, string, error) {
	if result.err != nil {
		_ = s.command.Wait()
		return serverStartup{}, "", fmt.Errorf("WADE server exited before starting: %w; see %s", result.err, s.logPath)
	}
	if result.startup.Error != "" {
		_ = s.command.Wait()
		return serverStartup{}, "", fmt.Errorf("failed to start WADE server: %s; see %s", result.startup.Error, s.logPath)
	}
	if result.startup.PID <= 0 || result.startup.Address == "" {
		s.terminate()
		return serverStartup{}, "", fmt.Errorf("WADE server returned an invalid startup response; see %s", s.logPath)
	}
	if err := s.command.Process.Release(); err != nil {
		s.terminate()
		return serverStartup{}, "", fmt.Errorf("releasing WADE server process: %w", err)
	}
	return result.startup, s.logPath, nil
}

func (s *backgroundServer) terminate() {
	_ = s.readyReader.Close()
	_ = s.command.Process.Kill()
	_ = s.command.Wait()
}

func openBackgroundServerLog(homeDirectory string) (string, *os.File, error) {
	stateDirectory := os.Getenv("XDG_STATE_HOME")
	if stateDirectory == "" {
		stateDirectory = filepath.Join(homeDirectory, ".local", "state")
	}
	stateDirectory = filepath.Join(stateDirectory, "wade")
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		return "", nil, fmt.Errorf("creating WADE state directory: %w", err)
	}

	logPath := filepath.Join(stateDirectory, "server.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("opening WADE server log: %w", err)
	}
	return logPath, logFile, nil
}

func backgroundServerCommand(
	executable string,
	homeDirectory string,
	logFile *os.File,
	readyWriter *os.File,
) *exec.Cmd {
	command := exec.Command(executable, "server", foregroundFlag)
	command.Dir = homeDirectory
	command.Stdout = logFile
	command.Stderr = logFile
	command.ExtraFiles = []*os.File{readyWriter}
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	command.Env = backgroundServerEnvironment()
	return command
}

func backgroundServerEnvironment() []string {
	readyFileEnvironmentPrefix := serverReadyFileEnv + "="
	environment := make([]string, 0, len(os.Environ())+1)
	for _, environmentVariable := range os.Environ() {
		if strings.HasPrefix(environmentVariable, readyFileEnvironmentPrefix) {
			continue
		}
		environment = append(environment, environmentVariable)
	}
	return append(environment, readyFileEnvironmentPrefix+strconv.Itoa(serverReadyFileDescriptor))
}

// newServerStartupReporter consumes the pipe descriptor used by a background
// parent to wait for startup success or failure. It clears the internal
// environment variable so terminals spawned by the server do not inherit it.
func newServerStartupReporter() (*serverStartupReporter, error) {
	readyFileDescriptor, found := os.LookupEnv(serverReadyFileEnv)
	if !found {
		return nil, nil
	}
	if err := os.Unsetenv(serverReadyFileEnv); err != nil {
		return nil, fmt.Errorf("clearing server readiness file descriptor: %w", err)
	}

	expectedFileDescriptor := strconv.Itoa(serverReadyFileDescriptor)
	if readyFileDescriptor != expectedFileDescriptor {
		return nil, fmt.Errorf("invalid server readiness file descriptor: %s", readyFileDescriptor)
	}

	readyFile := os.NewFile(serverReadyFileDescriptor, "wade-server-ready")
	return &serverStartupReporter{file: readyFile}, nil
}

func (r *serverStartupReporter) report(startup serverStartup) error {
	if err := json.NewEncoder(r.file).Encode(startup); err != nil {
		return err
	}
	_ = r.file.Close()
	r.file = nil
	return nil
}

func (r *serverStartupReporter) close(runError error) {
	if r.file == nil {
		return
	}
	if runError != nil {
		_ = json.NewEncoder(r.file).Encode(serverStartup{Error: runError.Error()})
	}
	_ = r.file.Close()
	r.file = nil
}
