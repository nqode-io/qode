package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupReviewTestRoot(t *testing.T, branch string) string {
	t.Helper()
	return setupPlanTestRoot(t, branch)
}

func TestRunReview_StrictEmptyDiff_Code(t *testing.T) {
	root := setupReviewTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview("code", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !strings.Contains(err.Error(), "no changes") {
		t.Errorf("error should mention no changes, got: %v", err)
	}
}

func TestRunReview_StrictEmptyDiff_Security(t *testing.T) {
	root := setupReviewTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview("security", false, false)
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !strings.Contains(err.Error(), "no changes") {
		t.Errorf("error should mention no changes, got: %v", err)
	}
}

func TestRunReview_NonStrict_EmptyDiff_ReturnsNil(t *testing.T) {
	root := setupReviewTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runReview("code", false, false)
	if err != nil {
		t.Errorf("expected nil error in non-strict mode with empty diff, got: %v", err)
	}
}

func TestRunReview_Force_EmptyDiff_Proceeds(t *testing.T) {
	root := setupReviewTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// force=true bypasses the strict diff-empty check.
	// It may fail later (context load etc.) but must not fail on "no changes".
	err := runReview("code", false, true)
	if err != nil && strings.Contains(err.Error(), "no changes") {
		t.Errorf("--force should bypass diff-empty check, got: %v", err)
	}
	_ = root // used above
}
