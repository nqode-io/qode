//go:build !integration

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStart_HappyPath(t *testing.T) {
	root := setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")

	// Write spec.md and analysis so guards pass.
	writeBranchFile(t, root, "test-branch", "refined-analysis.md",
		"<!-- qode:iteration=1 score=25/25 -->\n# Analysis\nContent.")
	writeBranchFile(t, root, "test-branch", "spec.md", "# Spec\nImplementation spec.")

	var buf bytes.Buffer
	err := runStart(&buf, io.Discard, false, true) // force=true to bypass guards
	if err != nil {
		t.Fatalf("runStart: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty start prompt output")
	}
}

func TestRunStart_NoSpec_HardError(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")

	err := runStart(io.Discard, io.Discard, false, true) // force bypasses guard, but spec still missing
	if err != ErrNoSpec {
		t.Errorf("expected ErrNoSpec, got: %v", err)
	}
}

func TestRunStart_GuardBlocked_NoSpec_Strict(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	cmd := newStartCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when spec is missing in strict mode")
	}
	if !strings.Contains(err.Error(), "spec.md") {
		t.Errorf("error should mention spec.md, got: %v", err)
	}
}

func TestRunStart_GuardBlocked_NonStrict(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	var buf bytes.Buffer
	cmd := newStartCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	runErr := cmd.Execute()

	if runErr != nil {
		t.Fatalf("expected nil error in non-strict mode, got: %v", runErr)
	}
	if !strings.Contains(buf.String(), "STOP") {
		t.Errorf("expected STOP instruction on stdout, got: %q", buf.String())
	}
}

func TestRunStart_Force_SkipsGuard_HardErrors_NoSpec(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// No spec.md — guard is bypassed by --force, but the hard error still fires.
	cmd := newStartCmd()
	cmd.SetArgs([]string{"--force"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected hard error for absent spec.md even with --force")
	}
	if !errors.Is(err, ErrNoSpec) {
		t.Errorf("expected ErrNoSpec, got: %v", err)
	}
}
