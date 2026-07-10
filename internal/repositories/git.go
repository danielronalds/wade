package repositories

// TODO: Review properly

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const gitCommandTimeout = 30 * time.Second
const reviewGitCommandTimeout = 5 * time.Second

type GitRepository struct{}

func NewGitRepository() GitRepository {
	return GitRepository{}
}

func (r GitRepository) CurrentBranch(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, time.Second, projectPath, "branch", "--show-current")
}

func (r GitRepository) OriginURL(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, time.Second, projectPath, "remote", "get-url", "origin")
}

func (r GitRepository) WorktreeListPorcelain(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "list", "--porcelain")
}

func (r GitRepository) Remotes(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, gitCommandTimeout, projectPath, "remote")
}

func (r GitRepository) FetchRemote(ctx context.Context, projectPath string, remote string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "fetch", remote, "--prune")
	return err
}

func (r GitRepository) RemoteBranches(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, gitCommandTimeout, projectPath, "branch", "-r", "--format=%(refname:short)")
}

func (r GitRepository) LocalBranches(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, gitCommandTimeout, projectPath, "branch", "--format=%(refname:short)")
}

func (r GitRepository) ValidateBranchName(ctx context.Context, projectPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "check-ref-format", "--branch", branch)
	return err
}

func (r GitRepository) AddWorktree(ctx context.Context, projectPath string, targetPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "add", targetPath, branch)
	return err
}

func (r GitRepository) AddTrackingWorktree(ctx context.Context, projectPath string, localBranch string, targetPath string, remoteBranch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "add", "--track", "-b", localBranch, targetPath, remoteBranch)
	return err
}

func (r GitRepository) AddNewBranchWorktree(ctx context.Context, projectPath string, localBranch string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "add", "-b", localBranch, targetPath)
	return err
}

func (r GitRepository) RemoveWorktree(ctx context.Context, projectPath string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "remove", targetPath)
	return err
}

func (r GitRepository) PruneWorktrees(ctx context.Context, projectPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "worktree", "prune")
	return err
}

func (r GitRepository) DeleteBranch(ctx context.Context, projectPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, projectPath, "branch", "-D", "--", branch)
	return err
}

func (r GitRepository) IgnoredPaths(ctx context.Context, projectPath string) (string, error) {
	return r.runString(ctx, gitCommandTimeout, projectPath, "ls-files", "--ignored", "--exclude-standard", "--others", "--directory")
}

func (r GitRepository) RepoRoot(ctx context.Context, cwd string) (string, error) {
	output, err := r.runBytes(ctx, reviewGitCommandTimeout, cwd, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (r GitRepository) VerifyHead(ctx context.Context, repoRoot string) error {
	_, err := r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "HEAD")
	return err
}

func (r GitRepository) ReviewCurrentBranch(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "branch", "--show-current")
}

func (r GitRepository) TrackedDiffNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", "HEAD", "--")
}

func (r GitRepository) UntrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--others", "--exclude-standard", "-z")
}

func (r GitRepository) TrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--cached", "-z")
}

func (r GitRepository) DeletedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--deleted", "-z")
}

func (r GitRepository) LastCommitNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff-tree", "--root", "--find-renames", "-M", "--name-status", "-z", "--no-commit-id", "-r", "HEAD")
}

func (r GitRepository) DiffNameStatusBetween(ctx context.Context, repoRoot string, originalRevision string, modifiedRevision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", originalRevision, modifiedRevision, "--")
}

func (r GitRepository) CommitRevision(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "--quiet", fmt.Sprintf("%s^{commit}", revision))
}

func (r GitRepository) MergeBase(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "merge-base", revision, "HEAD")
}

func (r GitRepository) RevisionContent(ctx context.Context, repoRoot string, revision string, filePath string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "show", fmt.Sprintf("%s:%s", revision, filePath))
}

func (r GitRepository) runString(ctx context.Context, timeout time.Duration, projectPath string, args ...string) (string, error) {
	output, err := r.runBytes(ctx, timeout, projectPath, "git", append([]string{"-C", projectPath}, args...)...)
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}

	return text, nil
}

func (r GitRepository) runBytes(ctx context.Context, timeout time.Duration, directory string, name string, args ...string) ([]byte, error) {
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(commandContext, name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%s %s failed: %s", name, strings.Join(args, " "), message)
	}

	return output, nil
}
