package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Store struct {
	directories []string
}

func NewStore(directories []string) Store {
	return Store{directories: directories}
}

func (s Store) Names() ([]string, error) {
	seenProjects := make(map[string]struct{})
	projectNames := make([]string, 0)

	for _, directory := range s.directories {
		if directory == "" {
			return nil, errors.New("invalid project directory")
		}

		entries, err := os.ReadDir(directory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectName := entry.Name()
			if _, exists := seenProjects[projectName]; exists {
				continue
			}

			seenProjects[projectName] = struct{}{}
			projectNames = append(projectNames, projectName)
		}
	}

	sort.Strings(projectNames)

	return projectNames, nil
}

func (s Store) Path(name string) (string, error) {
	if !isValidName(name) {
		return "", errors.New("invalid project name")
	}

	for _, directory := range s.directories {
		if directory == "" {
			return "", errors.New("invalid project directory")
		}

		path := filepath.Join(directory, name)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	return "", fmt.Errorf("project %q not found", name)
}

func isValidName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	if filepath.IsAbs(name) {
		return false
	}

	return filepath.Base(name) == name
}
