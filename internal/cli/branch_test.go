//go:build !integration

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunBranchCreate_WritesStubFiles verifies that runBranchCreate creates the
// context directory and both stub files when they don't already exist.
func TestRunBranchCreate_WritesStubFiles(t *testing.T) {
	root := setupTestRoot(t, "main")

	var buf bytes.Buffer
	if err := runBranchCreate(&buf, "my-feature", ""); err != nil {
		t.Fatalf("runBranchCreate: %v", err)
	}

	ctxDir := filepath.Join(root, ".qode", "branches", "my-feature", "context")
	for _, name := range []string{"ticket.md", "notes.md"} {
		if _, err := os.Stat(filepath.Join(ctxDir, name)); err != nil {
			t.Errorf("expected stub file %s to exist: %v", name, err)
		}
	}
}

// TestRunBranchCreate_OutputMentionsBranchName verifies that the success message
// includes the branch name.
func TestRunBranchCreate_OutputMentionsBranchName(t *testing.T) {
	setupTestRoot(t, "main")

	var buf bytes.Buffer
	if err := runBranchCreate(&buf, "feat-login", ""); err != nil {
		t.Fatalf("runBranchCreate: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "feat-login") {
		t.Errorf("expected output to mention branch name, got: %q", out)
	}
}

// TestRunBranchCreate_InvalidBase rejects a base branch starting with '-'.
func TestRunBranchCreate_InvalidBase(t *testing.T) {
	setupTestRoot(t, "main")

	err := runBranchCreate(io.Discard, "my-feature", "-bad-base")
	if err == nil {
		t.Fatal("expected error for base starting with '-'")
	}
	if !strings.Contains(err.Error(), "must not start with '-'") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRunBranchRemove_RemovesContextDir verifies that runBranchRemove deletes
// the branch context directory when keepCtx is false.
func TestRunBranchRemove_RemovesContextDir(t *testing.T) {
	root := setupTestRoot(t, "main")

	// Pre-create the branch context directory.
	branchDir := filepath.Join(root, ".qode", "branches", "old-feature")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Write a minimal qode.yaml so config.Load succeeds.
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := runBranchRemove(io.Discard, io.Discard, "old-feature", false); err != nil {
		t.Fatalf("runBranchRemove: %v", err)
	}

	if _, err := os.Stat(branchDir); !os.IsNotExist(err) {
		t.Errorf("expected branch dir to be removed, stat: %v", err)
	}
}

// TestRunBranchRemove_KeepCtxFromFlag verifies that the --keep-branch-context
// flag prevents the context directory from being deleted.
func TestRunBranchRemove_KeepCtxFromFlag(t *testing.T) {
	root := setupTestRoot(t, "main")

	branchDir := filepath.Join(root, ".qode", "branches", "keep-me")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// keepCtx=true — directory must survive.
	if err := runBranchRemove(io.Discard, io.Discard, "keep-me", true); err != nil {
		t.Fatalf("runBranchRemove: %v", err)
	}

	if _, err := os.Stat(branchDir); err != nil {
		t.Errorf("expected branch dir to be preserved, got: %v", err)
	}
}

// TestSafeBranchDir_SlashedName exercises the full safeBranchDir path:
// sanitization + path-traversal validation. This integration path is not
// covered by git.TestSanitizeBranchName (pure sanitization) or
// branchcontext.TestLoad_SlashedBranchName (load behavior).
func TestSafeBranchDir_SlashedName(t *testing.T) {
	root := t.TempDir()
	dir, err := safeBranchDir(root, "feat/jira-123")
	if err != nil {
		t.Fatalf("safeBranchDir: %v", err)
	}

	// Must be directly under .qode/branches/ with "--" separator.
	want := filepath.Join(root, ".qode", "branches", "feat--jira-123")
	if dir != want {
		t.Errorf("got %q, want %q", dir, want)
	}

	// Must not contain the nested path segment.
	base := filepath.Join(root, ".qode", "branches")
	rel, _ := filepath.Rel(base, dir)
	if strings.Contains(rel, string(filepath.Separator)) {
		t.Errorf("path %q is nested; expected flat directory under branches/", dir)
	}
}

// TestSafeBranchDir_MultipleSlashes verifies deep slash paths collapse to a
// single flat directory name.
func TestSafeBranchDir_MultipleSlashes(t *testing.T) {
	root := t.TempDir()
	dir, err := safeBranchDir(root, "feat/jira-123/description")
	if err != nil {
		t.Fatalf("safeBranchDir: %v", err)
	}

	want := filepath.Join(root, ".qode", "branches", "feat--jira-123--description")
	if dir != want {
		t.Errorf("got %q, want %q", dir, want)
	}
}

// TestSafeBranchDir_PlainName verifies that names without slashes are unchanged.
func TestSafeBranchDir_PlainName(t *testing.T) {
	root := t.TempDir()
	dir, err := safeBranchDir(root, "my-feature")
	if err != nil {
		t.Fatalf("safeBranchDir: %v", err)
	}

	want := filepath.Join(root, ".qode", "branches", "my-feature")
	if dir != want {
		t.Errorf("got %q, want %q", dir, want)
	}
}

// TestSafeBranchDir_PathTraversal verifies that ".." components are rejected
// even after slash sanitization.
func TestSafeBranchDir_PathTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := safeBranchDir(root, "../escape"); err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

// TestSafeBranchDir_SingleDot verifies that "." is rejected because it would
// resolve to the branches root directory rather than a branch subdirectory.
func TestSafeBranchDir_SingleDot(t *testing.T) {
	root := t.TempDir()
	if _, err := safeBranchDir(root, "."); err == nil {
		t.Error("expected error for \".\", got nil")
	}
}
