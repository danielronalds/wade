// NOTE: Vibecoded and not suppppppper reviewed
package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const gitCommandTimeout = 5 * time.Second

const (
	ScopePullRequest Scope = "pull-request"
	ScopeGitDiff     Scope = "git-diff"
	ScopeLastCommit  Scope = "last-commit"
	ScopeAllFiles    Scope = "all-files"
)

const (
	StatusModified ChangeStatus = "modified"
	StatusAdded    ChangeStatus = "added"
	StatusDeleted  ChangeStatus = "deleted"
	StatusRenamed  ChangeStatus = "renamed"
)

type Scope string

type ChangeStatus string

type PullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

type FileComparison struct {
	Status      ChangeStatus `json:"status"`
	OldPath     *string      `json:"oldPath"`
	NewPath     *string      `json:"newPath"`
	DisplayPath string       `json:"displayPath"`
	HasOriginal bool         `json:"hasOriginal"`
	HasModified bool         `json:"hasModified"`

	originalRevision string
	modifiedRevision string
}

type File struct {
	ID                 string          `json:"id"`
	Path               string          `json:"path"`
	WorktreeStatus     *ChangeStatus   `json:"worktreeStatus"`
	HasWorkingTreeFile bool            `json:"hasWorkingTreeFile"`
	InGitDiff          bool            `json:"inGitDiff"`
	InLastCommit       bool            `json:"inLastCommit"`
	InPullRequest      bool            `json:"inPullRequest"`
	GitDiff            *FileComparison `json:"gitDiff"`
	LastCommit         *FileComparison `json:"lastCommit"`
	PullRequest        *FileComparison `json:"pullRequest"`
}

type WindowData struct {
	RepoRoot    string       `json:"repoRoot"`
	BranchName  string       `json:"branchName"`
	PullRequest *PullRequest `json:"pullRequest"`
	Files       []File       `json:"files"`
}

type FileContents struct {
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

type changedPath struct {
	status  ChangeStatus
	oldPath *string
	newPath *string
}

type pullRequestResponse struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

type fileSeed struct {
	path               string
	worktreeStatus     *ChangeStatus
	hasWorkingTreeFile bool
	inGitDiff          bool
	inLastCommit       bool
	inPullRequest      bool
	gitDiff            *FileComparison
	lastCommit         *FileComparison
	pullRequest        *FileComparison
}

func BuildWindowData(ctx context.Context, cwd string) (WindowData, error) {
	repoRoot, err := repoRoot(ctx, cwd)
	if err != nil {
		return WindowData{}, err
	}

	branchName := currentBranchName(ctx, repoRoot)
	repositoryHasHead := hasHead(ctx, repoRoot)

	trackedDiffOutput := []byte(nil)
	if repositoryHasHead {
		trackedDiffOutput, err = runGit(ctx, repoRoot, "diff", "--find-renames", "-M", "--name-status", "-z", "HEAD", "--")
		if err != nil {
			return WindowData{}, err
		}
	}

	untrackedOutput := runGitAllowFailure(ctx, repoRoot, "ls-files", "--others", "--exclude-standard", "-z")
	trackedFilesOutput := runGitAllowFailure(ctx, repoRoot, "ls-files", "--cached", "-z")
	deletedFilesOutput := runGitAllowFailure(ctx, repoRoot, "ls-files", "--deleted", "-z")
	lastCommitOutput := []byte(nil)
	if repositoryHasHead {
		lastCommitOutput = runGitAllowFailure(ctx, repoRoot, "diff-tree", "--root", "--find-renames", "-M", "--name-status", "-z", "--no-commit-id", "-r", "HEAD")
	}

	var pullRequest *PullRequest
	if repositoryHasHead {
		pullRequest = openPullRequest(ctx, repoRoot, branchName)
	}

	pullRequestOriginalRevision := ""
	pullRequestChanges := []changedPath(nil)
	if pullRequest != nil {
		resolvedBaseRevision := resolvePullRequestBaseRevision(ctx, repoRoot, pullRequest.BaseRefName)
		pullRequestOriginalRevision = mergeBase(ctx, repoRoot, resolvedBaseRevision)
		if pullRequestOriginalRevision != "" {
			pullRequestOutput := runGitAllowFailure(ctx, repoRoot, "diff", "--find-renames", "-M", "--name-status", "-z", pullRequestOriginalRevision, "HEAD", "--")
			pullRequestChanges = filterReviewableChanges(parseNameStatusZ(pullRequestOutput))
		}
	}

	worktreeChanges := filterReviewableChanges(mergeChangedPaths(parseNameStatusZ(trackedDiffOutput), parseUntrackedPathsZ(untrackedOutput)))
	deletedPaths := stringSet(parsePathListZ(deletedFilesOutput))
	currentPaths := uniquePaths(append(parsePathListZ(trackedFilesOutput), parsePathListZ(untrackedOutput)...))
	currentPaths = filterReviewablePaths(filterDeletedPaths(currentPaths, deletedPaths))
	lastCommitChanges := filterReviewableChanges(parseNameStatusZ(lastCommitOutput))

	seeds := make(map[string]*fileSeed)

	for _, currentPath := range currentPaths {
		seeds[currentPath] = &fileSeed{
			path:               currentPath,
			hasWorkingTreeFile: true,
		}
	}

	for _, change := range worktreeChanges {
		key := changedPathKey(change)
		seed := upsertSeed(seeds, key, func() *fileSeed {
			return &fileSeed{
				path:               key,
				hasWorkingTreeFile: change.newPath != nil,
			}
		})
		status := change.status
		seed.worktreeStatus = &status
		seed.hasWorkingTreeFile = change.newPath != nil
		seed.inGitDiff = true
		seed.gitDiff = comparison(change)
	}

	for _, change := range lastCommitChanges {
		key := changedPathKey(change)
		seed := upsertSeed(seeds, key, func() *fileSeed {
			return &fileSeed{
				path:               key,
				hasWorkingTreeFile: change.newPath != nil && contains(currentPaths, *change.newPath),
			}
		})
		seed.inLastCommit = true
		seed.lastCommit = comparison(change)
	}

	for _, change := range pullRequestChanges {
		key := changedPathKey(change)
		seed := upsertSeed(seeds, key, func() *fileSeed {
			return &fileSeed{
				path:               key,
				hasWorkingTreeFile: change.newPath != nil && contains(currentPaths, *change.newPath),
			}
		})
		seed.inPullRequest = true
		seed.pullRequest = comparisonWithRevisions(change, pullRequestOriginalRevision, "HEAD")
	}

	files := make([]File, 0, len(seeds))
	for _, seed := range seeds {
		files = append(files, reviewFile(seed))
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})

	return WindowData{
		RepoRoot:    repoRoot,
		BranchName:  branchName,
		PullRequest: pullRequest,
		Files:       files,
	}, nil
}

func LoadFileContents(ctx context.Context, repoRoot string, file File, scope Scope) (FileContents, error) {
	if scope == ScopeAllFiles {
		content, err := workingTreeContent(repoRoot, file.Path)
		if err != nil {
			return FileContents{}, err
		}

		return FileContents{OriginalContent: content, ModifiedContent: content}, nil
	}

	comparison := file.LastCommit
	originalRevision := "HEAD^"
	modifiedRevision := "HEAD"

	if scope == ScopeGitDiff {
		comparison = file.GitDiff
		originalRevision = "HEAD"
		modifiedRevision = ""
	}

	if scope == ScopePullRequest {
		comparison = file.PullRequest
	}

	if comparison == nil {
		return FileContents{}, nil
	}

	if scope == ScopePullRequest {
		originalRevision = comparison.originalRevision
		modifiedRevision = comparison.modifiedRevision
	}

	originalContent := ""
	if comparison.OldPath != nil && originalRevision != "" {
		originalContent = revisionContent(ctx, repoRoot, originalRevision, *comparison.OldPath)
	}

	modifiedContent := ""
	if comparison.NewPath != nil {
		if modifiedRevision == "" {
			content, err := workingTreeContent(repoRoot, *comparison.NewPath)
			if err != nil {
				return FileContents{}, err
			}
			modifiedContent = content
		} else {
			modifiedContent = revisionContent(ctx, repoRoot, modifiedRevision, *comparison.NewPath)
		}
	}

	return FileContents{OriginalContent: originalContent, ModifiedContent: modifiedContent}, nil
}

func IsValidScope(scope Scope) bool {
	return scope == ScopePullRequest || scope == ScopeGitDiff || scope == ScopeLastCommit || scope == ScopeAllFiles
}

func repoRoot(ctx context.Context, cwd string) (string, error) {
	output, err := runCommand(ctx, gitCommandTimeout, cwd, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not inside a git repository")
	}

	return strings.TrimSpace(string(output)), nil
}

func hasHead(ctx context.Context, repoRoot string) bool {
	_, err := runGit(ctx, repoRoot, "rev-parse", "--verify", "HEAD")
	return err == nil
}

func currentBranchName(ctx context.Context, repoRoot string) string {
	output := runGitAllowFailure(ctx, repoRoot, "branch", "--show-current")
	return strings.TrimSpace(string(output))
}

func openPullRequest(ctx context.Context, repoRoot string, branchName string) *PullRequest {
	if branchName == "" {
		return nil
	}

	output, err := runCommand(ctx, gitCommandTimeout, repoRoot, "gh", "pr", "view", branchName, "--json", "number,url,state,baseRefName,headRefName")
	if err != nil {
		return nil
	}

	var response pullRequestResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil
	}

	if !strings.EqualFold(response.State, "OPEN") || response.URL == "" || response.BaseRefName == "" {
		return nil
	}

	headRefName := response.HeadRefName
	if headRefName == "" {
		headRefName = branchName
	}

	return &PullRequest{
		Number:      response.Number,
		URL:         response.URL,
		BaseRefName: response.BaseRefName,
		HeadRefName: headRefName,
	}
}

func resolvePullRequestBaseRevision(ctx context.Context, repoRoot string, baseRefName string) string {
	if baseRefName == "" {
		return ""
	}

	candidates := []string{
		"refs/remotes/origin/" + baseRefName,
		"refs/heads/" + baseRefName,
		baseRefName,
	}

	for _, candidate := range candidates {
		revision := commitRevision(ctx, repoRoot, candidate)
		if revision != "" {
			return revision
		}
	}

	return ""
}

func commitRevision(ctx context.Context, repoRoot string, revision string) string {
	output := runGitAllowFailure(ctx, repoRoot, "rev-parse", "--verify", "--quiet", fmt.Sprintf("%s^{commit}", revision))
	return strings.TrimSpace(string(output))
}

func mergeBase(ctx context.Context, repoRoot string, revision string) string {
	if revision == "" {
		return ""
	}

	output := runGitAllowFailure(ctx, repoRoot, "merge-base", revision, "HEAD")
	return strings.TrimSpace(string(output))
}

func runGit(ctx context.Context, repoRoot string, args ...string) ([]byte, error) {
	output, err := runCommand(ctx, gitCommandTimeout, repoRoot, "git", args...)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func runGitAllowFailure(ctx context.Context, repoRoot string, args ...string) []byte {
	output, err := runGit(ctx, repoRoot, args...)
	if err != nil {
		return nil
	}

	return output
}

func runCommand(ctx context.Context, timeout time.Duration, directory string, name string, args ...string) ([]byte, error) {
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

func parseNameStatusZ(output []byte) []changedPath {
	tokens := splitNUL(output)
	changes := make([]changedPath, 0)

	for index := 0; index < len(tokens); {
		rawStatus := tokens[index]
		index++
		if rawStatus == "" {
			continue
		}

		switch rawStatus[0] {
		case 'R':
			if index+1 >= len(tokens) {
				return changes
			}
			oldPath := tokens[index]
			newPath := tokens[index+1]
			index += 2
			changes = append(changes, changedPath{status: StatusRenamed, oldPath: stringPtr(oldPath), newPath: stringPtr(newPath)})
		case 'M', 'T':
			if index >= len(tokens) {
				return changes
			}
			filePath := tokens[index]
			index++
			changes = append(changes, changedPath{status: StatusModified, oldPath: stringPtr(filePath), newPath: stringPtr(filePath)})
		case 'A':
			if index >= len(tokens) {
				return changes
			}
			filePath := tokens[index]
			index++
			changes = append(changes, changedPath{status: StatusAdded, oldPath: nil, newPath: stringPtr(filePath)})
		case 'D':
			if index >= len(tokens) {
				return changes
			}
			filePath := tokens[index]
			index++
			changes = append(changes, changedPath{status: StatusDeleted, oldPath: stringPtr(filePath), newPath: nil})
		default:
			if index < len(tokens) {
				index++
			}
		}
	}

	return changes
}

func parseUntrackedPathsZ(output []byte) []changedPath {
	paths := parsePathListZ(output)
	changes := make([]changedPath, 0, len(paths))
	for _, filePath := range paths {
		changes = append(changes, changedPath{status: StatusAdded, oldPath: nil, newPath: stringPtr(filePath)})
	}

	return changes
}

func parsePathListZ(output []byte) []string {
	return splitNUL(output)
}

func splitNUL(output []byte) []string {
	if len(output) == 0 {
		return nil
	}

	rawParts := strings.Split(string(output), "\x00")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}

func mergeChangedPaths(tracked []changedPath, untracked []changedPath) []changedPath {
	seen := make(map[string]struct{})
	merged := make([]changedPath, 0, len(tracked)+len(untracked))

	for _, change := range tracked {
		key := changeIdentity(change)
		seen[key] = struct{}{}
		merged = append(merged, change)
	}

	for _, change := range untracked {
		key := changeIdentity(change)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		merged = append(merged, change)
	}

	return merged
}

func changeIdentity(change changedPath) string {
	return fmt.Sprintf("%s:%s:%s", change.status, dereference(change.oldPath), dereference(change.newPath))
}

func changedPathKey(change changedPath) string {
	if change.newPath != nil {
		return *change.newPath
	}
	if change.oldPath != nil {
		return *change.oldPath
	}

	return displayPath(change)
}

func displayPath(change changedPath) string {
	if change.status == StatusRenamed {
		return fmt.Sprintf("%s -> %s", dereference(change.oldPath), dereference(change.newPath))
	}

	if change.newPath != nil {
		return *change.newPath
	}
	if change.oldPath != nil {
		return *change.oldPath
	}

	return "(unknown)"
}

func comparison(change changedPath) *FileComparison {
	return comparisonWithRevisions(change, "", "")
}

func comparisonWithRevisions(change changedPath, originalRevision string, modifiedRevision string) *FileComparison {
	return &FileComparison{
		Status:           change.status,
		OldPath:          change.oldPath,
		NewPath:          change.newPath,
		DisplayPath:      displayPath(change),
		HasOriginal:      change.oldPath != nil,
		HasModified:      change.newPath != nil,
		originalRevision: originalRevision,
		modifiedRevision: modifiedRevision,
	}
}

func reviewFile(seed *fileSeed) File {
	return File{
		ID:                 reviewFileID(seed),
		Path:               seed.path,
		WorktreeStatus:     seed.worktreeStatus,
		HasWorkingTreeFile: seed.hasWorkingTreeFile,
		InGitDiff:          seed.inGitDiff,
		InLastCommit:       seed.inLastCommit,
		InPullRequest:      seed.inPullRequest,
		GitDiff:            seed.gitDiff,
		LastCommit:         seed.lastCommit,
		PullRequest:        seed.pullRequest,
	}
}

func reviewFileID(seed *fileSeed) string {
	workingTreeState := "gone"
	if seed.hasWorkingTreeFile {
		workingTreeState = "working"
	}

	gitDiffDisplayPath := ""
	if seed.gitDiff != nil {
		gitDiffDisplayPath = seed.gitDiff.DisplayPath
	}

	lastCommitDisplayPath := ""
	if seed.lastCommit != nil {
		lastCommitDisplayPath = seed.lastCommit.DisplayPath
	}

	pullRequestDisplayPath := ""
	if seed.pullRequest != nil {
		pullRequestDisplayPath = seed.pullRequest.DisplayPath
	}

	return strings.Join([]string{seed.path, workingTreeState, gitDiffDisplayPath, lastCommitDisplayPath, pullRequestDisplayPath}, "::")
}

func upsertSeed(seeds map[string]*fileSeed, key string, create func() *fileSeed) *fileSeed {
	if seed, exists := seeds[key]; exists {
		return seed
	}

	seed := create()
	seeds[key] = seed
	return seed
}

func filterReviewableChanges(changes []changedPath) []changedPath {
	filtered := make([]changedPath, 0, len(changes))
	for _, change := range changes {
		if !isReviewablePath(changedPathKey(change)) {
			continue
		}

		filtered = append(filtered, change)
	}

	return filtered
}

func filterReviewablePaths(paths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, filePath := range paths {
		if isReviewablePath(filePath) {
			filtered = append(filtered, filePath)
		}
	}

	return filtered
}

func filterDeletedPaths(paths []string, deletedPaths map[string]struct{}) []string {
	filtered := make([]string, 0, len(paths))
	for _, filePath := range paths {
		if _, deleted := deletedPaths[filePath]; deleted {
			continue
		}

		filtered = append(filtered, filePath)
	}

	return filtered
}

func isReviewablePath(filePath string) bool {
	lowerPath := strings.ToLower(filePath)
	fileName := path.Base(lowerPath)
	if fileName == "" || fileName == "." || fileName == "/" {
		return false
	}

	if strings.HasSuffix(fileName, ".min.js") || strings.HasSuffix(fileName, ".min.css") {
		return false
	}

	_, binary := binaryExtensions()[path.Ext(fileName)]
	return !binary
}

func binaryExtensions() map[string]struct{} {
	return map[string]struct{}{
		".7z": {}, ".a": {}, ".avi": {}, ".avif": {}, ".bin": {}, ".bmp": {},
		".class": {}, ".dll": {}, ".dylib": {}, ".eot": {}, ".exe": {}, ".gif": {},
		".gz": {}, ".ico": {}, ".jar": {}, ".jpeg": {}, ".jpg": {}, ".lockb": {},
		".map": {}, ".mov": {}, ".mp3": {}, ".mp4": {}, ".o": {}, ".otf": {},
		".pdf": {}, ".png": {}, ".pyc": {}, ".so": {}, ".svgz": {}, ".tar": {},
		".ttf": {}, ".wasm": {}, ".webm": {}, ".webp": {}, ".woff": {}, ".woff2": {},
		".zip": {},
	}
}

func revisionContent(ctx context.Context, repoRoot string, revision string, filePath string) string {
	output := runGitAllowFailure(ctx, repoRoot, "show", fmt.Sprintf("%s:%s", revision, filePath))
	return string(output)
}

func workingTreeContent(repoRoot string, filePath string) (string, error) {
	cleanPath, err := safeRelativePath(filePath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, cleanPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", err
	}

	return string(content), nil
}

func safeRelativePath(filePath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(filePath))
	if cleanPath == "." || cleanPath == "" || filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid review file path %q", filePath)
	}

	return cleanPath, nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{})
	unique := make([]string, 0, len(paths))
	for _, filePath := range paths {
		if _, exists := seen[filePath]; exists {
			continue
		}

		seen[filePath] = struct{}{}
		unique = append(unique, filePath)
	}

	return unique
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}

	return set
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}

	return false
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func stringPtr(value string) *string {
	return &value
}
