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

// isTTY reports whether os.Stdin is an interactive terminal.
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// RunInteractive runs claude as a foreground interactive process, wiring
// stdin/stdout/stderr to the current terminal. Falls back to batch mode
// when stdin is not a terminal (e.g., CI environments).
func (c *claudeCLI) RunInteractive(ctx context.Context, prompt string, opts Options) error {
	if !isTTY() {
		_, err := c.Run(ctx, prompt, opts)
		return err
	}

	f, err := os.CreateTemp("", "qode-prompt-*.md")
	if err != nil {
		return fmt.Errorf("create temp prompt file: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.WriteString(prompt); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close prompt file: %w", err)
	}

	readInstruction := fmt.Sprintf("Read and execute the instructions in %s", f.Name())

	cmd := exec.CommandContext(ctx, c.binaryPath,
		"--allowedTools", "Read,Write,Glob,Grep",
		"--model", "sonnet",
		readInstruction,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}
	return cmd.Run()
}

// RunInteractive resolves the best available dispatcher and runs it interactively.
// Falls back to clipboard dispatch if claude CLI is unavailable.
func RunInteractive(ctx context.Context, prompt string, opts Options) error {
	if d := newClaudeCLI(); d.Available() {
		return d.RunInteractive(ctx, prompt, opts)
	}
	d := &clipboardDispatcher{}
	_, err := d.Run(ctx, prompt, opts)
	return err
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
