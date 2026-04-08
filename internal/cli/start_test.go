//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupStartTestRoot(t *testing.T, branch string) string {
	t.Helper()
	return setupPlanTestRoot(t, branch)
}

func TestRunStart_GuardBlocked_NoSpec_Strict(t *testing.T) {
	root := setupStartTestRoot(t, "test-branch")

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
	root := setupStartTestRoot(t, "test-branch")

	cfg := "project:\n  name: test\n  stack: go\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	var output string
	var runErr error
	output = captureStdout(t, func() {
		cmd := newStartCmd()
		cmd.SetArgs([]string{})
		runErr = cmd.Execute()
	})

	if runErr != nil {
		t.Fatalf("expected nil error in non-strict mode, got: %v", runErr)
	}
	if !strings.Contains(output, "STOP") {
		t.Errorf("expected STOP instruction on stdout, got: %q", output)
	}
}

func TestRunStart_Force_SkipsGuard_HardErrors_NoSpec(t *testing.T) {
	root := setupStartTestRoot(t, "test-branch")

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
	if !strings.Contains(err.Error(), "no spec") {
		t.Errorf("expected hard error mentioning 'no spec', got: %v", err)
	}
}
