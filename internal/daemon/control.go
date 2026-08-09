package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

const (
	controlStatusError      = "error"
	controlStatusNotRunning = "not_running"
	controlStatusRunning    = "running"
	maxControlRequestBytes  = 4096
	maxSocketBindAttempts   = 3
)

// ControlServer owns the managed daemon's local lifecycle socket.
type ControlServer struct {
	listener   *net.UnixListener
	status     Status
	socketInfo os.FileInfo
	socketPath string

	acceptDone   chan struct{}
	closed       chan struct{}
	closeOnce    sync.Once
	ready        chan struct{}
	readyOnce    sync.Once
	stopOnce     sync.Once
	stopRequests chan struct{}
	waitGroup    sync.WaitGroup
}

// Acquire claims daemon ownership or reports the daemon that already owns it.
func (m *Manager) Acquire(address string) (*ControlServer, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return nil, err
	}
	if err := paths.ensureStateDirectory(); err != nil {
		return nil, err
	}

	status := Status{Version: m.version, PID: os.Getpid(), Address: address, LogPath: paths.LogPath}
	if err := validateStatus(status); err != nil {
		return nil, fmt.Errorf("creating daemon status: %w", err)
	}

	var lastListenError error
	for range maxSocketBindAttempts {
		listener, listenError := net.ListenUnix("unix", &net.UnixAddr{Name: paths.SocketPath, Net: "unix"})
		lastListenError = listenError
		if listenError != nil {
			if err := m.prepareControlSocketRetry(paths.SocketPath); err != nil {
				return nil, err
			}
			continue
		}

		listener.SetUnlinkOnClose(false)

		socketInfo, err := os.Lstat(paths.SocketPath)
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("inspecting WADE control socket: %w", err)
		}
		if err := os.Chmod(paths.SocketPath, 0o600); err != nil {
			_ = listener.Close()
			_ = removeSocketIfSame(paths.SocketPath, socketInfo)
			return nil, fmt.Errorf("securing WADE control socket: %w", err)
		}

		server := &ControlServer{
			listener:     listener,
			status:       status,
			socketInfo:   socketInfo,
			socketPath:   paths.SocketPath,
			acceptDone:   make(chan struct{}),
			closed:       make(chan struct{}),
			ready:        make(chan struct{}),
			stopRequests: make(chan struct{}),
		}
		go server.serve(m.controlTimeout)
		return server, nil
	}

	return nil, fmt.Errorf("acquiring WADE control socket after %d attempts: %w", maxSocketBindAttempts, lastListenError)
}

// MarkReady allows control requests to observe the running daemon.
func (s *ControlServer) MarkReady() {
	s.readyOnce.Do(func() {
		close(s.ready)
	})
}

// Status returns the identity and paths advertised by the managed daemon.
func (s *ControlServer) Status() Status {
	return s.status
}

// StopRequests is closed when a client requests graceful daemon shutdown.
func (s *ControlServer) StopRequests() <-chan struct{} {
	return s.stopRequests
}

// Close stops control handling and removes the socket if it is still owned.
func (s *ControlServer) Close() error {
	var closeError error
	s.closeOnce.Do(func() {
		close(s.closed)
		closeError = s.listener.Close()
		<-s.acceptDone
		s.waitGroup.Wait()
		if err := removeSocketIfSame(s.socketPath, s.socketInfo); err != nil && closeError == nil {
			closeError = err
		}
	})
	if errors.Is(closeError, net.ErrClosed) {
		return nil
	}
	return closeError
}

func (m *Manager) prepareControlSocketRetry(socketPath string) error {
	socketInfo, err := os.Lstat(socketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspecting WADE control socket after bind failure: %w", err)
	}
	if socketInfo.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("WADE control socket path is not a socket: %s", socketPath)
	}

	existingStatus, err := m.request(socketPath, controlCommandStatus)
	if err == nil {
		return AlreadyRunningError{Status: existingStatus}
	}
	var notRunningError NotRunningError
	if !errors.As(err, &notRunningError) {
		return err
	}
	return removeSocketIfSame(socketPath, socketInfo)
}

func (s *ControlServer) serve(timeout time.Duration) {
	defer close(s.acceptDone)

	for {
		connection, err := s.listener.Accept()
		if err != nil {
			return
		}

		s.waitGroup.Add(1)
		go s.handleConnection(connection, timeout)
	}
}

func (s *ControlServer) handleConnection(connection net.Conn, timeout time.Duration) {
	defer s.waitGroup.Done()
	defer connection.Close()

	_ = connection.SetDeadline(time.Now().Add(timeout))
	decoder := json.NewDecoder(io.LimitReader(connection, maxControlRequestBytes))
	var request controlRequest
	if err := decoder.Decode(&request); err != nil {
		s.writeError(connection, fmt.Sprintf("decoding request: %v", err))
		return
	}
	if request.Command != controlCommandStatus && request.Command != controlCommandStop {
		s.writeError(connection, fmt.Sprintf("unsupported command %q", request.Command))
		return
	}

	select {
	case <-s.ready:
	case <-s.closed:
		_ = json.NewEncoder(connection).Encode(controlResponse{Status: controlStatusNotRunning})
		return
	}

	if request.Command == controlCommandStop {
		s.stopOnce.Do(func() {
			close(s.stopRequests)
		})
	}
	_ = json.NewEncoder(connection).Encode(controlResponse{
		Status:  controlStatusRunning,
		Version: s.status.Version,
		PID:     s.status.PID,
		Address: s.status.Address,
		LogPath: s.status.LogPath,
	})
}

func (s *ControlServer) writeError(connection net.Conn, message string) {
	_ = json.NewEncoder(connection).Encode(controlResponse{Status: controlStatusError, Error: message})
}

func (m *Manager) request(socketPath string, command controlCommand) (Status, error) {
	dialer := net.Dialer{Timeout: m.controlTimeout}
	connection, err := dialer.Dial("unix", socketPath)
	if err != nil {
		return Status{}, controlConnectionError(err)
	}
	defer connection.Close()

	if err := connection.SetDeadline(time.Now().Add(m.controlTimeout)); err != nil {
		return Status{}, fmt.Errorf("setting WADE control connection deadline: %w", err)
	}
	if err := json.NewEncoder(connection).Encode(controlRequest{Command: command}); err != nil {
		return Status{}, controlConnectionError(err)
	}

	var response controlResponse
	if err := json.NewDecoder(io.LimitReader(connection, maxControlRequestBytes)).Decode(&response); err != nil {
		if isTimeout(err) {
			return Status{}, ConnectionTimeoutError{}
		}
		return Status{}, InvalidControlResponseError{Message: err.Error()}
	}

	switch response.Status {
	case controlStatusRunning:
		status := Status{
			Version: response.Version,
			PID:     response.PID,
			Address: response.Address,
			LogPath: response.LogPath,
		}
		if err := validateStatus(status); err != nil {
			return Status{}, InvalidControlResponseError{Message: err.Error()}
		}
		return status, nil
	case controlStatusNotRunning:
		return Status{}, NotRunningError{}
	case controlStatusError:
		if response.Error == "" {
			return Status{}, InvalidControlResponseError{Message: "error response did not include a message"}
		}
		return Status{}, InvalidControlResponseError{Message: response.Error}
	default:
		return Status{}, InvalidControlResponseError{Message: fmt.Sprintf("unknown status %q", response.Status)}
	}
}

func controlConnectionError(err error) error {
	if isTimeout(err) {
		return ConnectionTimeoutError{}
	}
	if os.IsNotExist(err) || errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
		return NotRunningError{}
	}
	return fmt.Errorf("connecting to WADE daemon: %w", err)
}

func isTimeout(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}

func removeSocketIfSame(path string, expected os.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspecting WADE control socket before removal: %w", err)
	}
	if !os.SameFile(expected, current) {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stale WADE control socket: %w", err)
	}
	return nil
}
