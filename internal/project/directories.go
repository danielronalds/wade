package project

import (
	"path/filepath"
	"sort"
)

func (s Store) NamesForDirectories(directories []string) []string {
	projectNames := make([]string, 0)
	seenProjects := make(map[string]struct{})

	for _, directory := range directories {
		projectName := filepath.Base(directory)
		projectPath, err := s.Path(projectName)
		if err != nil || filepath.Clean(projectPath) != filepath.Clean(directory) {
			continue
		}

		if _, seen := seenProjects[projectName]; seen {
			continue
		}

		seenProjects[projectName] = struct{}{}
		projectNames = append(projectNames, projectName)
	}

	sort.Strings(projectNames)

	return projectNames
}
