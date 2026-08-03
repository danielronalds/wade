package config

import (
	"os"
	"os/exec"

	configservice "wade/internal/services/config"
)

type Controller struct{}

func NewController() Controller {
	return Controller{}
}

func (c Controller) HandleArgs(args []string) error {
	configPath, err := configservice.EnsureFile()
	if err != nil {
		return err
	}

	cmd := exec.Command(getEditor(), configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func getEditor() string {
	if editor, ok := os.LookupEnv("EDITOR"); ok {
		return editor
	}

	return "nvim"
}
