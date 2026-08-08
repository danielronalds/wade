package terminals

import "fmt"

// WorkspaceNotFoundError reports an unknown workspace identity.
type WorkspaceNotFoundError struct {
	WorkspaceID string
}

func (e WorkspaceNotFoundError) Error() string {
	return fmt.Sprintf("workspace %q not found", e.WorkspaceID)
}

// InvalidTerminalIDError reports a malformed or unsupported terminal identity.
type InvalidTerminalIDError struct {
	TerminalID string
}

func (e InvalidTerminalIDError) Error() string {
	return fmt.Sprintf("invalid terminal ID %q", e.TerminalID)
}

// AgentNotConfiguredError reports an agent terminal absent from configuration.
type AgentNotConfiguredError struct {
	AgentName string
}

func (e AgentNotConfiguredError) Error() string {
	return fmt.Sprintf("agent %q is not configured", e.AgentName)
}

// TerminalNotFoundError reports an unknown terminal within a workspace.
type TerminalNotFoundError struct {
	WorkspaceID string
	TerminalID  string
}

func (e TerminalNotFoundError) Error() string {
	return fmt.Sprintf("terminal %q was not found in workspace %q", e.TerminalID, e.WorkspaceID)
}

// TerminalInputRequiredError reports an empty terminal input request.
type TerminalInputRequiredError struct{}

func (TerminalInputRequiredError) Error() string {
	return "terminal input text is required"
}

// InvalidInputModeError reports an unsupported terminal input mode.
type InvalidInputModeError struct {
	Mode InputMode
}

func (e InvalidInputModeError) Error() string {
	return fmt.Sprintf("invalid terminal input mode %q", e.Mode)
}
