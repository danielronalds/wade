package sessions

// TODO: Review properly

import "strings"

type ProjectService interface {
	Path(name string) (string, error)
	NamesForDirectories(directories []string) []string
}

type TerminalSessionService interface {
	ActiveDirectories() []string
	CloseSessionsForDirectory(directory string) int
	WriteToActiveAgent(directory string, data []byte) (int, error)
}

type Service struct {
	projects  ProjectService
	terminals TerminalSessionService
}

func NewService(projects ProjectService, terminals TerminalSessionService) Service {
	return Service{projects: projects, terminals: terminals}
}

func (s Service) List() []string {
	return s.projects.NamesForDirectories(s.terminals.ActiveDirectories())
}

func (s Service) Close(sessionName string) error {
	if err := validateSessionName(sessionName); err != nil {
		return err
	}

	projectPath, err := s.projects.Path(strings.TrimSpace(sessionName))
	if err != nil {
		return ErrSessionNotFound
	}

	s.terminals.CloseSessionsForDirectory(projectPath)
	return nil
}

func (s Service) SendToAgent(sessionName string, text string) error {
	if err := validateSessionName(sessionName); err != nil {
		return err
	}
	if err := validateAgentText(text); err != nil {
		return err
	}

	projectPath, err := s.projects.Path(strings.TrimSpace(sessionName))
	if err != nil {
		return ErrSessionNotFound
	}

	const bracketedPasteStart = "\x1b[200~"
	const bracketedPasteEnd = "\x1b[201~"

	activeAgentSessions, err := s.terminals.WriteToActiveAgent(
		projectPath,
		[]byte(bracketedPasteStart+text+bracketedPasteEnd),
	)
	if err != nil {
		return err
	}
	if activeAgentSessions == 0 {
		return ErrAgentSessionNotFound
	}
	if activeAgentSessions > 1 {
		return ErrAgentSessionAmbiguous
	}

	return nil
}
