package terminals

// TODO: Review properly

import (
	"sync"
	"sync/atomic"
)

const (
	terminalBufferBytes      = 1024 * 1024
	terminalClientOutputSize = 128
)

type ClientOutputKind string

const (
	ClientOutputKindData        ClientOutputKind = "data"
	ClientOutputKindReplayStart ClientOutputKind = "replayStart"
	ClientOutputKindReplayEnd   ClientOutputKind = "replayEnd"
)

type ClientOutput struct {
	Kind ClientOutputKind
	Data []byte
}

type Terminal struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Role        TerminalRole   `json:"role"`
	Agent       *string        `json:"agent"`
	Status      TerminalStatus `json:"status"`
	SocketURL   string         `json:"socketUrl"`

	key       string
	manager   *Service
	process   *Session
	directory string
	buffer    outputBuffer
	clients   map[*Client]struct{}
	mu        sync.Mutex
	writeMu   sync.Mutex
	closeOnce sync.Once
	closed    atomic.Bool
}

type Client struct {
	terminal *Terminal
	output   chan ClientOutput
	once     sync.Once
}

func (t *Terminal) IsClosed() bool {
	return t.closed.Load()
}

func (t *Terminal) Attach() *Client {
	client := &Client{
		terminal: t,
		output:   make(chan ClientOutput, terminalClientOutputSize),
	}

	t.mu.Lock()
	replay := t.buffer.Bytes()
	closed := t.closed.Load()
	if len(replay) > 0 {
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayStart})
		client.enqueue(ClientOutput{Kind: ClientOutputKindData, Data: replay})
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayEnd})
	}
	if !closed {
		t.clients[client] = struct{}{}
	}
	t.mu.Unlock()

	if closed {
		close(client.output)
	}

	return client
}

func (t *Terminal) Write(data []byte) (int, error) {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	return t.process.Write(data)
}

func (t *Terminal) ApplyControlMessage(data []byte) {
	message, ok := parseControlMessage(data)
	if !ok {
		return
	}

	if message.IsActivate() {
		t.manager.activateAgent(t)
		return
	}
	if !message.IsResize() {
		return
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	_ = t.process.Resize(Size{Cols: message.Cols, Rows: message.Rows})
}

func (t *Terminal) Close() {
	t.closeOnce.Do(func() {
		t.closed.Store(true)
		t.mu.Lock()
		clients := make([]*Client, 0, len(t.clients))
		for client := range t.clients {
			clients = append(clients, client)
			delete(t.clients, client)
		}
		t.mu.Unlock()

		for _, client := range clients {
			client.closeOutput()
		}

		t.process.Close()
		t.manager.remove(t.key, t)
	})
}

func (c *Client) Output() <-chan ClientOutput {
	return c.output
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.terminal.detach(c)
	})
}

func startTerminal(
	manager *Service,
	key string,
	shell string,
	environment WadeEnvironment,
	agentCommand string,
	role TerminalRole,
	workspaceID string,
	directory string,
	terminalID string,
	agent *string,
) (*Terminal, error) {
	process, err := startTerminalProcess(shell, environment, agentCommand, role, directory)
	if err != nil {
		return nil, err
	}

	terminal := &Terminal{
		ID:          terminalID,
		WorkspaceID: workspaceID,
		Role:        role,
		Agent:       agent,
		Status:      TerminalStatusRunning,
		SocketURL:   terminalSocketURL(workspaceID, terminalID),
		key:         key,
		manager:     manager,
		process:     process,
		directory:   directory,
		buffer:      newOutputBuffer(terminalBufferBytes),
		clients:     make(map[*Client]struct{}),
	}

	go terminal.readLoop()

	return terminal, nil
}

func startTerminalProcess(
	shell string,
	environment WadeEnvironment,
	agentCommand string,
	role TerminalRole,
	directory string,
) (*Session, error) {
	if role == TerminalRoleAgent && agentCommand != "" {
		return StartShellCommand(shell, directory, environment, agentCommand, Size{Cols: 80, Rows: 24})
	}

	return Start(shell, directory, environment, Size{Cols: 80, Rows: 24})
}

func (t *Terminal) readLoop() {
	buffer := make([]byte, 4096)

	for {
		bytesRead, err := t.process.Read(buffer)
		if err != nil {
			t.Close()
			return
		}

		t.broadcast(buffer[:bytesRead])
	}
}

func (t *Terminal) broadcast(data []byte) {
	message := append([]byte(nil), data...)

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed.Load() {
		return
	}

	t.buffer.Write(message)
	for client := range t.clients {
		client.send(ClientOutput{Kind: ClientOutputKindData, Data: message})
	}
}

func (t *Terminal) detach(client *Client) {
	t.mu.Lock()
	_, attached := t.clients[client]
	delete(t.clients, client)
	t.mu.Unlock()

	if attached {
		client.closeOutput()
	}
}

func (c *Client) enqueue(output ClientOutput) {
	c.output <- cloneClientOutput(output)
}

func (c *Client) send(output ClientOutput) {
	defer func() {
		_ = recover()
	}()

	select {
	case c.output <- cloneClientOutput(output):
	default:
	}
}

func cloneClientOutput(output ClientOutput) ClientOutput {
	return ClientOutput{
		Kind: output.Kind,
		Data: append([]byte(nil), output.Data...),
	}
}

func (c *Client) closeOutput() {
	defer func() {
		_ = recover()
	}()

	close(c.output)
}
