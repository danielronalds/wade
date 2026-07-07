package terminal

import (
	"os"
	"os/exec"

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

func Start(shell string, directory string, size Size) (*Session, error) {
	return start(interactiveShell(shell), directory, size)
}

func StartShellCommand(shell string, directory string, command string, size Size) (*Session, error) {
	return start(shellCommand(shell, command), directory, size)
}

func interactiveShell(shell string) *exec.Cmd {
	return exec.Command(shell)
}

func shellCommand(shell string, command string) *exec.Cmd {
	return exec.Command(shell, "-lc", command)
}

func start(command *exec.Cmd, directory string, size Size) (*Session, error) {
	command.Dir = directory
	command.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	terminalFile, err := pty.StartWithSize(command, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
	if err != nil {
		return nil, err
	}

	return &Session{
		command:  command,
		terminal: terminalFile,
	}, nil
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
