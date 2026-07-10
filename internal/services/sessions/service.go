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
