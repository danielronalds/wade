package controllers

import (
	"fmt"
	"io"
)

type Controller interface {
	HandleArgs(args []string) error
}

type Router struct {
	controllers map[string]Controller
}

func NewRouter(stdout io.Writer) Router {
	return Router{controllers: map[string]Controller{
		"config": NewConfigController(),
		"help":   NewHelpController(stdout),
		"server": NewServerController(),
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
