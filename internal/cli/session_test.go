//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	sess, err := loadSession(io.Discard)
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if sess.Config == nil {
		t.Fatal("Config is nil")
	}
	if !sess.Config.Agents.ClaudeCode.Enabled {
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

	_, err := loadSession(io.Discard)
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

	_, err := loadSessionCtx(ctx, io.Discard)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// bothKeysConfig carries the canonical and the deprecated agent key at once, the
// state the deprecation warning exists to report.
const bothKeysConfig = "agents:\n  cursor:\n    enabled: true\nide:\n  cursor:\n    enabled: false\n"

// bothKeysWarning is the distinguishing fragment of config's both-keys notice.
const bothKeysWarning = "both 'agents:' and 'ide:' are set"

func TestLoadSession_BothKeysConfig_WarnsOnce(t *testing.T) {
	// flagRoot is a package global, so this test cannot run in parallel.
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })
	writeConfigFile(t, root, bothKeysConfig)
	if err := qodecontext.Init(context.Background(), root, "test-context"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := qodecontext.Switch(context.Background(), root, "test-context"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	var errOut bytes.Buffer
	if _, err := loadSession(&errOut); err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if got := strings.Count(errOut.String(), bothKeysWarning); got != 1 {
		t.Errorf("warning printed %d times, want 1:\n%s", got, errOut.String())
	}
}
