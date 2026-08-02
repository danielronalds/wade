package gitrepositories

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceRepository interface {
	IDs() ([]string, error)
	Resolve(workspaceID string) (string, bool, error)
	CanonicalPath(workspaceID string) (string, bool, error)
	IDForDirectory(directory string) (string, bool, error)
	IDsForDirectories(directories []string) ([]string, error)
}

type GitRepository interface {
	IsGitWorktree(ctx context.Context, workspacePath string) (bool, error)
	WorktreePaths(ctx context.Context, workspacePath string) ([]string, error)
	CommonDirectory(ctx context.Context, workspacePath string) (string, error)
	HeadReference(ctx context.Context, workspacePath string) (string, bool, error)
	HeadCommit(ctx context.Context, workspacePath string) (string, bool, error)
	OriginRemoteURL(ctx context.Context, workspacePath string) (string, bool, error)
}

type Service struct {
	workspaces WorkspaceRepository
	git        GitRepository
}

type workspaceRecord struct {
	id              string
	path            string
	canonicalPath   string
	isGit           bool
	mainPath        string
	worktreePaths   []string
	commonDirectory string
	remoteURL       string
	remoteIdentity  string
	branch          Branch
}

func NewService(workspaces WorkspaceRepository, git GitRepository) Service {
	return Service{workspaces: workspaces, git: git}
}

func (s Service) List(ctx context.Context) ([]Context, error) {
	records, err := s.scan(ctx)
	if err != nil {
		return nil, err
	}

	repositories := make([]Context, 0)
	seenCommonDirectories := make(map[string]struct{})
	for index, record := range records {
		if !record.isGit {
			continue
		}
		if _, seen := seenCommonDirectories[record.commonDirectory]; seen {
			continue
		}

		seenCommonDirectories[record.commonDirectory] = struct{}{}
		repositories = append(repositories, buildContext(records, index))
	}

	sort.Slice(repositories, func(firstIndex int, secondIndex int) bool {
		return repositories[firstIndex].Repository.ID < repositories[secondIndex].Repository.ID
	})

	return repositories, nil
}

func (s Service) ListWorkspaceContexts(ctx context.Context) ([]WorkspaceContext, error) {
	records, err := s.scan(ctx)
	if err != nil {
		return nil, err
	}

	contexts := make([]WorkspaceContext, 0, len(records))
	for index, record := range records {
		if !record.isGit {
			continue
		}

		repositoryContext := buildContext(records, index)
		contexts = append(contexts, WorkspaceContext{
			RepositoryContext: repositoryContext,
			WorkspaceID:       record.id,
			WorkspacePath:     record.path,
			Branch:            record.branch,
			IsMain:            samePath(record.canonicalPath, record.mainPath),
			IsRemovable:       !samePath(record.canonicalPath, record.mainPath),
		})
	}

	return contexts, nil
}

func (s Service) Resolve(ctx context.Context, repositoryID string) (Context, error) {
	if err := validateRepositoryID(repositoryID); err != nil {
		return Context{}, err
	}

	repositories, err := s.List(ctx)
	if err != nil {
		return Context{}, err
	}

	var matched Context
	found := false
	for _, repository := range repositories {
		if repository.Repository.ID != repositoryID {
			continue
		}
		if found && matched.commonDirectory != repository.commonDirectory {
			return Context{}, RepositoryIDConflictError{RepositoryID: repositoryID}
		}

		matched = repository
		found = true
	}
	if !found {
		return Context{}, RepositoryNotFoundError{RepositoryID: repositoryID}
	}

	return matched, nil
}

func (s Service) ResolveWorkspace(ctx context.Context, workspaceID string) (WorkspaceContext, bool, error) {
	workspacePath, found, err := s.workspaces.Resolve(workspaceID)
	if err != nil {
		return WorkspaceContext{}, false, err
	}
	if !found {
		return WorkspaceContext{}, false, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	canonicalPath, found, err := s.workspaces.CanonicalPath(workspaceID)
	if err != nil {
		return WorkspaceContext{}, false, err
	}
	if !found {
		return WorkspaceContext{}, false, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	record, err := s.inspectWorkspace(ctx, workspaceID, workspacePath, canonicalPath)
	if err != nil {
		return WorkspaceContext{}, false, err
	}
	if !record.isGit {
		return WorkspaceContext{}, false, nil
	}

	workspaceIDs, err := s.workspaces.IDsForDirectories(record.worktreePaths)
	if err != nil {
		return WorkspaceContext{}, false, err
	}
	mainWorkspaceID := filepath.Base(record.mainPath)
	resolvedMainWorkspaceID, found, err := s.workspaces.IDForDirectory(record.mainPath)
	if err != nil {
		return WorkspaceContext{}, false, err
	}
	if found {
		mainWorkspaceID = resolvedMainWorkspaceID
	}

	repositoryContext := Context{
		Repository: Repository{
			ID:                 filepath.Base(record.mainPath),
			RemoteRepositoryID: githubRepositoryID(record.remoteURL),
			MainWorkspaceID:    mainWorkspaceID,
			WorkspaceIDs:       workspaceIDs,
		},
		mainWorktreePath: record.mainPath,
		commonDirectory:  record.commonDirectory,
		remoteURL:        record.remoteURL,
		remoteIdentity:   record.remoteIdentity,
	}
	isMain := samePath(record.canonicalPath, record.mainPath)

	return WorkspaceContext{
		RepositoryContext: repositoryContext,
		WorkspaceID:       record.id,
		WorkspacePath:     record.path,
		Branch:            record.branch,
		IsMain:            isMain,
		IsRemovable:       !isMain,
	}, true, nil
}

func (s Service) scan(ctx context.Context) ([]workspaceRecord, error) {
	workspaceIDs, err := s.workspaces.IDs()
	if err != nil {
		return nil, err
	}

	records := make([]workspaceRecord, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		workspacePath, found, err := s.workspaces.Resolve(workspaceID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		canonicalPath, found, err := s.workspaces.CanonicalPath(workspaceID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		record, err := s.inspectWorkspace(ctx, workspaceID, workspacePath, canonicalPath)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func (s Service) inspectWorkspace(
	ctx context.Context,
	workspaceID string,
	workspacePath string,
	canonicalPath string,
) (workspaceRecord, error) {
	record := workspaceRecord{
		id:            workspaceID,
		path:          workspacePath,
		canonicalPath: canonicalPath,
	}

	isGit, err := s.git.IsGitWorktree(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	if !isGit {
		return record, nil
	}

	worktreePaths, err := s.git.WorktreePaths(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	commonDirectory, err := s.git.CommonDirectory(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	headReference, hasHeadReference, err := s.git.HeadReference(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	headCommit, _, err := s.git.HeadCommit(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	remoteURL, hasRemote, err := s.git.OriginRemoteURL(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	if !hasRemote {
		remoteURL = ""
	}

	record.isGit = true
	record.mainPath = worktreePaths[0]
	record.worktreePaths = worktreePaths
	record.commonDirectory = commonDirectory
	record.remoteURL = remoteURL
	record.remoteIdentity = CanonicalRemoteIdentity(remoteURL)
	record.branch = Branch{
		Ref:        headReference,
		Name:       strings.TrimPrefix(headReference, "refs/heads/"),
		IsDetached: !hasHeadReference,
		Commit:     headCommit,
	}

	return record, nil
}

func buildContext(records []workspaceRecord, targetIndex int) Context {
	target := records[targetIndex]
	workspaceIDs := make([]string, 0)
	mainWorkspaceID := filepath.Base(target.mainPath)

	for _, record := range records {
		if !record.isGit || record.commonDirectory != target.commonDirectory {
			continue
		}

		workspaceIDs = append(workspaceIDs, record.id)
		if samePath(record.canonicalPath, target.mainPath) {
			mainWorkspaceID = record.id
		}
	}
	sort.Strings(workspaceIDs)

	return Context{
		Repository: Repository{
			ID:                 filepath.Base(target.mainPath),
			RemoteRepositoryID: githubRepositoryID(target.remoteURL),
			MainWorkspaceID:    mainWorkspaceID,
			WorkspaceIDs:       workspaceIDs,
		},
		mainWorktreePath: target.mainPath,
		commonDirectory:  target.commonDirectory,
		remoteURL:        target.remoteURL,
		remoteIdentity:   target.remoteIdentity,
	}
}

func samePath(firstPath string, secondPath string) bool {
	return filepath.Clean(firstPath) == filepath.Clean(secondPath)
}
