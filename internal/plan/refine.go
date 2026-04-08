// Package plan builds refine, spec, and start prompts and manages iteration file I/O.
package plan

import (
	"path/filepath"

	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scoring"
)

// RefineOutput holds the worker prompt for one iteration.
type RefineOutput struct {
	WorkerPrompt string
	Iteration    int
}

// BuildRefinePrompt generates the worker refinement prompt.
func BuildRefinePrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, ticketURL string, iteration int) (*RefineOutput, error) {
	return BuildRefinePromptWithOutput(engine, cfg, ctx, ticketURL, iteration, "")
}

// BuildRefinePromptWithOutput generates the worker refinement prompt with an
// optional output path so the AI writes its analysis directly to that file.
func BuildRefinePromptWithOutput(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, ticketURL string, iteration int, outputPath string) (*RefineOutput, error) {
	if iteration == 0 {
		iteration = ctx.NextIteration()
	}

	var extra string
	for _, e := range ctx.Extra {
		extra += e + "\n\n"
	}
	data := prompt.NewTemplateData(engine.ProjectName(), ctx.Branch).
		WithOutputPath(outputPath).
		WithBranchDir(ctx.ContextDir).
		WithExtra(extra).
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
func BuildSpecPrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context) (string, error) {
	return BuildSpecPromptWithOutput(engine, cfg, ctx, "")
}

// BuildSpecPromptWithOutput generates the spec creation prompt with an optional
// output path so the AI writes the spec directly to that file.
func BuildSpecPromptWithOutput(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, outputPath string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName(), ctx.Branch).
		WithOutputPath(outputPath).
		WithBranchDir(ctx.ContextDir).
		Build()
	return engine.Render("spec/base", data)
}

// BuildStartPrompt generates the implementation kickoff prompt.
func BuildStartPrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, kb string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName(), ctx.Branch).
		WithKB(kb).
		WithBranchDir(ctx.ContextDir).
		Build()
	return engine.Render("start/base", data)
}

// SaveIterationFiles writes the worker prompt to the branch context dir
// and returns its path.
func SaveIterationFiles(root, branch string, out *RefineOutput) (workerPath string, err error) {
	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
	if err := iokit.EnsureDir(branchDir); err != nil {
		return "", err
	}

	// Primary "current" prompt — always overwritten.
	workerPath = filepath.Join(branchDir, ".refine-prompt.md")
	if err := iokit.WriteFile(workerPath, []byte(out.WorkerPrompt), 0644); err != nil {
		return "", err
	}

	return workerPath, nil
}

// BuildJudgePrompt generates the judge scoring prompt.
// The template references refined-analysis.md by path; no file read is performed here.
func BuildJudgePrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context) (string, error) {
	rubric := scoring.BuildRubric(scoring.RubricRefine, cfg)
	targetScore := rubric.Total()
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		targetScore = cfg.Scoring.TargetScore
	}
	data := prompt.NewTemplateData("", "").
		WithBranchDir(ctx.ContextDir).
		WithRubric(rubric).
		WithTargetScore(targetScore).
		Build()
	return engine.Render("scoring/judge_refine", data)
}

