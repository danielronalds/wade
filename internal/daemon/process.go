package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// ExtraFiles[0] becomes descriptor 3 after stdin, stdout, and stderr.
	serverReadyFileDescriptor = 3
	serverReadyFileEnv        = "WADE_INTERNAL_SERVER_READY_FD"
)

type backgroundProcess struct {
	command     *exec.Cmd
	logPath     string
	readyReader *os.File
	timeout     time.Duration
}

// StartupReporter sends one readiness result to the process that started the daemon.
type StartupReporter struct {
	file *os.File
}

// Start launches the foreground command as a detached managed daemon.
func (m *Manager) Start(foregroundCommand ...string) (Status, error) {
	if len(foregroundCommand) == 0 {
		return Status{}, errors.New("foreground command is required")
	}

	status, err := m.Status()
	if err == nil {
		return Status{}, AlreadyRunningError{Status: status}
	}
	var notRunningError NotRunningError
	if !errors.As(err, &notRunningError) {
		return Status{}, err
	}

	process, err := m.startBackgroundProcess(foregroundCommand)
	if err != nil {
		return Status{}, err
	}
	return process.waitForStartup()
}

// ConsumeStartupReporter claims and clears the inherited readiness descriptor.
func ConsumeStartupReporter() (*StartupReporter, error) {
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
	if readyFile == nil {
		return nil, errors.New("opening server readiness file descriptor")
	}
	return &StartupReporter{file: readyFile}, nil
}

// ReportReady reports that the managed daemon is accepting requests.
func (r *StartupReporter) ReportReady(status Status) error {
	return r.report(startupMessage{Status: &status})
}

// ReportAlreadyRunning reports that another daemon owns the control socket.
func (r *StartupReporter) ReportAlreadyRunning(status Status) error {
	return r.report(startupMessage{Status: &status, AlreadyRunning: true})
}

// Close reports an unreported startup failure and releases the descriptor.
func (r *StartupReporter) Close(runError error) {
	if r == nil || r.file == nil {
		return
	}
	if runError != nil {
		_ = json.NewEncoder(r.file).Encode(startupMessage{Error: runError.Error()})
	}
	_ = r.file.Close()
	r.file = nil
}

func (r *StartupReporter) report(message startupMessage) error {
	if r == nil || r.file == nil {
		return errors.New("server readiness has already been reported")
	}

	err := json.NewEncoder(r.file).Encode(message)
	_ = r.file.Close()
	r.file = nil
	return err
}

func (m *Manager) startBackgroundProcess(foregroundCommand []string) (*backgroundProcess, error) {
	executable, err := m.executablePath()
	if err != nil {
		return nil, fmt.Errorf("finding WADE executable: %w", err)
	}

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	paths, err := ResolvePaths()
	if err != nil {
		return nil, err
	}
	logFile, err := paths.openLog()
	if err != nil {
		return nil, err
	}

	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("creating server readiness pipe: %w", err)
	}

	command := backgroundCommand(executable, foregroundCommand, homeDirectory, logFile, readyWriter)
	if err := command.Start(); err != nil {
		_ = readyReader.Close()
		_ = readyWriter.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("starting WADE server: %w", err)
	}
	_ = readyWriter.Close()
	_ = logFile.Close()

	return &backgroundProcess{
		command:     command,
		logPath:     paths.LogPath,
		readyReader: readyReader,
		timeout:     m.startupTimeout,
	}, nil
}

func (p *backgroundProcess) waitForStartup() (Status, error) {
	defer p.readyReader.Close()

	startupResults := make(chan startupResult, 1)
	go func() {
		var message startupMessage
		err := json.NewDecoder(p.readyReader).Decode(&message)
		startupResults <- startupResult{message: message, err: err}
	}()

	timer := time.NewTimer(p.timeout)
	defer timer.Stop()

	select {
	case result := <-startupResults:
		return p.handleStartupResult(result)
	case <-timer.C:
		p.terminate()
		return Status{}, StartupError{
			Message: "timed out waiting for WADE server to start",
			LogPath: p.logPath,
		}
	}
}

func (p *backgroundProcess) handleStartupResult(result startupResult) (Status, error) {
	if result.err != nil {
		p.terminate()
		return Status{}, StartupError{
			Message: fmt.Sprintf("server exited before starting: %v", result.err),
			LogPath: p.logPath,
		}
	}
	if result.message.Error != "" {
		p.terminate()
		return Status{}, StartupError{Message: result.message.Error, LogPath: p.logPath}
	}
	if result.message.Status == nil {
		p.terminate()
		return Status{}, StartupError{Message: "server returned an invalid startup response", LogPath: p.logPath}
	}
	if err := validateStatus(*result.message.Status); err != nil {
		p.terminate()
		return Status{}, StartupError{
			Message: fmt.Sprintf("server returned an invalid startup response: %v", err),
			LogPath: p.logPath,
		}
	}
	if result.message.AlreadyRunning {
		p.terminate()
		return Status{}, AlreadyRunningError{Status: *result.message.Status}
	}
	if err := p.command.Process.Release(); err != nil {
		p.terminate()
		return Status{}, fmt.Errorf("releasing WADE server process: %w", err)
	}
	return *result.message.Status, nil
}

func (p *backgroundProcess) terminate() {
	_ = p.readyReader.Close()
	_ = p.command.Process.Kill()
	_ = p.command.Wait()
}

func backgroundCommand(
	executable string,
	foregroundCommand []string,
	homeDirectory string,
	logFile *os.File,
	readyWriter *os.File,
) *exec.Cmd {
	command := exec.Command(executable, foregroundCommand...)
	command.Dir = homeDirectory
	command.Stdout = logFile
	command.Stderr = logFile
	command.ExtraFiles = []*os.File{readyWriter}
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	command.Env = backgroundEnvironment()
	return command
}

func backgroundEnvironment() []string {
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
