package manager

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const agentTerminalName = "agent"

type Agent struct {
	Name    string
	Command string
	Default bool
}

type Manager struct {
	shell    string
	agents   []Agent
	mu       sync.Mutex
	sessions map[string]*ProjectSession
}

func New(shell string, agents []Agent) *Manager {
	return &Manager{
		shell:    shell,
		agents:   cloneAgents(agents),
		sessions: make(map[string]*ProjectSession),
	}
}

func (m *Manager) Configure(shell string, agents []Agent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shell = shell
	m.agents = cloneAgents(agents)
}

func (m *Manager) GetOrStart(terminalName string, agentName string, directory string) (*ProjectSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := terminalSessionKey(directory, terminalName, m.agentKeyName(terminalName, agentName))
	if session, ok := m.sessions[key]; ok && !session.IsClosed() {
		return session, nil
	}

	// Closed sessions remove themselves asynchronously, so clear stale entries before
	// starting a replacement.
	delete(m.sessions, key)

	agentCommand, err := m.agentCommand(terminalName, agentName)
	if err != nil {
		return nil, err
	}

	session, err := startProjectSession(m, key, m.shell, agentCommand, terminalName, directory)
	if err != nil {
		return nil, err
	}

	m.sessions[key] = session
	return session, nil
}

func (m *Manager) CloseTerminal(terminalName string, agentName string, directory string) bool {
	return m.CloseSession(terminalSessionKey(directory, terminalName, m.agentKeyName(terminalName, agentName)))
}

func (m *Manager) CloseSession(key string) bool {
	m.mu.Lock()
	session, ok := m.sessions[key]
	m.mu.Unlock()

	if !ok || session.IsClosed() {
		return false
	}

	session.Close()
	return true
}

func (m *Manager) CloseSessionsForDirectory(directory string) int {
	m.mu.Lock()
	sessions := make([]*ProjectSession, 0)
	prefix := directory + "\x00"
	for key, session := range m.sessions {
		if strings.HasPrefix(key, prefix) && !session.IsClosed() {
			sessions = append(sessions, session)
		}
	}
	m.mu.Unlock()

	for _, session := range sessions {
		session.Close()
	}

	return len(sessions)
}

func (m *Manager) ActiveDirectories() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	directories := make(map[string]struct{})
	for key, session := range m.sessions {
		if session.IsClosed() {
			continue
		}

		directory, _, ok := strings.Cut(key, "\x00")
		if !ok || directory == "" {
			continue
		}

		directories[directory] = struct{}{}
	}

	activeDirectories := make([]string, 0, len(directories))
	for directory := range directories {
		activeDirectories = append(activeDirectories, directory)
	}

	sort.Strings(activeDirectories)

	return activeDirectories
}

func (m *Manager) remove(key string, session *ProjectSession) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sessions[key] == session {
		delete(m.sessions, key)
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	sessions := make([]*ProjectSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.sessions = make(map[string]*ProjectSession)
	m.mu.Unlock()

	for _, session := range sessions {
		session.Close()
	}
}

func (m *Manager) agentCommand(terminalName string, agentName string) (string, error) {
	if terminalName != agentTerminalName {
		return "", nil
	}

	resolvedAgentName := strings.TrimSpace(agentName)
	if resolvedAgentName == "" {
		resolvedAgentName = m.defaultAgentName()
	}

	for _, agent := range m.agents {
		if strings.EqualFold(agent.Name, resolvedAgentName) {
			return agent.Command, nil
		}
	}

	return "", fmt.Errorf("agent %q is not configured", resolvedAgentName)
}

func (m *Manager) agentKeyName(terminalName string, agentName string) string {
	if terminalName != agentTerminalName {
		return ""
	}

	resolvedAgentName := strings.TrimSpace(agentName)
	if resolvedAgentName == "" {
		resolvedAgentName = m.defaultAgentName()
	}

	return strings.ToLower(resolvedAgentName)
}

func (m *Manager) defaultAgentName() string {
	for _, agent := range m.agents {
		if agent.Default {
			return agent.Name
		}
	}

	if len(m.agents) == 0 {
		return ""
	}

	return m.agents[0].Name
}

func terminalSessionKey(directory string, terminalName string, agentName string) string {
	if terminalName == "" {
		terminalName = "terminal"
	}

	if terminalName == agentTerminalName && agentName != "" {
		return directory + "\x00" + terminalName + "\x00" + strings.ToLower(agentName)
	}

	return directory + "\x00" + terminalName
}

func cloneAgents(agents []Agent) []Agent {
	cloned := make([]Agent, len(agents))
	copy(cloned, agents)
	return cloned
}
