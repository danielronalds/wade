package terminals

import "wade/internal/infrastructure/pty"

// WorkspaceDiscovery resolves terminal working directories.
type WorkspaceDiscovery interface {
	Resolve(workspaceID string) (string, bool, error)
}

// PTY starts low-level terminal processes.
type PTY interface {
	StartInteractive(shell string, directory string, environment pty.WadeEnvironment, size pty.Size) (pty.Process, error)
	StartCommand(shell string, directory string, environment pty.WadeEnvironment, command string, size pty.Size) (pty.Process, error)
}
