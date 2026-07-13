package sessions

// TODO: Review properly

import "strings"

func validateSessionName(sessionName string) error {
	if strings.TrimSpace(sessionName) == "" {
		return ErrSessionRequired
	}

	return nil
}

func validateAgentText(text string) error {
	if text == "" {
		return ErrAgentTextRequired
	}

	return nil
}
