package terminals

// TerminalRole identifies a terminal's purpose.
type TerminalRole string // @name TerminalRole

// TerminalStatus identifies terminal process state.
type TerminalStatus string // @name TerminalStatus

// InputMode controls how input is written to the PTY.
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

// Agent configures one available agent terminal.
type Agent struct {
	Name    string
	Command string
	Default bool
}

// Configuration controls future terminal processes.
type Configuration struct {
	Shell         string
	ServerAddress string
	Agents        []Agent
}

// Input targets one exact terminal.
type Input struct {
	WorkspaceID string    `json:"-"`
	TerminalID  string    `json:"-"`
	Text        string    `json:"text"`
	Mode        InputMode `json:"mode"`
} // @name TerminalInputRequest

// Terminal is a detached serialisable terminal resource.
type Terminal struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Role        TerminalRole   `json:"role"`
	Agent       *string        `json:"agent" extensions:"x-nullable"`
	Status      TerminalStatus `json:"status"`
	SocketURL   string         `json:"socketUrl"`
} // @name Terminal

type terminalDescriptor struct {
	id      string
	role    TerminalRole
	agent   *string
	command string
}
