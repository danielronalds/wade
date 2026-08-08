package terminals

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"wade/internal/infrastructure/pty"
)

type terminalWorkspaceDiscoveryStub struct {
	path  string
	found bool
}

func (stub terminalWorkspaceDiscoveryStub) Resolve(string) (string, bool, error) {
	return stub.path, stub.found, nil
}

type fakePTY struct {
	mu         sync.Mutex
	startCount int
	processes  []*fakeProcess
}

func (pty *fakePTY) StartInteractive(string, string, pty.WadeEnvironment, pty.Size) (pty.Process, error) {
	return pty.start(), nil
}

func (pty *fakePTY) StartCommand(string, string, pty.WadeEnvironment, string, pty.Size) (pty.Process, error) {
	return pty.start(), nil
}

func (pty *fakePTY) start() *fakeProcess {
	pty.mu.Lock()
	defer pty.mu.Unlock()
	process := &fakeProcess{output: make(chan []byte, 8), closed: make(chan struct{})}
	pty.startCount++
	pty.processes = append(pty.processes, process)
	return process
}

type fakeProcess struct {
	mu      sync.Mutex
	writes  []byte
	output  chan []byte
	closed  chan struct{}
	closeMu sync.Once
}

func (process *fakeProcess) Read(data []byte) (int, error) {
	select {
	case output := <-process.output:
		return copy(data, output), nil
	case <-process.closed:
		return 0, io.EOF
	}
}

func (process *fakeProcess) Write(data []byte) (int, error) {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.writes = append(process.writes, data...)
	return len(data), nil
}

func (*fakeProcess) Resize(pty.Size) error { return nil }

func (process *fakeProcess) Close() {
	process.closeMu.Do(func() { close(process.closed) })
}

func TestModelPutIsIdempotentAndReturnsDetachedResources(t *testing.T) {
	pty := &fakePTY{}
	model := newTerminalTestModel(pty)
	defer model.Close()

	first, created, err := model.Put(context.Background(), "wade", "agent:pi")
	if err != nil || !created {
		t.Fatalf("first Put() = %#v/%v, error = %v", first, created, err)
	}
	second, created, err := model.Put(context.Background(), "wade", "AGENT:PI")
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("second Put() = %#v/%v, error = %v", second, created, err)
	}
	if pty.startCount != 1 {
		t.Fatalf("PTY starts = %d, want 1", pty.startCount)
	}

	*first.Agent = "changed"
	loaded, err := model.Get(context.Background(), "wade", "agent:pi")
	if err != nil || loaded.Agent == nil || *loaded.Agent != "Pi" {
		t.Fatalf("Get() = %#v, error = %v", loaded, err)
	}
}

func TestModelConcurrentPutResolvesToOnePTY(t *testing.T) {
	pty := &fakePTY{}
	model := newTerminalTestModel(pty)
	defer model.Close()

	var wait sync.WaitGroup
	for range 10 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, _, err := model.Put(context.Background(), "wade", "misc"); err != nil {
				t.Errorf("Put() error = %v", err)
			}
		}()
	}
	wait.Wait()
	if pty.startCount != 1 {
		t.Fatalf("PTY starts = %d, want 1", pty.startCount)
	}
}

func TestModelConnectStreamsOutputAndWritesInput(t *testing.T) {
	pty := &fakePTY{}
	model := newTerminalTestModel(pty)
	defer model.Close()
	if _, _, err := model.Put(context.Background(), "wade", "misc"); err != nil {
		t.Fatal(err)
	}

	session, err := model.Connect(context.Background(), "wade", "misc")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	pty.processes[0].output <- []byte("hello")

	select {
	case output := <-session.Output():
		if string(output.Data) != "hello" {
			t.Fatalf("output = %#v", output)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal output")
	}

	if _, err := session.Write([]byte("input")); err != nil {
		t.Fatal(err)
	}
	if string(pty.processes[0].writes) != "input" {
		t.Fatalf("writes = %q", pty.processes[0].writes)
	}
}

func TestModelDeleteAllRemovesWorkspaceActivity(t *testing.T) {
	pty := &fakePTY{}
	model := newTerminalTestModel(pty)
	if _, _, err := model.Put(context.Background(), "wade", "misc"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := model.Put(context.Background(), "wade", "server"); err != nil {
		t.Fatal(err)
	}

	deleted, err := model.DeleteAll(context.Background(), "wade")
	if err != nil || deleted != 2 {
		t.Fatalf("DeleteAll() = %d, error = %v", deleted, err)
	}
	if model.ActiveTerminalCount("wade") != 0 || len(model.ActiveWorkspaceIDs()) != 0 {
		t.Fatal("workspace remains active after DeleteAll()")
	}
}

func newTerminalTestModel(pty *fakePTY) *Model {
	return New(
		terminalWorkspaceDiscoveryStub{path: "/tmp/wade", found: true},
		pty,
		Configuration{
			Shell:         "/bin/sh",
			ServerAddress: "editor.localhost:8765",
			Agents:        []Agent{{Name: "Pi", Command: "pi", Default: true}},
		},
	)
}
