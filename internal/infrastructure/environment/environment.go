package environment

import (
	"os"
	"os/exec"
)

// Client reads process environment and executable lookup state.
type Client struct{}

// NewClient constructs an environment client.
func NewClient() Client {
	return Client{}
}

// HomeDirectory returns the current user's home directory.
func (Client) HomeDirectory() (string, error) {
	return os.UserHomeDir()
}

// Variable returns one process environment value.
func (Client) Variable(name string) string {
	return os.Getenv(name)
}

// InheritedShell returns the shell inherited by the current process.
func (Client) InheritedShell() string {
	return os.Getenv("SHELL")
}

// LookPath resolves an executable using the current process PATH.
func (Client) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
