package controllers

import (
	"context"

	"wade/internal/models/remoterepositories"
	"wade/internal/models/repositories"
	"wade/internal/models/reviewsnapshots"
	"wade/internal/models/settings"
	"wade/internal/models/terminals"
	"wade/internal/models/workspaces"
)

type fakeWorkspacesModel struct {
	listItems          []workspaces.WorkspaceSummary
	listByIDItems      []workspaces.WorkspaceSummary
	getItem            workspaces.Workspace
	materialised       workspaces.Workspace
	materialiseRequest *workspaces.MaterialiseRequest
	links              workspaces.WorkspaceLinks
	configuration      workspaces.Configuration
}

func (fake *fakeWorkspacesModel) List(context.Context) ([]workspaces.WorkspaceSummary, error) {
	return fake.listItems, nil
}
func (fake *fakeWorkspacesModel) ListByIDs(context.Context, []string) ([]workspaces.WorkspaceSummary, error) {
	return fake.listByIDItems, nil
}
func (fake *fakeWorkspacesModel) Get(context.Context, string) (workspaces.Workspace, error) {
	return fake.getItem, nil
}
func (fake *fakeWorkspacesModel) Materialise(_ context.Context, request workspaces.MaterialiseRequest) (workspaces.Workspace, error) {
	if fake.materialiseRequest != nil {
		*fake.materialiseRequest = request
	}
	return fake.materialised, nil
}
func (fake *fakeWorkspacesModel) ResolveLinks(context.Context, workspaces.LinkContext) (workspaces.WorkspaceLinks, error) {
	return fake.links, nil
}
func (fake *fakeWorkspacesModel) Configure(configuration workspaces.Configuration) {
	fake.configuration = configuration
}

type fakeRepositoriesModel struct {
	workspaceContexts       []repositories.WorkspaceContext
	targetedContexts        map[string]repositories.WorkspaceContext
	workspaceContext        *repositories.WorkspaceContext
	workspaceIDsByRemote    map[string][]string
	worktree                repositories.Worktree
	removed                 repositories.Worktree
	getWorktreeCalls        int
	removeWorktreeCalls     int
	workspaceMappingCalls   int
	targetedWorkspaceIDCall []string
	calls                   *[]string
	configuration           repositories.Configuration
}

func (*fakeRepositoriesModel) Get(context.Context, string) (repositories.Repository, error) {
	return repositories.Repository{}, nil
}
func (fake *fakeRepositoriesModel) ListWorkspaceContexts(context.Context) ([]repositories.WorkspaceContext, error) {
	return fake.workspaceContexts, nil
}
func (fake *fakeRepositoriesModel) ListWorkspaceContextsByIDs(_ context.Context, workspaceIDs []string) (map[string]repositories.WorkspaceContext, error) {
	fake.targetedWorkspaceIDCall = append([]string(nil), workspaceIDs...)
	return fake.targetedContexts, nil
}
func (fake *fakeRepositoriesModel) GetWorkspaceContext(context.Context, string) (*repositories.WorkspaceContext, error) {
	return fake.workspaceContext, nil
}
func (fake *fakeRepositoriesModel) WorkspaceIDsByRemoteRepository(context.Context, []string) (map[string][]string, error) {
	fake.workspaceMappingCalls++
	return fake.workspaceIDsByRemote, nil
}
func (*fakeRepositoriesModel) ListWorktrees(context.Context, string) ([]repositories.Worktree, error) {
	return nil, nil
}
func (fake *fakeRepositoriesModel) GetWorktree(context.Context, string, string) (repositories.Worktree, error) {
	fake.getWorktreeCalls++
	if fake.calls != nil {
		*fake.calls = append(*fake.calls, "get")
	}
	return fake.worktree, nil
}
func (*fakeRepositoriesModel) CreateWorktree(context.Context, string, repositories.CreateWorktreeRequest) (repositories.Worktree, error) {
	return repositories.Worktree{}, nil
}
func (fake *fakeRepositoriesModel) RemoveWorktree(context.Context, string, string) (repositories.Worktree, error) {
	fake.removeWorktreeCalls++
	if fake.calls != nil {
		*fake.calls = append(*fake.calls, "remove")
	}
	return fake.removed, nil
}
func (*fakeRepositoriesModel) ListBranches(context.Context, string, repositories.BranchKind) ([]repositories.Branch, error) {
	return nil, nil
}
func (fake *fakeRepositoriesModel) Configure(configuration repositories.Configuration) {
	fake.configuration = configuration
}

type fakeTerminalsModel struct {
	items              []terminals.Terminal
	putItem            terminals.Terminal
	putCreated         bool
	startDefaultItem   terminals.Terminal
	startDefaultError  error
	startDefaultCalls  int
	connectError       error
	activeWorkspaceIDs []string
	activeCounts       map[string]int
	deleteAllCalls     []string
	putCalls           int
	calls              *[]string
	configuration      terminals.Configuration
}

func (fake *fakeTerminalsModel) List(context.Context, string) ([]terminals.Terminal, error) {
	return fake.items, nil
}
func (*fakeTerminalsModel) Get(context.Context, string, string) (terminals.Terminal, error) {
	return terminals.Terminal{}, nil
}
func (fake *fakeTerminalsModel) Put(context.Context, string, string) (terminals.Terminal, bool, error) {
	fake.putCalls++
	return fake.putItem, fake.putCreated, nil
}
func (fake *fakeTerminalsModel) StartDefaultAgent(context.Context, string) (terminals.Terminal, error) {
	fake.startDefaultCalls++
	return fake.startDefaultItem, fake.startDefaultError
}
func (*fakeTerminalsModel) Delete(context.Context, string, string) error { return nil }
func (fake *fakeTerminalsModel) DeleteAll(_ context.Context, workspaceID string) (int, error) {
	fake.deleteAllCalls = append(fake.deleteAllCalls, workspaceID)
	if fake.calls != nil {
		*fake.calls = append(*fake.calls, "close")
	}
	return 1, nil
}
func (*fakeTerminalsModel) Input(context.Context, terminals.Input) error { return nil }
func (fake *fakeTerminalsModel) Connect(context.Context, string, string) (*terminals.TerminalSession, error) {
	return nil, fake.connectError
}
func (fake *fakeTerminalsModel) ActiveTerminalCount(workspaceID string) int {
	return fake.activeCounts[workspaceID]
}
func (fake *fakeTerminalsModel) ActiveWorkspaceIDs() []string {
	return append([]string(nil), fake.activeWorkspaceIDs...)
}
func (fake *fakeTerminalsModel) Configure(configuration terminals.Configuration) {
	fake.configuration = configuration
}

type fakeReviewSnapshotsModel struct {
	snapshot            reviewsnapshots.ReviewSnapshot
	contents            reviewsnapshots.FileContents
	createError         error
	getError            error
	fileContentsError   error
	deleteError         error
	createdWorkspaceID  string
	requestedSnapshotID string
	requestedFileID     string
	requestedScope      reviewsnapshots.Scope
	deletedSnapshotID   string
}

func (fake *fakeReviewSnapshotsModel) Create(_ context.Context, workspaceID string) (reviewsnapshots.ReviewSnapshot, error) {
	fake.createdWorkspaceID = workspaceID
	return fake.snapshot, fake.createError
}
func (fake *fakeReviewSnapshotsModel) Get(snapshotID string) (reviewsnapshots.ReviewSnapshot, error) {
	fake.requestedSnapshotID = snapshotID
	return fake.snapshot, fake.getError
}
func (fake *fakeReviewSnapshotsModel) FileContents(_ context.Context, snapshotID string, fileID string, scope reviewsnapshots.Scope) (reviewsnapshots.FileContents, error) {
	fake.requestedSnapshotID = snapshotID
	fake.requestedFileID = fileID
	fake.requestedScope = scope
	return fake.contents, fake.fileContentsError
}
func (fake *fakeReviewSnapshotsModel) Delete(snapshotID string) error {
	fake.deletedSnapshotID = snapshotID
	return fake.deleteError
}

type fakeSettingsModel struct {
	current      settings.Settings
	updateResult settings.UpdateResult
	reloadResult settings.UpdateResult
	getError     error
	updateError  error
	reloadError  error
	updated      settings.Settings
}

func (*fakeSettingsModel) EnsureFile() (string, error) {
	return "", nil
}
func (fake *fakeSettingsModel) Get() (settings.Settings, error) {
	return fake.current, fake.getError
}
func (*fakeSettingsModel) LoadRuntimeConfiguration() (settings.RuntimeConfiguration, error) {
	return settings.RuntimeConfiguration{}, nil
}
func (fake *fakeSettingsModel) Update(request settings.Settings) (settings.UpdateResult, error) {
	fake.updated = request
	return fake.updateResult, fake.updateError
}
func (fake *fakeSettingsModel) Reload() (settings.UpdateResult, error) {
	return fake.reloadResult, fake.reloadError
}

type fakeRemoteRepositoriesModel struct {
	items []remoterepositories.RemoteRepository
}

func (fake fakeRemoteRepositoriesModel) List(context.Context) ([]remoterepositories.RemoteRepository, error) {
	return append([]remoterepositories.RemoteRepository(nil), fake.items...), nil
}
