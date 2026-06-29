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

func Start(shell string, size Size) (*Session, error) {
	command := exec.Command(shell)
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
