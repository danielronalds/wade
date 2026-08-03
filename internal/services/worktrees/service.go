// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"wade/internal/services/config"
	"wade/internal/services/gitrepositories"
)

type gitRepository interface {
	WorktreeListPorcelain(ctx context.Context, repositoryPath string) (string, error)
	Remotes(ctx context.Context, repositoryPath string) (string, error)
	FetchRemote(ctx context.Context, repositoryPath string, remote string) error
	RemoteBranches(ctx context.Context, repositoryPath string) (string, error)
	LocalBranches(ctx context.Context, repositoryPath string) (string, error)
	ValidateBranchName(ctx context.Context, repositoryPath string, branch string) error
	AddWorktree(ctx context.Context, repositoryPath string, targetPath string, branch string) error
	AddTrackingWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string, remoteBranch string) error
	AddNewBranchWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string) error
	RemoveWorktree(ctx context.Context, repositoryPath string, targetPath string) error
	PruneWorktrees(ctx context.Context, repositoryPath string) error
	DeleteBranch(ctx context.Context, repositoryPath string, branch string) error
	IgnoredPaths(ctx context.Context, repositoryPath string) (string, error)
}

type fileRepository interface {
	CopyPath(source string, destination string) error
}

type workspaceRepository interface {
	IDForDirectory(directory string) (string, bool, error)
	Resolve(workspaceID string) (string, bool, error)
}

type terminalService interface {
	CloseTerminalsForDirectory(directory string) int
}

type Service struct {
	git        gitRepository
	files      fileRepository
	workspaces workspaceRepository
	terminals  terminalService
	state      *serviceState
}

type serviceState struct {
	mu                                 sync.RWMutex
	copyIgnoredFilesOnWorktreeCreation bool
	worktreeCopyExcludes               []string
}

func NewService(
	configuration config.Config,
	git gitRepository,
	files fileRepository,
	workspaces workspaceRepository,
	terminals terminalService,
) Service {
	return Service{
		git:        git,
		files:      files,
		workspaces: workspaces,
		terminals:  terminals,
		state: &serviceState{
			copyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
			worktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
		},
	}
}

func (s Service) Configure(configuration config.Config) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	s.state.copyIgnoredFilesOnWorktreeCreation = configuration.CopyIgnoredFilesOnWorktreeCreation
	s.state.worktreeCopyExcludes = append([]string(nil), configuration.WorktreeCopyExcludes...)
}

func (s Service) List(ctx context.Context, repository gitrepositories.Context) ([]Worktree, error) {
	entries, err := listWorktreeEntries(ctx, repository.MainWorktreePath(), s.git)
	if err != nil {
		return nil, err
	}

	worktrees := make([]Worktree, 0, len(entries))
	for _, entry := range entries {
		workspaceID, found, err := s.workspaces.IDForDirectory(entry.path)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		workspaceDirectory, found, err := s.workspaces.Resolve(workspaceID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		isMain := samePath(entry.path, repository.MainWorktreePath())
		name := workspaceID
		if !isMain {
			name = strings.TrimPrefix(workspaceID, repository.Repository.ID+"-")
		}

		var branch *Branch
		if entry.branch != "" {
			branch = &Branch{
				Ref:  entry.branch,
				Name: strings.TrimPrefix(entry.branch, "refs/heads/"),
			}
		}

		worktrees = append(worktrees, Worktree{
			ID:                 workspaceID,
			RepositoryID:       repository.Repository.ID,
			WorkspaceID:        workspaceID,
			Name:               name,
			Branch:             branch,
			IsMain:             isMain,
			IsRemovable:        !isMain,
			path:               entry.path,
			workspaceDirectory: workspaceDirectory,
		})
	}

	return worktrees, nil
}

func (s Service) Get(ctx context.Context, repository gitrepositories.Context, worktreeID string) (Worktree, error) {
	if err := validateWorktreeID(worktreeID); err != nil {
		return Worktree{}, err
	}

	worktrees, err := s.List(ctx, repository)
	if err != nil {
		return Worktree{}, err
	}

	for _, worktree := range worktrees {
		if worktree.ID == worktreeID {
			return worktree, nil
		}
	}

	return Worktree{}, WorktreeNotFoundError{WorktreeID: worktreeID}
}

func (s Service) Branches(ctx context.Context, repository gitrepositories.Context, kind BranchKind) ([]Branch, error) {
	if err := validateBranchKind(kind); err != nil {
		return nil, err
	}

	repositoryPath := repository.MainWorktreePath()
	worktrees, err := s.List(ctx, repository)
	if err != nil {
		return nil, err
	}
	checkedOutWorkspaces := make(map[string]string)
	for _, worktree := range worktrees {
		if worktree.Branch != nil {
			checkedOutWorkspaces[worktree.Branch.Name] = worktree.WorkspaceID
		}
	}

	if kind == BranchKindLocal {
		localBranches, err := listLocalBranchNames(ctx, repositoryPath, s.git)
		if err != nil {
			return nil, err
		}

		branches := make([]Branch, 0, len(localBranches))
		for _, branchName := range localBranches {
			branches = append(branches, Branch{
				Ref:                   "refs/heads/" + branchName,
				Name:                  branchName,
				HasLocalBranch:        true,
				CheckedOutWorkspaceID: stringReference(checkedOutWorkspaces[branchName]),
			})
		}
		return branches, nil
	}

	remote, found, err := preferredRemote(ctx, repositoryPath, s.git)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Branch{}, nil
	}
	if err := fetchRemote(ctx, repositoryPath, remote, s.git); err != nil {
		return nil, err
	}

	remoteBranches, err := listRemoteBranches(ctx, repositoryPath, remote, s.git)
	if err != nil {
		return nil, err
	}
	localBranches, err := listLocalBranches(ctx, repositoryPath, s.git)
	if err != nil {
		return nil, err
	}

	branches := make([]Branch, 0, len(remoteBranches))
	for _, branchName := range remoteBranches {
		branches = append(branches, Branch{
			Ref:                   fmt.Sprintf("refs/remotes/%s/%s", remote, branchName),
			Name:                  branchName,
			Remote:                stringReference(remote),
			HasLocalBranch:        localBranches[branchName],
			CheckedOutWorkspaceID: stringReference(checkedOutWorkspaces[branchName]),
		})
	}

	return branches, nil
}

func (s Service) Create(ctx context.Context, repository gitrepositories.Context, requestedBranchRef string) (Worktree, error) {
	branchRef := strings.TrimSpace(requestedBranchRef)
	if branchRef == "" {
		return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
	}

	repositoryPath := repository.MainWorktreePath()
	localBranch := strings.TrimPrefix(branchRef, "refs/heads/")
	isExplicitLocalBranch := strings.HasPrefix(branchRef, "refs/heads/")
	isExplicitRemoteBranch := false
	requestedRemote := ""
	hasRemote := false

	if remoteRef, found := strings.CutPrefix(branchRef, "refs/remotes/"); found {
		requestedRemote, localBranch, found = strings.Cut(remoteRef, "/")
		if !found || requestedRemote == "" || localBranch == "" {
			return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
		}
		isExplicitRemoteBranch = true
		hasRemote = true
	} else {
		if strings.HasPrefix(branchRef, "refs/") && !isExplicitLocalBranch {
			return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
		}

		preferredRemote, found, err := preferredRemote(ctx, repositoryPath, s.git)
		if err != nil {
			return Worktree{}, err
		}
		requestedRemote = preferredRemote
		hasRemote = found
		if hasRemote && strings.HasPrefix(branchRef, preferredRemote+"/") {
			localBranch = strings.TrimPrefix(branchRef, preferredRemote+"/")
			isExplicitRemoteBranch = true
		}
	}

	if err := validateBranchName(ctx, repositoryPath, localBranch, s.git); err != nil {
		return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
	}

	if isExplicitRemoteBranch {
		output, err := s.git.Remotes(ctx, repositoryPath)
		if err != nil {
			return Worktree{}, fmt.Errorf("listing remotes: %w", err)
		}

		remoteExists := false
		for _, remote := range parseLines(output) {
			if remote == requestedRemote {
				remoteExists = true
				break
			}
		}
		if !remoteExists {
			return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
		}
	}

	remoteBranches := map[string]bool{}
	if hasRemote {
		if err := fetchRemote(ctx, repositoryPath, requestedRemote, s.git); err != nil {
			return Worktree{}, err
		}

		branches, err := listRemoteBranches(ctx, repositoryPath, requestedRemote, s.git)
		if err != nil {
			return Worktree{}, err
		}
		for _, branch := range branches {
			remoteBranches[branch] = true
		}
	}

	worktrees, err := s.List(ctx, repository)
	if err != nil {
		return Worktree{}, err
	}
	for _, worktree := range worktrees {
		if worktree.Branch != nil && worktree.Branch.Name == localBranch {
			return worktree, nil
		}
	}

	localBranches, err := listLocalBranches(ctx, repositoryPath, s.git)
	if err != nil {
		return Worktree{}, err
	}
	targetPath, err := worktreePath(repositoryPath, repository.Repository.ID, localBranch)
	if err != nil {
		return Worktree{}, err
	}

	if localBranches[localBranch] {
		if err := s.git.AddWorktree(ctx, repositoryPath, targetPath, localBranch); err != nil {
			return Worktree{}, fmt.Errorf("creating worktree: %w", err)
		}
	} else if hasRemote && remoteBranches[localBranch] && !isExplicitLocalBranch {
		remoteBranch := requestedRemote + "/" + localBranch
		if err := s.git.AddTrackingWorktree(ctx, repositoryPath, localBranch, targetPath, remoteBranch); err != nil {
			return Worktree{}, fmt.Errorf("checking out remote worktree: %w", err)
		}
	} else if isExplicitRemoteBranch {
		return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
	} else {
		if err := s.git.AddNewBranchWorktree(ctx, repositoryPath, localBranch, targetPath); err != nil {
			return Worktree{}, fmt.Errorf("creating local branch worktree: %w", err)
		}
	}

	copyWarnings := s.copyIgnoredFiles(ctx, repositoryPath, targetPath)
	createdWorktrees, err := s.List(ctx, repository)
	if err != nil {
		return Worktree{}, err
	}
	for _, worktree := range createdWorktrees {
		if samePath(worktree.path, targetPath) {
			worktree.IgnoredFileCopyWarnings = copyWarnings
			return worktree, nil
		}
	}

	return Worktree{}, errors.New("created worktree is not discoverable as a workspace")
}

func (s Service) Remove(ctx context.Context, repository gitrepositories.Context, worktreeID string) (Worktree, error) {
	target, err := s.Get(ctx, repository, worktreeID)
	if err != nil {
		return Worktree{}, err
	}
	if !target.IsRemovable {
		return Worktree{}, WorktreeNotRemovableError{WorktreeID: worktreeID}
	}

	if s.terminals != nil {
		s.terminals.CloseTerminalsForDirectory(target.workspaceDirectory)
	}

	repositoryPath := repository.MainWorktreePath()
	if err := s.git.RemoveWorktree(ctx, repositoryPath, target.path); err != nil {
		return Worktree{}, fmt.Errorf("removing worktree: %w", err)
	}
	if err := s.git.PruneWorktrees(ctx, repositoryPath); err != nil {
		return Worktree{}, fmt.Errorf("pruning worktrees: %w", err)
	}
	if target.Branch == nil || target.Branch.Name == "" {
		return target, nil
	}
	if err := s.git.DeleteBranch(ctx, repositoryPath, target.Branch.Name); err != nil {
		return Worktree{}, fmt.Errorf("deleting local branch: %w", err)
	}

	return target, nil
}

func (s Service) copyIgnoredFiles(ctx context.Context, mainPath string, targetPath string) []string {
	s.state.mu.RLock()
	shouldCopy := s.state.copyIgnoredFilesOnWorktreeCreation
	excludes := append([]string(nil), s.state.worktreeCopyExcludes...)
	s.state.mu.RUnlock()

	if !shouldCopy {
		return nil
	}

	return copyIgnoredFiles(ctx, mainPath, targetPath, excludes, s.git, s.files)
}

func stringReference(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
