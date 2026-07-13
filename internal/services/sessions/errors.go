package sessions

// TODO: Review properly

import "errors"

var ErrSessionRequired = errors.New("session is required")
var ErrSessionNotFound = errors.New("session not found")
var ErrAgentTextRequired = errors.New("agent text is required")
var ErrAgentSessionNotFound = errors.New("agent session not found")
var ErrAgentSessionAmbiguous = errors.New("agent session is ambiguous")
