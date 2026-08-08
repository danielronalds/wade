package git

// TODO: Review properly

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitCommandTimeout = 30 * time.Second
const reviewGitCommandTimeout = 5 * time.Second

// Worktree is parsed technical data from Git's worktree porcelain format.
type Worktree struct {
	Path      string
	BranchRef string
}

type Client struct{}

func NewClient() Client {
	return Client{}
}

func (r Client) IsGitWorktree(ctx context.Context, workspacePath string) (bool, error) {
	output, err := r.runString(ctx, time.Second, workspacePath, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		return false, nil
	}

	return output == "true", nil
}

func (r Client) WorktreePaths(ctx context.Context, workspacePath string) ([]string, error) {
	worktrees, err := r.Worktrees(ctx, workspacePath)
	if err != nil {
		return nil, err
	}

	worktreePaths := make([]string, 0, len(worktrees))
	for index, worktree := range worktrees {
		worktreePath := worktree.Path
		if index == 0 {
			worktreePath, err = canonicalDirectoryPath(worktreePath)
			if err != nil {
				return nil, err
			}
		}
		worktreePaths = append(worktreePaths, worktreePath)
	}
	if len(worktreePaths) == 0 {
		return nil, errors.New("could not determine main worktree path")
	}
	return worktreePaths, nil
}

func (r Client) CommonDirectory(ctx context.Context, workspacePath string) (string, error) {
	commonDirectory, err := r.runString(ctx, time.Second, workspacePath, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}

	return canonicalDirectoryPath(commonDirectory)
}

func (r Client) HeadReference(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "symbolic-ref", "--quiet", "HEAD")
}

func (r Client) HeadCommit(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "rev-parse", "--verify", "--quiet", "HEAD")
}

func (r Client) OriginRemoteURL(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "config", "--get", "remote.origin.url")
}

func (r Client) Worktrees(ctx context.Context, repositoryPath string) ([]Worktree, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	worktrees := make([]Worktree, 0)
	var current *Worktree
	for _, line := range strings.Split(output, "\n") {
		if path, found := strings.CutPrefix(line, "worktree "); found {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{Path: path}
			continue
		}
		if current != nil {
			if branchRef, found := strings.CutPrefix(line, "branch "); found {
				current.BranchRef = branchRef
			}
		}
	}
	if current != nil {
		worktrees = append(worktrees, *current)
	}
	return worktrees, nil
}

func (r Client) Remotes(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "remote")
	return parseLines(output), err
}

func (r Client) FetchRemote(ctx context.Context, repositoryPath string, remote string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "fetch", remote, "--prune")
	return err
}

func (r Client) RemoteBranches(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "-r", "--format=%(refname:short)")
	return parseLines(output), err
}

func (r Client) LocalBranches(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "--format=%(refname:short)")
	return parseLines(output), err
}

func (r Client) ValidateBranchName(ctx context.Context, repositoryPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "check-ref-format", "--branch", branch)
	return err
}

func (r Client) AddWorktree(ctx context.Context, repositoryPath string, targetPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", targetPath, branch)
	return err
}

func (r Client) AddTrackingWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string, remoteBranch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", "--track", "-b", localBranch, targetPath, remoteBranch)
	return err
}

func (r Client) AddNewBranchWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", "-b", localBranch, targetPath)
	return err
}

func (r Client) RemoveWorktree(ctx context.Context, repositoryPath string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "remove", targetPath)
	return err
}

func (r Client) PruneWorktrees(ctx context.Context, repositoryPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "prune")
	return err
}

func (r Client) DeleteBranch(ctx context.Context, repositoryPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "-D", "--", branch)
	return err
}

func (r Client) IgnoredPaths(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "ls-files", "--ignored", "--exclude-standard", "--others", "--directory")
	return parseLines(output), err
}

func (r Client) RepoRoot(ctx context.Context, cwd string) (string, error) {
	output, err := r.runBytes(ctx, reviewGitCommandTimeout, cwd, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (r Client) VerifyHead(ctx context.Context, repoRoot string) error {
	_, err := r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "HEAD")
	return err
}

func (r Client) ReviewCurrentBranch(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "branch", "--show-current")
}

func (r Client) TrackedDiffNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", "HEAD", "--")
}

func (r Client) UntrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--others", "--exclude-standard", "-z")
}

func (r Client) TrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--cached", "-z")
}

func (r Client) DeletedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--deleted", "-z")
}

func (r Client) LastCommitNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff-tree", "--root", "--find-renames", "-M", "--name-status", "-z", "--no-commit-id", "-r", "HEAD")
}

func (r Client) DiffNameStatusBetween(ctx context.Context, repoRoot string, originalRevision string, modifiedRevision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", originalRevision, modifiedRevision, "--")
}

func (r Client) CommitRevision(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "--quiet", fmt.Sprintf("%s^{commit}", revision))
}

func (r Client) MergeBase(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "merge-base", revision, "HEAD")
}

func (r Client) RevisionContent(ctx context.Context, repoRoot string, revision string, filePath string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "show", fmt.Sprintf("%s:%s", revision, filePath))
}

func parseLines(output string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func canonicalDirectoryPath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonicalPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonicalPath), nil
}

func (r Client) runString(ctx context.Context, timeout time.Duration, repositoryPath string, args ...string) (string, error) {
	output, err := r.runBytes(ctx, timeout, repositoryPath, "git", append([]string{"-C", repositoryPath}, args...)...)
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}

	return text, nil
}

func (r Client) runOptionalString(ctx context.Context, timeout time.Duration, repositoryPath string, args ...string) (string, bool, error) {
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(commandContext, "git", append([]string{"-C", repositoryPath}, args...)...)
	command.Dir = repositoryPath
	output, err := command.CombinedOutput()
	if err == nil {
		return strings.TrimSpace(string(output)), true, nil
	}
	if commandContext.Err() != nil {
		return "", false, commandContext.Err()
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", false, nil
	}

	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return "", false, fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
}

func (r Client) runBytes(ctx context.Context, timeout time.Duration, directory string, name string, args ...string) ([]byte, error) {
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
