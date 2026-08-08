package pty

const fallbackShell = "/bin/bash"

// ResolveShell returns the configured shell or the platform fallback.
func ResolveShell(shell string) string {
	if shell == "" {
		return fallbackShell
	}

	return shell
}
