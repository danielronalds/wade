package app

// TODO: Review properly

import (
	"wade/internal/services/config"
	"wade/internal/services/terminalsessions"
	"wade/internal/services/workspaces"
)

type configReloader struct {
	workspaces workspaces.Service
	terminals  *terminalsessions.Service
}

func (r configReloader) ReloadConfig() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}

	r.workspaces.Reload(configuration.ProjectDirs)
	r.terminals.Configure(configuration.Shell, terminalAgents(configuration.Agents))
	return nil
}
