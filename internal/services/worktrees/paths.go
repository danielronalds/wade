// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func worktreePath(mainPath string, repositoryName string, branch string) (string, error) {
	sanitised := sanitiseForDirectory(branch)
	if sanitised == "" {
		return "", fmt.Errorf("invalid branch %q: results in empty directory name after sanitisation", branch)
	}

	return filepath.Join(filepath.Dir(mainPath), repositoryName+"-"+sanitised), nil
}

func samePath(first string, second string) bool {
	if first == "" || second == "" {
		return false
	}

	firstPath := cleanPath(first)
	secondPath := cleanPath(second)
	return firstPath == secondPath
}

func cleanPath(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		absolutePath = path
	}

	resolvedPath, err := os.Readlink(absolutePath)
	if err == nil && !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(filepath.Dir(absolutePath), resolvedPath)
	}
	if err == nil {
		absolutePath = resolvedPath
	}

	if evaluatedPath, err := filepath.EvalSymlinks(absolutePath); err == nil {
		absolutePath = evaluatedPath
	}

	return filepath.Clean(absolutePath)
}

var nonAlphanumericDashDotUnderscore = regexp.MustCompile(`[^a-zA-Z0-9\-._]`)
var multipleDashes = regexp.MustCompile(`-{2,}`)

func sanitiseForDirectory(name string) string {
	result := strings.ReplaceAll(name, "/", "-")
	result = strings.ReplaceAll(result, " ", "-")
	result = nonAlphanumericDashDotUnderscore.ReplaceAllString(result, "")
	result = multipleDashes.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")
	return result
}
