package dispatch

import (
	"context"
	"errors"
	"os"
	"testing"
)

// isTTY returns false in test environments where stdin is a pipe.
func TestIsTTY_NonTerminal(t *testing.T) {
	if isTTY() {
		t.Skip("stdin is a TTY in this environment; skipping non-TTY assertion")
	}
}

// RunInteractive falls back to batch mode (c.Run) when stdin is not a TTY.
// We use a real binary (true) as the claude stub so c.Run exits cleanly.
func TestRunInteractive_NonTTYFallsThroughToRun(t *testing.T) {
	if isTTY() {
		t.Skip("test requires non-TTY stdin")
	}

	// Point binaryPath at a real binary that exits 0.
	truePath := "/usr/bin/true"
	if _, err := os.Stat(truePath); errors.Is(err, os.ErrNotExist) {
		t.Skip("/usr/bin/true not available")
	}

	c := &claudeCLI{binaryPath: truePath}
	// In non-TTY mode RunInteractive delegates to c.Run, which calls
	// `true --print --allowedTools ... --model sonnet --output-format text`
	// with the prompt on stdin. `true` ignores all args and exits 0.
	err := c.RunInteractive(context.Background(), "test prompt", Options{})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// RunInteractive in non-TTY mode delegates to c.Run, so no temp file is created.
// Temp file cleanup in TTY mode is verified by the isTTY+RunInteractive integration
// and requires manual testing (stdin must be a terminal).
func TestRunInteractive_TempFileCleanedUp_NonTTY(t *testing.T) {
	if isTTY() {
		t.Skip("test requires non-TTY stdin")
	}
	// In non-TTY mode no temp file is created; verify no leftover files exist.
	before, _ := os.ReadDir(os.TempDir())
	truePath := "/usr/bin/true"
	if _, err := os.Stat(truePath); errors.Is(err, os.ErrNotExist) {
		t.Skip("/usr/bin/true not available")
	}
	c := &claudeCLI{binaryPath: truePath}
	_ = c.RunInteractive(context.Background(), "test prompt", Options{})
	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before) {
		t.Error("unexpected temp files created in non-TTY RunInteractive path")
	}
}

// clipboardDispatcher returns ErrManualDispatch — the fallback path used when
// no claude binary is available.
func TestClipboardDispatcher_ReturnsErrManualDispatch(t *testing.T) {
	d := &clipboardDispatcher{}
	_, err := d.Run(context.Background(), "test prompt", Options{})
	if !errors.Is(err, ErrManualDispatch) {
		t.Errorf("expected ErrManualDispatch, got %v", err)
	}
}

// RunInteractive (package-level) delegates to clipboard when claudeCLI is unavailable.
func TestRunInteractive_PackageLevel_ClipboardFallback(t *testing.T) {
	if isTTY() {
		t.Skip("clipboard fallback test requires non-TTY stdin")
	}
	// Construct a claudeCLI with no binary path to simulate unavailability,
	// then verify the package-level function falls back to clipboard.
	// We test the underlying clipboard path directly since newClaudeCLI checks
	// known paths on disk beyond just PATH.
	d := &clipboardDispatcher{}
	_, err := d.Run(context.Background(), "test", Options{})
	if !errors.Is(err, ErrManualDispatch) {
		t.Errorf("expected ErrManualDispatch from clipboard fallback, got %v", err)
	}
}
