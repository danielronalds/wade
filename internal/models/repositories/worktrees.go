package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CreateWorktreeRequest describes the branch to materialise as a worktree.
type CreateWorktreeRequest struct {
	BranchRef string `json:"branchRef"`
} // @name CreateWorktreeRequest

// ListWorktrees returns detached worktrees for a local repository.
func (model *Model) ListWorktrees(ctx context.Context, repositoryID string) ([]Worktree, error) {
	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	return model.listWorktrees(ctx, repository)
}

// GetWorktree returns one detached worktree.
func (model *Model) GetWorktree(ctx context.Context, repositoryID string, worktreeID string) (Worktree, error) {
	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return Worktree{}, err
	}
	return model.getWorktree(ctx, repository, worktreeID)
}

// CreateWorktree serialises repository mutation and idempotently creates the requested branch worktree.
func (model *Model) CreateWorktree(ctx context.Context, repositoryID string, request CreateWorktreeRequest) (Worktree, error) {
	lock := model.repositoryMutationLock(repositoryID)
	lock.Lock()
	defer lock.Unlock()

	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return Worktree{}, err
	}
	return model.createWorktree(ctx, repository, request.BranchRef)
}

// RemoveWorktree revalidates removability while holding the repository mutation lock.
func (model *Model) RemoveWorktree(ctx context.Context, repositoryID string, worktreeID string) (Worktree, error) {
	lock := model.repositoryMutationLock(repositoryID)
	lock.Lock()
	defer lock.Unlock()

	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return Worktree{}, err
	}
	return model.removeWorktree(ctx, repository, worktreeID)
}

// ListBranches returns local or remote branches for a repository.
func (model *Model) ListBranches(ctx context.Context, repositoryID string, kind BranchKind) ([]Branch, error) {
	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	return model.listBranches(ctx, repository, kind)
}

func (model *Model) listWorktrees(ctx context.Context, repository repositoryContext) ([]Worktree, error) {
	entries, err := listWorktreeEntries(ctx, repository.mainWorktreePath, model.git)
	if err != nil {
		return nil, err
	}

	worktrees := make([]Worktree, 0, len(entries))
	for _, entry := range entries {
		workspaceID, found, err := model.workspaces.IDForDirectory(entry.Path)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		isMain := samePath(entry.Path, repository.mainWorktreePath)
		name := workspaceID
		if !isMain {
			name = strings.TrimPrefix(workspaceID, repository.repository.ID+"-")
		}

		var branch *Branch
		if entry.BranchRef != "" {
			branch = &Branch{
				Ref:  entry.BranchRef,
				Name: strings.TrimPrefix(entry.BranchRef, "refs/heads/"),
			}
		}

		worktrees = append(worktrees, Worktree{
			ID:           workspaceID,
			RepositoryID: repository.repository.ID,
			WorkspaceID:  workspaceID,
			Name:         name,
			Branch:       branch,
			IsMain:       isMain,
			IsRemovable:  !isMain,
			path:         entry.Path,
		})
	}

	return worktrees, nil
}

func (model *Model) getWorktree(ctx context.Context, repository repositoryContext, worktreeID string) (Worktree, error) {
	if err := validateWorktreeID(worktreeID); err != nil {
		return Worktree{}, err
	}

	worktrees, err := model.listWorktrees(ctx, repository)
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

func (model *Model) listBranches(ctx context.Context, repository repositoryContext, kind BranchKind) ([]Branch, error) {
	if err := validateBranchKind(kind); err != nil {
		return nil, err
	}

	repositoryPath := repository.mainWorktreePath
	worktrees, err := model.listWorktrees(ctx, repository)
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
		localBranches, err := listLocalBranchNames(ctx, repositoryPath, model.git)
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

	remote, found, err := preferredRemote(ctx, repositoryPath, model.git)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Branch{}, nil
	}
	if err := fetchRemote(ctx, repositoryPath, remote, model.git); err != nil {
		return nil, err
	}

	remoteBranches, err := listRemoteBranches(ctx, repositoryPath, remote, model.git)
	if err != nil {
		return nil, err
	}
	localBranches, err := listLocalBranches(ctx, repositoryPath, model.git)
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

func (model *Model) createWorktree(ctx context.Context, repository repositoryContext, requestedBranchRef string) (Worktree, error) {
	branchRef := strings.TrimSpace(requestedBranchRef)
	if branchRef == "" {
		return Worktree{}, BranchReferenceRequiredError{}
	}

	repositoryPath := repository.mainWorktreePath
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

		preferredRemote, found, err := preferredRemote(ctx, repositoryPath, model.git)
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

	if err := validateBranchName(ctx, repositoryPath, localBranch, model.git); err != nil {
		return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
	}

	if isExplicitRemoteBranch {
		output, err := model.git.Remotes(ctx, repositoryPath)
		if err != nil {
			return Worktree{}, fmt.Errorf("listing remotes: %w", err)
		}

		remoteExists := false
		for _, remote := range output {
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
		if err := fetchRemote(ctx, repositoryPath, requestedRemote, model.git); err != nil {
			return Worktree{}, err
		}

		branches, err := listRemoteBranches(ctx, repositoryPath, requestedRemote, model.git)
		if err != nil {
			return Worktree{}, err
		}
		for _, branch := range branches {
			remoteBranches[branch] = true
		}
	}

	worktrees, err := model.listWorktrees(ctx, repository)
	if err != nil {
		return Worktree{}, err
	}
	for _, worktree := range worktrees {
		if worktree.Branch != nil && worktree.Branch.Name == localBranch {
			return worktree, nil
		}
	}

	localBranches, err := listLocalBranches(ctx, repositoryPath, model.git)
	if err != nil {
		return Worktree{}, err
	}
	targetPath, err := worktreePath(repositoryPath, repository.repository.ID, localBranch)
	if err != nil {
		return Worktree{}, err
	}

	if localBranches[localBranch] {
		if err := model.git.AddWorktree(ctx, repositoryPath, targetPath, localBranch); err != nil {
			return Worktree{}, fmt.Errorf("creating worktree: %w", err)
		}
	} else if hasRemote && remoteBranches[localBranch] && !isExplicitLocalBranch {
		remoteBranch := requestedRemote + "/" + localBranch
		if err := model.git.AddTrackingWorktree(ctx, repositoryPath, localBranch, targetPath, remoteBranch); err != nil {
			return Worktree{}, fmt.Errorf("checking out remote worktree: %w", err)
		}
	} else if isExplicitRemoteBranch {
		return Worktree{}, InvalidBranchReferenceError{BranchRef: branchRef}
	} else {
		if err := model.git.AddNewBranchWorktree(ctx, repositoryPath, localBranch, targetPath); err != nil {
			return Worktree{}, fmt.Errorf("creating local branch worktree: %w", err)
		}
	}

	copyWarnings := model.copyIgnoredFiles(ctx, repositoryPath, targetPath)
	createdWorktrees, err := model.listWorktrees(ctx, repository)
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

func (model *Model) removeWorktree(ctx context.Context, repository repositoryContext, worktreeID string) (Worktree, error) {
	target, err := model.getWorktree(ctx, repository, worktreeID)
	if err != nil {
		return Worktree{}, err
	}
	if !target.IsRemovable {
		return Worktree{}, WorktreeNotRemovableError{WorktreeID: worktreeID}
	}

	repositoryPath := repository.mainWorktreePath
	if err := model.git.RemoveWorktree(ctx, repositoryPath, target.path); err != nil {
		return Worktree{}, fmt.Errorf("removing worktree: %w", err)
	}
	if err := model.git.PruneWorktrees(ctx, repositoryPath); err != nil {
		return Worktree{}, fmt.Errorf("pruning worktrees: %w", err)
	}
	if target.Branch == nil || target.Branch.Name == "" {
		return target, nil
	}
	if err := model.git.DeleteBranch(ctx, repositoryPath, target.Branch.Name); err != nil {
		return Worktree{}, fmt.Errorf("deleting local branch: %w", err)
	}

	return target, nil
}

func (model *Model) copyIgnoredFiles(ctx context.Context, mainPath string, targetPath string) []string {
	model.configurationMu.RLock()
	shouldCopy := model.configuration.CopyIgnoredFilesOnWorktreeCreation
	excludes := append([]string(nil), model.configuration.WorktreeCopyExcludes...)
	model.configurationMu.RUnlock()

	if !shouldCopy {
		return nil
	}

	return copyIgnoredFiles(ctx, mainPath, targetPath, excludes, model.git, model.files)
}

func stringReference(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
