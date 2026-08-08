// NOTE: Vibecoded and not suppppppper reviewed
package repositories

// TODO: Review properly

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func copyIgnoredFiles(ctx context.Context, mainPath string, targetPath string, excludes []string, git Git, files FileSystem) []string {
	paths, err := listIgnoredPaths(ctx, mainPath, git)
	if err != nil {
		return []string{fmt.Sprintf("listing ignored files: %v", err)}
	}

	warnings := make([]string, 0)
	for _, relPath := range paths {
		if isExcludedFromCopy(relPath, excludes) {
			continue
		}

		source := filepath.Join(mainPath, relPath)
		destination := filepath.Join(targetPath, relPath)
		if err := files.CopyPath(source, destination); err != nil {
			warnings = append(warnings, fmt.Sprintf("copying %s: %v", relPath, err))
		}
	}

	return warnings
}

func listIgnoredPaths(ctx context.Context, repositoryPath string, git Git) ([]string, error) {
	paths, err := git.IgnoredPaths(ctx, repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	return paths, nil
}

func isExcludedFromCopy(path string, excludes []string) bool {
	matchPath := strings.TrimSuffix(path, "/")
	for _, pattern := range excludes {
		if matched, _ := doublestar.Match(pattern, matchPath); matched {
			return true
		}
	}

	return false
}
