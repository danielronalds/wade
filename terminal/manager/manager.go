package manager

import "sync"

type Manager struct {
	shell    string
	mu       sync.Mutex
	sessions map[string]*ProjectSession
}

func New(shell string) *Manager {
	return &Manager{
		shell:    shell,
		sessions: make(map[string]*ProjectSession),
	}
}

func (m *Manager) GetOrStart(key string, directory string) (*ProjectSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[key]; ok && !session.IsClosed() {
		return session, nil
	}

	// Closed sessions remove themselves asynchronously, so clear stale entries before
	// starting a replacement.
	delete(m.sessions, key)

	session, err := startProjectSession(m, key, m.shell, directory)
	if err != nil {
		return nil, err
	}

	m.sessions[key] = session
	return session, nil
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
