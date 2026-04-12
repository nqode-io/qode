package qodecontext

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/scoring"
)

// SaveIterationResult writes a scored iteration file and updates the canonical
// refined-analysis.md. Use this when the score is already known from judge output.
func SaveIterationResult(ctx context.Context, contextDir string, iteration int, analysisText string, result scoring.Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := iokit.EnsureDir(contextDir); err != nil {
		return err
	}

	iterFile := filepath.Join(contextDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := iokit.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return err
	}

	latestFile := filepath.Join(contextDir, "refined-analysis.md")
	header := buildAnalysisHeader(iteration, result)
	return iokit.WriteFile(latestFile, []byte(header+analysisText), 0644)
}

// ParseAndSaveIteration extracts a score from analysis text and saves iteration files.
func ParseAndSaveIteration(ctx context.Context, contextDir string, iteration int, analysisText string, cfg *config.Config) (scoring.Result, error) {
	if err := ctx.Err(); err != nil {
		return scoring.Result{}, err
	}
	rubric := scoring.BuildRubric(scoring.RubricRefine, cfg)
	result := scoring.ParseScore(analysisText, rubric)
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		result.TargetScore = cfg.Scoring.TargetScore
	}

	if err := iokit.EnsureDir(contextDir); err != nil {
		return result, fmt.Errorf("create context directory %q: %w", contextDir, err)
	}

	iterFile := filepath.Join(contextDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", iteration, result.TotalScore))
	if err := iokit.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
		return result, fmt.Errorf("write iteration file %q: %w", iterFile, err)
	}

	latestFile := filepath.Join(contextDir, "refined-analysis.md")
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
