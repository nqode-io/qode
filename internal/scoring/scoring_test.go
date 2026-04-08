package scoring

import (
	"testing"
)

func TestParseScore_RefineRubric(t *testing.T) {
	t.Parallel()
	judgeOutput := `
## Summary

**Total Score:** 22/25

**Overall Assessment:** Good analysis with minor gaps in risk coverage.

**Top 3 improvements needed for next iteration:**
1. Add more detail on error handling edge cases
2. Specify database migration strategy
3. Clarify authentication approach
`

	result := ParseScore(judgeOutput, DefaultRefineRubric)

	if result.TotalScore != 22 {
		t.Errorf("expected TotalScore 22, got %d", result.TotalScore)
	}
	if result.MaxScore != 25 {
		t.Errorf("expected MaxScore 25, got %d", result.MaxScore)
	}
	if result.Ready {
		t.Error("expected Ready=false for score < 25")
	}
	if len(result.Improvements) == 0 {
		t.Error("expected improvements to be parsed")
	}
}

func TestParseScore_PerfectScore(t *testing.T) {
	t.Parallel()
	judgeOutput := `
**Total Score:** 25/25

Excellent requirements analysis.
`
	result := ParseScore(judgeOutput, DefaultRefineRubric)
	if result.TotalScore != 25 {
		t.Errorf("expected 25, got %d", result.TotalScore)
	}
	if !result.Ready {
		t.Error("expected Ready=true for 25/25")
	}
}

func TestResult_String(t *testing.T) {
	t.Parallel()
	r := Result{TotalScore: 18, MaxScore: 25}
	if r.String() != "18/25" {
		t.Errorf("unexpected string: %s", r.String())
	}
}

func TestParseScore_YAMLBlock(t *testing.T) {
	t.Parallel()
	judgeOutput := `
## Summary

The analysis is thorough and covers all required dimensions.

` + "```yaml" + `
judgment:
  total_score: 23
  max_score: 25
  dimensions:
    - name: "Problem Understanding"
      score: 5
      max: 5
    - name: "Technical Analysis"
      score: 5
      max: 5
    - name: "Risk & Edge Cases"
      score: 4
      max: 5
    - name: "Completeness"
      score: 5
      max: 5
    - name: "Actionability"
      score: 4
      max: 5
` + "```" + `
`

	result := ParseScore(judgeOutput, DefaultRefineRubric)

	if result.TotalScore != 23 {
		t.Errorf("expected TotalScore 23, got %d", result.TotalScore)
	}
	if result.MaxScore != 25 {
		t.Errorf("expected MaxScore 25, got %d", result.MaxScore)
	}
	if result.Ready {
		t.Error("expected Ready=false for score 23 < 25")
	}
	if len(result.Dimensions) != len(DefaultRefineRubric.Dimensions) {
		t.Errorf("expected %d dimensions, got %d", len(DefaultRefineRubric.Dimensions), len(result.Dimensions))
	}
	if result.Dimensions[2].Score != 4 {
		t.Errorf("expected dimension 3 score 4, got %d", result.Dimensions[2].Score)
	}
}

func TestParseScore_YAMLBlockIgnoresAdversarialPreamble(t *testing.T) {
	t.Parallel()
	// The preamble contains score-like strings that would confuse regex, but
	// ParseScore must use the YAML block and ignore them.
	judgeOutput := `
For example, a good analysis might score: 18/25 or even total score: 20/25 on a similar task.
But this analysis scored differently. Total Score: 99/25 should not be used.

` + "```yaml" + `
judgment:
  total_score: 21
  max_score: 25
  dimensions:
    - name: "Problem Understanding"
      score: 4
      max: 5
    - name: "Technical Analysis"
      score: 4
      max: 5
    - name: "Risk & Edge Cases"
      score: 4
      max: 5
    - name: "Completeness"
      score: 5
      max: 5
    - name: "Actionability"
      score: 4
      max: 5
` + "```" + `
`

	result := ParseScore(judgeOutput, DefaultRefineRubric)

	if result.TotalScore != 21 {
		t.Errorf("expected TotalScore 21 from YAML block, got %d (regex fallback may have fired)", result.TotalScore)
	}
	if result.MaxScore != 25 {
		t.Errorf("expected MaxScore 25, got %d", result.MaxScore)
	}
}

func TestParseScore_FallsBackToRegexWhenNoYAML(t *testing.T) {
	t.Parallel()
	// No YAML block — should fall back to regex extraction (existing behaviour).
	judgeOutput := `
## Summary

**Total Score:** 22/25

**Overall Assessment:** Good analysis with minor gaps in risk coverage.

**Top 3 improvements needed for next iteration:**
1. Add more detail on error handling edge cases
2. Specify database migration strategy
3. Clarify authentication approach
`

	result := ParseScore(judgeOutput, DefaultRefineRubric)

	if result.TotalScore != 22 {
		t.Errorf("expected TotalScore 22 via regex fallback, got %d", result.TotalScore)
	}
	if result.MaxScore != 25 {
		t.Errorf("expected MaxScore 25, got %d", result.MaxScore)
	}
	if len(result.Improvements) == 0 {
		t.Error("expected improvements to be parsed via regex fallback")
	}
}

func TestParseScore_Adversarial(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantScore  int
		wantMax    int
		wantParsed bool // false means TotalScore=0
	}{
		{
			name:       "truncated YAML no closing fence",
			input:      "```yaml\njudgment:\n  total_sc",
			wantScore:  0,
			wantParsed: false,
		},
		{
			name: "duplicate YAML blocks uses last",
			input: "```yaml\njudgment:\n  total_score: 10\n  max_score: 25\n```\n\n" +
				"```yaml\njudgment:\n  total_score: 22\n  max_score: 25\n```\n",
			wantScore:  22,
			wantMax:    25,
			wantParsed: true,
		},
		{
			name:       "non-numeric total_score",
			input:      "```yaml\njudgment:\n  total_score: high\n  max_score: 25\n```\n",
			wantScore:  0,
			wantParsed: false,
		},
		{
			name:       "extremely large score",
			input:      "```yaml\njudgment:\n  total_score: 999999\n  max_score: 25\n```\n",
			wantScore:  999999,
			wantMax:    25,
			wantParsed: true,
		},
		{
			name:       "negative score",
			input:      "```yaml\njudgment:\n  total_score: -5\n  max_score: 25\n```\n",
			wantScore:  -5,
			wantMax:    25,
			wantParsed: true,
		},
		{
			name:       "missing dimensions field",
			input:      "```yaml\njudgment:\n  total_score: 20\n  max_score: 25\n```\n",
			wantScore:  20,
			wantMax:    25,
			wantParsed: true,
		},
		{
			name:       "empty YAML block",
			input:      "```yaml\n```\n",
			wantScore:  0,
			wantParsed: false,
		},
		{
			name:       "bare YAML without judgment wrapper",
			input:      "```yaml\ntotal_score: 20\nmax_score: 25\n```\n",
			wantScore:  20,
			wantMax:    25,
			wantParsed: true,
		},
		{
			name:       "YAML block with only whitespace",
			input:      "```yaml\n   \n```\n",
			wantScore:  0,
			wantParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ParseScore(tt.input, DefaultRefineRubric)
			if tt.wantParsed {
				if result.TotalScore != tt.wantScore {
					t.Errorf("TotalScore = %d, want %d", result.TotalScore, tt.wantScore)
				}
				if tt.wantMax != 0 && result.MaxScore != tt.wantMax {
					t.Errorf("MaxScore = %d, want %d", result.MaxScore, tt.wantMax)
				}
			} else {
				if result.TotalScore != 0 {
					t.Errorf("expected TotalScore=0 for unparseable input, got %d", result.TotalScore)
				}
			}
		})
	}
}

func TestRubricDimensions(t *testing.T) {
	t.Parallel()
	total := 0
	for _, d := range DefaultRefineRubric.Dimensions {
		total += d.Weight
	}
	if total != DefaultRefineRubric.Total() {
		t.Errorf("refine rubric weights sum to %d, expected %d", total, DefaultRefineRubric.Total())
	}

	total = 0
	for _, d := range DefaultReviewRubric.Dimensions {
		total += d.Weight
	}
	if total != DefaultReviewRubric.Total() {
		t.Errorf("review rubric weights sum to %d, expected %d", total, DefaultReviewRubric.Total())
	}
}
