package terminals

// TODO: Review properly

import "encoding/json"

type ControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func parseControlMessage(data []byte) (ControlMessage, bool) {
	var message ControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return ControlMessage{}, false
	}

	return message, true
}

func (message ControlMessage) IsResize() bool {
	return message.Type == "resize" && message.Cols > 0 && message.Rows > 0
}

func (message ControlMessage) IsActivate() bool {
	return message.Type == "activate"
}
