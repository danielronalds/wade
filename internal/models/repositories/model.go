package repositories

import "sync"

// Configuration controls runtime worktree behaviour.
type Configuration struct {
	CopyIgnoredFilesOnWorktreeCreation bool
	WorktreeCopyExcludes               []string
}

// Model owns local repositories, worktrees, branches, and workspace Git contexts.
type Model struct {
	workspaces WorkspaceDiscovery
	git        Git
	files      FileSystem

	configurationMu sync.RWMutex
	configuration   Configuration
	mutationLocks   sync.Map
}

// New constructs an application-scoped Repositories Model.
func New(workspaces WorkspaceDiscovery, git Git, files FileSystem, configuration Configuration) *Model {
	return &Model{
		workspaces: workspaces,
		git:        git,
		files:      files,
		configuration: Configuration{
			CopyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
			WorktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
		},
	}
}

// Configure atomically replaces runtime worktree configuration.
func (model *Model) Configure(configuration Configuration) {
	model.configurationMu.Lock()
	defer model.configurationMu.Unlock()

	model.configuration = Configuration{
		CopyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
	}
}

func (model *Model) repositoryMutationLock(repositoryID string) *sync.Mutex {
	lock, _ := model.mutationLocks.LoadOrStore(repositoryID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
