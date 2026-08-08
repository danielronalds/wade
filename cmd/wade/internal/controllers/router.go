package controllers

import (
	"fmt"
	"io"

	"wade/cmd/wade/internal/controllers/config"
	"wade/cmd/wade/internal/controllers/help"
	"wade/cmd/wade/internal/controllers/server"
	httpcontrollers "wade/internal/controllers"
)

const (
	configCommand = "config"
	helpCommand   = "help"
)

// Controller handles one command-line command.
type Controller interface {
	HandleArgs(args []string) (int, error)
}

// Router dispatches command-line arguments to command controllers.
type Router struct {
	controllers map[string]Controller
}

// NewRouter constructs the command-line router with shared dependencies.
func NewRouter(stdout io.Writer, settingsModel httpcontrollers.SettingsModel) Router {
	serverController := server.NewController(stdout, settingsModel)

	return Router{controllers: map[string]Controller{
		configCommand:        config.NewController(settingsModel),
		helpCommand:          help.NewController(stdout),
		server.ServerCommand: serverController,
		server.StatusCommand: serverController,
		server.StopCommand:   serverController,
	}}
}

// HandleArgs dispatches arguments and returns the requested process exit code.
func (r Router) HandleArgs(args []string) (int, error) {
	command := helpCommand
	if len(args) > 0 {
		command = args[0]
	}

	controller, ok := r.controllers[command]
	if !ok {
		return 0, fmt.Errorf("unknown command: %s", command)
	}

	return controller.HandleArgs(args)
}
