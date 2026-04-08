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

func TestRunPlanRefine_HappyPath(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")

	var buf bytes.Buffer
	err := runPlanRefine(&buf, io.Discard, "", false)
	if err != nil {
		t.Fatalf("runPlanRefine: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty refine prompt output")
	}
}

func TestRunPlanRefine_ToFile(t *testing.T) {
	setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")

	var errBuf bytes.Buffer
	err := runPlanRefine(io.Discard, &errBuf, "", true)
	if err != nil {
		t.Fatalf("runPlanRefine --to-file: %v", err)
	}
	if !strings.Contains(errBuf.String(), "worker prompt saved") {
		t.Errorf("expected save confirmation on stderr, got: %q", errBuf.String())
	}
}

func TestRunPlanJudge_NoAnalysis(t *testing.T) {
	_ = setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")

	err := runPlanJudge(io.Discard, io.Discard, false)
	if err != ErrNoAnalysis {
		t.Errorf("expected ErrNoAnalysis, got: %v", err)
	}
}

func TestRunPlanJudge_HappyPath(t *testing.T) {
	root := setupTestRootWithConfig(t, "test-branch", "project:\n  name: test\n  stack: go\n")
	writeBranchFile(t, root, "test-branch", "refined-analysis.md",
		"<!-- qode:iteration=1 -->\n# Analysis\nContent.")

	var buf bytes.Buffer
	err := runPlanJudge(&buf, io.Discard, false)
	if err != nil {
		t.Fatalf("runPlanJudge: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty judge prompt output")
	}
}

func TestRunPlanSpec_GuardBlocked_NoAnalysis_Strict(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	// Write a qode.yaml with strict: true
	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runPlanSpec(io.Discard, io.Discard, false, false)
	if err == nil {
		t.Fatal("expected error for missing analysis in strict mode")
	}
	if !strings.Contains(err.Error(), "refined-analysis.md") {
		t.Errorf("error message should mention refined-analysis.md, got: %v", err)
	}
}

func TestRunPlanSpec_GuardBlocked_NonStrict(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := ""
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	var buf bytes.Buffer
	runErr := runPlanSpec(&buf, io.Discard, false, false)

	if runErr != nil {
		t.Fatalf("expected nil error in non-strict mode, got: %v", runErr)
	}
	if !strings.Contains(buf.String(), "STOP") {
		t.Errorf("expected STOP instruction on stdout, got: %q", buf.String())
	}
}

func TestRunPlanSpec_GuardBlocked_Unscored(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write analysis without a score header — file present but unscored.
	writeBranchFile(t, root, "test-branch", "refined-analysis.md",
		"<!-- qode:iteration=1 -->\n# Analysis\nContent.")

	err := runPlanSpec(io.Discard, io.Discard, false, false)
	if err == nil {
		t.Fatal("expected error for unscored analysis in strict mode")
	}
	if !strings.Contains(err.Error(), "qode-plan-judge") {
		t.Errorf("error should mention /qode-plan-judge, got: %v", err)
	}
}

func TestRunPlanSpec_Force_SkipsGuard(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// No refined-analysis.md — guard would block, but force bypasses it.
	// The hard-error ("no refined analysis") fires instead of the guard error.
	err := runPlanSpec(io.Discard, io.Discard, false, true)
	if err == nil {
		t.Fatal("expected hard error for missing analysis file")
	}
	// Must be the hard-error, not the guard message.
	if strings.Contains(err.Error(), "Run /qode-plan-refine") {
		t.Errorf("force should bypass guard, but got guard message: %v", err)
	}
}

func TestRunPlanSpec_Pass(t *testing.T) {
	root := setupTestRoot(t, "test-branch")

	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write a scored refined-analysis.md that meets the threshold.
	writeBranchFile(t, root, "test-branch", "refined-analysis.md",
		"<!-- qode:iteration=1 score=25/25 -->\n# Analysis\nContent.")

	var buf bytes.Buffer
	runErr := runPlanSpec(&buf, io.Discard, false, false)

	if runErr != nil {
		t.Fatalf("expected nil error for passing score, got: %v", runErr)
	}
	output := buf.String()
	if strings.Contains(output, "STOP") {
		t.Errorf("guard should not emit STOP for a passing score, got: %q", output)
	}
	// Templates are embedded — the rendered spec prompt must contain expected content.
	if !strings.Contains(output, "Technical Specification") {
		t.Errorf("expected spec prompt on stdout, got: %q", output)
	}
}
