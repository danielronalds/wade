package projects

// TODO: Review properly

import "context"

type Repository interface {
	Names() ([]string, error)
	Path(name string) (string, error)
	Directories() []string
	NamesForDirectories(directories []string) []string
	Reload(directories []string)
}

type GitRepository interface {
	CurrentBranch(ctx context.Context, projectPath string) (string, error)
	OriginURL(ctx context.Context, projectPath string) (string, error)
}

type GitHubRepository interface {
	PullRequestURL(ctx context.Context, repo string, branch string) (string, error)
}

type Service struct {
	projects Repository
	git      GitRepository
	github   GitHubRepository
}

func NewService(projects Repository, git GitRepository, github GitHubRepository) Service {
	return Service{projects: projects, git: git, github: github}
}

func (s Service) List() ([]string, error) {
	return s.projects.Names()
}

func (s Service) Path(name string) (string, error) {
	return s.projects.Path(name)
}

func (s Service) Directories() []string {
	return s.projects.Directories()
}

func (s Service) NamesForDirectories(directories []string) []string {
	return s.projects.NamesForDirectories(directories)
}

func (s Service) Reload(directories []string) {
	s.projects.Reload(directories)
}

func (s Service) Details(ctx context.Context, name string) (Metadata, error) {
	projectPath, err := s.projects.Path(name)
	if err != nil {
		return Metadata{}, err
	}

	return Details(ctx, projectPath, s.git, s.github), nil
}
