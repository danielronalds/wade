package terminals

type TerminalRole string

type TerminalStatus string

type InputMode string

const (
	TerminalRoleAgent      TerminalRole = "agent"
	TerminalRoleMisc       TerminalRole = "misc"
	TerminalRoleServer     TerminalRole = "server"
	TerminalRoleScratchpad TerminalRole = "scratchpad"
)

const TerminalStatusRunning TerminalStatus = "running"

const (
	InputModeRaw            InputMode = "raw"
	InputModeBracketedPaste InputMode = "bracketed-paste"
)

type terminalDescriptor struct {
	id      string
	role    TerminalRole
	agent   *string
	command string
}
