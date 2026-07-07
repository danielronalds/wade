package remote

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
	"time"
)

const githubCommandTimeout = 2 * time.Minute

var nameWithOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type Service struct {
	runner CommandRunner
}

type CommandRunner func(ctx context.Context, name string, args ...string) (string, error)

type githubProject struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
}

func NewService(runner CommandRunner) Service {
	return Service{runner: runner}
}

func (s Service) List(ctx context.Context, localProjectNames []string) ([]Project, error) {
	output, err := s.runGithub(ctx, "repo", "list", "--json", "name,nameWithOwner,url,sshUrl", "--limit", "5000")
	if err != nil {
		return nil, fmt.Errorf("listing GitHub projects: %w", err)
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
	if err := ensureTargetDoesNotExist(targetPath); err != nil {
		return ClonedProject{}, err
	}

	if err := os.MkdirAll(projectDirectory, 0o755); err != nil {
		return ClonedProject{}, fmt.Errorf("creating project directory: %w", err)
	}

	if _, err := s.runGithub(ctx, "repo", "clone", repository.nameWithOwner, targetPath); err != nil {
		return ClonedProject{}, fmt.Errorf("cloning GitHub project: %w", err)
	}

	return ClonedProject{Name: repository.name, Path: targetPath}, nil
}

func (s Service) runGithub(ctx context.Context, args ...string) (string, error) {
	if s.runner == nil {
		return "", errors.New("command runner is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, githubCommandTimeout)
	defer cancel()

	return s.runner(commandContext, "gh", args...)
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

func ensureTargetDoesNotExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("project already exists at %s", path)
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
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
