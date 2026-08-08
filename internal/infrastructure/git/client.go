package git

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

// Client executes bounded Git commands and parses their technical results.
type Client struct{}

// NewClient constructs a Git command client.
func NewClient() Client {
	return Client{}
}

// IsGitWorktree reports whether a path is inside a Git worktree.
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

// WorktreePaths returns the main worktree first, followed by linked worktrees.
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

// CommonDirectory returns the canonical Git common directory for a worktree.
func (r Client) CommonDirectory(ctx context.Context, workspacePath string) (string, error) {
	commonDirectory, err := r.runString(ctx, time.Second, workspacePath, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}

	return canonicalDirectoryPath(commonDirectory)
}

// HeadReference returns the symbolic HEAD reference when the worktree is attached.
func (r Client) HeadReference(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "symbolic-ref", "--quiet", "HEAD")
}

// HeadCommit returns the current commit when HEAD exists.
func (r Client) HeadCommit(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "rev-parse", "--verify", "--quiet", "HEAD")
}

// OriginRemoteURL returns the configured origin URL when present.
func (r Client) OriginRemoteURL(ctx context.Context, workspacePath string) (string, bool, error) {
	return r.runOptionalString(ctx, time.Second, workspacePath, "config", "--get", "remote.origin.url")
}

// Worktrees parses the repository's worktree porcelain output.
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

// Remotes returns the configured remote names.
func (r Client) Remotes(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "remote")
	return parseLines(output), err
}

// FetchRemote fetches and prunes one remote.
func (r Client) FetchRemote(ctx context.Context, repositoryPath string, remote string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "fetch", remote, "--prune")
	return err
}

// RemoteBranches returns short remote branch names.
func (r Client) RemoteBranches(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "-r", "--format=%(refname:short)")
	return parseLines(output), err
}

// LocalBranches returns short local branch names.
func (r Client) LocalBranches(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "--format=%(refname:short)")
	return parseLines(output), err
}

// ValidateBranchName checks a proposed local branch name with Git.
func (r Client) ValidateBranchName(ctx context.Context, repositoryPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "check-ref-format", "--branch", branch)
	return err
}

// AddWorktree creates a worktree for an existing local branch.
func (r Client) AddWorktree(ctx context.Context, repositoryPath string, targetPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", targetPath, branch)
	return err
}

// AddTrackingWorktree creates a local tracking branch and its worktree.
func (r Client) AddTrackingWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string, remoteBranch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", "--track", "-b", localBranch, targetPath, remoteBranch)
	return err
}

// AddNewBranchWorktree creates a new local branch and its worktree.
func (r Client) AddNewBranchWorktree(ctx context.Context, repositoryPath string, localBranch string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "add", "-b", localBranch, targetPath)
	return err
}

// RemoveWorktree removes a linked worktree through Git.
func (r Client) RemoveWorktree(ctx context.Context, repositoryPath string, targetPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "remove", targetPath)
	return err
}

// PruneWorktrees removes stale worktree administrative data.
func (r Client) PruneWorktrees(ctx context.Context, repositoryPath string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "worktree", "prune")
	return err
}

// DeleteBranch force-deletes a local branch.
func (r Client) DeleteBranch(ctx context.Context, repositoryPath string, branch string) error {
	_, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "branch", "-D", "--", branch)
	return err
}

// IgnoredPaths returns untracked paths ignored by repository rules.
func (r Client) IgnoredPaths(ctx context.Context, repositoryPath string) ([]string, error) {
	output, err := r.runString(ctx, gitCommandTimeout, repositoryPath, "ls-files", "--ignored", "--exclude-standard", "--others", "--directory")
	return parseLines(output), err
}

// RepoRoot returns the repository root containing a directory.
func (r Client) RepoRoot(ctx context.Context, cwd string) (string, error) {
	output, err := r.runBytes(ctx, reviewGitCommandTimeout, cwd, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// VerifyHead checks that the repository has a current commit.
func (r Client) VerifyHead(ctx context.Context, repoRoot string) error {
	_, err := r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "HEAD")
	return err
}

// ReviewCurrentBranch returns the raw current branch name for snapshot parsing.
func (r Client) ReviewCurrentBranch(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "branch", "--show-current")
}

// TrackedDiffNameStatus returns NUL-delimited tracked working-tree changes.
func (r Client) TrackedDiffNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", "HEAD", "--")
}

// UntrackedFiles returns NUL-delimited untracked file paths.
func (r Client) UntrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--others", "--exclude-standard", "-z")
}

// TrackedFiles returns NUL-delimited tracked file paths.
func (r Client) TrackedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--cached", "-z")
}

// DeletedFiles returns NUL-delimited deleted file paths.
func (r Client) DeletedFiles(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "ls-files", "--deleted", "-z")
}

// LastCommitNameStatus returns NUL-delimited changes from the last commit.
func (r Client) LastCommitNameStatus(ctx context.Context, repoRoot string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff-tree", "--root", "--find-renames", "-M", "--name-status", "-z", "--no-commit-id", "-r", "HEAD")
}

// DiffNameStatusBetween returns NUL-delimited changes between two revisions.
func (r Client) DiffNameStatusBetween(ctx context.Context, repoRoot string, originalRevision string, modifiedRevision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "diff", "--find-renames", "-M", "--name-status", "-z", originalRevision, modifiedRevision, "--")
}

// CommitRevision resolves a revision to a verified commit identifier.
func (r Client) CommitRevision(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "rev-parse", "--verify", "--quiet", fmt.Sprintf("%s^{commit}", revision))
}

// MergeBase returns the merge base between a revision and HEAD.
func (r Client) MergeBase(ctx context.Context, repoRoot string, revision string) ([]byte, error) {
	return r.runBytes(ctx, reviewGitCommandTimeout, repoRoot, "git", "merge-base", revision, "HEAD")
}

// RevisionContent returns a file's contents at a pinned revision.
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
