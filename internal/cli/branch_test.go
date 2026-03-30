package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeBranchDir_SlashedName verifies that a branch name containing "/"
// produces a flat sanitized directory, not nested subdirectories.
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
