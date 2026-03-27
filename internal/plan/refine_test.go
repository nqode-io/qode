package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/prompt"
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

func TestBuildSpecPromptWithOutput_OmitsAnalysis(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:          "test-branch",
		ContextDir:      filepath.Join(root, ".qode", "branches", "test-branch"),
		RefinedAnalysis: "full refined analysis sentinel text",
	}

	got, err := BuildSpecPromptWithOutput(engine, &config.Config{}, ctx, "")
	if err != nil {
		t.Fatalf("BuildSpecPromptWithOutput: %v", err)
	}

	if strings.Contains(got, "full refined analysis sentinel text") {
		t.Error("prompt must not inline analysis content")
	}
	if !strings.Contains(got, "refined-analysis.md") {
		t.Error("prompt must reference refined-analysis.md")
	}
}

func TestBuildStartPrompt_OmitsSpec(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
		Spec:       "spec sentinel content here",
	}

	got, err := BuildStartPrompt(engine, &config.Config{}, ctx, "")
	if err != nil {
		t.Fatalf("BuildStartPrompt: %v", err)
	}

	if strings.Contains(got, "spec sentinel content here") {
		t.Error("prompt must not inline spec content")
	}
	if !strings.Contains(got, "spec.md") {
		t.Error("prompt must reference spec.md")
	}
}

func TestBuildRefinePromptWithOutput_OmitsAnalysisAndTicket(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:          "test-branch",
		ContextDir:      filepath.Join(root, ".qode", "branches", "test-branch"),
		RefinedAnalysis: "previous iteration analysis sentinel",
		Ticket:          "ticket sentinel text",
	}

	out, err := BuildRefinePromptWithOutput(engine, &config.Config{}, ctx, "", 2, "")
	if err != nil {
		t.Fatalf("BuildRefinePromptWithOutput: %v", err)
	}

	if strings.Contains(out.WorkerPrompt, "previous iteration analysis sentinel") {
		t.Error("prompt must not inline analysis content")
	}
	if strings.Contains(out.WorkerPrompt, "ticket sentinel text") {
		t.Error("prompt must not inline ticket content")
	}
}
