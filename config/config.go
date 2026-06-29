package config

import (
	"os"

	"web-terminal/terminal"
)

type Config struct {
	Address string
	Shell   string
}

func Load() Config {
	return Config{
		Address: envOrDefault("WEB_TERMINAL_ADDR", "127.0.0.1:8765"),
		Shell:   terminal.ResolveShell(os.Getenv("SHELL")),
	}
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
