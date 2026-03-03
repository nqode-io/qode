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

	result := ParseScore(judgeOutput, RefineRubric)

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
	result := ParseScore(judgeOutput, RefineRubric)
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

func TestRubricDimensions(t *testing.T) {
	total := 0
	for _, d := range RefineRubric.Dimensions {
		total += d.Weight
	}
	if total != RefineRubric.MaxScore {
		t.Errorf("dimension weights sum to %d, expected %d", total, RefineRubric.MaxScore)
	}

	total = 0
	for _, d := range ReviewRubric.Dimensions {
		total += d.Weight
	}
	if total != ReviewRubric.MaxScore {
		t.Errorf("review rubric weights sum to %d, expected %d", total, ReviewRubric.MaxScore)
	}
}
