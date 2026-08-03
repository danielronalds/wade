package controllers

import (
	"fmt"
	"io"

	"wade/cmd/wade/internal/controllers/config"
	"wade/cmd/wade/internal/controllers/help"
	"wade/cmd/wade/internal/controllers/server"
)

type Controller interface {
	HandleArgs(args []string) error
}

type Router struct {
	controllers map[string]Controller
}

func NewRouter(stdout io.Writer) Router {
	return Router{controllers: map[string]Controller{
		"config": config.NewController(),
		"help":   help.NewController(stdout),
		"server": server.NewController(stdout),
	}}
}

func (r Router) HandleArgs(args []string) error {
	command := "help"
	if len(args) > 0 {
		command = args[0]
	}

	controller, ok := r.controllers[command]
	if !ok {
		return fmt.Errorf("unknown command: %s", command)
	}

	return controller.HandleArgs(args)
}
