package daemon

type controlCommand string

const (
	controlCommandStatus controlCommand = "status"
	controlCommandStop   controlCommand = "stop"
)

// Status identifies a running managed daemon and its local resources.
type Status struct {
	PID     int    `json:"pid"`
	Address string `json:"address"`
	LogPath string `json:"logPath"`
}

type startupMessage struct {
	Status         *Status `json:"status,omitempty"`
	AlreadyRunning bool    `json:"alreadyRunning,omitempty"`
	Error          string  `json:"error,omitempty"`
}

type startupResult struct {
	message startupMessage
	err     error
}

type controlRequest struct {
	Command controlCommand `json:"command"`
}

type controlResponse struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	Address string `json:"address,omitempty"`
	LogPath string `json:"logPath,omitempty"`
	Error   string `json:"error,omitempty"`
}
