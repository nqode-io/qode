package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/scoring"
)

func setupRefineDir(t *testing.T) (root, branchDir string) {
	t.Helper()
	root = t.TempDir()
	branchDir = filepath.Join(root, ".qode", "branches", "test-branch")
	return root, branchDir
}

func TestSaveIterationResult_WritesCorrectScore(t *testing.T) {
	root, branchDir := setupRefineDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	result := scoring.Result{TotalScore: 22, MaxScore: 25}
	if err := SaveIterationResult(root, "test-branch", 1, "analysis content", result); err != nil {
		t.Fatalf("SaveIterationResult: %v", err)
	}

	iterFile := filepath.Join(branchDir, "refined-analysis-1-score-22.md")
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("expected %s to exist: %v", iterFile, err)
	}

	data, err := os.ReadFile(filepath.Join(branchDir, "refined-analysis.md"))
	if err != nil {
		t.Fatalf("ReadFile refined-analysis.md: %v", err)
	}
	if !strings.Contains(string(data), "score=22/25") {
		t.Errorf("refined-analysis.md missing score=22/25, got: %s", string(data))
	}
}

func TestSaveIterationResult_ZeroScore(t *testing.T) {
	root, branchDir := setupRefineDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	result := scoring.Result{TotalScore: 0, MaxScore: 25}
	if err := SaveIterationResult(root, "test-branch", 1, "analysis content", result); err != nil {
		t.Fatalf("SaveIterationResult: %v", err)
	}

	iterFile := filepath.Join(branchDir, "refined-analysis-1-score-0.md")
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("expected %s to exist: %v", iterFile, err)
	}

	data, err := os.ReadFile(filepath.Join(branchDir, "refined-analysis.md"))
	if err != nil {
		t.Fatalf("ReadFile refined-analysis.md: %v", err)
	}
	if !strings.Contains(string(data), "score=0/25") {
		t.Errorf("refined-analysis.md missing score=0/25, got: %s", string(data))
	}
}

func TestSaveIterationResult_CreatesDir(t *testing.T) {
	root, _ := setupRefineDir(t)
	// branchDir does NOT exist yet

	result := scoring.Result{TotalScore: 18, MaxScore: 25}
	if err := SaveIterationResult(root, "test-branch", 2, "content", result); err != nil {
		t.Fatalf("SaveIterationResult should create dir: %v", err)
	}

	branchDir := filepath.Join(root, ".qode", "branches", "test-branch")
	iterFile := filepath.Join(branchDir, "refined-analysis-2-score-18.md")
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("expected %s to exist: %v", iterFile, err)
	}
}
