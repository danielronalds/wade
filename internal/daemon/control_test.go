package daemon

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestControlServerReportsRunningStatus(t *testing.T) {
	manager, server := startReadyControlServer(t)

	status, err := manager.Status()
	if err != nil {
		t.Fatalf("Status() error = %v, want nil", err)
	}
	if status != server.Status() {
		t.Fatalf("Status() = %#v, want %#v", status, server.Status())
	}
}

func TestStatusReturnsNotRunningWithoutSocket(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", shortStateRoot(t))
	manager := NewManager()

	_, err := manager.Status()
	var notRunningError NotRunningError
	if !errors.As(err, &notRunningError) {
		t.Fatalf("Status() error = %v, want NotRunningError", err)
	}
}

func TestAcquireRemovesStaleSocket(t *testing.T) {
	stateRoot := shortStateRoot(t)
	t.Setenv("XDG_STATE_HOME", stateRoot)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}
	if err := paths.ensureStateDirectory(); err != nil {
		t.Fatalf("ensureStateDirectory() error = %v, want nil", err)
	}

	staleListener, err := net.ListenUnix("unix", &net.UnixAddr{Name: paths.SocketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("ListenUnix() error = %v, want nil", err)
	}
	staleListener.SetUnlinkOnClose(false)
	if err := staleListener.Close(); err != nil {
		t.Fatalf("stale listener Close() error = %v, want nil", err)
	}

	manager := NewManager()
	server, err := manager.Acquire("test.localhost:1234")
	if err != nil {
		t.Fatalf("Acquire() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.MarkReady()

	if _, err := manager.Status(); err != nil {
		t.Fatalf("Status() after stale cleanup error = %v, want nil", err)
	}
}

func TestControlServerWaitsForReadiness(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", shortStateRoot(t))
	manager := NewManager()
	manager.controlTimeout = time.Second

	server, err := manager.Acquire("test.localhost:1234")
	if err != nil {
		t.Fatalf("Acquire() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}
	connection, err := net.Dial("unix", paths.SocketPath)
	if err != nil {
		t.Fatalf("Dial() error = %v, want nil", err)
	}
	defer connection.Close()
	if err := json.NewEncoder(connection).Encode(controlRequest{Command: controlCommandStatus}); err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}

	if err := connection.SetReadDeadline(time.Now().Add(25 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v, want nil", err)
	}
	_, err = connection.Read(make([]byte, 1))
	var networkError net.Error
	if !errors.As(err, &networkError) || !networkError.Timeout() {
		t.Fatalf("Read() error = %v, want timeout before readiness", err)
	}

	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clearing read deadline error = %v, want nil", err)
	}
	server.MarkReady()

	var response controlResponse
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v, want nil", err)
	}
	want := controlResponse{
		Status:  controlStatusRunning,
		Version: server.Status().Version,
		PID:     server.Status().PID,
		Address: server.Status().Address,
		LogPath: server.Status().LogPath,
	}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
}

func TestAcquireReportsExistingDaemon(t *testing.T) {
	manager, server := startReadyControlServer(t)

	_, err := manager.Acquire("other.localhost:5678")
	var alreadyRunningError AlreadyRunningError
	if !errors.As(err, &alreadyRunningError) {
		t.Fatalf("Acquire() error = %v, want AlreadyRunningError", err)
	}
	if alreadyRunningError.Status != server.Status() {
		t.Fatalf("existing status = %#v, want %#v", alreadyRunningError.Status, server.Status())
	}
}

func TestConcurrentAcquireCreatesOneOwner(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", shortStateRoot(t))
	manager := NewManager()
	manager.controlTimeout = time.Second

	const attempts = 8
	results := make(chan *ControlServer, attempts)
	errorsChannel := make(chan error, attempts)
	var waitGroup sync.WaitGroup
	for range attempts {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			server, err := manager.Acquire("test.localhost:1234")
			if server != nil {
				server.MarkReady()
			}
			results <- server
			errorsChannel <- err
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsChannel)

	var owner *ControlServer
	for server := range results {
		if server == nil {
			continue
		}
		if owner != nil {
			t.Fatal("concurrent Acquire() created more than one owner")
		}
		owner = server
	}
	if owner == nil {
		t.Fatal("concurrent Acquire() did not create an owner")
	}
	t.Cleanup(func() { _ = owner.Close() })

	alreadyRunningErrors := 0
	for err := range errorsChannel {
		if err == nil {
			continue
		}
		var alreadyRunningError AlreadyRunningError
		if !errors.As(err, &alreadyRunningError) {
			t.Fatalf("Acquire() error = %v, want AlreadyRunningError", err)
		}
		alreadyRunningErrors++
	}
	if alreadyRunningErrors != attempts-1 {
		t.Fatalf("AlreadyRunningError count = %d, want %d", alreadyRunningErrors, attempts-1)
	}
}

func TestManagerStartReportsExistingDaemon(t *testing.T) {
	manager, server := startReadyControlServer(t)

	_, err := manager.Start(testDaemonArgument)
	var alreadyRunningError AlreadyRunningError
	if !errors.As(err, &alreadyRunningError) {
		t.Fatalf("Start() error = %v, want AlreadyRunningError", err)
	}
	if alreadyRunningError.Status != server.Status() {
		t.Fatalf("existing status = %#v, want %#v", alreadyRunningError.Status, server.Status())
	}
}

func TestControlSocketHasRestrictivePermissions(t *testing.T) {
	_, _ = startReadyControlServer(t)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}

	info, err := os.Stat(paths.SocketPath)
	if err != nil {
		t.Fatalf("Stat(socket) error = %v, want nil", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode = %o, want 600", info.Mode().Perm())
	}
}

func TestStatusReturnsConnectionTimeout(t *testing.T) {
	stateRoot := shortStateRoot(t)
	t.Setenv("XDG_STATE_HOME", stateRoot)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}
	if err := paths.ensureStateDirectory(); err != nil {
		t.Fatalf("ensureStateDirectory() error = %v, want nil", err)
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: paths.SocketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("ListenUnix() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	connectionAccepted := make(chan struct{})
	releaseConnection := make(chan struct{})
	connectionClosed := make(chan struct{})
	go func() {
		connection, acceptError := listener.Accept()
		if acceptError != nil {
			return
		}
		defer connection.Close()
		defer close(connectionClosed)
		close(connectionAccepted)
		<-releaseConnection
	}()

	manager := NewManager()
	manager.controlTimeout = 25 * time.Millisecond
	_, err = manager.Status()
	var timeoutError ConnectionTimeoutError
	if !errors.As(err, &timeoutError) {
		t.Fatalf("Status() error = %v, want ConnectionTimeoutError", err)
	}
	<-connectionAccepted
	close(releaseConnection)
	<-connectionClosed
}

func TestStatusRejectsInvalidControlResponse(t *testing.T) {
	stateRoot := shortStateRoot(t)
	t.Setenv("XDG_STATE_HOME", stateRoot)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}
	if err := paths.ensureStateDirectory(); err != nil {
		t.Fatalf("ensureStateDirectory() error = %v, want nil", err)
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: paths.SocketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("ListenUnix() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		connection, acceptError := listener.Accept()
		if acceptError != nil {
			return
		}
		defer connection.Close()
		var request controlRequest
		_ = json.NewDecoder(connection).Decode(&request)
		_ = json.NewEncoder(connection).Encode(controlResponse{Status: controlStatusRunning})
	}()

	manager := NewManager()
	_, err = manager.Status()
	var invalidResponseError InvalidControlResponseError
	if !errors.As(err, &invalidResponseError) {
		t.Fatalf("Status() error = %v, want InvalidControlResponseError", err)
	}
}

func TestControlServerRejectsMalformedAndUnsupportedRequests(t *testing.T) {
	_, _ = startReadyControlServer(t)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}

	tests := []struct {
		name    string
		request string
	}{
		{name: "malformed", request: "not json\n"},
		{name: "unsupported", request: "{\"command\":\"restart\"}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection, err := net.Dial("unix", paths.SocketPath)
			if err != nil {
				t.Fatalf("Dial() error = %v, want nil", err)
			}
			defer connection.Close()
			if _, err := connection.Write([]byte(test.request)); err != nil {
				t.Fatalf("Write() error = %v, want nil", err)
			}

			var response controlResponse
			if err := json.NewDecoder(connection).Decode(&response); err != nil {
				t.Fatalf("Decode() error = %v, want nil", err)
			}
			if response.Status != controlStatusError || response.Error == "" {
				t.Fatalf("response = %#v, want control error", response)
			}
		})
	}
}

func TestStopWaitsForControlServerShutdown(t *testing.T) {
	manager, server := startReadyControlServer(t)
	server.status.PID = 99999999
	manager.shutdownTimeout = time.Second

	shutdownComplete := make(chan struct{})
	go func() {
		<-server.StopRequests()
		_ = server.Close()
		close(shutdownComplete)
	}()

	if err := manager.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}
	<-shutdownComplete
}

func TestStopTimesOutWhenDaemonDoesNotExit(t *testing.T) {
	manager, _ := startReadyControlServer(t)
	manager.shutdownTimeout = 50 * time.Millisecond
	manager.pollInterval = 5 * time.Millisecond

	err := manager.Stop()
	var timeoutError ShutdownTimeoutError
	if !errors.As(err, &timeoutError) {
		t.Fatalf("Stop() error = %v, want ShutdownTimeoutError", err)
	}
}

func TestCloseOnlyRemovesOwnedSocket(t *testing.T) {
	_, server := startReadyControlServer(t)
	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v, want nil", err)
	}

	if err := os.Remove(paths.SocketPath); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	replacementPath := filepath.Join(filepath.Dir(paths.SocketPath), "replacement.sock")
	replacement, err := net.ListenUnix("unix", &net.UnixAddr{Name: replacementPath, Net: "unix"})
	if err != nil {
		t.Fatalf("replacement ListenUnix() error = %v, want nil", err)
	}
	replacement.SetUnlinkOnClose(false)
	t.Cleanup(func() {
		_ = replacement.Close()
		_ = os.Remove(replacementPath)
		_ = os.Remove(paths.SocketPath)
	})
	if err := os.Rename(replacementPath, paths.SocketPath); err != nil {
		t.Fatalf("Rename() error = %v, want nil", err)
	}

	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if _, err := os.Stat(paths.SocketPath); err != nil {
		t.Fatalf("replacement socket was removed: %v", err)
	}
}

func startReadyControlServer(t *testing.T) (*Manager, *ControlServer) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", shortStateRoot(t))

	manager := NewManager()
	server, err := manager.Acquire("test.localhost:1234")
	if err != nil {
		t.Fatalf("Acquire() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.MarkReady()
	return manager, server
}

func shortStateRoot(t *testing.T) string {
	t.Helper()

	// Use /tmp directly to stay within the Unix socket path limit on macOS.
	stateRoot, err := os.MkdirTemp("/tmp", "wade-state-")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(stateRoot) })
	return stateRoot
}
