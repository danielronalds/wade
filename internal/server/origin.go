package server

import (
	"net/http"
	"net/url"
)

// AllowSameOrigin prevents arbitrary browser tabs from connecting to local shells.
func AllowSameOrigin(r *http.Request) bool {
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
