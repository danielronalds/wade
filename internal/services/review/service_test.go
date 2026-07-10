// NOTE: Vibecoded and not suppppppper reviewed
package review

// TODO: Review properly

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"wade/internal/repositories"
)

func TestParseNameStatusZ(t *testing.T) {
	output := []byte("M\x00src/app.ts\x00A\x00new file.ts\x00D\x00old.txt\x00R100\x00from name.txt\x00to name.txt\x00")

	got := parseNameStatusZ(output)
	want := []changedPath{
		{status: StatusModified, oldPath: stringPtr("src/app.ts"), newPath: stringPtr("src/app.ts")},
		{status: StatusAdded, oldPath: nil, newPath: stringPtr("new file.ts")},
		{status: StatusDeleted, oldPath: stringPtr("old.txt"), newPath: nil},
		{status: StatusRenamed, oldPath: stringPtr("from name.txt"), newPath: stringPtr("to name.txt")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNameStatusZ() = %#v, want %#v", got, want)
	}
}

func TestWindowDataAndLoadFileContents(t *testing.T) {
	requireGit(t)

	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init")
	runGitCommand(t, repoRoot, "config", "user.email", "wade@example.com")
	runGitCommand(t, repoRoot, "config", "user.name", "WADE")

	writeReviewTestFile(t, repoRoot, "tracked.txt", "old\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "initial")

	writeReviewTestFile(t, repoRoot, "last.txt", "last\n")
	runGitCommand(t, repoRoot, "add", "last.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "add last")

	writeReviewTestFile(t, repoRoot, "tracked.txt", "new\n")
	writeReviewTestFile(t, repoRoot, "untracked.txt", "fresh\n")

	service := newTestService()
	data, err := service.BuildWindowData(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("BuildWindowData() error = %v, want nil", err)
	}

	trackedFile := findReviewFile(t, data.Files, "tracked.txt")
	if !trackedFile.InGitDiff || trackedFile.GitDiff == nil || trackedFile.GitDiff.Status != StatusModified {
		t.Fatalf("tracked file = %#v, want modified git diff file", trackedFile)
	}

	untrackedFile := findReviewFile(t, data.Files, "untracked.txt")
	if !untrackedFile.InGitDiff || untrackedFile.GitDiff == nil || untrackedFile.GitDiff.Status != StatusAdded {
		t.Fatalf("untracked file = %#v, want added git diff file", untrackedFile)
	}

	lastCommitFile := findReviewFile(t, data.Files, "last.txt")
	if !lastCommitFile.InLastCommit || lastCommitFile.LastCommit == nil || lastCommitFile.LastCommit.Status != StatusAdded {
		t.Fatalf("last commit file = %#v, want added last commit file", lastCommitFile)
	}

	trackedContents, err := service.LoadFileContents(context.Background(), data.RepoRoot, trackedFile, ScopeGitDiff)
	if err != nil {
		t.Fatalf("LoadFileContents(tracked) error = %v, want nil", err)
	}

	if trackedContents.OriginalContent != "old\n" || trackedContents.ModifiedContent != "new\n" {
		t.Fatalf("tracked contents = %#v, want old/new", trackedContents)
	}

	untrackedContents, err := service.LoadFileContents(context.Background(), data.RepoRoot, untrackedFile, ScopeGitDiff)
	if err != nil {
		t.Fatalf("LoadFileContents(untracked) error = %v, want nil", err)
	}

	if untrackedContents.OriginalContent != "" || untrackedContents.ModifiedContent != "fresh\n" {
		t.Fatalf("untracked contents = %#v, want empty/fresh", untrackedContents)
	}
}

func TestWindowDataAndLoadFileContentsIncludesOpenPullRequest(t *testing.T) {
	requireGit(t)

	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init")
	runGitCommand(t, repoRoot, "config", "user.email", "wade@example.com")
	runGitCommand(t, repoRoot, "config", "user.name", "WADE")

	writeReviewTestFile(t, repoRoot, "tracked.txt", "base\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "base")
	runGitCommand(t, repoRoot, "branch", "-M", "main")
	runGitCommand(t, repoRoot, "checkout", "-b", "feature/pr-view")

	writeReviewTestFile(t, repoRoot, "tracked.txt", "branch\n")
	writeReviewTestFile(t, repoRoot, "pr-only.txt", "created\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt", "pr-only.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "pr changes")
	writeReviewTestFile(t, repoRoot, "local-only.txt", "draft\n")

	installFakeGH(t, `{"number":12,"url":"https://github.com/example/wade/pull/12","state":"OPEN","baseRefName":"main","headRefName":"feature/pr-view"}`)

	service := newTestService()
	data, err := service.BuildWindowData(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("BuildWindowData() error = %v, want nil", err)
	}

	if data.PullRequest == nil || data.PullRequest.Number != 12 || data.PullRequest.BaseRefName != "main" {
		t.Fatalf("pull request = %#v, want open PR metadata", data.PullRequest)
	}

	trackedFile := findReviewFile(t, data.Files, "tracked.txt")
	if !trackedFile.InPullRequest || trackedFile.PullRequest == nil || trackedFile.PullRequest.Status != StatusModified {
		t.Fatalf("tracked file = %#v, want modified pull request file", trackedFile)
	}

	trackedContents, err := service.LoadFileContents(context.Background(), data.RepoRoot, trackedFile, ScopePullRequest)
	if err != nil {
		t.Fatalf("LoadFileContents(tracked PR) error = %v, want nil", err)
	}

	if trackedContents.OriginalContent != "base\n" || trackedContents.ModifiedContent != "branch\n" {
		t.Fatalf("tracked PR contents = %#v, want base/branch", trackedContents)
	}

	addedFile := findReviewFile(t, data.Files, "pr-only.txt")
	if !addedFile.InPullRequest || addedFile.PullRequest == nil || addedFile.PullRequest.Status != StatusAdded {
		t.Fatalf("added file = %#v, want added pull request file", addedFile)
	}

	localFile := findReviewFile(t, data.Files, "local-only.txt")
	if localFile.InPullRequest || localFile.PullRequest != nil {
		t.Fatalf("local file = %#v, want no pull request comparison", localFile)
	}
}

func TestWindowDataExcludesClosedPullRequest(t *testing.T) {
	requireGit(t)

	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init")
	runGitCommand(t, repoRoot, "config", "user.email", "wade@example.com")
	runGitCommand(t, repoRoot, "config", "user.name", "WADE")

	writeReviewTestFile(t, repoRoot, "tracked.txt", "base\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "base")
	runGitCommand(t, repoRoot, "branch", "-M", "main")
	runGitCommand(t, repoRoot, "checkout", "-b", "feature/pr-view")
	writeReviewTestFile(t, repoRoot, "tracked.txt", "branch\n")
	runGitCommand(t, repoRoot, "add", "tracked.txt")
	runGitCommand(t, repoRoot, "commit", "-m", "pr changes")

	installFakeGH(t, `{"number":12,"url":"https://github.com/example/wade/pull/12","state":"CLOSED","baseRefName":"main","headRefName":"feature/pr-view"}`)

	service := newTestService()
	data, err := service.BuildWindowData(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("BuildWindowData() error = %v, want nil", err)
	}

	if data.PullRequest != nil {
		t.Fatalf("pull request = %#v, want nil", data.PullRequest)
	}

	trackedFile := findReviewFile(t, data.Files, "tracked.txt")
	if trackedFile.InPullRequest || trackedFile.PullRequest != nil {
		t.Fatalf("tracked file = %#v, want no pull request comparison", trackedFile)
	}
}

func newTestService() Service {
	return NewService(repositories.NewGitRepository(), repositories.NewGitHubRepository(repositories.RunCommand), repositories.NewFileRepository())
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func installFakeGH(t *testing.T, response string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"pr\" ] && [ \"$2\" = \"view\" ]; then\ncat <<'JSON'\n" + response + "\nJSON\nexit 0\nfi\nexit 1\n"

	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", ghPath, err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func runGitCommand(t *testing.T, directory string, args ...string) {
	t.Helper()

	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v error = %v\n%s", args, err, output)
	}
}

func writeReviewTestFile(t *testing.T, repoRoot string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(repoRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(fullPath), err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fullPath, err)
	}
}

func findReviewFile(t *testing.T, files []File, path string) File {
	t.Helper()

	for _, file := range files {
		if file.Path == path {
			return file
		}
	}

	t.Fatalf("file %q not found in %#v", path, files)
	return File{}
}
