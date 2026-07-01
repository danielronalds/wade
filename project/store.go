package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Store struct {
	state *storeState
}

type storeState struct {
	mu          sync.RWMutex
	directories []string
}

// NewStore creates a project store for the supplied discovery directories.
func NewStore(directories []string) Store {
	return Store{state: &storeState{directories: cloneDirectories(directories)}}
}

// Reload swaps the discovery directories used by future project lookups.
func (s Store) Reload(directories []string) {
	if s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	s.state.directories = cloneDirectories(directories)
}

// Names lists unique project names across configured discovery directories.
func (s Store) Names() ([]string, error) {
	seenProjects := make(map[string]struct{})
	projectNames := make([]string, 0)

	for _, directory := range s.directories() {
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

// Path resolves a project name to its first matching directory.
func (s Store) Path(name string) (string, error) {
	if !isValidName(name) {
		return "", errors.New("invalid project name")
	}

	for _, directory := range s.directories() {
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

// directories returns a copy of the current discovery directories.
func (s Store) directories() []string {
	if s.state == nil {
		return nil
	}

	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	return cloneDirectories(s.state.directories)
}

// cloneDirectories prevents callers from mutating store state through slices.
func cloneDirectories(directories []string) []string {
	return append([]string(nil), directories...)
}

// isValidName checks that a project name cannot escape discovery directories.
func isValidName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	if filepath.IsAbs(name) {
		return false
	}

	return filepath.Base(name) == name
}
