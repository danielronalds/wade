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
}

type GitRepository interface {
	IsGitWorktree(ctx context.Context, workspacePath string) (bool, error)
	MainWorktreePath(ctx context.Context, workspacePath string) (string, error)
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
	records, err := s.scan(ctx)
	if err != nil {
		return WorkspaceContext{}, false, err
	}

	for index, record := range records {
		if record.id != workspaceID {
			continue
		}
		if !record.isGit {
			return WorkspaceContext{}, false, nil
		}

		repositoryContext := buildContext(records, index)
		return WorkspaceContext{
			RepositoryContext: repositoryContext,
			WorkspaceID:       record.id,
			WorkspacePath:     record.path,
			Branch:            record.branch,
			IsMain:            samePath(record.canonicalPath, record.mainPath),
			IsRemovable:       !samePath(record.canonicalPath, record.mainPath),
		}, true, nil
	}

	return WorkspaceContext{}, false, WorkspaceNotFoundError{WorkspaceID: workspaceID}
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

		isGit, err := s.git.IsGitWorktree(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		if !isGit {
			records = append(records, workspaceRecord{
				id:            workspaceID,
				path:          workspacePath,
				canonicalPath: canonicalPath,
			})
			continue
		}

		mainPath, err := s.git.MainWorktreePath(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		commonDirectory, err := s.git.CommonDirectory(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		headReference, hasHeadReference, err := s.git.HeadReference(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		headCommit, _, err := s.git.HeadCommit(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		remoteURL, hasRemote, err := s.git.OriginRemoteURL(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		if !hasRemote {
			remoteURL = ""
		}

		branchName := strings.TrimPrefix(headReference, "refs/heads/")
		records = append(records, workspaceRecord{
			id:              workspaceID,
			path:            workspacePath,
			canonicalPath:   canonicalPath,
			isGit:           true,
			mainPath:        mainPath,
			commonDirectory: commonDirectory,
			remoteURL:       remoteURL,
			remoteIdentity:  CanonicalRemoteIdentity(remoteURL),
			branch: Branch{
				Ref:        headReference,
				Name:       branchName,
				IsDetached: !hasHeadReference,
				Commit:     headCommit,
			},
		})
	}

	return records, nil
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
