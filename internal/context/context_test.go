package context

import (
	"os"
	"path/filepath"
	"testing"
)

// setupBranchDir creates a temp dir and the full .qode/branches/<branch> tree,
// returning the branch directory path.
func setupBranchDir(t *testing.T) (root, branchDir string) {
	t.Helper()
	root = t.TempDir()
	branchDir = filepath.Join(root, ".qode", "branches", "test-branch")
	if err := os.MkdirAll(filepath.Join(branchDir, "context"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return root, branchDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

// TestParseIterationFromAnalysis_HeaderPresent verifies happy-path parsing.
func TestParseIterationFromAnalysis_HeaderPresent(t *testing.T) {
	_, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"),
		"<!-- qode:iteration=2 score=23/25 -->\n# Analysis\n...")

	n, score, ok := parseIterationFromAnalysis(branchDir)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if n != 2 {
		t.Errorf("iteration: want 2, got %d", n)
	}
	if score != 23 {
		t.Errorf("score: want 23, got %d", score)
	}
}

// TestParseIterationFromAnalysis_MissingFile returns false gracefully.
func TestParseIterationFromAnalysis_MissingFile(t *testing.T) {
	_, branchDir := setupBranchDir(t)

	_, _, ok := parseIterationFromAnalysis(branchDir)
	if ok {
		t.Fatal("expected ok=false for missing file")
	}
}

// TestParseIterationFromAnalysis_NoHeader returns false when header is absent.
func TestParseIterationFromAnalysis_NoHeader(t *testing.T) {
	_, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"), "# Analysis\nNo header here.\n")

	_, _, ok := parseIterationFromAnalysis(branchDir)
	if ok {
		t.Fatal("expected ok=false for missing header")
	}
}

// TestParseIterationFromAnalysis_CorruptedHeader returns false on bad values.
func TestParseIterationFromAnalysis_CorruptedHeader(t *testing.T) {
	_, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"),
		"<!-- qode:iteration=abc score=x/25 -->\n")

	_, _, ok := parseIterationFromAnalysis(branchDir)
	if ok {
		t.Fatal("expected ok=false for corrupted header")
	}
}

// TestMaxIterationNumber covers empty and non-empty slices.
func TestMaxIterationNumber(t *testing.T) {
	if got := maxIterationNumber(nil); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
	its := []Iteration{{Number: 1, Score: 20}, {Number: 3, Score: 22}, {Number: 2, Score: 21}}
	if got := maxIterationNumber(its); got != 3 {
		t.Errorf("want 3, got %d", got)
	}
}

// TestLoad_OnlyAnalysisMdHeader — no numbered files; header is the only source.
func TestLoad_OnlyAnalysisMdHeader(t *testing.T) {
	root, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"),
		"<!-- qode:iteration=1 score=22/25 -->\n# Analysis\n")

	ctx, err := Load(root, "test-branch")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ctx.Iterations) != 1 {
		t.Fatalf("want 1 iteration, got %d", len(ctx.Iterations))
	}
	if ctx.Iterations[0].Number != 1 || ctx.Iterations[0].Score != 22 {
		t.Errorf("want {1,22}, got %+v", ctx.Iterations[0])
	}
}

// TestLoad_NumberedFilesMatchHeader — header matches glob; no duplicate.
func TestLoad_NumberedFilesMatchHeader(t *testing.T) {
	root, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis-1-score-22.md"), "content")
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"),
		"<!-- qode:iteration=1 score=22/25 -->\n")

	ctx, err := Load(root, "test-branch")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ctx.Iterations) != 1 {
		t.Errorf("want 1 iteration (no duplicate), got %d", len(ctx.Iterations))
	}
}

// TestLoad_HeaderNewerThanGlob — header has higher iteration than any numbered file.
func TestLoad_HeaderNewerThanGlob(t *testing.T) {
	root, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis-1-score-22.md"), "content")
	writeFile(t, filepath.Join(branchDir, "refined-analysis.md"),
		"<!-- qode:iteration=2 score=23/25 -->\n")

	ctx, err := Load(root, "test-branch")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ctx.Iterations) != 2 {
		t.Fatalf("want 2 iterations, got %d", len(ctx.Iterations))
	}
	if maxIterationNumber(ctx.Iterations) != 2 {
		t.Errorf("max iteration should be 2, got %d", maxIterationNumber(ctx.Iterations))
	}
}

// TestLoad_NoAnalysisMd — graceful fallback when file is absent.
func TestLoad_NoAnalysisMd(t *testing.T) {
	root, branchDir := setupBranchDir(t)
	writeFile(t, filepath.Join(branchDir, "refined-analysis-1-score-20.md"), "content")

	ctx, err := Load(root, "test-branch")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ctx.Iterations) != 1 {
		t.Errorf("want 1 iteration from glob, got %d", len(ctx.Iterations))
	}
}
