package controllers

import (
	"fmt"
	"io"

	"wade/cmd/wade/internal/controllers/config"
	"wade/cmd/wade/internal/controllers/help"
	"wade/cmd/wade/internal/controllers/server"
)

const (
	configCommand = "config"
	helpCommand   = "help"
)

type Controller interface {
	HandleArgs(args []string) (int, error)
}

type Router struct {
	controllers map[string]Controller
}

func NewRouter(stdout io.Writer) Router {
	serverController := server.NewController(stdout)

	return Router{controllers: map[string]Controller{
		configCommand:        config.NewController(),
		helpCommand:          help.NewController(stdout),
		server.ServerCommand: serverController,
		server.StatusCommand: serverController,
		server.StopCommand:   serverController,
	}}
}

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
