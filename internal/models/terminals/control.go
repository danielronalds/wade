package terminals

import "encoding/json"

// ControlMessage is a decoded WebSocket terminal control message.
type ControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// IsResize reports whether the message contains a valid terminal size.
func (message ControlMessage) IsResize() bool {
	return message.Type == "resize" && message.Cols > 0 && message.Rows > 0
}

// IsActivate reports whether the terminal should become the active agent.
func (message ControlMessage) IsActivate() bool {
	return message.Type == "activate"
}

func parseControlMessage(data []byte) (ControlMessage, bool) {
	var message ControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return ControlMessage{}, false
	}

	return message, true
}
