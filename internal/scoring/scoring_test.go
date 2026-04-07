package scoring

import (
	"testing"
)

func TestParseScore_RefineRubric(t *testing.T) {
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
	r := Result{TotalScore: 18, MaxScore: 25}
	if r.String() != "18/25" {
		t.Errorf("unexpected string: %s", r.String())
	}
}

func TestParseScore_YAMLBlock(t *testing.T) {
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
	if len(result.Dimensions) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(result.Dimensions))
	}
	if result.Dimensions[2].Score != 4 {
		t.Errorf("expected dimension 3 score 4, got %d", result.Dimensions[2].Score)
	}
}

func TestParseScore_YAMLBlockIgnoresAdversarialPreamble(t *testing.T) {
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

func TestRubricDimensions(t *testing.T) {
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
