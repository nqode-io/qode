package cli

import (
	"context"
	"runtime"
	"testing"
)

func TestRunDiffCommand_Success(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("echo behaves differently on Windows")
	}
	root := t.TempDir()
	got := runDiffCommand(root, "echo hello")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestRunDiffCommand_EmptyConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got := runDiffCommand(root, "")
	if got != "" {
		t.Errorf("expected empty string for empty command, got %q", got)
	}
}

func TestRunDiffCommand_CommandFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A command that always exits non-zero.
	got := runDiffCommand(root, "false")
	if got != "" {
		t.Errorf("expected empty string when command fails, got %q", got)
	}
}

func TestRunDiffCommandCtx_Cancelled(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Even a fast command should return "" when context is already cancelled.
	got := runDiffCommandCtx(ctx, root, "echo hello")
	if got != "" {
		t.Errorf("expected empty string for cancelled context, got %q", got)
	}
}
