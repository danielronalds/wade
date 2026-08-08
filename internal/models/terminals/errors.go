package terminals

import "fmt"

type WorkspaceNotFoundError struct {
	WorkspaceID string
}

func (e WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace %q not found", e.WorkspaceID)
}

type InvalidTerminalIDError struct {
	TerminalID string
}

func (e InvalidTerminalIDError) Error() string {
	return fmt.Sprintf("invalid terminal ID %q", e.TerminalID)
}

type AgentNotConfiguredError struct {
	AgentName string
}

func (e AgentNotConfiguredError) Error() string {
	return fmt.Sprintf("agent %q is not configured", e.AgentName)
}

type TerminalNotFoundError struct {
	WorkspaceID string
	TerminalID  string
}

func (e TerminalNotFoundError) Error() string {
	return fmt.Sprintf("terminal %q was not found in workspace %q", e.TerminalID, e.WorkspaceID)
}

type TerminalInputRequiredError struct{}

func (TerminalInputRequiredError) Error() string {
	return "terminal input text is required"
}

type InvalidInputModeError struct {
	Mode InputMode
}

func (e InvalidInputModeError) Error() string {
	return fmt.Sprintf("invalid terminal input mode %q", e.Mode)
}
