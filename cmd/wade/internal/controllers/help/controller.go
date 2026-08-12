package help

import (
	"fmt"
	"io"

	"wade/internal/buildinfo"
)

// Controller writes command-line usage information.
type Controller struct {
	stdout  io.Writer
	version string
}

// NewController constructs the help command controller.
func NewController(stdout io.Writer) Controller {
	return Controller{stdout: stdout, version: buildinfo.Version()}
}

// HandleArgs writes help text and returns a successful exit code.
func (c Controller) HandleArgs(_ []string) (int, error) {
	_, err := fmt.Fprintf(c.stdout, "wade %s\n\n%s", c.version, helpText())
	return 0, err
}

func helpText() string {
	return `Usage: wade [command]

WADE is a local-first browser workspace for agentic coding sessions.

Commands
  start     Start the WADE daemon
  status    Show the WADE daemon status
  stop      Stop the WADE daemon
  config    Open the WADE config in your editor
  help      Show this menu

Run wade start --foreground to keep the server attached to the terminal.
`
}
