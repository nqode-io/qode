//go:build !integration

package cli

import (
	"context"
	"testing"
)

func TestLoadSession_HappyPath(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", testYAMLMinimal)

	sess, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if sess.Branch != "test-branch" {
		t.Errorf("Branch = %q, want %q", sess.Branch, "test-branch")
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

func TestLoadSession_NotGitRepo(t *testing.T) {
	// Point flagRoot to a non-git directory.
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	_, err := loadSession()
	if err == nil {
		t.Fatal("expected error when root is not a git repo")
	}
}

func TestLoadSessionCtx_CancelledContext(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", testYAMLMinimal)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := loadSessionCtx(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
