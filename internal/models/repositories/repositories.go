package repositories

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Eight workers kept measured multi-workspace scan latency close to one inspection while bounding concurrent Git subprocesses.
const workspaceInspectionConcurrency = 8

type workspaceRecord struct {
	id              string
	path            string
	canonicalPath   string
	isGit           bool
	mainPath        string
	worktreePaths   []string
	commonDirectory string
	remoteURL       string
	remoteIdentity  string
	branch          WorkspaceBranch
}

// Get returns a detached local repository snapshot.
func (model *Model) Get(ctx context.Context, repositoryID string) (Repository, error) {
	repository, err := model.resolveRepository(ctx, repositoryID)
	if err != nil {
		return Repository{}, err
	}

	return cloneRepository(repository.repository), nil
}

// ListWorkspaceContexts returns Git context for every discovered Git workspace.
func (model *Model) ListWorkspaceContexts(ctx context.Context) ([]WorkspaceContext, error) {
	records, err := model.scan(ctx)
	if err != nil {
		return nil, err
	}

	contexts := make([]WorkspaceContext, 0, len(records))
	for index, record := range records {
		if !record.isGit {
			continue
		}

		contexts = append(contexts, workspaceContextFromRecord(records, index))
	}

	return contexts, nil
}

// ListWorkspaceContextsByIDs loads Git context only for the requested workspace IDs.
func (model *Model) ListWorkspaceContextsByIDs(ctx context.Context, workspaceIDs []string) (map[string]WorkspaceContext, error) {
	contexts := make(map[string]WorkspaceContext, len(workspaceIDs))
	seen := make(map[string]struct{}, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		if _, exists := seen[workspaceID]; exists {
			continue
		}
		seen[workspaceID] = struct{}{}

		workspaceContext, err := model.GetWorkspaceContext(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if workspaceContext != nil {
			contexts[workspaceID] = *workspaceContext
		}
	}

	return contexts, nil
}

// GetWorkspaceContext returns nil for a valid non-Git workspace.
func (model *Model) GetWorkspaceContext(ctx context.Context, workspaceID string) (*WorkspaceContext, error) {
	if err := validateWorkspaceID(workspaceID); err != nil {
		return nil, err
	}

	workspacePath, found, err := model.workspaces.Resolve(workspaceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	canonicalPath, found, err := model.workspaces.CanonicalPath(workspaceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	record, err := model.inspectWorkspace(ctx, workspaceID, workspacePath, canonicalPath)
	if err != nil {
		return nil, err
	}
	if !record.isGit {
		return nil, nil
	}

	workspaceIDs, err := model.workspaces.IDsForDirectories(record.worktreePaths)
	if err != nil {
		return nil, err
	}
	mainWorkspaceID := filepath.Base(record.mainPath)
	resolvedMainWorkspaceID, found, err := model.workspaces.IDForDirectory(record.mainPath)
	if err != nil {
		return nil, err
	}
	if found {
		mainWorkspaceID = resolvedMainWorkspaceID
	}

	repository := Repository{
		ID:                 filepath.Base(record.mainPath),
		RemoteRepositoryID: githubRepositoryID(record.remoteURL),
		MainWorkspaceID:    mainWorkspaceID,
		WorkspaceIDs:       append([]string(nil), workspaceIDs...),
	}
	isMain := samePath(record.canonicalPath, record.mainPath)

	return &WorkspaceContext{
		WorkspaceID: workspaceID,
		Repository:  repository,
		Branch:      record.branch,
		IsMain:      isMain,
		IsRemovable: !isMain,
	}, nil
}

// WorkspaceIDsByRemoteRepository performs one bulk local scan and groups workspace IDs by provider repository ID.
func (model *Model) WorkspaceIDsByRemoteRepository(ctx context.Context, remoteRepositoryIDs []string) (map[string][]string, error) {
	requested := make(map[string]struct{}, len(remoteRepositoryIDs))
	for _, remoteRepositoryID := range remoteRepositoryIDs {
		requested[remoteRepositoryID] = struct{}{}
	}

	repositories, err := model.listRepositories(ctx)
	if err != nil {
		return nil, err
	}

	workspaceIDs := make(map[string][]string, len(requested))
	for _, repository := range repositories {
		remoteRepositoryID := repository.repository.RemoteRepositoryID
		if remoteRepositoryID == nil {
			continue
		}
		if _, wanted := requested[*remoteRepositoryID]; !wanted {
			continue
		}
		workspaceIDs[*remoteRepositoryID] = append(workspaceIDs[*remoteRepositoryID], repository.repository.WorkspaceIDs...)
	}
	for remoteRepositoryID, ids := range workspaceIDs {
		sort.Strings(ids)
		workspaceIDs[remoteRepositoryID] = compactStrings(ids)
	}

	return workspaceIDs, nil
}

func (model *Model) listRepositories(ctx context.Context) ([]repositoryContext, error) {
	records, err := model.scan(ctx)
	if err != nil {
		return nil, err
	}

	repositories := make([]repositoryContext, 0)
	seenCommonDirectories := make(map[string]struct{})
	for index, record := range records {
		if !record.isGit {
			continue
		}
		if _, seen := seenCommonDirectories[record.commonDirectory]; seen {
			continue
		}

		seenCommonDirectories[record.commonDirectory] = struct{}{}
		repositories = append(repositories, buildRepositoryContext(records, index))
	}

	sort.Slice(repositories, func(firstIndex int, secondIndex int) bool {
		return repositories[firstIndex].repository.ID < repositories[secondIndex].repository.ID
	})

	return repositories, nil
}

func (model *Model) resolveRepository(ctx context.Context, repositoryID string) (repositoryContext, error) {
	if err := validateRepositoryID(repositoryID); err != nil {
		return repositoryContext{}, err
	}

	repositories, err := model.listRepositories(ctx)
	if err != nil {
		return repositoryContext{}, err
	}

	var matched repositoryContext
	found := false
	for _, repository := range repositories {
		if repository.repository.ID != repositoryID {
			continue
		}
		if found && matched.commonDirectory != repository.commonDirectory {
			return repositoryContext{}, RepositoryIDConflictError{RepositoryID: repositoryID}
		}

		matched = repository
		found = true
	}
	if !found {
		return repositoryContext{}, RepositoryNotFoundError{RepositoryID: repositoryID}
	}

	return matched, nil
}

func (model *Model) scan(ctx context.Context) ([]workspaceRecord, error) {
	workspaceIDs, err := model.workspaces.IDs()
	if err != nil {
		return nil, err
	}

	type workspaceInput struct {
		id            string
		path          string
		canonicalPath string
	}
	workspaceInputs := make([]workspaceInput, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		workspacePath, found, err := model.workspaces.Resolve(workspaceID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		canonicalPath, found, err := model.workspaces.CanonicalPath(workspaceID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		workspaceInputs = append(workspaceInputs, workspaceInput{
			id:            workspaceID,
			path:          workspacePath,
			canonicalPath: canonicalPath,
		})
	}

	inspectionContext, cancelInspections := context.WithCancel(ctx)
	defer cancelInspections()

	records := make([]workspaceRecord, len(workspaceInputs))
	workspaceIndexes := make(chan int, len(workspaceInputs))
	for index := range workspaceInputs {
		workspaceIndexes <- index
	}
	close(workspaceIndexes)

	workerCount := min(workspaceInspectionConcurrency, len(workspaceInputs))
	var workers sync.WaitGroup
	var firstInspectionError error
	var inspectionErrorOnce sync.Once
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()

			for index := range workspaceIndexes {
				if inspectionContext.Err() != nil {
					return
				}

				input := workspaceInputs[index]
				record, err := model.inspectWorkspace(
					inspectionContext,
					input.id,
					input.path,
					input.canonicalPath,
				)
				if err != nil {
					inspectionErrorOnce.Do(func() {
						firstInspectionError = err
						cancelInspections()
					})
					return
				}
				records[index] = record
			}
		}()
	}
	workers.Wait()

	if firstInspectionError != nil {
		return nil, firstInspectionError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (model *Model) inspectWorkspace(ctx context.Context, workspaceID string, workspacePath string, canonicalPath string) (workspaceRecord, error) {
	record := workspaceRecord{id: workspaceID, path: workspacePath, canonicalPath: canonicalPath}

	isGit, err := model.git.IsGitWorktree(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	if !isGit {
		return record, nil
	}

	worktreePaths, err := model.git.WorktreePaths(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	commonDirectory, err := model.git.CommonDirectory(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	headReference, hasHeadReference, err := model.git.HeadReference(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	headCommit, _, err := model.git.HeadCommit(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	remoteURL, hasRemote, err := model.git.OriginRemoteURL(ctx, workspacePath)
	if err != nil {
		return workspaceRecord{}, err
	}
	if !hasRemote {
		remoteURL = ""
	}

	record.isGit = true
	record.mainPath = worktreePaths[0]
	record.worktreePaths = append([]string(nil), worktreePaths...)
	record.commonDirectory = commonDirectory
	record.remoteURL = remoteURL
	record.remoteIdentity = CanonicalRemoteIdentity(remoteURL)
	record.branch = WorkspaceBranch{
		Ref:        headReference,
		Name:       strings.TrimPrefix(headReference, "refs/heads/"),
		IsDetached: !hasHeadReference,
		Commit:     headCommit,
	}

	return record, nil
}

func workspaceContextFromRecord(records []workspaceRecord, targetIndex int) WorkspaceContext {
	target := records[targetIndex]
	repository := buildRepositoryContext(records, targetIndex).repository
	isMain := samePath(target.canonicalPath, target.mainPath)

	return WorkspaceContext{
		WorkspaceID: target.id,
		Repository:  cloneRepository(repository),
		Branch:      target.branch,
		IsMain:      isMain,
		IsRemovable: !isMain,
	}
}

func buildRepositoryContext(records []workspaceRecord, targetIndex int) repositoryContext {
	target := records[targetIndex]
	workspaceIDs := make([]string, 0)
	mainWorkspaceID := filepath.Base(target.mainPath)

	for _, record := range records {
		if !record.isGit || record.commonDirectory != target.commonDirectory {
			continue
		}

		workspaceIDs = append(workspaceIDs, record.id)
		if samePath(record.canonicalPath, target.mainPath) {
			mainWorkspaceID = record.id
		}
	}
	sort.Strings(workspaceIDs)

	return repositoryContext{
		repository: Repository{
			ID:                 filepath.Base(target.mainPath),
			RemoteRepositoryID: githubRepositoryID(target.remoteURL),
			MainWorkspaceID:    mainWorkspaceID,
			WorkspaceIDs:       workspaceIDs,
		},
		mainWorktreePath: target.mainPath,
		commonDirectory:  target.commonDirectory,
		remoteURL:        target.remoteURL,
		remoteIdentity:   target.remoteIdentity,
	}
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	compacted := values[:1]
	for _, value := range values[1:] {
		if value != compacted[len(compacted)-1] {
			compacted = append(compacted, value)
		}
	}
	return compacted
}
