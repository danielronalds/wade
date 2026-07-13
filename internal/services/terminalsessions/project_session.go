package terminalsessions

// TODO: Review properly

import (
	"sync"
	"sync/atomic"
)

const (
	projectSessionBufferBytes = 1024 * 1024
	projectClientOutputSize   = 128
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

type ProjectSession struct {
	key          string
	manager      *Service
	session      *Session
	terminalName string
	directory    string
	buffer       outputBuffer
	clients      map[*Client]struct{}
	mu           sync.Mutex
	writeMu      sync.Mutex
	closeMux     sync.Once
	closed       atomic.Bool
}

type Client struct {
	session *ProjectSession
	output  chan ClientOutput
	once    sync.Once
}

func (s *ProjectSession) IsClosed() bool {
	return s.closed.Load()
}

func (s *ProjectSession) Attach() *Client {
	client := &Client{
		session: s,
		output:  make(chan ClientOutput, projectClientOutputSize),
	}

	s.mu.Lock()
	replay := s.buffer.Bytes()
	closed := s.closed.Load()
	if len(replay) > 0 {
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayStart})
		client.enqueue(ClientOutput{Kind: ClientOutputKindData, Data: replay})
		client.enqueue(ClientOutput{Kind: ClientOutputKindReplayEnd})
	}
	if !closed {
		s.clients[client] = struct{}{}
	}
	s.mu.Unlock()

	if closed {
		close(client.output)
	}

	return client
}

func (s *ProjectSession) Write(data []byte) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	return s.session.Write(data)
}

func (s *ProjectSession) ApplyControlMessage(data []byte) {
	message, ok := parseControlMessage(data)
	if !ok {
		return
	}

	if message.IsActivate() {
		s.manager.activateAgent(s)
		return
	}
	if !message.IsResize() {
		return
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	_ = s.session.Resize(Size{Cols: message.Cols, Rows: message.Rows})
}

func (s *ProjectSession) Close() {
	s.closeMux.Do(func() {
		s.closed.Store(true)
		s.mu.Lock()
		clients := make([]*Client, 0, len(s.clients))
		for client := range s.clients {
			clients = append(clients, client)
			delete(s.clients, client)
		}
		s.mu.Unlock()

		for _, client := range clients {
			client.closeOutput()
		}

		s.session.Close()
		s.manager.remove(s.key, s)
	})
}

func (c *Client) Output() <-chan ClientOutput {
	return c.output
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.session.detach(c)
	})
}

func startProjectSession(manager *Service, key string, shell string, agentCommand string, terminalName string, projectName string, directory string) (*ProjectSession, error) {
	session, err := startTerminalSession(shell, agentCommand, terminalName, projectName, directory)
	if err != nil {
		return nil, err
	}

	projectSession := &ProjectSession{
		key:          key,
		manager:      manager,
		session:      session,
		terminalName: terminalName,
		directory:    directory,
		buffer:       newOutputBuffer(projectSessionBufferBytes),
		clients:      make(map[*Client]struct{}),
	}

	go projectSession.readLoop()

	return projectSession, nil
}

func startTerminalSession(shell string, agentCommand string, terminalName string, projectName string, directory string) (*Session, error) {
	if shouldStartAgentCommand(terminalName, agentCommand) {
		return StartShellCommand(shell, directory, projectName, agentCommand, Size{Cols: 80, Rows: 24})
	}

	return Start(shell, directory, projectName, Size{Cols: 80, Rows: 24})
}

func shouldStartAgentCommand(terminalName string, agentCommand string) bool {
	return terminalName == agentTerminalName && agentCommand != ""
}

func (s *ProjectSession) readLoop() {
	buffer := make([]byte, 4096)

	for {
		bytesRead, err := s.session.Read(buffer)
		if err != nil {
			s.Close()
			return
		}

		s.broadcast(buffer[:bytesRead])
	}
}

func (s *ProjectSession) broadcast(data []byte) {
	message := append([]byte(nil), data...)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	s.buffer.Write(message)
	for client := range s.clients {
		client.send(ClientOutput{Kind: ClientOutputKindData, Data: message})
	}
}

func (s *ProjectSession) detach(client *Client) {
	s.mu.Lock()
	_, attached := s.clients[client]
	delete(s.clients, client)
	s.mu.Unlock()

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
