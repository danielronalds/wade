package server

import (
	"net/http"
	"net/url"
)

// Prevents arbitrary browser tabs from connecting to the local shell via WebSocket.
func allowSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https" {
		return false
	}

	return parsedOrigin.Host == r.Host
}
