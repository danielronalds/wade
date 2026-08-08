package terminals

import (
	"sync"
	"sync/atomic"

	"wade/internal/infrastructure/pty"
)

const (
	terminalBufferBytes      = 1024 * 1024
	terminalClientOutputSize = 128
)

// ClientOutputKind identifies terminal stream framing messages.
type ClientOutputKind string

// Terminal client stream framing kinds.
const (
	ClientOutputKindData        ClientOutputKind = "data"
	ClientOutputKindReplayStart ClientOutputKind = "replayStart"
	ClientOutputKindReplayEnd   ClientOutputKind = "replayEnd"
)

// ClientOutput is one detached terminal stream message.
type ClientOutput struct {
	Kind ClientOutputKind
	Data []byte
}

type terminalProcess struct {
	resource  Terminal
	key       string
	manager   *Model
	process   pty.Process
	buffer    outputBuffer
	clients   map[*terminalClient]struct{}
	mu        sync.Mutex
	writeMu   sync.Mutex
	closeOnce sync.Once
	closed    atomic.Bool
}

type terminalClient struct {
	terminal *terminalProcess
	output   chan ClientOutput
	once     sync.Once
}

// TerminalSession is an explicit live handle for WebSocket streaming.
type TerminalSession struct {
	terminal *terminalProcess
	client   *terminalClient
	once     sync.Once
}

// Output returns detached process output and replay framing messages.
func (session *TerminalSession) Output() <-chan ClientOutput {
	return session.client.output
}

// Write writes raw bytes to the live PTY.
func (session *TerminalSession) Write(data []byte) (int, error) {
	return session.terminal.write(data)
}

// ApplyControlMessage applies a JSON terminal control message.
func (session *TerminalSession) ApplyControlMessage(data []byte) {
	session.terminal.applyControlMessage(data)
}

// Close detaches this session without closing the terminal process.
func (session *TerminalSession) Close() {
	session.once.Do(func() {
		session.client.close()
	})
}

func (terminal *terminalProcess) snapshot() Terminal {
	snapshot := terminal.resource
	if terminal.resource.Agent != nil {
		agent := *terminal.resource.Agent
		snapshot.Agent = &agent
	}
	return snapshot
}

func (terminal *terminalProcess) isClosed() bool {
	return terminal.closed.Load()
}

func (terminal *terminalProcess) attach() *TerminalSession {
	client := &terminalClient{
		terminal: terminal,
		output:   make(chan ClientOutput, terminalClientOutputSize),
	}

	terminal.mu.Lock()
	replay := terminal.buffer.Bytes()
	closed := terminal.closed.Load()
	if len(replay) > 0 {
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayStart})
		client.enqueue(ClientOutput{Kind: ClientOutputKindData, Data: replay})
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayEnd})
	}
	if !closed {
		terminal.clients[client] = struct{}{}
	}
	terminal.mu.Unlock()

	if closed {
		close(client.output)
	}
	return &TerminalSession{terminal: terminal, client: client}
}

func (terminal *terminalProcess) write(data []byte) (int, error) {
	terminal.writeMu.Lock()
	defer terminal.writeMu.Unlock()
	return terminal.process.Write(data)
}

func (terminal *terminalProcess) applyControlMessage(data []byte) {
	message, ok := parseControlMessage(data)
	if !ok {
		return
	}
	if message.IsActivate() {
		terminal.manager.activateAgent(terminal)
		return
	}
	if !message.IsResize() {
		return
	}

	terminal.writeMu.Lock()
	defer terminal.writeMu.Unlock()
	_ = terminal.process.Resize(pty.Size{Cols: message.Cols, Rows: message.Rows})
}

func (terminal *terminalProcess) close() {
	terminal.closeOnce.Do(func() {
		terminal.closed.Store(true)
		terminal.mu.Lock()
		clients := make([]*terminalClient, 0, len(terminal.clients))
		for client := range terminal.clients {
			clients = append(clients, client)
			delete(terminal.clients, client)
		}
		terminal.mu.Unlock()

		for _, client := range clients {
			client.closeOutput()
		}
		terminal.process.Close()
		terminal.manager.remove(terminal.key, terminal)
	})
}

func (terminal *terminalProcess) readLoop() {
	buffer := make([]byte, 4096)
	for {
		bytesRead, err := terminal.process.Read(buffer)
		if err != nil {
			terminal.close()
			return
		}
		terminal.broadcast(buffer[:bytesRead])
	}
}

func (terminal *terminalProcess) broadcast(data []byte) {
	message := append([]byte(nil), data...)
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	if terminal.closed.Load() {
		return
	}

	terminal.buffer.Write(message)
	for client := range terminal.clients {
		client.send(ClientOutput{Kind: ClientOutputKindData, Data: message})
	}
}

func (terminal *terminalProcess) detach(client *terminalClient) {
	terminal.mu.Lock()
	_, attached := terminal.clients[client]
	delete(terminal.clients, client)
	terminal.mu.Unlock()
	if attached {
		client.closeOutput()
	}
}

func (client *terminalClient) close() {
	client.once.Do(func() {
		client.terminal.detach(client)
	})
}

func (client *terminalClient) enqueue(output ClientOutput) {
	client.output <- cloneClientOutput(output)
}

func (client *terminalClient) send(output ClientOutput) {
	defer func() { _ = recover() }()
	select {
	case client.output <- cloneClientOutput(output):
	default:
	}
}

func (client *terminalClient) closeOutput() {
	defer func() { _ = recover() }()
	close(client.output)
}

func cloneClientOutput(output ClientOutput) ClientOutput {
	return ClientOutput{Kind: output.Kind, Data: append([]byte(nil), output.Data...)}
}
