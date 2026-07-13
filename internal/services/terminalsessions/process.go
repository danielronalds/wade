package terminalsessions

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

type Session struct {
	command  *exec.Cmd
	terminal *os.File
}

func Start(shell string, directory string, projectName string, size Size) (*Session, error) {
	return start(interactiveShell(shell, projectName), directory, size)
}

func StartShellCommand(shell string, directory string, projectName string, command string, size Size) (*Session, error) {
	return start(shellCommand(shell, projectName, command), directory, size)
}

func (s *Session) Read(data []byte) (int, error) {
	return s.terminal.Read(data)
}

func (s *Session) Write(data []byte) (int, error) {
	return s.terminal.Write(data)
}

func (s *Session) Resize(size Size) error {
	return pty.Setsize(s.terminal, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
}

func (s *Session) Close() {
	_ = s.terminal.Close()

	if s.command.Process != nil {
		_ = s.command.Process.Kill()
	}

	_ = s.command.Wait()
}

func interactiveShell(shell string, projectName string) *exec.Cmd {
	return withShellEnvironment(exec.Command(shell), shell, projectName)
}

func shellCommand(shell string, projectName string, command string) *exec.Cmd {
	return withShellEnvironment(exec.Command(shell, "-lc", command), shell, projectName)
}

func withShellEnvironment(command *exec.Cmd, shell string, projectName string) *exec.Cmd {
	command.Env = setEnvironmentValues(
		os.Environ(),
		environmentVariable{name: "SHELL", value: shell},
		environmentVariable{name: "WADE_SESSION", value: projectName},
	)
	return command
}

func start(command *exec.Cmd, directory string, size Size) (*Session, error) {
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

	return &Session{
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
