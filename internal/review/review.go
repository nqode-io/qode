package review

import (
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scoring"
)

// BuildCodePrompt generates the code review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildCodePrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:      prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:       ctx.Branch,
		OutputPath:   outputPath,
		BranchDir:    ctx.ContextDir,
		Rubric:       scoring.BuildRubric(scoring.RubricReview, cfg),
		MinPassScore: cfg.Review.MinCodeScore,
	}
	return engine.Render("review/code", data)
}

// BuildSecurityPrompt generates the security review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildSecurityPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:      prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:       ctx.Branch,
		OutputPath:   outputPath,
		BranchDir:    ctx.ContextDir,
		Rubric:       scoring.BuildRubric(scoring.RubricSecurity, cfg),
		MinPassScore: cfg.Review.MinSecurityScore,
	}
	return engine.Render("review/security", data)
}
