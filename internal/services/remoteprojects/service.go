package remoteprojects

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
)

var nameWithOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type gitHubRepository interface {
	ListProjects(ctx context.Context) (string, error)
	CloneProject(ctx context.Context, nameWithOwner string, targetPath string) error
}

type fileRepository interface {
	EnsureDirectory(path string) error
	EnsurePathDoesNotExist(path string) error
}

type Service struct {
	github gitHubRepository
	files  fileRepository
}

type githubProject struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
}

func NewService(github gitHubRepository, files fileRepository) Service {
	return Service{github: github, files: files}
}

func (s Service) List(ctx context.Context, localProjectNames []string) ([]Project, error) {
	if s.github == nil {
		return nil, errors.New("GitHub repository is required")
	}

	output, err := s.github.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	githubProjects, err := parseGithubProjects(output)
	if err != nil {
		return nil, err
	}

	localNames := localProjectNameSet(localProjectNames)
	projects := make([]Project, 0, len(githubProjects))
	for _, githubProject := range githubProjects {
		project, err := buildProject(githubProject, localNames)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	sort.Slice(projects, func(firstIndex int, secondIndex int) bool {
		return projects[firstIndex].NameWithOwner < projects[secondIndex].NameWithOwner
	})

	return projects, nil
}

func (s Service) Clone(ctx context.Context, request CloneRequest) (ClonedProject, error) {
	repository, err := parseNameWithOwner(request.NameWithOwner)
	if err != nil {
		return ClonedProject{}, err
	}

	if _, isLocal := localProjectNameSet(request.LocalProjectNames)[repository.name]; isLocal {
		return ClonedProject{}, fmt.Errorf("project %q already exists locally", repository.name)
	}

	projectDirectory, err := projectDirectoryAt(request.ProjectDirectories, request.DirectoryIndex)
	if err != nil {
		return ClonedProject{}, err
	}

	targetPath := filepath.Join(projectDirectory, repository.name)
	if s.files == nil {
		return ClonedProject{}, errors.New("file repository is required")
	}

	if err := ensureTargetDoesNotExist(s.files, targetPath); err != nil {
		return ClonedProject{}, err
	}

	if err := s.files.EnsureDirectory(projectDirectory); err != nil {
		return ClonedProject{}, fmt.Errorf("creating project directory: %w", err)
	}

	if s.github == nil {
		return ClonedProject{}, errors.New("GitHub repository is required")
	}

	if err := s.github.CloneProject(ctx, repository.nameWithOwner, targetPath); err != nil {
		return ClonedProject{}, err
	}

	return ClonedProject{Name: repository.name, Path: targetPath}, nil
}

func parseGithubProjects(output string) ([]githubProject, error) {
	projects := make([]githubProject, 0)
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		return nil, fmt.Errorf("parsing GitHub projects: %w", err)
	}

	return projects, nil
}

func buildProject(project githubProject, localNames map[string]struct{}) (Project, error) {
	name := strings.TrimSpace(project.Name)
	nameWithOwner := strings.TrimSpace(project.NameWithOwner)
	if name == "" || nameWithOwner == "" {
		return Project{}, errors.New("GitHub project response is missing required fields")
	}

	_, isLocal := localNames[name]
	localName := ""
	if isLocal {
		localName = name
	}

	return Project{
		Name:          name,
		NameWithOwner: nameWithOwner,
		URL:           strings.TrimSpace(project.URL),
		SSHURL:        strings.TrimSpace(project.SSHURL),
		IsLocal:       isLocal,
		LocalName:     localName,
	}, nil
}

type repositoryIdentity struct {
	nameWithOwner string
	name          string
}

func parseNameWithOwner(value string) (repositoryIdentity, error) {
	nameWithOwner := strings.TrimSpace(value)
	if !nameWithOwnerPattern.MatchString(nameWithOwner) {
		return repositoryIdentity{}, errors.New("invalid GitHub project name")
	}

	parts := strings.Split(nameWithOwner, "/")
	return repositoryIdentity{nameWithOwner: nameWithOwner, name: parts[1]}, nil
}

func projectDirectoryAt(projectDirectories []string, directoryIndex int) (string, error) {
	if directoryIndex < 0 || directoryIndex >= len(projectDirectories) {
		return "", errors.New("invalid project directory")
	}

	projectDirectory := strings.TrimSpace(projectDirectories[directoryIndex])
	if projectDirectory == "" {
		return "", errors.New("invalid project directory")
	}

	return projectDirectory, nil
}

func ensureTargetDoesNotExist(files fileRepository, path string) error {
	if err := files.EnsurePathDoesNotExist(path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("project already exists at %s", path)
		}

		return err
	}

	return nil
}

func localProjectNameSet(projectNames []string) map[string]struct{} {
	localNames := make(map[string]struct{}, len(projectNames))
	for _, projectName := range projectNames {
		projectName = strings.TrimSpace(projectName)
		if projectName == "" {
			continue
		}
		localNames[projectName] = struct{}{}
	}

	return localNames
}
