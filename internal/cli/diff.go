package cli

import (
	"context"
	"os/exec"
	"strings"

	"github.com/nqode/qode/internal/log"
)

// runDiffCommand executes the configured diff command in root and returns its
// stdout as a string. Returns an empty string (no error) when the command
// produces no output or when the diff command is not configured.
func runDiffCommand(root, command string) string {
	return runDiffCommandCtx(context.Background(), root, command)
}

// runDiffCommandCtx is like runDiffCommand but respects context cancellation.
func runDiffCommandCtx(ctx context.Context, root, command string) string {
	if command == "" {
		return ""
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = root

	out, err := cmd.Output()
	if err != nil {
		log.Warn("diff command failed", "command", command, "error", err)
		return ""
	}

	return strings.TrimRight(string(out), "\n")
}
