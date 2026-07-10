package controllers

// TODO: Review properly

type Controllers struct {
	Config         ConfigHandler
	Projects       Projects
	RemoteProjects RemoteProjects
	Sessions       Sessions
	Terminals      Terminals
	Worktrees      Worktrees
	Review         Review
	Docs           Docs
	Page           Page
}
