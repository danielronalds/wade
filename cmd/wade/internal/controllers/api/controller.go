// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"wade/internal/daemon"
	"wade/internal/infrastructure/environment"
	"wade/internal/openapi"
)

// Command is the CLI command handled by the api controller.
const Command = "api"

const (
	addressFlag                    = "address"
	bodyFlag                       = "body"
	addressFlagUsage               = "WADE server address (host:port)"
	addressEnvironmentVariable     = "WADE_ADDR"
	developmentEnvironmentVariable = "WADE_DEV"
	developmentAddress             = "editor-dev.localhost:8090"
	defaultAddress                 = "editor.localhost:8765"
)

type daemonLifecycle interface {
	Status() (daemon.Status, error)
}

type environmentClient interface {
	Variable(name string) string
}

// Controller executes WADE HTTP API operations derived from the embedded
// OpenAPI specification.
type Controller struct {
	stdout      io.Writer
	stdin       io.Reader
	environment environmentClient
	daemon      daemonLifecycle
	httpClient  *http.Client
}

// NewController constructs the api command controller.
func NewController(stdout io.Writer) Controller {
	return Controller{
		stdout:      stdout,
		stdin:       os.Stdin,
		environment: environment.NewClient(),
		daemon:      daemon.NewManager(),
		httpClient:  &http.Client{},
	}
}

// HandleArgs executes one API command or writes API command help.
func (c Controller) HandleArgs(args []string) (int, error) {
	operations, err := parseOperations(openapi.JSON())
	if err != nil {
		return 0, err
	}

	commandArgs := args[1:]
	if len(commandArgs) == 0 || isHelpArgument(commandArgs[0]) {
		return 0, c.writeCommandList(operations)
	}

	operation, found := findOperation(operations, commandArgs[0])
	if !found {
		return 0, fmt.Errorf("unknown api command: %s (run \"wade api\" to list commands)", commandArgs[0])
	}

	return c.runOperation(operation, commandArgs[1:])
}

func isHelpArgument(argument string) bool {
	return argument == "help" || argument == "--help" || argument == "-h"
}

func findOperation(operations []Operation, command string) (Operation, bool) {
	for _, operation := range operations {
		if operation.Command == command {
			return operation, true
		}
	}
	return Operation{}, false
}

func (c Controller) writeCommandList(operations []Operation) error {
	widest := 0
	for _, operation := range operations {
		widest = max(widest, len(operation.Command))
	}

	var listing strings.Builder
	listing.WriteString("Usage: wade api <command> [flags]\n\n")
	listing.WriteString("Call the WADE HTTP API. Commands are derived from the OpenAPI specification.\n\n")
	listing.WriteString("Commands\n")
	for _, operation := range operations {
		fmt.Fprintf(&listing, "  %-*s  %s\n", widest, operation.Command, operation.Summary)
	}
	listing.WriteString("\nRun wade api <command> --help for the flags of one command.\n")

	_, err := io.WriteString(c.stdout, listing.String())
	return err
}

func (c Controller) writeOperationHelp(operation Operation) error {
	flags := make([][2]string, 0, len(operation.Parameters)+2)
	for _, parameter := range operation.Parameters {
		flags = append(flags, [2]string{parameter.Flag, parameterUsage(parameter)})
	}
	if operation.HasBody {
		flags = append(flags, [2]string{bodyFlag, bodyUsage(operation)})
	}
	flags = append(flags, [2]string{addressFlag, addressFlagUsage})

	widest := 0
	for _, entry := range flags {
		widest = max(widest, len(entry[0]))
	}

	var help strings.Builder
	fmt.Fprintf(&help, "Usage: wade api %s [flags]\n\n", operation.Command)
	if operation.Summary != "" {
		fmt.Fprintf(&help, "%s\n", operation.Summary)
	}
	fmt.Fprintf(&help, "%s %s\n\nFlags\n", operation.Method, operation.Path)
	for _, entry := range flags {
		fmt.Fprintf(&help, "  --%-*s  %s\n", widest, entry[0], entry[1])
	}

	_, err := io.WriteString(c.stdout, help.String())
	return err
}

func parameterUsage(parameter Parameter) string {
	usage := parameter.Description
	if usage == "" {
		usage = parameter.Name
	}
	if len(parameter.Enum) > 0 {
		usage += " (one of: " + strings.Join(parameter.Enum, ", ") + ")"
	}
	if parameter.Required {
		usage += " (required)"
	}
	return usage
}

func bodyUsage(operation Operation) string {
	usage := operation.BodyDescription
	if usage == "" {
		usage = "Request body"
	}
	usage += " as inline JSON, @file, or - for stdin"
	if operation.BodyRequired {
		usage += " (required)"
	}
	return usage
}

func (c Controller) resolveAddress() string {
	if isEnabled(c.environment.Variable(developmentEnvironmentVariable)) {
		return developmentAddress
	}
	if address := c.environment.Variable(addressEnvironmentVariable); address != "" {
		return address
	}
	if status, err := c.daemon.Status(); err == nil {
		return status.Address
	}
	return defaultAddress
}

func isEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}
