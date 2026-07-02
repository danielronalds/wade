package manager

import (
	"sync"
	"sync/atomic"

	"wade/terminal"
)

const (
	projectSessionBufferBytes = 1024 * 1024
	projectClientOutputSize   = 128
)

type ProjectSession struct {
	key      string
	manager  *Manager
	session  *terminal.Session
	buffer   outputBuffer
	clients  map[*Client]struct{}
	mu       sync.Mutex
	writeMu  sync.Mutex
	closeMux sync.Once
	closed   atomic.Bool
}

type Client struct {
	session *ProjectSession
	output  chan []byte
	once    sync.Once
}

func startProjectSession(manager *Manager, key string, shell string, agentPaneCommand string, terminalName string, directory string) (*ProjectSession, error) {
	session, err := startTerminalSession(shell, agentPaneCommand, terminalName, directory)
	if err != nil {
		return nil, err
	}

	projectSession := &ProjectSession{
		key:     key,
		manager: manager,
		session: session,
		buffer:  newOutputBuffer(projectSessionBufferBytes),
		clients: make(map[*Client]struct{}),
	}

	go projectSession.readLoop()

	return projectSession, nil
}

func startTerminalSession(shell string, agentPaneCommand string, terminalName string, directory string) (*terminal.Session, error) {
	if shouldStartAgentPaneCommand(terminalName, agentPaneCommand) {
		return terminal.StartShellCommand(shell, directory, agentPaneCommand, terminal.Size{Cols: 80, Rows: 24})
	}

	return terminal.Start(shell, directory, terminal.Size{Cols: 80, Rows: 24})
}

func shouldStartAgentPaneCommand(terminalName string, agentPaneCommand string) bool {
	return terminalName == agentTerminalName && agentPaneCommand != ""
}

func (s *ProjectSession) IsClosed() bool {
	return s.closed.Load()
}

func (s *ProjectSession) Attach() *Client {
	client := &Client{
		session: s,
		output:  make(chan []byte, projectClientOutputSize),
	}

	s.mu.Lock()
	replay := s.buffer.Bytes()
	closed := s.closed.Load()
	if !closed {
		s.clients[client] = struct{}{}
	}
	s.mu.Unlock()

	if len(replay) > 0 {
		client.send(replay)
	}

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
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.session.ApplyControlMessage(data)
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
		client.send(message)
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

func (c *Client) Output() <-chan []byte {
	return c.output
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.session.detach(c)
	})
}

func (c *Client) send(data []byte) {
	defer func() {
		_ = recover()
	}()

	message := append([]byte(nil), data...)
	select {
	case c.output <- message:
	default:
	}
}

func (c *Client) closeOutput() {
	defer func() {
		_ = recover()
	}()

	close(c.output)
}
