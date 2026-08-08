package filesystem

// TODO: Review properly

import (
	"errors"
	"os"
	"sort"
)

// IDForDirectory resolves a directory to its discovered workspace ID.
func (s WorkspaceDiscovery) IDForDirectory(directory string) (string, bool, error) {
	workspaces, err := s.discover()
	if err != nil {
		return "", false, err
	}

	return workspaceIDForDirectory(workspaces, directory)
}

// IDsForDirectories maps exact workspace directories to discovered workspace IDs.
func (s WorkspaceDiscovery) IDsForDirectories(directories []string) ([]string, error) {
	workspaces, err := s.discover()
	if err != nil {
		return nil, err
	}

	workspaceIDs := make([]string, 0, len(directories))
	seenIDs := make(map[string]struct{})

	for _, directory := range directories {
		workspaceID, found, err := workspaceIDForDirectory(workspaces, directory)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if _, seen := seenIDs[workspaceID]; seen {
			continue
		}

		seenIDs[workspaceID] = struct{}{}
		workspaceIDs = append(workspaceIDs, workspaceID)
	}

	sort.Strings(workspaceIDs)

	return workspaceIDs, nil
}

func workspaceIDForDirectory(workspaces []discoveredWorkspace, directory string) (string, bool, error) {
	canonicalDirectory, err := canonicalDirectoryPath(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}

		return "", false, err
	}

	for _, workspace := range workspaces {
		if workspace.canonicalPath == canonicalDirectory {
			return workspace.id, true, nil
		}
	}

	return "", false, nil
}
