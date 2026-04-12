// Package plan builds refine, spec, and start prompts and manages iteration file I/O.
package plan

import (
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/qodecontext"
	"github.com/nqode/qode/internal/scoring"
)

// RefineOutput holds the worker prompt for one iteration.
type RefineOutput struct {
	WorkerPrompt string
	Iteration    int
}

// BuildRefinePrompt generates the worker refinement prompt.
func BuildRefinePrompt(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context, ticketURL string, iteration int) (*RefineOutput, error) {
	return BuildRefinePromptWithOutput(engine, cfg, ctx, ticketURL, iteration, "")
}

// BuildRefinePromptWithOutput generates the worker refinement prompt with an
// optional output path so the AI writes its analysis directly to that file.
func BuildRefinePromptWithOutput(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context, ticketURL string, iteration int, outputPath string) (*RefineOutput, error) {
	if iteration == 0 {
		iteration = ctx.NextIteration()
	}

	data := prompt.NewTemplateData(engine.ProjectName()).
		WithOutputPath(outputPath).
		Build()

	workerPrompt, err := engine.Render("refine/base", data)
	if err != nil {
		return nil, err
	}

	out := &RefineOutput{
		WorkerPrompt: workerPrompt,
		Iteration:    iteration,
	}

	return out, nil
}

// BuildSpecPrompt generates the spec creation prompt.
func BuildSpecPrompt(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context) (string, error) {
	return BuildSpecPromptWithOutput(engine, cfg, ctx, "")
}

// BuildSpecPromptWithOutput generates the spec creation prompt with an optional
// output path so the AI writes the spec directly to that file.
func BuildSpecPromptWithOutput(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context, outputPath string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName()).
		WithOutputPath(outputPath).
		Build()
	return engine.Render("spec/base", data)
}

// BuildStartPrompt generates the implementation kickoff prompt.
func BuildStartPrompt(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context, kb string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName()).
		WithKB(kb).
		Build()
	return engine.Render("start/base", data)
}

// SaveIterationFiles writes the worker prompt to the context directory
// and returns its path.
func SaveIterationFiles(contextDir string, out *RefineOutput) (workerPath string, err error) {
	if err := iokit.EnsureDir(contextDir); err != nil {
		return "", err
	}

	// Primary "current" prompt — always overwritten.
	workerPath = filepath.Join(contextDir, ".refine-prompt.md")
	if err := iokit.WriteFile(workerPath, []byte(out.WorkerPrompt), 0644); err != nil {
		return "", err
	}

	return workerPath, nil
}

// BuildJudgePrompt generates the judge scoring prompt.
// The template references refined-analysis.md by path; no file read is performed here.
func BuildJudgePrompt(engine prompt.Renderer, cfg *config.Config, ctx *qodecontext.Context) (string, error) {
	rubric := scoring.BuildRubric(scoring.RubricRefine, cfg)
	targetScore := rubric.Total()
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		targetScore = cfg.Scoring.TargetScore
	}
	data := prompt.NewTemplateData("").
		WithRubric(rubric).
		WithTargetScore(targetScore).
		Build()
	return engine.Render("scoring/judge_refine", data)
}
