package review

import (
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/prompt"
)

// BuildCodePrompt generates the code review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildCodePrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, diff, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:    cfg.Project,
		Layers:     cfg.Layers(),
		Branch:     ctx.Branch,
		Spec:       ctx.Spec,
		Diff:       diff,
		OutputPath: outputPath,
	}
	return engine.Render("review/code", data)
}

// BuildSecurityPrompt generates the security review prompt.
// When outputPath is non-empty the rendered prompt includes file-write instructions.
func BuildSecurityPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, diff, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:    cfg.Project,
		Layers:     cfg.Layers(),
		Branch:     ctx.Branch,
		Diff:       diff,
		OutputPath: outputPath,
	}
	return engine.Render("review/security", data)
}
