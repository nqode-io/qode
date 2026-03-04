package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scoring"
)

// RefineOutput holds both the worker prompt and the judge prompt for one iteration.
type RefineOutput struct {
	WorkerPrompt string
	JudgePrompt  string
	Iteration    int
}

// BuildRefinePrompt generates the worker refinement prompt.
// If cfg.Scoring.TwoPass is true, it also generates the judge prompt.
func BuildRefinePrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, ticketURL string, iteration int) (*RefineOutput, error) {
	return BuildRefinePromptWithOutput(engine, cfg, ctx, ticketURL, iteration, "")
}

// BuildRefinePromptWithOutput generates the worker refinement prompt with an
// optional output path so the AI writes its analysis directly to that file.
func BuildRefinePromptWithOutput(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, ticketURL string, iteration int, outputPath string) (*RefineOutput, error) {
	if iteration == 0 {
		iteration = len(ctx.Iterations) + 1
	}

	data := prompt.TemplateData{
		Project:    cfg.Project,
		Layers:     cfg.Layers(),
		Branch:     ctx.Branch,
		Ticket:     ctx.Ticket,
		Notes:      ctx.Notes,
		Analysis:   ctx.RefinedAnalysis,
		OutputPath: outputPath,
	}
	for _, e := range ctx.Extra {
		data.Extra += e + "\n\n"
	}

	workerPrompt, err := engine.Render("refine/base", data)
	if err != nil {
		return nil, err
	}

	out := &RefineOutput{
		WorkerPrompt: workerPrompt,
		Iteration:    iteration,
	}

	if cfg.Scoring.TwoPass {
		scoreEngine := scoring.NewEngine(engine, cfg)
		judgePrompt, err := scoreEngine.BuildJudgePrompt(
			"[Worker output will be pasted here by the user after running the worker prompt above]",
			scoring.RefineRubric,
		)
		if err != nil {
			return nil, fmt.Errorf("building judge prompt: %w", err)
		}
		out.JudgePrompt = judgePrompt
	}

	return out, nil
}

// BuildSpecPrompt generates the spec creation prompt.
func BuildSpecPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context) (string, error) {
	return BuildSpecPromptWithOutput(engine, cfg, ctx, "")
}

// BuildSpecPromptWithOutput generates the spec creation prompt with an optional
// output path so the AI writes the spec directly to that file.
func BuildSpecPromptWithOutput(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:    cfg.Project,
		Layers:     cfg.Layers(),
		Branch:     ctx.Branch,
		Analysis:   ctx.RefinedAnalysis,
		OutputPath: outputPath,
	}
	return engine.Render("spec/base", data)
}

// BuildStartPrompt generates the implementation kickoff prompt.
func BuildStartPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, kb string) (string, error) {
	data := prompt.TemplateData{
		Project: cfg.Project,
		Layers:  cfg.Layers(),
		Branch:  ctx.Branch,
		Spec:    ctx.Spec,
		KB:      kb,
	}
	return engine.Render("start/base", data)
}

// SaveIterationFiles writes worker and judge prompts to the branch context dir
// and returns the path of the worker prompt file.
func SaveIterationFiles(root, branch string, out *RefineOutput) (workerPath, judgePath string, err error) {
	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return "", "", err
	}

	// Primary "current" prompt — always overwritten.
	workerPath = filepath.Join(branchDir, ".refine-prompt.md")
	if err := os.WriteFile(workerPath, []byte(out.WorkerPrompt), 0644); err != nil {
		return "", "", err
	}

	if out.JudgePrompt != "" {
		judgePath = filepath.Join(branchDir, ".refine-judge-prompt.md")
		header := fmt.Sprintf("# Judge Scoring — Iteration %d\n\n"+
			"**Instructions:** Paste the worker's analysis below the separator, then run this prompt.\n\n"+
			"---\n\n", out.Iteration)
		if err := os.WriteFile(judgePath, []byte(header+out.JudgePrompt), 0644); err != nil {
			return workerPath, "", err
		}
	}

	return workerPath, judgePath, nil
}

// SaveIterationResult writes iteration files using a pre-computed score result.
// Use this instead of ParseIterationFromOutput when the score is known from
// judge output rather than parsed from the analysis text.
func SaveIterationResult(root, branch string, iteration int, analysisText string, result scoring.Result) error {
	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return err
	}

	iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := os.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return err
	}

	latestFile := filepath.Join(branchDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	return os.WriteFile(latestFile, []byte(header+analysisText), 0644)
}

// ParseIterationFromOutput tries to extract a score and save the analysis file.
func ParseIterationFromOutput(root, branch string, iteration int, analysisText string) (scoring.Result, error) {
	result := scoring.ParseScore(analysisText, scoring.RefineRubric)
	result.TargetScore = 25

	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)

	// Save numbered iteration file.
	iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	_ = os.WriteFile(iterFile, []byte(analysisText), 0644)

	// Always update the canonical "latest" file.
	latestFile := filepath.Join(branchDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	_ = os.WriteFile(latestFile, []byte(header+analysisText), 0644)

	return result, nil
}

func buildAnalysisHeader(iteration int, result scoring.Result) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<!-- qode:iteration=%d score=%d/%d -->\n\n", iteration, result.TotalScore, result.MaxScore)
	return sb.String()
}
