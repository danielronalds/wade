package reviewsnapshots

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func (model *Model) buildWindowData(ctx context.Context, cwd string) (windowData, error) {
	repoRoot, err := repoRoot(ctx, cwd, model.git)
	if err != nil {
		return windowData{}, err
	}

	branchName := currentBranchName(ctx, repoRoot, model.git)
	repositoryHasHead := hasHead(ctx, repoRoot, model.git)

	trackedDiffOutput := []byte(nil)
	if repositoryHasHead {
		trackedDiffOutput, err = model.git.TrackedDiffNameStatus(ctx, repoRoot)
		if err != nil {
			return windowData{}, err
		}
	}

	untrackedOutput := runGitAllowFailure(func() ([]byte, error) { return model.git.UntrackedFiles(ctx, repoRoot) })
	trackedFilesOutput := runGitAllowFailure(func() ([]byte, error) { return model.git.TrackedFiles(ctx, repoRoot) })
	deletedFilesOutput := runGitAllowFailure(func() ([]byte, error) { return model.git.DeletedFiles(ctx, repoRoot) })
	lastCommitOutput := []byte(nil)
	if repositoryHasHead {
		lastCommitOutput = runGitAllowFailure(func() ([]byte, error) { return model.git.LastCommitNameStatus(ctx, repoRoot) })
	}

	var pullRequest *pullRequest
	if repositoryHasHead {
		pullRequest = openPullRequest(ctx, repoRoot, branchName, model.github)
	}

	pullRequestOriginalRevision := ""
	pullRequestChanges := []changedPath(nil)
	if pullRequest != nil {
		resolvedBaseRevision := resolvePullRequestBaseRevision(ctx, repoRoot, pullRequest.baseRefName, model.git)
		pullRequestOriginalRevision = mergeBase(ctx, repoRoot, resolvedBaseRevision, model.git)
		if pullRequestOriginalRevision != "" {
			pullRequestOutput := runGitAllowFailure(func() ([]byte, error) {
				return model.git.DiffNameStatusBetween(ctx, repoRoot, pullRequestOriginalRevision, "HEAD")
			})
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

	return windowData{
		repoRoot:    repoRoot,
		branchName:  branchName,
		pullRequest: pullRequest,
		files:       files,
	}, nil
}

func (model *Model) loadFileContents(ctx context.Context, repoRoot string, file File, scope Scope) (FileContents, error) {
	if scope == ScopeCurrent {
		content, err := workingTreeContent(model.files, repoRoot, file.Path)
		if err != nil {
			return FileContents{}, err
		}

		return FileContents{OriginalContent: content, ModifiedContent: content}, nil
	}

	comparison := file.LastCommit
	if scope == ScopeWorkingTree {
		comparison = file.GitDiff
	}
	if scope == ScopePullRequest {
		comparison = file.PullRequest
	}
	if comparison == nil {
		return FileContents{}, nil
	}

	originalRevision := comparison.originalRevision
	modifiedRevision := comparison.modifiedRevision
	if scope == ScopeLastCommit && originalRevision == "" && modifiedRevision == "" {
		originalRevision = "HEAD^"
		modifiedRevision = "HEAD"
	}
	if scope == ScopeWorkingTree && originalRevision == "" {
		originalRevision = "HEAD"
	}

	originalContent := ""
	if comparison.OldPath != nil && originalRevision != "" {
		originalContent = revisionContent(ctx, model.git, repoRoot, originalRevision, *comparison.OldPath)
	}

	modifiedContent := ""
	if comparison.NewPath != nil {
		if comparison.capturedModifiedContent != nil {
			modifiedContent = *comparison.capturedModifiedContent
		} else if modifiedRevision == "" {
			content, err := workingTreeContent(model.files, repoRoot, *comparison.NewPath)
			if err != nil {
				return FileContents{}, err
			}
			modifiedContent = content
		} else {
			modifiedContent = revisionContent(ctx, model.git, repoRoot, modifiedRevision, *comparison.NewPath)
		}
	}

	return FileContents{OriginalContent: originalContent, ModifiedContent: modifiedContent}, nil
}

func isValidScope(scope Scope) bool {
	return scope == ScopePullRequest || scope == ScopeWorkingTree || scope == ScopeLastCommit || scope == ScopeCurrent
}

func repoRoot(ctx context.Context, cwd string, git Git) (string, error) {
	root, err := git.RepoRoot(ctx, cwd)
	if err != nil {
		return "", WorkspaceNotGitRepositoryError{}
	}

	return root, nil
}

func hasHead(ctx context.Context, repoRoot string, git Git) bool {
	return git.VerifyHead(ctx, repoRoot) == nil
}

func currentBranchName(ctx context.Context, repoRoot string, git Git) string {
	output := runGitAllowFailure(func() ([]byte, error) { return git.ReviewCurrentBranch(ctx, repoRoot) })
	return strings.TrimSpace(string(output))
}

func openPullRequest(ctx context.Context, repoRoot string, branchName string, github GitHub) *pullRequest {
	if branchName == "" || github == nil {
		return nil
	}

	response, err := github.PullRequest(ctx, repoRoot, branchName)
	if err != nil || response == nil {
		return nil
	}

	if !strings.EqualFold(response.State, "OPEN") || response.URL == "" || response.BaseRefName == "" {
		return nil
	}

	headRefName := response.HeadRefName
	if headRefName == "" {
		headRefName = branchName
	}

	return &pullRequest{
		number:      response.Number,
		url:         response.URL,
		baseRefName: response.BaseRefName,
		headRefName: headRefName,
	}
}

func resolvePullRequestBaseRevision(ctx context.Context, repoRoot string, baseRefName string, git Git) string {
	if baseRefName == "" {
		return ""
	}

	candidates := []string{
		"refs/remotes/origin/" + baseRefName,
		"refs/heads/" + baseRefName,
		baseRefName,
	}

	for _, candidate := range candidates {
		revision := commitRevision(ctx, repoRoot, candidate, git)
		if revision != "" {
			return revision
		}
	}

	return ""
}

func commitRevision(ctx context.Context, repoRoot string, revision string, git Git) string {
	output := runGitAllowFailure(func() ([]byte, error) { return git.CommitRevision(ctx, repoRoot, revision) })
	return strings.TrimSpace(string(output))
}

func mergeBase(ctx context.Context, repoRoot string, revision string, git Git) string {
	if revision == "" {
		return ""
	}

	output := runGitAllowFailure(func() ([]byte, error) { return git.MergeBase(ctx, repoRoot, revision) })
	return strings.TrimSpace(string(output))
}

func runGitAllowFailure(run func() ([]byte, error)) []byte {
	output, err := run()
	if err != nil {
		return nil
	}

	return output
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

func revisionContent(ctx context.Context, git Git, repoRoot string, revision string, filePath string) string {
	output := runGitAllowFailure(func() ([]byte, error) { return git.RevisionContent(ctx, repoRoot, revision, filePath) })
	return string(output)
}

func workingTreeContent(files FileSystem, repoRoot string, filePath string) (string, error) {
	cleanPath, err := safeRelativePath(filePath)
	if err != nil {
		return "", err
	}

	content, err := files.ReadFile(filepath.Join(repoRoot, cleanPath))
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
