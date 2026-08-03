package help

import (
	"fmt"
	"io"
)

type Controller struct {
	stdout io.Writer
}

func NewController(stdout io.Writer) Controller {
	return Controller{stdout: stdout}
}

func (c Controller) HandleArgs(args []string) error {
	_, err := fmt.Fprint(c.stdout, helpText())
	return err
}

func helpText() string {
	return `Usage: wade [command]

WADE is a local-first browser workspace for agentic coding sessions.

Commands
  server    Start the WADE web server in the background
  config    Open the WADE config in your editor
  help      Show this menu

Run wade server --foreground to keep the server attached to the terminal.
`
}
