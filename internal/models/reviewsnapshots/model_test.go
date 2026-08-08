package reviewsnapshots

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"wade/internal/infrastructure/github"
)

type workspaceDiscoveryStub struct {
	path  string
	found bool
	err   error
}

func (stub workspaceDiscoveryStub) Resolve(string) (string, bool, error) {
	return stub.path, stub.found, stub.err
}

type gitStub struct {
	repoRoot         string
	repoRootError    error
	verifyHeadError  error
	currentBranch    []byte
	trackedDiff      []byte
	untrackedFiles   []byte
	trackedFiles     []byte
	deletedFiles     []byte
	lastCommit       []byte
	diffBetween      []byte
	commitRevisions  map[string][]byte
	mergeBases       map[string][]byte
	revisionContents map[string][]byte
}

func (stub gitStub) RepoRoot(context.Context, string) (string, error) {
	return stub.repoRoot, stub.repoRootError
}
func (stub gitStub) VerifyHead(context.Context, string) error {
	return stub.verifyHeadError
}
func (stub gitStub) ReviewCurrentBranch(context.Context, string) ([]byte, error) {
	return stub.currentBranch, nil
}
func (stub gitStub) TrackedDiffNameStatus(context.Context, string) ([]byte, error) {
	return stub.trackedDiff, nil
}
func (stub gitStub) UntrackedFiles(context.Context, string) ([]byte, error) {
	return stub.untrackedFiles, nil
}
func (stub gitStub) TrackedFiles(context.Context, string) ([]byte, error) {
	return stub.trackedFiles, nil
}
func (stub gitStub) DeletedFiles(context.Context, string) ([]byte, error) {
	return stub.deletedFiles, nil
}
func (stub gitStub) LastCommitNameStatus(context.Context, string) ([]byte, error) {
	return stub.lastCommit, nil
}
func (stub gitStub) DiffNameStatusBetween(context.Context, string, string, string) ([]byte, error) {
	return stub.diffBetween, nil
}
func (stub gitStub) CommitRevision(_ context.Context, _ string, revision string) ([]byte, error) {
	output, found := stub.commitRevisions[revision]
	if !found {
		return nil, errors.New("revision not found")
	}
	return output, nil
}
func (stub gitStub) MergeBase(_ context.Context, _ string, revision string) ([]byte, error) {
	output, found := stub.mergeBases[revision]
	if !found {
		return nil, errors.New("merge base not found")
	}
	return output, nil
}
func (stub gitStub) RevisionContent(_ context.Context, _ string, revision string, filePath string) ([]byte, error) {
	output, found := stub.revisionContents[revision+":"+filePath]
	if !found {
		return nil, errors.New("content not found")
	}
	return output, nil
}

type gitHubStub struct {
	pullRequest *github.PullRequest
	err         error
}

func (stub gitHubStub) PullRequest(context.Context, string, string) (*github.PullRequest, error) {
	return stub.pullRequest, stub.err
}

type fileSystemStub struct {
	mu      sync.RWMutex
	content map[string][]byte
}

func (stub *fileSystemStub) ReadFile(path string) ([]byte, error) {
	stub.mu.RLock()
	defer stub.mu.RUnlock()
	content, found := stub.content[path]
	if !found {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), content...), nil
}

func (stub *fileSystemStub) set(path string, content string) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.content[path] = []byte(content)
}

func TestModelSnapshotLifecycleAndPinnedContents(t *testing.T) {
	model, files := newSnapshotTestModel()

	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if snapshot.ID == "" || snapshot.WorkspaceID != "wade" {
		t.Fatalf("Create() identity = %q/%q, want generated/wade", snapshot.ID, snapshot.WorkspaceID)
	}
	if snapshot.Branch == nil || snapshot.Branch.Ref != "refs/heads/feature/review" {
		t.Fatalf("Create() Branch = %#v, want feature/review", snapshot.Branch)
	}
	if snapshot.PullRequest == nil || snapshot.PullRequest.Number != 12 || snapshot.PullRequest.BaseRef != "refs/heads/main" {
		t.Fatalf("Create() PullRequest = %#v", snapshot.PullRequest)
	}

	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	files.set(filepath.Join("/repo", "tracked.txt"), "changed after snapshot\n")

	workingTreeContents, err := model.FileContents(context.Background(), snapshot.ID, file.ID, ScopeWorkingTree)
	if err != nil {
		t.Fatalf("FileContents(working tree) error = %v, want nil", err)
	}
	if workingTreeContents.OriginalContent != "head\n" || workingTreeContents.ModifiedContent != "working\n" {
		t.Fatalf("FileContents(working tree) = %#v, want pinned head/working", workingTreeContents)
	}

	untrackedFile := findReviewFile(t, snapshot.Files, "untracked.txt")
	untrackedContents, err := model.FileContents(context.Background(), snapshot.ID, untrackedFile.ID, ScopeWorkingTree)
	if err != nil {
		t.Fatalf("FileContents(untracked) error = %v, want nil", err)
	}
	if untrackedContents.OriginalContent != "" || untrackedContents.ModifiedContent != "fresh\n" {
		t.Fatalf("FileContents(untracked) = %#v, want empty/fresh", untrackedContents)
	}

	lastCommitContents, err := model.FileContents(context.Background(), snapshot.ID, file.ID, ScopeLastCommit)
	if err != nil {
		t.Fatalf("FileContents(last commit) error = %v, want nil", err)
	}
	if lastCommitContents.OriginalContent != "parent\n" || lastCommitContents.ModifiedContent != "head\n" {
		t.Fatalf("FileContents(last commit) = %#v, want pinned parent/head", lastCommitContents)
	}

	pullRequestContents, err := model.FileContents(context.Background(), snapshot.ID, file.ID, ScopePullRequest)
	if err != nil {
		t.Fatalf("FileContents(pull request) error = %v, want nil", err)
	}
	if pullRequestContents.OriginalContent != "base\n" || pullRequestContents.ModifiedContent != "head\n" {
		t.Fatalf("FileContents(pull request) = %#v, want pinned base/head", pullRequestContents)
	}

	currentContents, err := model.FileContents(context.Background(), snapshot.ID, file.ID, ScopeCurrent)
	if err != nil {
		t.Fatalf("FileContents(current) error = %v, want nil", err)
	}
	if currentContents.OriginalContent != "changed after snapshot\n" || currentContents.ModifiedContent != "changed after snapshot\n" {
		t.Fatalf("FileContents(current) = %#v, want current working tree content", currentContents)
	}

	if err := model.Delete(snapshot.ID); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	_, err = model.Get(snapshot.ID)
	var notFoundError SnapshotNotFoundError
	if !errors.As(err, &notFoundError) {
		t.Fatalf("Get() error = %v, want SnapshotNotFoundError", err)
	}
}

func TestModelReturnsDefensiveSnapshotCopies(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	originalFileID := file.ID
	*snapshot.Files[0].GitDiff.OldPath = "mutated.txt"
	*snapshot.Files[0].WorktreeStatus = StatusDeleted
	snapshot.Files[0].ID = "mutated"
	snapshot.Branch.Name = "mutated"
	snapshot.PullRequest.URL = "https://example.invalid"

	loaded, err := model.Get(snapshot.ID)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	loadedFile := findReviewFile(t, loaded.Files, "tracked.txt")
	if loadedFile.ID != originalFileID || *loadedFile.GitDiff.OldPath != "tracked.txt" || *loadedFile.WorktreeStatus != StatusModified {
		t.Fatalf("Get() file = %#v, want unchanged detached value", loadedFile)
	}
	if loaded.Branch.Name != "feature/review" || loaded.PullRequest.URL != "https://github.com/example/wade/pull/12" {
		t.Fatalf("Get() metadata = %#v/%#v, want unchanged", loaded.Branch, loaded.PullRequest)
	}
}

func TestModelValidatesSnapshotAndFileLookups(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	_, err = model.FileContents(context.Background(), snapshot.ID, "unknown", ScopeCurrent)
	var fileNotFoundError SnapshotFileNotFoundError
	if !errors.As(err, &fileNotFoundError) {
		t.Fatalf("FileContents() error = %v, want SnapshotFileNotFoundError", err)
	}

	_, err = model.FileContents(context.Background(), snapshot.ID, snapshot.Files[0].ID, Scope("invalid"))
	var invalidScopeError InvalidScopeError
	if !errors.As(err, &invalidScopeError) {
		t.Fatalf("FileContents() error = %v, want InvalidScopeError", err)
	}

	_, err = model.FileContents(context.Background(), "unknown", snapshot.Files[0].ID, ScopeCurrent)
	var snapshotNotFoundError SnapshotNotFoundError
	if !errors.As(err, &snapshotNotFoundError) {
		t.Fatalf("FileContents() error = %v, want SnapshotNotFoundError", err)
	}
}

func TestModelExcludesClosedPullRequests(t *testing.T) {
	model, _ := newSnapshotTestModel()
	model.github = gitHubStub{pullRequest: &github.PullRequest{
		Number:      12,
		URL:         "https://github.com/example/wade/pull/12",
		State:       "CLOSED",
		BaseRefName: "main",
		HeadRefName: "feature/review",
	}}

	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if snapshot.PullRequest != nil {
		t.Fatalf("Create() PullRequest = %#v, want nil", snapshot.PullRequest)
	}
	file := findReviewFile(t, snapshot.Files, "tracked.txt")
	if file.InPullRequest || file.PullRequest != nil {
		t.Fatalf("Create() file = %#v, want no pull request comparison", file)
	}
}

func TestModelReportsUnknownAndNonGitWorkspaces(t *testing.T) {
	model := New(workspaceDiscoveryStub{}, gitStub{}, gitHubStub{}, &fileSystemStub{})
	_, err := model.Create(context.Background(), "missing")
	var workspaceNotFoundError WorkspaceNotFoundError
	if !errors.As(err, &workspaceNotFoundError) {
		t.Fatalf("Create() error = %v, want WorkspaceNotFoundError", err)
	}

	model = New(
		workspaceDiscoveryStub{path: "/repo", found: true},
		gitStub{repoRootError: errors.New("not a repository")},
		gitHubStub{},
		&fileSystemStub{},
	)
	_, err = model.Create(context.Background(), "notes")
	var notGitError WorkspaceNotGitRepositoryError
	if !errors.As(err, &notGitError) {
		t.Fatalf("Create() error = %v, want WorkspaceNotGitRepositoryError", err)
	}
}

func TestModelSnapshotRegistryIsEphemeral(t *testing.T) {
	model, _ := newSnapshotTestModel()
	snapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	restarted, _ := newSnapshotTestModel()
	_, err = restarted.Get(snapshot.ID)
	var notFoundError SnapshotNotFoundError
	if !errors.As(err, &notFoundError) {
		t.Fatalf("Get() error = %v, want SnapshotNotFoundError", err)
	}
}

func TestModelSnapshotRegistrySupportsConcurrentAccess(t *testing.T) {
	model, _ := newSnapshotTestModel()
	baseSnapshot, err := model.Create(context.Background(), "wade")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	var wait sync.WaitGroup
	for range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 10 {
				if _, err := model.Get(baseSnapshot.ID); err != nil {
					t.Errorf("Get() error = %v, want nil", err)
				}
				snapshot, err := model.Create(context.Background(), "wade")
				if err != nil {
					t.Errorf("Create() error = %v, want nil", err)
					return
				}
				if err := model.Delete(snapshot.ID); err != nil {
					t.Errorf("Delete() error = %v, want nil", err)
				}
			}
		}()
	}
	wait.Wait()
}

func newSnapshotTestModel() (*Model, *fileSystemStub) {
	files := &fileSystemStub{content: map[string][]byte{
		filepath.Join("/repo", "tracked.txt"):   []byte("working\n"),
		filepath.Join("/repo", "untracked.txt"): []byte("fresh\n"),
	}}
	git := gitStub{
		repoRoot:       "/repo",
		currentBranch:  []byte("feature/review\n"),
		trackedDiff:    []byte("M\x00tracked.txt\x00"),
		untrackedFiles: []byte("untracked.txt\x00"),
		trackedFiles:   []byte("tracked.txt\x00"),
		lastCommit:     []byte("M\x00tracked.txt\x00"),
		diffBetween:    []byte("M\x00tracked.txt\x00"),
		commitRevisions: map[string][]byte{
			"HEAD":                     []byte("head-1\n"),
			"HEAD^":                    []byte("parent-1\n"),
			"refs/remotes/origin/main": []byte("base-1\n"),
		},
		mergeBases: map[string][]byte{
			"base-1": []byte("base-1\n"),
		},
		revisionContents: map[string][]byte{
			"head-1:tracked.txt":   []byte("head\n"),
			"parent-1:tracked.txt": []byte("parent\n"),
			"base-1:tracked.txt":   []byte("base\n"),
		},
	}
	github := gitHubStub{pullRequest: &github.PullRequest{
		Number:      12,
		URL:         "https://github.com/example/wade/pull/12",
		State:       "OPEN",
		BaseRefName: "main",
		HeadRefName: "feature/review",
	}}
	return New(workspaceDiscoveryStub{path: "/repo", found: true}, git, github, files), files
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
