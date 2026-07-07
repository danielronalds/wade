package controllers

import (
	"os"
	"os/exec"

	"wade/internal/config"
)

type ConfigController struct{}

func NewConfigController() ConfigController {
	return ConfigController{}
}

func (c ConfigController) HandleArgs(args []string) error {
	configPath, err := config.EnsureFile()
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
