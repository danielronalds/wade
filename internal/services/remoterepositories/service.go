package remoterepositories

// TODO: Review properly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"wade/internal/services/gitrepositories"
	"wade/internal/services/workspaces"
)

var remoteRepositoryIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type GitHubRepository interface {
	ListRepositories(ctx context.Context) (string, error)
	CloneRepository(ctx context.Context, nameWithOwner string, targetPath string) error
}

type FileRepository interface {
	EnsureDirectory(path string) error
	EnsurePathDoesNotExist(path string) error
}

type LocalRepositoryService interface {
	List(ctx context.Context) ([]gitrepositories.Context, error)
}

type WorkspaceRepository interface {
	IDs() ([]string, error)
}

type WorkspaceService interface {
	Get(ctx context.Context, workspaceID string) (workspaces.Workspace, error)
}

type Service struct {
	github              GitHubRepository
	files               FileRepository
	localRepositories   LocalRepositoryService
	workspaceRepository WorkspaceRepository
	workspaces          WorkspaceService
	state               *serviceState
}

type serviceState struct {
	mu                   sync.RWMutex
	workspaceDirectories []WorkspaceDirectory
}

type githubRepository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
}

func NewService(
	github GitHubRepository,
	files FileRepository,
	localRepositories LocalRepositoryService,
	workspaceRepository WorkspaceRepository,
	workspaces WorkspaceService,
	workspaceDirectories []WorkspaceDirectory,
) *Service {
	return &Service{
		github:              github,
		files:               files,
		localRepositories:   localRepositories,
		workspaceRepository: workspaceRepository,
		workspaces:          workspaces,
		state: &serviceState{
			workspaceDirectories: cloneWorkspaceDirectories(workspaceDirectories),
		},
	}
}

func (s *Service) List(ctx context.Context) ([]RemoteRepository, error) {
	if s.github == nil {
		return nil, errors.New("GitHub repository is required")
	}

	output, err := s.github.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}

	githubRepositories, err := parseGitHubRepositories(output)
	if err != nil {
		return nil, err
	}
	localRepositories, err := s.localRepositories.List(ctx)
	if err != nil {
		return nil, err
	}

	workspaceIDsByRemote := make(map[string]map[string]struct{})
	for _, localRepository := range localRepositories {
		remoteIdentity := localRepository.RemoteIdentity()
		if remoteIdentity == "" {
			continue
		}

		workspaceIDs := workspaceIDsByRemote[remoteIdentity]
		if workspaceIDs == nil {
			workspaceIDs = make(map[string]struct{})
			workspaceIDsByRemote[remoteIdentity] = workspaceIDs
		}
		for _, workspaceID := range localRepository.Repository.WorkspaceIDs {
			workspaceIDs[workspaceID] = struct{}{}
		}
	}

	remoteRepositories := make([]RemoteRepository, 0, len(githubRepositories))
	for _, repository := range githubRepositories {
		remoteRepository, err := buildRemoteRepository(repository, workspaceIDsByRemote)
		if err != nil {
			return nil, err
		}
		remoteRepositories = append(remoteRepositories, remoteRepository)
	}
	sort.Slice(remoteRepositories, func(firstIndex int, secondIndex int) bool {
		return remoteRepositories[firstIndex].ID < remoteRepositories[secondIndex].ID
	})

	return remoteRepositories, nil
}

func (s *Service) Clone(ctx context.Context, request CloneRequest) (workspaces.Workspace, error) {
	remoteRepositoryID := strings.TrimSpace(request.RemoteRepositoryID)
	if !remoteRepositoryIDPattern.MatchString(remoteRepositoryID) {
		return workspaces.Workspace{}, InvalidRemoteRepositoryIDError{RemoteRepositoryID: request.RemoteRepositoryID}
	}

	workspaceDirectory, found := s.workspaceDirectory(request.WorkspaceDirectory)
	if !found {
		return workspaces.Workspace{}, WorkspaceDirectoryNotConfiguredError{
			WorkspaceDirectory: request.WorkspaceDirectory,
		}
	}

	workspaceID := strings.Split(remoteRepositoryID, "/")[1]
	workspaceIDs, err := s.workspaceRepository.IDs()
	if err != nil {
		return workspaces.Workspace{}, err
	}
	for _, existingWorkspaceID := range workspaceIDs {
		if existingWorkspaceID == workspaceID {
			return workspaces.Workspace{}, WorkspaceAlreadyExistsError{WorkspaceID: workspaceID}
		}
	}

	targetPath := filepath.Join(workspaceDirectory.Path, workspaceID)
	if s.files == nil {
		return workspaces.Workspace{}, errors.New("file repository is required")
	}
	if err := ensureTargetDoesNotExist(s.files, targetPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return workspaces.Workspace{}, WorkspaceAlreadyExistsError{WorkspaceID: workspaceID}
		}
		return workspaces.Workspace{}, err
	}
	if err := s.files.EnsureDirectory(workspaceDirectory.Path); err != nil {
		return workspaces.Workspace{}, fmt.Errorf("creating workspace directory: %w", err)
	}
	if s.github == nil {
		return workspaces.Workspace{}, errors.New("GitHub repository is required")
	}
	if err := s.github.CloneRepository(ctx, remoteRepositoryID, targetPath); err != nil {
		return workspaces.Workspace{}, err
	}

	return s.workspaces.Get(ctx, workspaceID)
}

func (s *Service) WorkspaceDirectories() []WorkspaceDirectory {
	if s == nil || s.state == nil {
		return nil
	}

	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	return cloneWorkspaceDirectories(s.state.workspaceDirectories)
}

func (s *Service) Configure(workspaceDirectories []WorkspaceDirectory) {
	if s == nil || s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	s.state.workspaceDirectories = cloneWorkspaceDirectories(workspaceDirectories)
}

func (s *Service) workspaceDirectory(setting string) (WorkspaceDirectory, bool) {
	for _, workspaceDirectory := range s.WorkspaceDirectories() {
		if workspaceDirectory.Setting == setting {
			return workspaceDirectory, true
		}
	}

	return WorkspaceDirectory{}, false
}

func parseGitHubRepositories(output string) ([]githubRepository, error) {
	repositories := make([]githubRepository, 0)
	if err := json.Unmarshal([]byte(output), &repositories); err != nil {
		return nil, fmt.Errorf("parsing GitHub repositories: %w", err)
	}

	return repositories, nil
}

func buildRemoteRepository(
	repository githubRepository,
	workspaceIDsByRemote map[string]map[string]struct{},
) (RemoteRepository, error) {
	name := strings.TrimSpace(repository.Name)
	remoteRepositoryID := strings.TrimSpace(repository.NameWithOwner)
	webURL := strings.TrimSpace(repository.URL)
	cloneURL := strings.TrimSpace(repository.SSHURL)
	if name == "" || remoteRepositoryID == "" || webURL == "" || cloneURL == "" {
		return RemoteRepository{}, errors.New("GitHub repository response is missing required fields")
	}
	if !remoteRepositoryIDPattern.MatchString(remoteRepositoryID) {
		return RemoteRepository{}, InvalidRemoteRepositoryIDError{RemoteRepositoryID: remoteRepositoryID}
	}

	remoteIdentity := gitrepositories.CanonicalRemoteIdentity(cloneURL)
	workspaceIDSet := workspaceIDsByRemote[remoteIdentity]
	localWorkspaceIDs := make([]string, 0, len(workspaceIDSet))
	for workspaceID := range workspaceIDSet {
		localWorkspaceIDs = append(localWorkspaceIDs, workspaceID)
	}
	sort.Strings(localWorkspaceIDs)

	return RemoteRepository{
		ID:                remoteRepositoryID,
		Name:              name,
		WebURL:            webURL,
		CloneURL:          cloneURL,
		LocalWorkspaceIDs: localWorkspaceIDs,
	}, nil
}

func ensureTargetDoesNotExist(files FileRepository, path string) error {
	return files.EnsurePathDoesNotExist(path)
}

func cloneWorkspaceDirectories(workspaceDirectories []WorkspaceDirectory) []WorkspaceDirectory {
	return append([]WorkspaceDirectory(nil), workspaceDirectories...)
}
