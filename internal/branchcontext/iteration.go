package branchcontext

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/scoring"
)

// NextIteration returns the next iteration number based on existing iterations.
func (c *Context) NextIteration() int {
	return maxIterationNumber(c.Iterations) + 1
}

// SaveIterationResult writes a scored iteration file and updates the canonical
// refined-analysis.md. Use this when the score is already known from judge output.
func SaveIterationResult(root, branch string, iteration int, analysisText string, result scoring.Result) error {
	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
	if err := iokit.EnsureDir(branchDir); err != nil {
		return err
	}

	iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := iokit.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return err
	}

	latestFile := filepath.Join(branchDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	return iokit.WriteFile(latestFile, []byte(header+analysisText), 0644)
}

// ParseAndSaveIteration extracts a score from analysis text and saves iteration files.
func ParseAndSaveIteration(root, branch string, iteration int, analysisText string, cfg *config.Config) (scoring.Result, error) {
	rubric := scoring.BuildRubric(scoring.RubricRefine, cfg)
	result := scoring.ParseScore(analysisText, rubric)
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		result.TargetScore = cfg.Scoring.TargetScore
	}

	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
	if err := iokit.EnsureDir(branchDir); err != nil {
		return result, fmt.Errorf("create branch directory %q: %w", branchDir, err)
	}

	// Save numbered iteration file.
	iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := iokit.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return result, fmt.Errorf("write iteration file %q: %w", iterFile, err)
	}

	// Always update the canonical "latest" file.
	latestFile := filepath.Join(branchDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	if err := iokit.WriteFile(latestFile, []byte(header+analysisText), 0644); err != nil {
		return result, fmt.Errorf("write canonical analysis file %q: %w", latestFile, err)
	}

	return result, nil
}

func buildAnalysisHeader(iteration int, result scoring.Result) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<!-- qode:iteration=%d score=%d/%d -->\n\n", iteration, result.TotalScore, result.MaxScore)
	return sb.String()
}
