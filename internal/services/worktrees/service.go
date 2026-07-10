// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"wade/internal/services/config"
)

type gitRepository interface {
	WorktreeListPorcelain(ctx context.Context, projectPath string) (string, error)
	Remotes(ctx context.Context, projectPath string) (string, error)
	FetchRemote(ctx context.Context, projectPath string, remote string) error
	RemoteBranches(ctx context.Context, projectPath string) (string, error)
	LocalBranches(ctx context.Context, projectPath string) (string, error)
	ValidateBranchName(ctx context.Context, projectPath string, branch string) error
	AddWorktree(ctx context.Context, projectPath string, targetPath string, branch string) error
	AddTrackingWorktree(ctx context.Context, projectPath string, localBranch string, targetPath string, remoteBranch string) error
	AddNewBranchWorktree(ctx context.Context, projectPath string, localBranch string, targetPath string) error
	RemoveWorktree(ctx context.Context, projectPath string, targetPath string) error
	PruneWorktrees(ctx context.Context, projectPath string) error
	DeleteBranch(ctx context.Context, projectPath string, branch string) error
	IgnoredPaths(ctx context.Context, projectPath string) (string, error)
}

type fileRepository interface {
	CopyPath(source string, destination string) error
}

type Service struct {
	copyIgnoredFilesOnWorktreeCreation bool
	worktreeCopyExcludes               []string
	git                                gitRepository
	files                              fileRepository
}

func NewService(configuration config.Config, git gitRepository, files fileRepository) Service {
	return Service{
		copyIgnoredFilesOnWorktreeCreation: configuration.CopyIgnoredFilesOnWorktreeCreation,
		worktreeCopyExcludes:               append([]string(nil), configuration.WorktreeCopyExcludes...),
		git:                                git,
		files:                              files,
	}
}

func (s Service) List(ctx context.Context, projectPath string) ([]Worktree, error) {
	data, err := resolveContext(ctx, projectPath, s.git)
	if err != nil {
		return nil, err
	}

	return buildWorktrees(data, projectPath), nil
}

func (s Service) RemoteBranches(ctx context.Context, projectPath string) (RemoteBranchList, error) {
	data, err := resolveContext(ctx, projectPath, s.git)
	if err != nil {
		return RemoteBranchList{}, err
	}

	remote, found, err := preferredRemote(ctx, data.mainPath, s.git)
	if err != nil {
		return RemoteBranchList{}, err
	}
	if !found {
		return RemoteBranchList{}, errors.New("no git remote configured")
	}

	if err := fetchRemote(ctx, data.mainPath, remote, s.git); err != nil {
		return RemoteBranchList{}, err
	}

	remoteBranches, err := listRemoteBranches(ctx, data.mainPath, remote, s.git)
	if err != nil {
		return RemoteBranchList{}, err
	}

	localBranches, err := listLocalBranches(ctx, data.mainPath, s.git)
	if err != nil {
		return RemoteBranchList{}, err
	}

	worktrees := buildWorktrees(data, projectPath)
	branchWorktrees := worktreesByBranch(worktrees)
	branches := make([]RemoteBranch, 0, len(remoteBranches))

	for _, branch := range remoteBranches {
		worktree, isCheckedOut := branchWorktrees[branch]
		branches = append(branches, RemoteBranch{
			Name:                remote + "/" + branch,
			Branch:              branch,
			HasLocalBranch:      localBranches[branch],
			IsCheckedOut:        isCheckedOut,
			WorktreeName:        worktree.Name,
			WorktreeProjectName: worktree.ProjectName,
		})
	}

	return RemoteBranchList{Remote: remote, Branches: branches}, nil
}

func (s Service) Create(ctx context.Context, projectPath string, requestedBranch string) (Worktree, error) {
	branch := strings.TrimSpace(requestedBranch)
	if branch == "" {
		return Worktree{}, errors.New("branch is required")
	}

	data, err := resolveContext(ctx, projectPath, s.git)
	if err != nil {
		return Worktree{}, err
	}

	remote, hasRemote, err := preferredRemote(ctx, data.mainPath, s.git)
	if err != nil {
		return Worktree{}, err
	}

	remoteBranches := map[string]bool{}
	isExplicitRemoteBranch := false
	localBranch := branch

	if hasRemote {
		if err := fetchRemote(ctx, data.mainPath, remote, s.git); err != nil {
			return Worktree{}, err
		}

		branches, err := listRemoteBranches(ctx, data.mainPath, remote, s.git)
		if err != nil {
			return Worktree{}, err
		}
		for _, branch := range branches {
			remoteBranches[branch] = true
		}

		if strings.HasPrefix(branch, remote+"/") {
			isExplicitRemoteBranch = true
			localBranch = strings.TrimPrefix(branch, remote+"/")
		}
	}

	if err := validateBranchName(ctx, data.mainPath, localBranch, s.git); err != nil {
		return Worktree{}, err
	}

	worktrees := buildWorktrees(data, projectPath)
	if existing, ok := worktreesByBranch(worktrees)[localBranch]; ok {
		return existing, nil
	}

	localBranches, err := listLocalBranches(ctx, data.mainPath, s.git)
	if err != nil {
		return Worktree{}, err
	}

	targetPath, err := worktreePath(data.mainPath, data.projectName, localBranch)
	if err != nil {
		return Worktree{}, err
	}

	if localBranches[localBranch] {
		if err := s.git.AddWorktree(ctx, data.mainPath, targetPath, localBranch); err != nil {
			return Worktree{}, fmt.Errorf("creating worktree: %w", err)
		}
	} else if hasRemote && remoteBranches[localBranch] {
		if err := s.git.AddTrackingWorktree(ctx, data.mainPath, localBranch, targetPath, remote+"/"+localBranch); err != nil {
			return Worktree{}, fmt.Errorf("checking out remote worktree: %w", err)
		}
	} else if isExplicitRemoteBranch {
		return Worktree{}, fmt.Errorf("remote branch %q not found", branch)
	} else {
		if err := s.git.AddNewBranchWorktree(ctx, data.mainPath, localBranch, targetPath); err != nil {
			return Worktree{}, fmt.Errorf("creating local branch worktree: %w", err)
		}
	}

	copyWarnings := s.copyIgnoredFiles(ctx, data.mainPath, targetPath)

	createdData, err := resolveContext(ctx, projectPath, s.git)
	if err != nil {
		return Worktree{}, err
	}

	createdWorktrees := buildWorktrees(createdData, targetPath)
	for _, worktree := range createdWorktrees {
		if samePath(worktree.Path, targetPath) {
			worktree.IgnoredFileCopyWarnings = copyWarnings
			return worktree, nil
		}
	}

	return Worktree{
		Name:                    strings.TrimPrefix(filepath.Base(targetPath), data.projectName+"-"),
		ProjectName:             filepath.Base(targetPath),
		Path:                    targetPath,
		Branch:                  localBranch,
		IsBase:                  false,
		IsCurrent:               true,
		IsRemovable:             true,
		IgnoredFileCopyWarnings: copyWarnings,
	}, nil
}

func (s Service) Find(ctx context.Context, projectPath string, query string) (Worktree, error) {
	target := strings.TrimSpace(query)
	if target == "" {
		return Worktree{}, errors.New("worktree is required")
	}

	worktrees, err := s.List(ctx, projectPath)
	if err != nil {
		return Worktree{}, err
	}

	for _, worktree := range worktrees {
		if worktree.ProjectName == target || worktree.Name == target || worktree.Branch == target || samePath(worktree.Path, target) {
			return worktree, nil
		}
	}

	return Worktree{}, fmt.Errorf("worktree %q not found", target)
}

func (s Service) Remove(ctx context.Context, projectPath string, target Worktree) error {
	if target.IsBase || !target.IsRemovable {
		return errors.New("cannot remove base worktree")
	}

	data, err := resolveContext(ctx, projectPath, s.git)
	if err != nil {
		return err
	}

	if err := s.git.RemoveWorktree(ctx, data.mainPath, target.Path); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}

	if err := s.git.PruneWorktrees(ctx, data.mainPath); err != nil {
		return fmt.Errorf("pruning worktrees: %w", err)
	}

	if target.Branch == "" {
		return nil
	}

	if err := s.git.DeleteBranch(ctx, data.mainPath, target.Branch); err != nil {
		return fmt.Errorf("deleting local branch: %w", err)
	}

	return nil
}

func (s Service) copyIgnoredFiles(ctx context.Context, mainPath string, targetPath string) []string {
	if !s.copyIgnoredFilesOnWorktreeCreation {
		return nil
	}

	return copyIgnoredFiles(ctx, mainPath, targetPath, s.worktreeCopyExcludes, s.git, s.files)
}
