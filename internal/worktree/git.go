// NOTE: Vibecoded and not suppppppper reviewed
package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const gitCommandTimeout = 30 * time.Second

func gitOutput(ctx context.Context, projectPath string, args ...string) (string, error) {
	commandContext, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	defer cancel()

	command := exec.CommandContext(commandContext, "git", append([]string{"-C", projectPath}, args...)...)
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}

	return text, nil
}

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
