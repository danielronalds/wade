package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	StateDirectory string
	LogPath        string
	SocketPath     string
}

func ResolvePaths() (Paths, error) {
	stateRoot := os.Getenv("XDG_STATE_HOME")
	if stateRoot == "" {
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("finding home directory: %w", err)
		}
		stateRoot = filepath.Join(homeDirectory, ".local", "state")
	}

	stateDirectory := filepath.Join(stateRoot, "wade")
	return Paths{
		StateDirectory: stateDirectory,
		LogPath:        filepath.Join(stateDirectory, "server.log"),
		SocketPath:     filepath.Join(stateDirectory, "server.sock"),
	}, nil
}

func (p Paths) ensureStateDirectory() error {
	if err := os.MkdirAll(p.StateDirectory, 0o700); err != nil {
		return fmt.Errorf("creating WADE state directory: %w", err)
	}
	if err := os.Chmod(p.StateDirectory, 0o700); err != nil {
		return fmt.Errorf("securing WADE state directory: %w", err)
	}
	return nil
}

func (p Paths) openLog() (*os.File, error) {
	if err := p.ensureStateDirectory(); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(p.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening WADE server log: %w", err)
	}
	if err := logFile.Chmod(0o600); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("securing WADE server log: %w", err)
	}
	return logFile, nil
}
