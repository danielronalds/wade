package config

import (
	"os"
	"os/exec"

	"wade/internal/controllers"
)

// Controller opens the persisted settings file in the user's editor.
type Controller struct {
	settings controllers.SettingsModel
}

// NewController constructs the config command controller.
func NewController(settings controllers.SettingsModel) Controller {
	return Controller{settings: settings}
}

// HandleArgs ensures and opens the settings file.
func (c Controller) HandleArgs(_ []string) (int, error) {
	configPath, err := c.settings.EnsureFile()
	if err != nil {
		return 0, err
	}

	command := exec.Command(getEditor(), configPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return 0, command.Run()
}

func getEditor() string {
	if editor, found := os.LookupEnv("EDITOR"); found {
		return editor
	}
	return "nvim"
}
