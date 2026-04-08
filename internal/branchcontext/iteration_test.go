package branchcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/scoring"
)

func setupIterationDir(t *testing.T) (root, branchDir string) {
	t.Helper()
	root = t.TempDir()
	branchDir = filepath.Join(root, ".qode", "branches", "test-branch")
	return root, branchDir
}

func TestSaveIterationResult_WritesCorrectScore(t *testing.T) {
	root, branchDir := setupIterationDir(t)
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
	root, branchDir := setupIterationDir(t)
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
	root, _ := setupIterationDir(t)
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

func TestParseAndSaveIteration_TargetScoreOverride(t *testing.T) {
	root, branchDir := setupIterationDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	analysisText := "<!-- qode:iteration=1 -->\n\n**Total Score:** 18/25\n\nsome analysis"
	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			TargetScore: 20,
		},
	}

	result, err := ParseAndSaveIteration(root, "test-branch", 1, analysisText, cfg)
	if err != nil {
		t.Fatalf("ParseAndSaveIteration: %v", err)
	}
	if result.TargetScore != 20 {
		t.Errorf("expected TargetScore 20 from cfg override, got %d", result.TargetScore)
	}
	if result.TotalScore != 18 {
		t.Errorf("expected TotalScore 18, got %d", result.TotalScore)
	}

	iterFile := filepath.Join(branchDir, "refined-analysis-1-score-18.md")
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("expected iteration file %s to exist: %v", iterFile, err)
	}
	if _, err := os.Stat(filepath.Join(branchDir, "refined-analysis.md")); err != nil {
		t.Errorf("expected canonical file refined-analysis.md to exist: %v", err)
	}
}

func TestParseAndSaveIteration_TargetScoreDefaultsToRubricTotal(t *testing.T) {
	root, branchDir := setupIterationDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	analysisText := "<!-- qode:iteration=1 -->\n\n**Total Score:** 22/25\n\nsome analysis"
	// TargetScore == 0 → should default to rubric.Total() == 25
	cfg := &config.Config{}

	result, err := ParseAndSaveIteration(root, "test-branch", 1, analysisText, cfg)
	if err != nil {
		t.Fatalf("ParseAndSaveIteration: %v", err)
	}
	if result.TargetScore != 25 {
		t.Errorf("expected TargetScore 25 (rubric total), got %d", result.TargetScore)
	}

	iterFile := filepath.Join(branchDir, "refined-analysis-1-score-22.md")
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("expected iteration file %s to exist: %v", iterFile, err)
	}
	if _, err := os.Stat(filepath.Join(branchDir, "refined-analysis.md")); err != nil {
		t.Errorf("expected canonical file refined-analysis.md to exist: %v", err)
	}
}

func TestNextIteration_Empty(t *testing.T) {
	ctx := &Context{}
	if got := ctx.NextIteration(); got != 1 {
		t.Errorf("NextIteration on empty = %d, want 1", got)
	}
}

func TestNextIteration_WithExisting(t *testing.T) {
	ctx := &Context{
		Iterations: []Iteration{
			{Number: 1, Score: 20},
			{Number: 3, Score: 22},
		},
	}
	if got := ctx.NextIteration(); got != 4 {
		t.Errorf("NextIteration = %d, want 4", got)
	}
}
