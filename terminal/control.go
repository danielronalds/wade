package terminal

import "encoding/json"

type ControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Session) ApplyControlMessage(data []byte) {
	var message ControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return
	}

	if !message.IsResize() {
		return
	}

	_ = s.Resize(Size{Cols: message.Cols, Rows: message.Rows})
}

func (message ControlMessage) IsResize() bool {
	return message.Type == "resize" && message.Cols > 0 && message.Rows > 0
}
