package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// resolveProjectDirectories expands configured project directory paths.
func resolveProjectDirectories(homeDir string, directories []string) ([]string, error) {
	projectDirs := make([]string, 0, len(directories))
	for _, directory := range directories {
		projectDir, err := resolveProjectDirectory(homeDir, directory)
		if err != nil {
			return nil, err
		}

		projectDirs = append(projectDirs, projectDir)
	}

	return projectDirs, nil
}

// resolveProjectDirectory expands a single path using ~ or an absolute path.
func resolveProjectDirectory(homeDir string, directory string) (string, error) {
	if directory == "" {
		return "", errors.New("project directory cannot be empty")
	}

	if directory == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(directory, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(directory, "~/")), nil
	}

	if filepath.IsAbs(directory) {
		return filepath.Clean(directory), nil
	}

	return "", fmt.Errorf("project directory %q must use ~ or an absolute path", directory)
}
