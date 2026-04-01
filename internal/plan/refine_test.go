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

func TestBuildJudgePrompt_ReferencesRefinedAnalysis(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	branchDir := filepath.Join(root, ".qode", "branches", "test-branch")
	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: branchDir,
	}

	got, err := BuildJudgePrompt(engine, &config.Config{}, ctx)
	if err != nil {
		t.Fatalf("BuildJudgePrompt: %v", err)
	}
	if !strings.Contains(got, branchDir) {
		t.Errorf("judge prompt must reference branch dir %q, got:\n%s", branchDir, got)
	}
	if !strings.Contains(got, "refined-analysis.md") {
		t.Error("judge prompt must reference refined-analysis.md")
	}
}

func TestBuildJudgePrompt_CustomRubric(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			Rubrics: map[string]config.RubricConfig{
				"refine": {
					Dimensions: []config.DimensionConfig{
						{Name: "Domain Fit", Weight: 5, Description: "Fits domain requirements"},
						{Name: "Data Clarity", Weight: 5, Description: "Data model is clear"},
					},
				},
			},
		},
	}

	branchDir := filepath.Join(root, ".qode", "branches", "test-branch")
	ctx := &context.Context{Branch: "test-branch", ContextDir: branchDir}

	got, err := BuildJudgePrompt(engine, cfg, ctx)
	if err != nil {
		t.Fatalf("BuildJudgePrompt: %v", err)
	}
	if !strings.Contains(got, "Domain Fit") {
		t.Error("judge prompt must contain custom dimension 'Domain Fit'")
	}
	if !strings.Contains(got, "Data Clarity") {
		t.Error("judge prompt must contain custom dimension 'Data Clarity'")
	}
	if strings.Contains(got, "Problem Understanding") {
		t.Error("judge prompt must not contain default dimension 'Problem Understanding' when overridden")
	}
	if !strings.Contains(got, "= 10 maximum") {
		t.Error("judge prompt must show total '= 10 maximum' for two 5-weight dimensions")
	}
	// Ensure no raw Go template action syntax leaked through
	if strings.Contains(got, ".Rubric.Total") {
		t.Error("judge prompt must not contain unrendered '.Rubric.Total'")
	}
}

func TestParseIterationFromOutput_TargetScoreOverride(t *testing.T) {
	root, branchDir := setupRefineDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	analysisText := "<!-- qode:iteration=1 -->\n\n**Total Score:** 18/25\n\nsome analysis"
	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			TargetScore: 20,
		},
	}

	result, err := ParseIterationFromOutput(root, "test-branch", 1, analysisText, cfg)
	if err != nil {
		t.Fatalf("ParseIterationFromOutput: %v", err)
	}
	if result.TargetScore != 20 {
		t.Errorf("expected TargetScore 20 from cfg override, got %d", result.TargetScore)
	}
	if result.TotalScore != 18 {
		t.Errorf("expected TotalScore 18, got %d", result.TotalScore)
	}
}

func TestParseIterationFromOutput_TargetScoreDefaultsToRubricTotal(t *testing.T) {
	root, branchDir := setupRefineDir(t)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	analysisText := "<!-- qode:iteration=1 -->\n\n**Total Score:** 22/25\n\nsome analysis"
	// TargetScore == 0 → should default to rubric.Total() == 25
	cfg := &config.Config{}

	result, err := ParseIterationFromOutput(root, "test-branch", 1, analysisText, cfg)
	if err != nil {
		t.Fatalf("ParseIterationFromOutput: %v", err)
	}
	if result.TargetScore != 25 {
		t.Errorf("expected TargetScore 25 (rubric total), got %d", result.TargetScore)
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

func TestBuildRefinePromptWithOutput_ContainsProjectName(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "myproject")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
	}

	out, err := BuildRefinePromptWithOutput(engine, &config.Config{}, ctx, "", 1, "")
	if err != nil {
		t.Fatalf("BuildRefinePromptWithOutput: %v", err)
	}

	if !strings.Contains(out.WorkerPrompt, "myproject") {
		t.Errorf("prompt must contain project name %q derived from root dir, got:\n%s", "myproject", out.WorkerPrompt)
	}
}
