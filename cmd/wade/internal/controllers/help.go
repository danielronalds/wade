package controllers

import (
	"fmt"
	"io"
)

type HelpController struct {
	stdout io.Writer
}

func NewHelpController(stdout io.Writer) HelpController {
	return HelpController{stdout: stdout}
}

func (c HelpController) HandleArgs(args []string) error {
	_, err := fmt.Fprint(c.stdout, helpText())
	return err
}

func helpText() string {
	return `Usage: wade [command]

WADE is a local-first browser workspace for agentic coding sessions.

Commands
  server    Start the WADE web server
  config    Open the WADE config in your editor
  help      Show this menu
`
}
