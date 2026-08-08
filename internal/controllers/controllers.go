package controllers

// Controllers groups the HTTP controllers registered by the server.
type Controllers struct {
	Workspaces         Workspaces
	Repositories       Repositories
	RemoteRepositories RemoteRepositories
	Worktrees          Worktrees
	Terminals          Terminals
	ReviewSnapshots    ReviewSnapshots
	Settings           *Settings
	Docs               Docs
	Page               Page
}
