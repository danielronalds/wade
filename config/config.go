package config

import (
	"net"
	"os"
	"strings"

	"web-terminal/terminal"
)

const (
	addressEnv  = "WEB_TERMINAL_ADDR"
	devModeEnv  = "WEB_TERMINAL_DEV"
	defaultPort = "8765"
	devHost     = "editor-dev.localhost"
	runHost     = "editor.localhost"
)

type Config struct {
	Address string
	Shell   string
}

func Load() Config {
	return Config{
		Address: envOrDefault(addressEnv, defaultAddress(os.Getenv(devModeEnv))),
		Shell:   terminal.ResolveShell(os.Getenv("SHELL")),
	}
}

func defaultAddress(devMode string) string {
	if isEnabled(devMode) {
		return net.JoinHostPort(devHost, defaultPort)
	}

	return net.JoinHostPort(runHost, defaultPort)
}

func isEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
