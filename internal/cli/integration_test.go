//go:build integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/iokit"
)

// setupOption is a functional option for setupProject.
type setupOption func(root, branchDir string, t *testing.T)

// withTicket writes a ticket.md file to the branch context directory.
func withTicket(content string) setupOption {
	return func(root, branchDir string, t *testing.T) {
		t.Helper()
		ctxDir := filepath.Join(branchDir, "context")
		if err := iokit.WriteFile(filepath.Join(ctxDir, "ticket.md"), []byte(content), 0644); err != nil {
			t.Fatalf("withTicket: %v", err)
		}
	}
}

// withRefinedAnalysis writes a refined-analysis.md to the branch directory.
func withRefinedAnalysis(content string) setupOption {
	return func(root, branchDir string, t *testing.T) {
		t.Helper()
		if err := iokit.WriteFile(filepath.Join(branchDir, "refined-analysis.md"), []byte(content), 0644); err != nil {
			t.Fatalf("withRefinedAnalysis: %v", err)
		}
	}
}

// withQodeYAML writes a qode.yaml to the project root.
func withQodeYAML(content string) setupOption {
	return func(root, branchDir string, t *testing.T) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(content), 0644); err != nil {
			t.Fatalf("withQodeYAML: %v", err)
		}
	}
}

// setupProject creates a temporary git repository with a branch and minimal
// qode.yaml, then applies any functional options. It returns the project root.
func setupProject(t *testing.T, branch string, opts ...setupOption) string {
	t.Helper()

	// Delegate git init, branch dir creation, and flagRoot to shared helper.
	root := setupTestRootWithConfig(t, branch, testYAMLNonStrict)

	// Resolve the branch directory for functional options.
	sanitized := strings.ReplaceAll(branch, "/", "--")
	branchDir := filepath.Join(root, ".qode", "branches", sanitized)

	for _, opt := range opts {
		opt(root, branchDir, t)
	}

	// Integration tests also need to reset cobra state, including child command
	// contexts — Cobra only propagates parent context when cmd.ctx == nil.
	t.Cleanup(func() {
		flagStrict = false
		rootCmd.SetArgs(nil)
		rootCmd.SetContext(nil)
		for _, cmd := range rootCmd.Commands() {
			cmd.SetContext(nil)
			for _, sub := range cmd.Commands() {
				sub.SetContext(nil)
			}
		}
	})

	return root
}

// runCommand executes a CLI command (e.g. "plan", "refine") in the context of
// the already-configured project root and captures stdout.
//
// It sets rootCmd.SetOut so cobra routes output through the buffer instead of
// os.Stdout. Each RunE closure uses cmd.OutOrStdout(), which resolves to the
// buffer set on the root command.
func runCommand(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs(args)
	runErr := rootCmd.Execute()
	rootCmd.SetOut(nil)
	return buf.String(), runErr
}

// TestIntegration_PlanRefine_DefaultRubric verifies that 'plan refine' renders
// a non-empty worker prompt containing the canonical section headings when a
// ticket is present.
func TestIntegration_PlanRefine_DefaultRubric(t *testing.T) {
	const ticket = "# Feature: Add login page\n\nUsers need a login page with email and password.\n"
	setupProject(t, "test-integration",
		withTicket(ticket),
	)

	out, err := runCommand(t, "plan", "refine")
	if err != nil {
		t.Fatalf("plan refine returned error: %v", err)
	}
	if out == "" {
		t.Fatal("plan refine produced no output")
	}

	// The refine/base template must contain these canonical headings.
	for _, section := range []string{
		"Requirements Refinement",
		"Problem Understanding",
		"Technical Analysis",
		"Risk & Edge Cases",
		"Actionable Implementation Plan",
	} {
		if !strings.Contains(out, section) {
			t.Errorf("output missing expected section %q", section)
		}
	}
}

// TestIntegration_PlanRefine_MissingTicket verifies that 'plan refine'
// succeeds (exit 0) even when no ticket.md has been written, because the
// template references the file by path and the AI reads it at runtime.
func TestIntegration_PlanRefine_MissingTicket(t *testing.T) {
	// No withTicket option — context/ticket.md is absent.
	setupProject(t, "test-no-ticket")

	out, err := runCommand(t, "plan", "refine")
	if err != nil {
		t.Fatalf("plan refine should not fail when ticket is missing, got: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty prompt even without a ticket")
	}
	// Output must still be a valid prompt template render.
	if !strings.Contains(out, "Requirements Refinement") {
		t.Errorf("expected prompt header in output, got: %q", out)
	}
}

// TestIntegration_PlanSpec_GuardBlocked verifies that 'plan spec' emits a STOP
// message when no refined-analysis.md exists in non-strict mode.
func TestIntegration_PlanSpec_GuardBlocked(t *testing.T) {
	// Non-strict mode (default minimal config) — guard emits STOP instead of error.
	setupProject(t, "test-spec-guard")

	out, err := runCommand(t, "plan", "spec")
	if err != nil {
		t.Fatalf("plan spec should not return an error in non-strict mode, got: %v", err)
	}
	if !strings.Contains(out, "STOP") {
		t.Errorf("expected STOP message when guard blocks, got: %q", out)
	}
}

// TestIntegration_PlanSpec_PassesWithAnalysis verifies that 'plan spec' renders
// a spec prompt when a scored refined-analysis.md is present.
func TestIntegration_PlanSpec_PassesWithAnalysis(t *testing.T) {
	const analysis = "<!-- qode:iteration=1 score=25/25 -->\n\n# Analysis\n\nFull analysis here.\n"
	setupProject(t, "test-spec-pass",
		withQodeYAML(testYAMLStrictMode),
		withRefinedAnalysis(analysis),
	)

	out, err := runCommand(t, "plan", "spec")
	if err != nil {
		t.Fatalf("plan spec returned error: %v", err)
	}
	if out == "" {
		t.Fatal("plan spec produced no output")
	}
	if !strings.Contains(out, "Technical Specification") {
		t.Errorf("expected spec prompt in output, got: %q", out)
	}
}

// TestIntegration_ReviewCode_NoDiff verifies that 'review code' with an empty
// diff in non-strict mode returns nil and writes nothing to stdout (because
// there is nothing to review).
func TestIntegration_ReviewCode_NoDiff(t *testing.T) {
	// Non-strict mode: empty diff yields a message to stderr, nil error, empty stdout.
	setupProject(t, "test-review-no-diff",
		withQodeYAML(testYAMLFullNonStrict),
	)

	out, err := runCommand(t, "review", "code")
	if err != nil {
		t.Fatalf("review code should not error with empty diff in non-strict mode, got: %v", err)
	}
	// In non-strict mode with empty diff the command exits cleanly after printing
	// to stderr; stdout should be empty.
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty stdout for empty-diff non-strict review, got: %q", out)
	}
}

// TestIntegration_ReviewCode_StrictNoDiff verifies that 'review code' returns
// an error in strict mode when there is no diff.
func TestIntegration_ReviewCode_StrictNoDiff(t *testing.T) {
	setupProject(t, "test-review-strict",
		withQodeYAML(testYAMLStrictMode),
	)

	_, err := runCommand(t, "review", "code")
	if err == nil {
		t.Fatal("expected error for empty diff in strict mode")
	}
	if !errors.Is(err, ErrNoChanges) {
		t.Errorf("expected ErrNoChanges, got: %v", err)
	}
}

// TestIntegration_PlanRefine_CancelledContext verifies that CLI commands
// respect context cancellation and return an error promptly.
func TestIntegration_PlanRefine_CancelledContext(t *testing.T) {
	setupProject(t, "test-cancel")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"plan", "refine"})
	rootCmd.SetContext(ctx)
	err := rootCmd.Execute()
	rootCmd.SetOut(nil)
	rootCmd.SetContext(nil)

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
