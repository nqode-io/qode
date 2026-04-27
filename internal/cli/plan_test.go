//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPlanRefine_HappyPath(t *testing.T) {
	_ = setupTestRootWithConfig(t, testYAMLWithStack)

	var buf bytes.Buffer
	err := runPlanRefine(context.Background(), &buf, io.Discard, "", false)
	if err != nil {
		t.Fatalf("runPlanRefine: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty refine prompt output")
	}
}

func TestRunPlanRefine_ToFile(t *testing.T) {
	root := setupTestRootWithConfig(t, testYAMLWithStack)

	var errBuf bytes.Buffer
	err := runPlanRefine(context.Background(), io.Discard, &errBuf, "", true)
	if err != nil {
		t.Fatalf("runPlanRefine --to-file: %v", err)
	}
	if !strings.Contains(errBuf.String(), "worker prompt saved") {
		t.Errorf("expected save confirmation on stderr, got: %q", errBuf.String())
	}

	// Verify the saved file has actual prompt content.
	promptPath := filepath.Join(root, ".qode", "contexts", "test-context", ".refine-prompt.md")
	data, readErr := os.ReadFile(promptPath)
	if readErr != nil {
		t.Fatalf("reading saved prompt: %v", readErr)
	}
	if len(data) == 0 {
		t.Error("saved prompt file is empty")
	}
	if !strings.Contains(string(data), "Requirements Refinement") {
		t.Errorf("saved prompt missing expected heading, got: %q", string(data)[:min(200, len(data))])
	}
}

func TestRunPlanJudge_NoAnalysis(t *testing.T) {
	_ = setupTestRootWithConfig(t, testYAMLWithStack)

	err := runPlanJudge(context.Background(), io.Discard, io.Discard, false)
	if !errors.Is(err, ErrNoAnalysis) {
		t.Errorf("expected ErrNoAnalysis, got: %v", err)
	}
}

func TestRunPlanJudge_HappyPath(t *testing.T) {
	root := setupTestRootWithConfig(t, testYAMLWithStack)
	writeContextFile(t, root, "refined-analysis.md",
		"<!-- qode:iteration=1 -->\n# Analysis\nContent.")

	var buf bytes.Buffer
	err := runPlanJudge(context.Background(), &buf, io.Discard, false)
	if err != nil {
		t.Fatalf("runPlanJudge: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty judge prompt output")
	}
}

func TestRunPlanSpec_GuardBlocked_NoAnalysis_Strict(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	// Write a qode.yaml with strict: true
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	err := runPlanSpec(context.Background(), io.Discard, io.Discard, false, false)
	if err == nil {
		t.Fatal("expected error for missing analysis in strict mode")
	}
	if !strings.Contains(err.Error(), "refined-analysis.md") {
		t.Errorf("error message should mention refined-analysis.md, got: %v", err)
	}
}

func TestRunPlanSpec_GuardBlocked_NonStrict(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLNonStrict), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	var buf bytes.Buffer
	runErr := runPlanSpec(context.Background(), &buf, io.Discard, false, false)

	if runErr != nil {
		t.Fatalf("expected nil error in non-strict mode, got: %v", runErr)
	}
	if !strings.Contains(buf.String(), "STOP") {
		t.Errorf("expected STOP instruction on stdout, got: %q", buf.String())
	}
}

func TestRunPlanSpec_GuardBlocked_Unscored(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write analysis without a score header — file present but unscored.
	writeContextFile(t, root, "refined-analysis.md",
		"<!-- qode:iteration=1 -->\n# Analysis\nContent.")

	err := runPlanSpec(context.Background(), io.Discard, io.Discard, false, false)
	if err == nil {
		t.Fatal("expected error for unscored analysis in strict mode")
	}
	if !strings.Contains(err.Error(), "qode-plan-judge") {
		t.Errorf("error should mention qode-plan-judge, got: %v", err)
	}
}

func TestRunPlanSpec_Force_SkipsGuard(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// No refined-analysis.md — guard would block, but force bypasses it.
	// The hard-error ("no refined analysis") fires instead of the guard error.
	err := runPlanSpec(context.Background(), io.Discard, io.Discard, false, true)
	if err == nil {
		t.Fatal("expected hard error for missing analysis file")
	}
	// Must be the hard-error, not the guard message.
	if strings.Contains(err.Error(), "Run the `qode-plan-refine` step") {
		t.Errorf("force should bypass guard, but got guard message: %v", err)
	}
}

func TestRunPlanSpec_Pass(t *testing.T) {
	root := setupTestRoot(t, "test-context")

	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(testYAMLStrictMode), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write a scored refined-analysis.md that meets the threshold.
	writeContextFile(t, root, "refined-analysis.md",
		"<!-- qode:iteration=1 score=25/25 -->\n# Analysis\nContent.")

	var buf bytes.Buffer
	runErr := runPlanSpec(context.Background(), &buf, io.Discard, false, false)

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
