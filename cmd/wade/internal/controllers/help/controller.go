package help

import (
	"fmt"
	"io"
)

// Controller writes command-line usage information.
type Controller struct {
	stdout io.Writer
}

// NewController constructs the help command controller.
func NewController(stdout io.Writer) Controller {
	return Controller{stdout: stdout}
}

// HandleArgs writes help text and returns a successful exit code.
func (c Controller) HandleArgs(args []string) (int, error) {
	_, err := fmt.Fprint(c.stdout, helpText())
	return 0, err
}

func helpText() string {
	return `Usage: wade [command]

WADE is a local-first browser workspace for agentic coding sessions.

Commands
  server    Start the WADE web server in the background
  status    Show the background server status
  stop      Stop the background server
  config    Open the WADE config in your editor
  help      Show this menu

Run wade server --foreground to keep the server attached to the terminal.
`
}
