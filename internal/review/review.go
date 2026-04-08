// Package review builds code-review and security-review prompts.
package review

import (
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scoring"
)

// BuildCodePrompt generates the code review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildCodePrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, outputPath string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName(), ctx.Branch).
		WithOutputPath(outputPath).
		WithBranchDir(ctx.ContextDir).
		WithRubric(scoring.BuildRubric(scoring.RubricReview, cfg)).
		WithMinPassScore(cfg.Review.MinCodeScore).
		Build()
	return engine.Render("review/code", data)
}

// BuildSecurityPrompt generates the security review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildSecurityPrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, outputPath string) (string, error) {
	data := prompt.NewTemplateData(engine.ProjectName(), ctx.Branch).
		WithOutputPath(outputPath).
		WithBranchDir(ctx.ContextDir).
		WithRubric(scoring.BuildRubric(scoring.RubricSecurity, cfg)).
		WithMinPassScore(cfg.Review.MinSecurityScore).
		Build()
	return engine.Render("review/security", data)
}
