package sessions

// TODO: Review properly

import "errors"

var ErrSessionRequired = errors.New("session is required")
var ErrSessionNotFound = errors.New("session not found")
