package qodecontext

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/scoring"
)

func TestSaveIterationResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		iteration int
		text      string
		result    scoring.Result
		wantIter  string
		wantHead  string
	}{
		{
			name:      "writes scored iteration and canonical file",
			iteration: 2,
			text:      "analysis body",
			result:    scoring.Result{TotalScore: 20, MaxScore: 25},
			wantIter:  "refined-analysis-2-score-20.md",
			wantHead:  "<!-- qode:iteration=2 score=20/25 -->",
		},
		{
			name:      "zero score",
			iteration: 1,
			text:      "empty analysis",
			result:    scoring.Result{TotalScore: 0, MaxScore: 25},
			wantIter:  "refined-analysis-1-score-0.md",
			wantHead:  "<!-- qode:iteration=1 score=0/25 -->",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			if err := SaveIterationResult(context.Background(), dir, tc.iteration, tc.text, tc.result); err != nil {
				t.Fatalf("SaveIterationResult: %v", err)
			}

			// Verify iteration file.
			iterData, err := os.ReadFile(filepath.Join(dir, tc.wantIter))
			if err != nil {
				t.Fatalf("read iteration file: %v", err)
			}
			if string(iterData) != tc.text {
				t.Errorf("iteration file content = %q, want %q", iterData, tc.text)
			}

			// Verify canonical file has header + body.
			canonical, err := os.ReadFile(filepath.Join(dir, "refined-analysis.md"))
			if err != nil {
				t.Fatalf("read canonical file: %v", err)
			}
			if !strings.HasPrefix(string(canonical), tc.wantHead) {
				t.Errorf("canonical file missing header %q, got prefix %q", tc.wantHead, string(canonical)[:min(len(canonical), 60)])
			}
			if !strings.Contains(string(canonical), tc.text) {
				t.Errorf("canonical file missing body text")
			}
		})
	}
}

func TestSaveIterationResult_CreatesDir(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "nested", "context")

	result := scoring.Result{TotalScore: 15, MaxScore: 25}
	if err := SaveIterationResult(context.Background(), dir, 1, "body", result); err != nil {
		t.Fatalf("SaveIterationResult: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "refined-analysis-1-score-15.md")); err != nil {
		t.Errorf("iteration file not created: %v", err)
	}
}

func TestParseAndSaveIteration_TargetScoreOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Scoring.TargetScore = 18

	result, err := ParseAndSaveIteration(context.Background(), dir, 1, "some analysis", &cfg)
	if err != nil {
		t.Fatalf("ParseAndSaveIteration: %v", err)
	}

	if result.TargetScore != 18 {
		t.Errorf("TargetScore = %d, want 18", result.TargetScore)
	}
}

func TestParseAndSaveIteration_DefaultsToRubricTotal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result, err := ParseAndSaveIteration(context.Background(), dir, 1, "some analysis", nil)
	if err != nil {
		t.Fatalf("ParseAndSaveIteration: %v", err)
	}

	rubric := scoring.BuildRubric(scoring.RubricRefine, nil)
	if result.TargetScore != rubric.Total() {
		t.Errorf("TargetScore = %d, want rubric total %d", result.TargetScore, rubric.Total())
	}
}

func TestParseAndSaveIteration_WritesFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result, err := ParseAndSaveIteration(context.Background(), dir, 3, "analysis text", nil)
	if err != nil {
		t.Fatalf("ParseAndSaveIteration: %v", err)
	}

	iterFile := filepath.Join(dir, fmt.Sprintf("refined-analysis-3-score-%d.md", result.TotalScore))
	if _, err := os.Stat(iterFile); err != nil {
		t.Errorf("iteration file not created: %v", err)
	}

	canonical, err := os.ReadFile(filepath.Join(dir, "refined-analysis.md"))
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	if !strings.Contains(string(canonical), "analysis text") {
		t.Error("canonical file missing body")
	}
}

func TestBuildAnalysisHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		iteration int
		result    scoring.Result
		want      string
	}{
		{
			name:      "standard header",
			iteration: 2,
			result:    scoring.Result{TotalScore: 20, MaxScore: 25},
			want:      "<!-- qode:iteration=2 score=20/25 -->\n\n",
		},
		{
			name:      "zero score",
			iteration: 1,
			result:    scoring.Result{TotalScore: 0, MaxScore: 25},
			want:      "<!-- qode:iteration=1 score=0/25 -->\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildAnalysisHeader(tc.iteration, tc.result)
			if got != tc.want {
				t.Errorf("buildAnalysisHeader() = %q, want %q", got, tc.want)
			}
		})
	}
}
