package workspaces

// TODO: Review properly

import (
	"context"
	"sort"

	"wade/internal/services/gitrepositories"
)

type Repository interface {
	IDs() ([]string, error)
	Resolve(workspaceID string) (string, bool, error)
	IDForDirectory(directory string) (string, bool, error)
	IDsForDirectories(directories []string) ([]string, error)
	Directories() []string
	Reload(directories []string)
}

type LocalRepositoryService interface {
	ListWorkspaceContexts(ctx context.Context) ([]gitrepositories.WorkspaceContext, error)
	ResolveWorkspace(ctx context.Context, workspaceID string) (gitrepositories.WorkspaceContext, bool, error)
}

type GitHubRepository interface {
	PullRequestURL(ctx context.Context, repository string, branch string) (string, error)
}

type Service struct {
	workspaces        Repository
	localRepositories LocalRepositoryService
	github            GitHubRepository
}

func NewService(workspaces Repository, localRepositories LocalRepositoryService, github GitHubRepository) Service {
	return Service{
		workspaces:        workspaces,
		localRepositories: localRepositories,
		github:            github,
	}
}

func (s Service) List(ctx context.Context) ([]WorkspaceSummary, error) {
	workspaceIDs, err := s.workspaces.IDs()
	if err != nil {
		return nil, err
	}

	localContexts, err := s.localRepositories.ListWorkspaceContexts(ctx)
	if err != nil {
		return nil, err
	}
	contextsByWorkspaceID := make(map[string]gitrepositories.WorkspaceContext, len(localContexts))
	for _, localContext := range localContexts {
		contextsByWorkspaceID[localContext.WorkspaceID] = localContext
	}

	workspaceSummaries := make([]WorkspaceSummary, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		localContext, isGit := contextsByWorkspaceID[workspaceID]
		workspace := s.buildWorkspace(ctx, workspaceID, localContext, isGit, false)
		workspaceSummaries = append(workspaceSummaries, WorkspaceSummary(workspace))
	}

	return workspaceSummaries, nil
}

func (s Service) ListByIDs(ctx context.Context, workspaceIDs []string) ([]WorkspaceSummary, error) {
	requestedWorkspaceIDs := append([]string(nil), workspaceIDs...)
	if len(requestedWorkspaceIDs) == 0 {
		return []WorkspaceSummary{}, nil
	}
	sort.Strings(requestedWorkspaceIDs)

	discoveredWorkspaceIDs, err := s.workspaces.IDs()
	if err != nil {
		return nil, err
	}
	discoveredWorkspaces := make(map[string]struct{}, len(discoveredWorkspaceIDs))
	for _, workspaceID := range discoveredWorkspaceIDs {
		discoveredWorkspaces[workspaceID] = struct{}{}
	}

	workspaceSummaries := make([]WorkspaceSummary, 0, len(requestedWorkspaceIDs))
	seenWorkspaceIDs := make(map[string]struct{}, len(requestedWorkspaceIDs))
	for _, workspaceID := range requestedWorkspaceIDs {
		if _, seen := seenWorkspaceIDs[workspaceID]; seen {
			continue
		}
		seenWorkspaceIDs[workspaceID] = struct{}{}
		if _, found := discoveredWorkspaces[workspaceID]; !found {
			continue
		}

		localContext, isGit, err := s.localRepositories.ResolveWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		workspace := s.buildWorkspace(ctx, workspaceID, localContext, isGit, false)
		workspaceSummaries = append(workspaceSummaries, WorkspaceSummary(workspace))
	}

	return workspaceSummaries, nil
}

func (s Service) Get(ctx context.Context, workspaceID string) (Workspace, error) {
	if _, err := s.Path(workspaceID); err != nil {
		return Workspace{}, err
	}

	localContext, isGit, err := s.localRepositories.ResolveWorkspace(ctx, workspaceID)
	if err != nil {
		return Workspace{}, err
	}

	return s.buildWorkspace(ctx, workspaceID, localContext, isGit, true), nil
}

func (s Service) Path(workspaceID string) (string, error) {
	if err := validateWorkspaceID(workspaceID); err != nil {
		return "", err
	}

	workspacePath, found, err := s.workspaces.Resolve(workspaceID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	return workspacePath, nil
}

func (s Service) IDForDirectory(directory string) (string, bool, error) {
	return s.workspaces.IDForDirectory(directory)
}

func (s Service) IDsForDirectories(directories []string) []string {
	workspaceIDs, err := s.workspaces.IDsForDirectories(directories)
	if err != nil {
		return nil
	}

	return workspaceIDs
}

func (s Service) Directories() []string {
	return s.workspaces.Directories()
}

func (s Service) Reload(directories []string) {
	s.workspaces.Reload(directories)
}

func (s Service) buildWorkspace(
	ctx context.Context,
	workspaceID string,
	localContext gitrepositories.WorkspaceContext,
	isGit bool,
	includePullRequest bool,
) Workspace {
	workspace := Workspace{ID: workspaceID, Name: workspaceID}
	if !isGit {
		return workspace
	}

	repository := localContext.RepositoryContext.Repository
	workspace.RepositoryID = stringReference(repository.ID)
	workspace.RemoteRepositoryID = repository.RemoteRepositoryID
	workspace.Worktree = &WorktreeReference{
		ID:          workspaceID,
		IsMain:      localContext.IsMain,
		IsRemovable: localContext.IsRemovable,
	}
	workspace.Branch = &Branch{
		Ref:        localContext.Branch.Ref,
		Name:       localContext.Branch.Name,
		IsDetached: localContext.Branch.IsDetached,
		Commit:     localContext.Branch.Commit,
	}
	github := s.github
	if !includePullRequest {
		github = nil
	}
	workspace.Links = workspaceLinks(ctx, repository.RemoteRepositoryID, localContext.Branch.Name, github)

	return workspace
}
