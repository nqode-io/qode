//go:build !integration

package cli

import (
	"testing"
)

func TestLoadSession_HappyPath(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n")

	sess, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if sess.Branch != "test-branch" {
		t.Errorf("Branch = %q, want %q", sess.Branch, "test-branch")
	}
	if sess.Config == nil {
		t.Error("Config is nil")
	}
	if sess.Context == nil {
		t.Error("Context is nil")
	}
	if sess.Engine == nil {
		t.Error("Engine is nil")
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
