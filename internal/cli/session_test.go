//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nqode/qode/internal/qodecontext"
)

func TestLoadSession_HappyPath(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })
	writeConfigFile(t, root, testYAMLMinimal)
	if err := qodecontext.Init(context.Background(), root, "test-context"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := qodecontext.Switch(context.Background(), root, "test-context"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	sess, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if sess.Config == nil {
		t.Fatal("Config is nil")
	}
	if !sess.Config.IDE.ClaudeCode.Enabled {
		t.Error("expected ClaudeCode enabled from default config")
	}
	if sess.Context == nil {
		t.Fatal("Context is nil")
	}
	if sess.Context.ContextDir == "" {
		t.Error("expected non-empty ContextDir")
	}
	if sess.Engine == nil {
		t.Fatal("Engine is nil")
	}
	if sess.Engine.ProjectName() == "" {
		t.Error("expected non-empty ProjectName")
	}
}

func TestLoadSession_NoCurrentContext(t *testing.T) {
	// Setup root with config but no context symlink (no Init/Switch).
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })
	writeConfigFile(t, root, testYAMLMinimal)
	// Create .qode/contexts/ dir but no symlink.
	if err := os.MkdirAll(filepath.Join(root, ".qode", "contexts"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := loadSession()
	if err == nil {
		t.Fatal("expected error when no current context")
	}
	if !errors.Is(err, qodecontext.ErrNoCurrentContext) {
		t.Errorf("want ErrNoCurrentContext, got: %v", err)
	}
}

func TestLoadSessionCtx_CancelledContext(t *testing.T) {
	_ = setupTestRootWithConfig(t, testYAMLMinimal)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := loadSessionCtx(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
