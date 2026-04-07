package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupPlanTestRoot initialises a real git repo so git.CurrentBranch works,
// sets flagRoot, and returns the root path.
func setupPlanTestRoot(t *testing.T, branch string) (root string) {
	t.Helper()
	root = t.TempDir()
	flagRoot = root

	gitCmds := [][]string{
		{"init", "-b", branch},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	branchDir := filepath.Join(root, ".qode", "branches", branch)
	if err := os.MkdirAll(filepath.Join(branchDir, "context"), 0755); err != nil {
		t.Fatalf("MkdirAll branch dir: %v", err)
	}

	t.Cleanup(func() { flagRoot = "" })
	return root
}

// writePlanFile writes content to a file in the branch context dir.
func writePlanFile(t *testing.T, root, branch, name, content string) {
	t.Helper()
	path := filepath.Join(root, ".qode", "branches", branch, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

// captureStdout redirects os.Stdout to a pipe for the duration of fn, then
// returns everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("w.Close: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("r.Close: %v", err)
	}
	return buf.String()
}

func TestRunPlanSpec_GuardBlocked_NoAnalysis_Strict(t *testing.T) {
	root := setupPlanTestRoot(t, "test-branch")

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
	root := setupPlanTestRoot(t, "test-branch")

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
	root := setupPlanTestRoot(t, "test-branch")

	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write analysis without a score header — file present but unscored.
	writePlanFile(t, root, "test-branch", "refined-analysis.md",
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
	root := setupPlanTestRoot(t, "test-branch")

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
	root := setupPlanTestRoot(t, "test-branch")

	cfg := "scoring:\n  strict: true\n"
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}

	// Write a scored refined-analysis.md that meets the threshold.
	writePlanFile(t, root, "test-branch", "refined-analysis.md",
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
