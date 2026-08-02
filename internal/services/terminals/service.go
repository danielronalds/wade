package terminals

// TODO: Review properly

import (
	"fmt"
	"io"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
)

type WorkspaceRepository interface {
	Path(workspaceID string) (string, error)
	IDForDirectory(directory string) (string, bool, error)
}

type Agent struct {
	Name    string
	Command string
	Default bool
}

type Service struct {
	workspaces            WorkspaceRepository
	serverAddress         string
	mu                    sync.Mutex
	shell                 string
	agents                []Agent
	terminals             map[string]*Terminal
	selectedAgentTerminal map[string]*Terminal
}

func NewService(
	workspaces WorkspaceRepository,
	shell string,
	serverAddress string,
	agents []Agent,
) *Service {
	return &Service{
		workspaces:            workspaces,
		shell:                 shell,
		serverAddress:         serverAddress,
		agents:                cloneAgents(agents),
		terminals:             make(map[string]*Terminal),
		selectedAgentTerminal: make(map[string]*Terminal),
	}
}

func (s *Service) Configure(shell string, agents []Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shell = shell
	s.agents = cloneAgents(agents)
}

func (s *Service) Put(workspaceID string, terminalID string) (*Terminal, bool, error) {
	workspacePath, err := s.workspaces.Path(workspaceID)
	if err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	descriptor, err := resolveTerminalDescriptor(terminalID, s.agents)
	if err != nil {
		return nil, false, err
	}

	key := terminalKey(workspaceID, descriptor.id)
	if terminal, found := s.terminals[key]; found && !terminal.IsClosed() {
		return terminal, false, nil
	}
	delete(s.terminals, key)

	environment := WadeEnvironment{
		WorkspaceID: workspaceID,
		TerminalID:  descriptor.id,
		Address:     s.serverAddress,
	}
	terminal, err := startTerminal(
		s,
		key,
		s.shell,
		environment,
		descriptor.command,
		descriptor.role,
		workspaceID,
		workspacePath,
		descriptor.id,
		descriptor.agent,
	)
	if err != nil {
		return nil, false, err
	}

	s.terminals[key] = terminal
	return terminal, true, nil
}

func (s *Service) Get(workspaceID string, terminalID string) (*Terminal, error) {
	if _, err := s.workspaces.Path(workspaceID); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	descriptor, err := resolveTerminalDescriptor(terminalID, s.agents)
	if err != nil {
		return nil, err
	}

	terminal, found := s.terminals[terminalKey(workspaceID, descriptor.id)]
	if !found || terminal.IsClosed() {
		return nil, TerminalNotFoundError{WorkspaceID: workspaceID, TerminalID: descriptor.id}
	}

	return terminal, nil
}

func (s *Service) List(workspaceID string) ([]*Terminal, error) {
	if _, err := s.workspaces.Path(workspaceID); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := workspaceID + "\x00"
	workspaceTerminals := make([]*Terminal, 0)
	for key, terminal := range s.terminals {
		if strings.HasPrefix(key, prefix) && !terminal.IsClosed() {
			workspaceTerminals = append(workspaceTerminals, terminal)
		}
	}
	sort.Slice(workspaceTerminals, func(firstIndex int, secondIndex int) bool {
		return workspaceTerminals[firstIndex].ID < workspaceTerminals[secondIndex].ID
	})

	return workspaceTerminals, nil
}

func (s *Service) Delete(workspaceID string, terminalID string) error {
	if _, err := s.workspaces.Path(workspaceID); err != nil {
		return err
	}

	s.mu.Lock()
	descriptor, err := resolveTerminalDescriptor(terminalID, s.agents)
	if err != nil {
		s.mu.Unlock()
		return err
	}

	key := terminalKey(workspaceID, descriptor.id)
	terminal, found := s.terminals[key]
	if !found || terminal.IsClosed() {
		s.mu.Unlock()
		return TerminalNotFoundError{WorkspaceID: workspaceID, TerminalID: descriptor.id}
	}

	delete(s.terminals, key)
	if s.selectedAgentTerminal[workspaceID] == terminal {
		delete(s.selectedAgentTerminal, workspaceID)
	}
	s.mu.Unlock()

	terminal.Close()
	return nil
}

func (s *Service) DeleteAll(workspaceID string) int {
	s.mu.Lock()
	prefix := workspaceID + "\x00"
	workspaceTerminals := make([]*Terminal, 0)
	for key, terminal := range s.terminals {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		delete(s.terminals, key)
		if !terminal.IsClosed() {
			workspaceTerminals = append(workspaceTerminals, terminal)
		}
	}
	delete(s.selectedAgentTerminal, workspaceID)
	s.mu.Unlock()

	for _, terminal := range workspaceTerminals {
		terminal.Close()
	}

	return len(workspaceTerminals)
}

func (s *Service) Input(workspaceID string, terminalID string, text string, mode InputMode) error {
	if text == "" {
		return TerminalInputRequiredError{}
	}
	if mode != InputModeRaw && mode != InputModeBracketedPaste {
		return InvalidInputModeError{Mode: mode}
	}

	terminal, err := s.Get(workspaceID, terminalID)
	if err != nil {
		return err
	}

	input := []byte(text)
	if mode == InputModeBracketedPaste {
		input = []byte("\x1b[200~" + text + "\x1b[201~")
	}
	bytesWritten, err := terminal.Write(input)
	if err == nil && bytesWritten != len(input) {
		return io.ErrShortWrite
	}

	return err
}

func (s *Service) ActiveTerminalCount(workspaceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	prefix := workspaceID + "\x00"
	for key, terminal := range s.terminals {
		if strings.HasPrefix(key, prefix) && !terminal.IsClosed() {
			count++
		}
	}
	return count
}

func (s *Service) ActiveWorkspaceIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeWorkspaceIDs := make([]string, 0, len(s.terminals))
	for _, terminal := range s.terminals {
		if !terminal.IsClosed() {
			activeWorkspaceIDs = append(activeWorkspaceIDs, terminal.WorkspaceID)
		}
	}
	sort.Strings(activeWorkspaceIDs)

	return slices.Compact(activeWorkspaceIDs)
}

func (s *Service) CloseTerminalsForDirectory(directory string) int {
	workspaceID, found, err := s.workspaces.IDForDirectory(directory)
	if err != nil || !found {
		return 0
	}

	return s.DeleteAll(workspaceID)
}

func (s *Service) Close() {
	s.mu.Lock()
	activeTerminals := make([]*Terminal, 0, len(s.terminals))
	for _, terminal := range s.terminals {
		activeTerminals = append(activeTerminals, terminal)
	}
	s.terminals = make(map[string]*Terminal)
	s.selectedAgentTerminal = make(map[string]*Terminal)
	s.mu.Unlock()

	for _, terminal := range activeTerminals {
		terminal.Close()
	}
}

func (s *Service) remove(key string, terminal *Terminal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.terminals[key] == terminal {
		delete(s.terminals, key)
	}
	if s.selectedAgentTerminal[terminal.WorkspaceID] == terminal {
		delete(s.selectedAgentTerminal, terminal.WorkspaceID)
	}
}

func (s *Service) activateAgent(terminal *Terminal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isActiveAgentTerminal(terminal, terminal.WorkspaceID) {
		s.selectedAgentTerminal[terminal.WorkspaceID] = terminal
	}
}

func (s *Service) isActiveAgentTerminal(terminal *Terminal, workspaceID string) bool {
	return terminal != nil &&
		terminal.WorkspaceID == workspaceID &&
		terminal.Role == TerminalRoleAgent &&
		s.terminals[terminal.key] == terminal &&
		!terminal.IsClosed()
}

func terminalKey(workspaceID string, terminalID string) string {
	return workspaceID + "\x00" + terminalID
}

func terminalSocketURL(workspaceID string, terminalID string) string {
	return fmt.Sprintf(
		"/api/v1/workspaces/%s/terminals/%s/socket",
		url.PathEscape(workspaceID),
		url.PathEscape(terminalID),
	)
}

func cloneAgents(agents []Agent) []Agent {
	return append([]Agent(nil), agents...)
}
