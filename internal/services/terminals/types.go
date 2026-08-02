package terminals

type TerminalRole string // @name TerminalRole

type TerminalStatus string // @name TerminalStatus

type InputMode string // @name TerminalInputMode

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
