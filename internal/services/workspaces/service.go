package workspaces

// TODO: Review properly

import "context"

type Repository interface {
	IDs() ([]string, error)
	Resolve(workspaceID string) (string, bool, error)
	IDForDirectory(directory string) (string, bool, error)
	IDsForDirectories(directories []string) ([]string, error)
	Directories() []string
	Reload(directories []string)
}

type GitRepository interface {
	CurrentBranch(ctx context.Context, workspacePath string) (string, error)
	OriginURL(ctx context.Context, workspacePath string) (string, error)
}

type GitHubRepository interface {
	PullRequestURL(ctx context.Context, repository string, branch string) (string, error)
}

type Service struct {
	workspaces Repository
	git        GitRepository
	github     GitHubRepository
}

func NewService(workspaces Repository, git GitRepository, github GitHubRepository) Service {
	return Service{workspaces: workspaces, git: git, github: github}
}

func (s Service) List() ([]WorkspaceSummary, error) {
	workspaceIDs, err := s.workspaces.IDs()
	if err != nil {
		return nil, err
	}

	workspaces := make([]WorkspaceSummary, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		workspaces = append(workspaces, WorkspaceSummary{
			ID:   workspaceID,
			Name: workspaceID,
		})
	}

	return workspaces, nil
}

func (s Service) Get(ctx context.Context, workspaceID string) (Workspace, error) {
	workspacePath, err := s.Path(workspaceID)
	if err != nil {
		return Workspace{}, err
	}

	metadata := loadMetadata(ctx, workspacePath, s.git, s.github)
	return Workspace{
		ID:                 workspaceID,
		Name:               workspaceID,
		RemoteRepositoryID: metadata.remoteRepositoryID,
		Branch:             metadata.branch,
		Links:              metadata.links,
	}, nil
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
