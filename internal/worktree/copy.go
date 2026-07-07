// NOTE: Vibecoded and not suppppppper reviewed
package worktree

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func copyIgnoredFiles(ctx context.Context, mainPath string, targetPath string, excludes []string) []string {
	paths, err := listIgnoredPaths(ctx, mainPath)
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
		if err := copyPath(source, destination); err != nil {
			warnings = append(warnings, fmt.Sprintf("copying %s: %v", relPath, err))
		}
	}

	return warnings
}

func listIgnoredPaths(ctx context.Context, projectPath string) ([]string, error) {
	output, err := gitOutput(ctx, projectPath, "ls-files", "--ignored", "--exclude-standard", "--others", "--directory")
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	return parseLines(output), nil
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

func copyPath(source string, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(source, destination)
	}

	if info.IsDir() {
		return copyDirectory(source, destination)
	}

	return copyFile(source, destination, info.Mode())
}

func copySymlink(source string, destination string) error {
	linkTarget, err := os.Readlink(source)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	if err := os.RemoveAll(destination); err != nil {
		return err
	}

	return os.Symlink(linkTarget, destination)
}

func copyDirectory(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		target := filepath.Join(destination, relPath)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return copySymlink(path, target)
		}

		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(source string, destination string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}
