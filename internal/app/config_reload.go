package app

// TODO: Review properly

import (
	"wade/internal/services/config"
	projectservice "wade/internal/services/projects"
	"wade/internal/services/terminalsessions"
)

type configReloader struct {
	projects  projectservice.Service
	terminals *terminalsessions.Service
}

func (r configReloader) ReloadConfig() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}

	r.projects.Reload(configuration.ProjectDirs)
	r.terminals.Configure(configuration.Shell, terminalAgents(configuration.Agents))
	return nil
}
