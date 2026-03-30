// Package scoring implements the two-pass worker/judge scoring engine.
//
// The two-pass approach eliminates the self-scoring bias that plagues single-pass
// AI evaluations. The worker produces output without knowing it will be scored;
// the judge receives only that output and applies the rubric independently.
//
// Flow:
//
//	Context → [Worker prompt] → Worker output (no self-score)
//	                                   ↓
//	                          [Judge prompt] → Score + per-dimension feedback
package scoring

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Result holds the judge's scoring output.
type Result struct {
	TotalScore   int
	MaxScore     int
	Dimensions   []DimensionScore
	Assessment   string
	Improvements []string
	Ready        bool // true when score meets the target
	TargetScore  int
}

// DimensionScore holds the score and justification for one rubric dimension.
type DimensionScore struct {
	Name          string
	Score         int
	MaxScore      int
	Justification string
}

// String returns a compact representation of the result.
func (r Result) String() string {
	return fmt.Sprintf("%d/%d", r.TotalScore, r.MaxScore)
}

// ParseScore attempts to extract a total score from judge output.
// It looks for patterns like "Total Score: 22/25" or "**Total Score:** 22/25".
// TargetScore defaults to rubric.Total(); callers may override it afterward.
func ParseScore(judgeOutput string, rubric Rubric) Result {
	result := Result{
		MaxScore:    rubric.Total(),
		TargetScore: rubric.Total(),
	}

	// Try to find "Total Score: N/M" or "N/M" near "total".
	totalRe := regexp.MustCompile(`(?i)total\s*score[:\s*]*(\d+)\s*/\s*(\d+)`)
	if m := totalRe.FindStringSubmatch(judgeOutput); len(m) >= 3 {
		result.TotalScore, _ = strconv.Atoi(m[1])
		result.MaxScore, _ = strconv.Atoi(m[2])
	}

	// Parse per-dimension scores: "Score: N/5" or "**Score:** N/5".
	dimRe := regexp.MustCompile(`(?i)(?:score|points)[:\s*]*(\d+)\s*/\s*(\d+)`)
	matches := dimRe.FindAllStringSubmatch(judgeOutput, -1)
	for i, dim := range rubric.Dimensions {
		score := 0
		if i < len(matches) && len(matches[i]) >= 2 {
			score, _ = strconv.Atoi(matches[i][1])
		}
		result.Dimensions = append(result.Dimensions, DimensionScore{
			Name:     dim.Name,
			Score:    score,
			MaxScore: dim.Weight,
		})
	}

	// Extract numbered improvements from anywhere in the output.
	// Matches lines like "1. Some improvement text"
	lineRe := regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)
	lineMatches := lineRe.FindAllStringSubmatch(judgeOutput, 5)
	for _, m := range lineMatches {
		if len(m) >= 2 {
			line := strings.TrimSpace(m[1])
			if line != "" {
				result.Improvements = append(result.Improvements, line)
			}
		}
	}

	result.Ready = result.TotalScore >= result.TargetScore
	return result
}
