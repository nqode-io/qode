//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunReview_HappyPath_Code(t *testing.T) {
	root := setupTestRootWithConfig(t, "test-branch", testYAMLWithStack)

	// Create a change so diff is non-empty.
	if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := exec.Command("git", "-C", root, "add", "app.go").Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := exec.Command("git", "-C", root, "commit", "-m", "add app").Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	var buf bytes.Buffer
	err := runReview(context.Background(),&buf, io.Discard, "code", false, true) // force=true
	if err != nil {
		t.Fatalf("runReview code: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty review prompt output")
	}
}

func TestRunReview_StrictEmptyDiff_Code(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := testYAMLStrictMode
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(),io.Discard, io.Discard, "code", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !errors.Is(err, ErrNoChanges) {
		t.Errorf("expected ErrNoChanges, got: %v", err)
	}
}

func TestRunReview_StrictEmptyDiff_Security(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := testYAMLStrictMode
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(),io.Discard, io.Discard, "security", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !errors.Is(err, ErrNoChanges) {
		t.Errorf("expected ErrNoChanges, got: %v", err)
	}
}

func TestRunReview_NonStrict_EmptyDiff_ReturnsNil(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := testYAMLWithStack
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(),io.Discard, io.Discard, "code", false, false)
	if err != nil {
		t.Errorf("expected nil error in non-strict mode with empty diff, got: %v", err)
	}
}

func TestRunReview_Force_EmptyDiff_Proceeds(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := testYAMLStrictMode
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// force=true bypasses the strict diff-empty check.
	// It may fail later (context load etc.) but must not fail on "no changes".
	err := runReview(context.Background(),io.Discard, io.Discard, "code", false, true)
	if err != nil && errors.Is(err, ErrNoChanges) {
		t.Errorf("--force should bypass diff-empty check, got: %v", err)
	}
}
