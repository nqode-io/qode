//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nqode/qode/internal/qodecontext"
)

func TestRunContextInit_CreatesWithoutSwitch(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	var buf bytes.Buffer
	if err := runContextInit(context.Background(), &buf, "feat-login", false); err != nil {
		t.Fatalf("runContextInit: %v", err)
	}

	// Context directory should exist but no symlink should be created.
	_, err := qodecontext.CurrentName(context.Background(), root)
	if !errors.Is(err, qodecontext.ErrNoCurrentContext) {
		t.Errorf("want ErrNoCurrentContext, got: %v", err)
	}
}

func TestRunContextInit_WithAutoSwitch(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	var buf bytes.Buffer
	if err := runContextInit(context.Background(), &buf, "feat-login", true); err != nil {
		t.Fatalf("runContextInit: %v", err)
	}

	name, err := qodecontext.CurrentName(context.Background(), root)
	if err != nil {
		t.Fatalf("CurrentName: %v", err)
	}
	if name != "feat-login" {
		t.Errorf("got %q, want %q", name, "feat-login")
	}
}

func TestRunContextInit_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	if err := qodecontext.Init(context.Background(), root, "existing"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := runContextInit(context.Background(), &bytes.Buffer{}, "existing", false); err == nil {
		t.Fatal("expected error for duplicate context name")
	}
}

func TestRunContextSwitch_SwitchesActive(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	if err := qodecontext.Init(context.Background(), root, "ctx-a"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := qodecontext.Init(context.Background(), root, "ctx-b"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := qodecontext.Switch(context.Background(), root, "ctx-a"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var buf bytes.Buffer
	if err := runContextSwitch(context.Background(), &buf, "ctx-b"); err != nil {
		t.Fatalf("runContextSwitch: %v", err)
	}

	name, err := qodecontext.CurrentName(context.Background(), root)
	if err != nil {
		t.Fatalf("CurrentName: %v", err)
	}
	if name != "ctx-b" {
		t.Errorf("got %q, want %q", name, "ctx-b")
	}
}

func TestRunContextClear_ClearsFiles(t *testing.T) {
	root := setupTestRoot(t, "clear-test")

	ctxDir := filepath.Join(root, ".qode", "contexts", "clear-test")
	if err := os.WriteFile(filepath.Join(ctxDir, "spec.md"), []byte("spec"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var buf bytes.Buffer
	if err := runContextClear(context.Background(), &buf, "clear-test"); err != nil {
		t.Fatalf("runContextClear: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ctxDir, "spec.md")); !os.IsNotExist(err) {
		t.Error("expected spec.md to be removed after clear")
	}
	if _, err := os.Stat(filepath.Join(ctxDir, "ticket.md")); err != nil {
		t.Errorf("expected ticket.md to exist after clear: %v", err)
	}
}

func TestRunContextRemove_RemovesDir(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	if err := qodecontext.Init(context.Background(), root, "removable"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := qodecontext.Switch(context.Background(), root, "removable"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var buf bytes.Buffer
	if err := runContextRemove(context.Background(), &buf, "removable"); err != nil {
		t.Fatalf("runContextRemove: %v", err)
	}

	ctxDir := filepath.Join(root, ".qode", "contexts", "removable")
	if _, err := os.Stat(ctxDir); !os.IsNotExist(err) {
		t.Error("expected context directory to be removed")
	}
}

func TestRunContextReset_ClearsSymlink(t *testing.T) {
	root := setupTestRoot(t, "reset-test")

	var buf bytes.Buffer
	if err := runContextReset(context.Background(), &buf); err != nil {
		t.Fatalf("runContextReset: %v", err)
	}

	link := filepath.Join(root, ".qode", "contexts", "current")
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("expected current symlink to be removed after reset")
	}
}
