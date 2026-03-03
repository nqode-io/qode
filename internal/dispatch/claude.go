package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// claudeCLI dispatches prompts via the `claude -p` CLI.
type claudeCLI struct {
	binaryPath string
}

// knownClaudePaths lists common installation locations tried when "claude" is
// not found on PATH.
var knownClaudePaths = []string{
	"~/.local/bin/claude",
	"/usr/local/bin/claude",
}

// newClaudeCLI attempts to locate the claude binary.
// Returns a dispatcher regardless; check Available() before use.
func newClaudeCLI() *claudeCLI {
	if path, err := exec.LookPath("claude"); err == nil {
		return &claudeCLI{binaryPath: path}
	}
	// PATH may not include the user's local bin — try known locations.
	home, _ := os.UserHomeDir()
	for _, candidate := range knownClaudePaths {
		if strings.HasPrefix(candidate, "~/") {
			candidate = home + candidate[1:]
		}
		if _, err := os.Stat(candidate); err == nil {
			return &claudeCLI{binaryPath: candidate}
		}
	}
	return &claudeCLI{}
}

func (c *claudeCLI) Name() string { return "claude" }

func (c *claudeCLI) Available() bool { return c.binaryPath != "" }

// Run executes the prompt via `claude -p` and returns the output.
// The prompt is sent via stdin to avoid shell argument length limits.
func (c *claudeCLI) Run(ctx context.Context, prompt string, opts Options) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, opts.timeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binaryPath,
		"--print",
		"--allowedTools", "Read,Write,Glob,Grep",
		"--model", "sonnet",
		"--output-format", "text",
	)
	cmd.Stdin = strings.NewReader(prompt)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}
	// Strip CLAUDECODE so the subprocess isn't blocked by nested-session detection.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// claude writes auth/API errors to stdout, not stderr — check both.
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("claude CLI: %s", errMsg)
	}

	return stdout.String(), nil
}

// filterEnv returns a copy of env with all entries for the given key removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
