package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// WorkspaceDiscovery resolves configured workspace directories through fresh scans.
type WorkspaceDiscovery struct {
	state *workspaceDiscoveryState
}

type workspaceDiscoveryState struct {
	mu          sync.RWMutex
	directories []string
}

type discoveredWorkspace struct {
	id            string
	path          string
	canonicalPath string
}

// NewWorkspaceDiscovery creates a workspace store for the supplied discovery directories.
func NewWorkspaceDiscovery(directories []string) WorkspaceDiscovery {
	return WorkspaceDiscovery{state: &workspaceDiscoveryState{directories: cloneDirectories(directories)}}
}

// IDs lists the unique workspace IDs in configured discovery directories.
func (s WorkspaceDiscovery) IDs() ([]string, error) {
	workspaces, err := s.discover()
	if err != nil {
		return nil, err
	}

	workspaceIDs := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		workspaceIDs = append(workspaceIDs, workspace.id)
	}
	sort.Strings(workspaceIDs)

	return workspaceIDs, nil
}

// Resolve returns the path for the first workspace matching the supplied ID.
func (s WorkspaceDiscovery) Resolve(workspaceID string) (string, bool, error) {
	workspaces, err := s.discover()
	if err != nil {
		return "", false, err
	}

	for _, workspace := range workspaces {
		if workspace.id == workspaceID {
			return workspace.path, true, nil
		}
	}

	return "", false, nil
}

// CanonicalPath returns the canonical path for the supplied workspace ID.
func (s WorkspaceDiscovery) CanonicalPath(workspaceID string) (string, bool, error) {
	workspaces, err := s.discover()
	if err != nil {
		return "", false, err
	}

	for _, workspace := range workspaces {
		if workspace.id == workspaceID {
			return workspace.canonicalPath, true, nil
		}
	}

	return "", false, nil
}

// Directories returns the configured workspace discovery directories.
func (s WorkspaceDiscovery) Directories() []string {
	if s.state == nil {
		return nil
	}

	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	return cloneDirectories(s.state.directories)
}

// Reload swaps the discovery directories used by future workspace lookups.
func (s WorkspaceDiscovery) Reload(directories []string) {
	if s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	s.state.directories = cloneDirectories(directories)
}

func (s WorkspaceDiscovery) discover() ([]discoveredWorkspace, error) {
	workspaceDirectories := s.Directories()
	workspaces := make([]discoveredWorkspace, 0)
	seenIDs := make(map[string]struct{})
	seenPaths := make(map[string]struct{})

	for _, directory := range workspaceDirectories {
		if directory == "" {
			return nil, errors.New("invalid workspace directory")
		}

		entries, err := os.ReadDir(directory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		for _, entry := range entries {
			workspaceID := entry.Name()
			if _, exists := seenIDs[workspaceID]; exists {
				continue
			}

			workspacePath := filepath.Join(directory, workspaceID)
			info, err := os.Stat(workspacePath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}

				return nil, err
			}
			if !info.IsDir() {
				continue
			}

			canonicalPath, err := canonicalDirectoryPath(workspacePath)
			if err != nil {
				return nil, err
			}
			if _, exists := seenPaths[canonicalPath]; exists {
				continue
			}

			seenIDs[workspaceID] = struct{}{}
			seenPaths[canonicalPath] = struct{}{}
			workspaces = append(workspaces, discoveredWorkspace{
				id:            workspaceID,
				path:          workspacePath,
				canonicalPath: canonicalPath,
			})
		}
	}

	return workspaces, nil
}

func canonicalDirectoryPath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	canonicalPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return "", err
	}

	return filepath.Clean(canonicalPath), nil
}

func cloneDirectories(directories []string) []string {
	return append([]string(nil), directories...)
}
