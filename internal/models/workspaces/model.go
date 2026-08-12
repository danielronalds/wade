package workspaces

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var remoteRepositoryIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// WorkspaceDirectory maps a persisted setting to its resolved local path.
type WorkspaceDirectory struct {
	Setting string
	Path    string
}

// LinearConfiguration controls optional Linear issue resolution.
type LinearConfiguration struct {
	Enabled   bool
	Workspace string
}

// Configuration controls workspace discovery, materialisation, and provider integration.
type Configuration struct {
	WorkspaceDirectories []WorkspaceDirectory
	Linear               LinearConfiguration
}

// MaterialiseRequest creates a workspace from a remote repository.
type MaterialiseRequest struct {
	RemoteRepositoryID string `json:"remoteRepositoryId"`
	WorkspaceDirectory string `json:"workspaceDirectory"`
} // @name MaterialiseWorkspaceRequest

// LinkContext contains the repository context required for provider links.
type LinkContext struct {
	RemoteRepositoryID *string
	BranchName         string
	ResolvePullRequest bool
}

// Model owns workspace identity, materialisation, and provider links.
type Model struct {
	files     FileSystem
	discovery WorkspaceDiscovery
	github    GitHub
	linear    Linear

	configurationMu sync.RWMutex
	configuration   Configuration
	materialiseMu   sync.Map
}

// New constructs an application-scoped Workspaces Model.
func New(files FileSystem, discovery WorkspaceDiscovery, github GitHub, linear Linear, configuration Configuration) *Model {
	return &Model{
		files:         files,
		discovery:     discovery,
		github:        github,
		linear:        linear,
		configuration: cloneConfiguration(configuration),
	}
}

// Configure atomically updates future workspace discovery and materialisation.
func (model *Model) Configure(configuration Configuration) {
	cloned := cloneConfiguration(configuration)

	model.configurationMu.Lock()
	model.configuration = cloned
	model.configurationMu.Unlock()

	paths := make([]string, 0, len(cloned.WorkspaceDirectories))
	for _, directory := range cloned.WorkspaceDirectories {
		paths = append(paths, directory.Path)
	}
	model.discovery.Reload(paths)
}

// List returns base snapshots for all discovered workspaces.
func (model *Model) List(_ context.Context) ([]WorkspaceSummary, error) {
	workspaceIDs, err := model.discovery.IDs()
	if err != nil {
		return nil, fmt.Errorf("discovering workspaces: %w", err)
	}

	workspaces := make([]WorkspaceSummary, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		workspaces = append(workspaces, WorkspaceSummary(newWorkspace(workspaceID)))
	}
	return workspaces, nil
}

// ListByIDs returns base snapshots for only discovered requested workspace IDs.
func (model *Model) ListByIDs(_ context.Context, workspaceIDs []string) ([]WorkspaceSummary, error) {
	requested := append([]string(nil), workspaceIDs...)
	if len(requested) == 0 {
		return []WorkspaceSummary{}, nil
	}
	sort.Strings(requested)

	workspaces := make([]WorkspaceSummary, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, workspaceID := range requested {
		if _, exists := seen[workspaceID]; exists {
			continue
		}
		seen[workspaceID] = struct{}{}

		_, found, err := model.discovery.Resolve(workspaceID)
		if err != nil {
			return nil, fmt.Errorf("resolving workspace %q: %w", workspaceID, err)
		}
		if found {
			workspaces = append(workspaces, WorkspaceSummary(newWorkspace(workspaceID)))
		}
	}
	return workspaces, nil
}

// Get returns a detached base workspace snapshot.
func (model *Model) Get(_ context.Context, workspaceID string) (Workspace, error) {
	if err := validateWorkspaceID(workspaceID); err != nil {
		return Workspace{}, err
	}

	_, found, err := model.discovery.Resolve(workspaceID)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolving workspace %q: %w", workspaceID, err)
	}
	if !found {
		return Workspace{}, WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}

	return newWorkspace(workspaceID), nil
}

// Materialise validates and clones a remote repository into a configured workspace directory.
func (model *Model) Materialise(ctx context.Context, request MaterialiseRequest) (Workspace, error) {
	remoteRepositoryID := strings.TrimSpace(request.RemoteRepositoryID)
	if !remoteRepositoryIDPattern.MatchString(remoteRepositoryID) {
		return Workspace{}, InvalidRemoteRepositoryIDError{RemoteRepositoryID: request.RemoteRepositoryID}
	}

	workspaceDirectory, found := model.workspaceDirectory(request.WorkspaceDirectory)
	if !found {
		return Workspace{}, WorkspaceDirectoryNotConfiguredError{WorkspaceDirectory: request.WorkspaceDirectory}
	}

	workspaceID := strings.Split(remoteRepositoryID, "/")[1]
	targetPath := filepath.Join(workspaceDirectory.Path, workspaceID)
	lock := model.materialisationLock(workspaceID)
	lock.Lock()
	defer lock.Unlock()

	if _, found, err := model.discovery.Resolve(workspaceID); err != nil {
		return Workspace{}, fmt.Errorf("checking workspace identity: %w", err)
	} else if found {
		return Workspace{}, WorkspaceAlreadyExistsError{WorkspaceID: workspaceID}
	}

	if err := model.files.EnsurePathDoesNotExist(targetPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Workspace{}, WorkspaceAlreadyExistsError{WorkspaceID: workspaceID}
		}
		return Workspace{}, fmt.Errorf("checking workspace target: %w", err)
	}
	if err := model.files.EnsureDirectory(workspaceDirectory.Path); err != nil {
		return Workspace{}, fmt.Errorf("creating workspace directory: %w", err)
	}
	if err := model.github.CloneRepository(ctx, remoteRepositoryID, targetPath); err != nil {
		return Workspace{}, fmt.Errorf("materialising workspace: %w", err)
	}

	return model.Get(ctx, workspaceID)
}

// ResolveLinks returns all successfully resolved links and any optional provider failures.
func (model *Model) ResolveLinks(ctx context.Context, linkContext LinkContext) (WorkspaceLinks, error) {
	configuration := model.configurationSnapshot()
	links := WorkspaceLinks{}
	var linkErrors []error

	if linkContext.RemoteRepositoryID != nil && *linkContext.RemoteRepositoryID != "" {
		links.Repository = stringReference(repositoryURL(*linkContext.RemoteRepositoryID))
		if linkContext.ResolvePullRequest && linkContext.BranchName != "" {
			pullRequestURL, err := model.github.PullRequestURL(ctx, *linkContext.RemoteRepositoryID, linkContext.BranchName)
			if err != nil {
				linkErrors = append(linkErrors, fmt.Errorf("resolving pull request link: %w", err))
			} else {
				links.PullRequest = stringReference(pullRequestURL)
			}
		}
	}

	if model.linear != nil && configuration.Linear.Enabled {
		ticket, err := model.linear.TicketForBranch(configuration.Linear.Workspace, linkContext.BranchName)
		if err != nil {
			linkErrors = append(linkErrors, fmt.Errorf("resolving issue link: %w", err))
		} else if ticket != nil {
			links.Issue = &IssueReference{Provider: "linear", Key: ticket.Key, URL: ticket.URL}
		}
	}

	return links, errors.Join(linkErrors...)
}

func (model *Model) workspaceDirectories() []WorkspaceDirectory {
	return model.configurationSnapshot().WorkspaceDirectories
}

func (model *Model) configurationSnapshot() Configuration {
	model.configurationMu.RLock()
	defer model.configurationMu.RUnlock()
	return cloneConfiguration(model.configuration)
}

func (model *Model) materialisationLock(key string) *sync.Mutex {
	lock, _ := model.materialiseMu.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (model *Model) workspaceDirectory(setting string) (WorkspaceDirectory, bool) {
	for _, directory := range model.workspaceDirectories() {
		if directory.Setting == setting {
			return directory, true
		}
	}
	return WorkspaceDirectory{}, false
}

func cloneConfiguration(configuration Configuration) Configuration {
	return Configuration{
		WorkspaceDirectories: append([]WorkspaceDirectory(nil), configuration.WorkspaceDirectories...),
		Linear:               configuration.Linear,
	}
}

func newWorkspace(workspaceID string) Workspace {
	return Workspace{ID: workspaceID, Name: workspaceID}
}
