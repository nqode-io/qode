package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/git"
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
		iteration = len(ctx.Iterations) + 1
	}

	data := prompt.TemplateData{
		Project:    prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:     ctx.Branch,
		OutputPath: outputPath,
		BranchDir:  ctx.ContextDir,
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

	return out, nil
}

// BuildSpecPrompt generates the spec creation prompt.
func BuildSpecPrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context) (string, error) {
	return BuildSpecPromptWithOutput(engine, cfg, ctx, "")
}

// BuildSpecPromptWithOutput generates the spec creation prompt with an optional
// output path so the AI writes the spec directly to that file.
func BuildSpecPromptWithOutput(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, outputPath string) (string, error) {
	data := prompt.TemplateData{
		Project:    prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:     ctx.Branch,
		OutputPath: outputPath,
		BranchDir:  ctx.ContextDir,
	}
	return engine.Render("spec/base", data)
}

// BuildStartPrompt generates the implementation kickoff prompt.
func BuildStartPrompt(engine prompt.Renderer, cfg *config.Config, ctx *branchcontext.Context, kb string) (string, error) {
	data := prompt.TemplateData{
		Project:   prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:    ctx.Branch,
		KB:        kb,
		BranchDir: ctx.ContextDir,
	}
	return engine.Render("start/base", data)
}

// SaveIterationFiles writes the worker prompt to the branch context dir
// and returns its path.
func SaveIterationFiles(root, branch string, out *RefineOutput) (workerPath string, err error) {
	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return "", err
	}

	// Primary "current" prompt — always overwritten.
	workerPath = filepath.Join(branchDir, ".refine-prompt.md")
	if err := os.WriteFile(workerPath, []byte(out.WorkerPrompt), 0644); err != nil {
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
	data := prompt.TemplateData{
		BranchDir:   ctx.ContextDir,
		Rubric:      rubric,
		TargetScore: targetScore,
	}
	return engine.Render("scoring/judge_refine", data)
}

// SaveIterationResult writes iteration files using a pre-computed score result.
// Use this instead of ParseIterationFromOutput when the score is known from
// judge output rather than parsed from the analysis text.
func SaveIterationResult(root, branch string, iteration int, analysisText string, result scoring.Result) error {
	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
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
func ParseIterationFromOutput(root, branch string, iteration int, analysisText string, cfg *config.Config) (scoring.Result, error) {
	rubric := scoring.BuildRubric(scoring.RubricRefine, cfg)
	result := scoring.ParseScore(analysisText, rubric)
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		result.TargetScore = cfg.Scoring.TargetScore
	}

	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return result, fmt.Errorf("create branch directory %q: %w", branchDir, err)
	}

	// Save numbered iteration file.
	iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := os.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return result, fmt.Errorf("write iteration file %q: %w", iterFile, err)
	}

	// Always update the canonical "latest" file.
	latestFile := filepath.Join(branchDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	if err := os.WriteFile(latestFile, []byte(header+analysisText), 0644); err != nil {
		return result, fmt.Errorf("write canonical analysis file %q: %w", latestFile, err)
	}

	return result, nil
}

func buildAnalysisHeader(iteration int, result scoring.Result) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<!-- qode:iteration=%d score=%d/%d -->\n\n", iteration, result.TotalScore, result.MaxScore)
	return sb.String()
}
