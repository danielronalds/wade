package terminals

// TODO: Review properly

import (
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
)

type Size struct {
	Cols uint16
	Rows uint16
}

type WadeEnvironment struct {
	WorkspaceID string
	TerminalID  string
	Address     string
}

type Process struct {
	command  *exec.Cmd
	terminal *os.File
}

func Start(shell string, directory string, environment WadeEnvironment, size Size) (*Process, error) {
	return start(interactiveShell(shell, environment), directory, size)
}

func StartShellCommand(shell string, directory string, environment WadeEnvironment, command string, size Size) (*Process, error) {
	return start(shellCommand(shell, environment, command), directory, size)
}

func (s *Process) Read(data []byte) (int, error) {
	return s.terminal.Read(data)
}

func (s *Process) Write(data []byte) (int, error) {
	return s.terminal.Write(data)
}

func (s *Process) Resize(size Size) error {
	return pty.Setsize(s.terminal, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
}

func (s *Process) Close() {
	_ = s.terminal.Close()

	if s.command.Process != nil {
		_ = s.command.Process.Kill()
	}

	_ = s.command.Wait()
}

func interactiveShell(shell string, environment WadeEnvironment) *exec.Cmd {
	return withShellEnvironment(exec.Command(shell), shell, environment)
}

func shellCommand(shell string, environment WadeEnvironment, command string) *exec.Cmd {
	return withShellEnvironment(exec.Command(shell, "-lc", command), shell, environment)
}

func withShellEnvironment(command *exec.Cmd, shell string, environment WadeEnvironment) *exec.Cmd {
	inheritedEnvironment := os.Environ()
	environmentWithoutLegacySession := make([]string, 0, len(inheritedEnvironment))
	for _, entry := range inheritedEnvironment {
		if !strings.HasPrefix(entry, "WADE_SESSION=") {
			environmentWithoutLegacySession = append(environmentWithoutLegacySession, entry)
		}
	}

	command.Env = setEnvironmentValues(
		environmentWithoutLegacySession,
		environmentVariable{name: "SHELL", value: shell},
		environmentVariable{name: "WADE_WORKSPACE_ID", value: environment.WorkspaceID},
		environmentVariable{name: "WADE_TERMINAL_ID", value: environment.TerminalID},
		environmentVariable{name: "WADE_ADDR", value: environment.Address},
	)
	return command
}

func start(command *exec.Cmd, directory string, size Size) (*Process, error) {
	command.Dir = directory
	if command.Env == nil {
		command.Env = os.Environ()
	}
	command.Env = setEnvironmentValues(
		command.Env,
		environmentVariable{name: "TERM", value: "xterm-256color"},
		environmentVariable{name: "COLORTERM", value: "truecolor"},
	)

	terminalFile, err := pty.StartWithSize(command, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
	if err != nil {
		return nil, err
	}

	return &Process{
		command:  command,
		terminal: terminalFile,
	}, nil
}

type environmentVariable struct {
	name  string
	value string
}

func setEnvironmentValues(environment []string, variables ...environmentVariable) []string {
	updated := append([]string(nil), environment...)
	for _, variable := range variables {
		prefix := variable.name + "="
		value := prefix + variable.value
		next := make([]string, 0, len(updated)+1)

		replaced := false
		for _, entry := range updated {
			if !strings.HasPrefix(entry, prefix) {
				next = append(next, entry)
				continue
			}

			if replaced {
				continue
			}

			next = append(next, value)
			replaced = true
		}

		if !replaced {
			next = append(next, value)
		}

		updated = next
	}

	return updated
}
