package terminal

const fallbackShell = "/bin/bash"

func ResolveShell(shell string) string {
	if shell == "" {
		return fallbackShell
	}

	return shell
}
