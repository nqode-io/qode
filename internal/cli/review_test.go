//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunReview_HappyPath_Code(t *testing.T) {
	// force=true bypasses the empty-diff check; no VCS operations needed.
	_ = setupTestRootWithConfig(t, testYAMLWithStack)

	var buf bytes.Buffer
	err := runReview(context.Background(), &buf, io.Discard, "code", false, true)
	if err != nil {
		t.Fatalf("runReview code: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty review prompt output")
	}
}

func TestRunReview_StrictEmptyDiff_Code(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(), io.Discard, io.Discard, "code", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !errors.Is(err, ErrNoChanges) {
		t.Errorf("expected ErrNoChanges, got: %v", err)
	}
}

func TestRunReview_StrictEmptyDiff_Security(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(), io.Discard, io.Discard, "security", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !errors.Is(err, ErrNoChanges) {
		t.Errorf("expected ErrNoChanges, got: %v", err)
	}
}

func TestRunReview_NonStrict_EmptyDiff_ReturnsNil(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLWithStack), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview(context.Background(), io.Discard, io.Discard, "code", false, false)
	if err != nil {
		t.Errorf("expected nil error in non-strict mode with empty diff, got: %v", err)
	}
}

func TestRunReview_Force_EmptyDiff_Proceeds(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// force=true bypasses the strict diff-empty check.
	err := runReview(context.Background(), io.Discard, io.Discard, "code", false, true)
	if err != nil && errors.Is(err, ErrNoChanges) {
		t.Errorf("--force should bypass diff-empty check, got: %v", err)
	}
}
