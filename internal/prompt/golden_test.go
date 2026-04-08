package prompt

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/nqode/qode/internal/scoring"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_Templates(t *testing.T) {
	t.Parallel()

	e, err := NewEngine(t.TempDir())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	rubric := scoring.BuildRubric(scoring.RubricRefine, nil)

	tests := []struct {
		name string
		data TemplateData
	}{
		{
			name: "refine/base",
			data: NewTemplateData("test-project", "feat-login").
				WithTicket("# Login Feature\nUsers need a login page.").
				WithOutputPath("/tmp/refined-analysis.md").
				WithBranchDir("/tmp/.qode/branches/feat-login").
				WithRubric(rubric).
				WithTargetScore(rubric.Total()).
				Build(),
		},
		{
			name: "spec/base",
			data: NewTemplateData("test-project", "feat-login").
				WithAnalysis("# Analysis\nWell-structured analysis.").
				WithOutputPath("/tmp/spec.md").
				Build(),
		},
		{
			name: "start/base",
			data: NewTemplateData("test-project", "feat-login").
				WithSpec("# Spec\nImplementation specification.").
				WithKB("- docs/architecture.md").
				Build(),
		},
		{
			name: "review/code",
			data: NewTemplateData("test-project", "feat-login").
				WithDiff("diff --git a/main.go b/main.go\n+package main").
				WithSpec("# Spec\nImplementation specification.").
				WithOutputPath("/tmp/code-review.md").
				WithRubric(scoring.BuildRubric(scoring.RubricReview, nil)).
				WithMinPassScore(10.0).
				Build(),
		},
		{
			name: "review/security",
			data: NewTemplateData("test-project", "feat-login").
				WithDiff("diff --git a/main.go b/main.go\n+package main").
				WithSpec("# Spec\nImplementation specification.").
				WithOutputPath("/tmp/security-review.md").
				WithRubric(scoring.BuildRubric(scoring.RubricSecurity, nil)).
				WithMinPassScore(10.0).
				Build(),
		},
		{
			name: "scoring/judge_refine",
			data: NewTemplateData("test-project", "feat-login").
				WithAnalysis("# Analysis\nDetailed analysis content.").
				WithRubric(rubric).
				WithTargetScore(rubric.Total()).
				WithOutputPath("/tmp/judge-score.md").
				Build(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.Render(tt.name, tt.data)
			if err != nil {
				t.Fatalf("Render(%q): %v", tt.name, err)
			}

			goldenFile := filepath.Join("testdata", "golden", filepath.FromSlash(tt.name)+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v\nRun with -update to generate golden files", goldenFile, err)
			}
			if got != string(want) {
				t.Errorf("output does not match golden file %s\n\nGot (first 200 chars):\n%s\n\nRun with -update to regenerate",
					goldenFile, truncate(got, 200))
			}
		})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
