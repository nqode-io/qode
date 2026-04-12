package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/qodecontext"
)

func TestBuildSpecPromptWithOutput_OmitsAnalysis(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName:     "test-context",
		ContextDir:      filepath.Join(root, ".qode", "contexts", "test-context"),
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
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
		Spec:        "spec sentinel content here",
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
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
	}

	got, err := BuildJudgePrompt(engine, &config.Config{}, ctx)
	if err != nil {
		t.Fatalf("BuildJudgePrompt: %v", err)
	}
	if !strings.Contains(got, "refined-analysis.md") {
		t.Error("judge prompt must reference refined-analysis.md")
	}
	if !strings.Contains(got, ".qode/contexts/current/") {
		t.Errorf("judge prompt must reference fixed context path, got:\n%s", got)
	}
}

func TestBuildJudgePrompt_CustomRubric(t *testing.T) {
	t.Parallel()
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

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
	}

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
	if strings.Contains(got, ".Rubric.Total") {
		t.Error("judge prompt must not contain unrendered '.Rubric.Total'")
	}
}

func TestBuildRefinePromptWithOutput_OmitsAnalysisAndTicket(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName:     "test-context",
		ContextDir:      filepath.Join(root, ".qode", "contexts", "test-context"),
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

func TestBuildStartPrompt_WithKB(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
	}

	kb := "- lessons/error-handling.md\n- lessons/naming.md"
	got, err := BuildStartPrompt(engine, &config.Config{}, ctx, kb)
	if err != nil {
		t.Fatalf("BuildStartPrompt: %v", err)
	}
	if !strings.Contains(got, "error-handling.md") {
		t.Error("prompt must contain KB file references when provided")
	}
}

func TestBuildRefinePromptWithOutput_WithOutputPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
	}

	out, err := BuildRefinePromptWithOutput(engine, &config.Config{}, ctx, "", 1, "/tmp/output.md")
	if err != nil {
		t.Fatalf("BuildRefinePromptWithOutput: %v", err)
	}
	if !strings.Contains(out.WorkerPrompt, "/tmp/output.md") {
		t.Error("prompt must contain output path when specified")
	}
}

func TestBuildRefinePromptWithOutput_AutoIteration(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
		Iterations:  []qodecontext.Iteration{{Number: 2, Score: 15}},
	}

	// iteration=0 should trigger auto-increment via ctx.NextIteration().
	out, err := BuildRefinePromptWithOutput(engine, &config.Config{}, ctx, "", 0, "")
	if err != nil {
		t.Fatalf("BuildRefinePromptWithOutput: %v", err)
	}
	if out.Iteration != 3 {
		t.Errorf("expected auto-iteration 3, got %d", out.Iteration)
	}
}

func TestBuildRefinePromptWithOutput_ContainsProjectName(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	root := filepath.Join(base, "myproject")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &qodecontext.Context{
		ContextName: "test-context",
		ContextDir:  filepath.Join(root, ".qode", "contexts", "test-context"),
	}

	out, err := BuildRefinePromptWithOutput(engine, &config.Config{}, ctx, "", 1, "")
	if err != nil {
		t.Fatalf("BuildRefinePromptWithOutput: %v", err)
	}

	if !strings.Contains(out.WorkerPrompt, "myproject") {
		t.Errorf("prompt must contain project name %q derived from root dir, got:\n%s", "myproject", out.WorkerPrompt)
	}
}
