// NOTE: Vibecoded and not suppppppper reviewed
package worktrees

// TODO: Review properly

import "strings"

func parseLines(output string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
