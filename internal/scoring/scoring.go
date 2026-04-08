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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrEmptyJudgment is returned when the judge output contains no scoring data.
var ErrEmptyJudgment = errors.New("empty judgment")

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

// judgmentYAML is the expected structure of the YAML block appended by the judge.
type judgmentYAML struct {
	TotalScore int `yaml:"total_score"`
	MaxScore   int `yaml:"max_score"`
	Dimensions []struct {
		Name  string `yaml:"name"`
		Score int    `yaml:"score"`
		Max   int    `yaml:"max"`
	} `yaml:"dimensions"`
}

// extractYAMLBlock returns the content inside the last ```yaml ... ``` fenced block,
// or an empty string if no such block is found.
func extractYAMLBlock(text string) string {
	const fence = "```yaml"
	const closeFence = "```"
	last := strings.LastIndex(text, fence)
	if last == -1 {
		return ""
	}
	inner := text[last+len(fence):]
	end := strings.Index(inner, closeFence)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(inner[:end])
}

// parseYAMLBlock tries to parse a judgmentYAML from raw YAML text.
// The YAML may be wrapped under a top-level "judgment:" key or bare.
func parseYAMLBlock(raw string, rubric Rubric) (Result, bool) {
	// Try bare struct first, then wrapped under "judgment:".
	var jy judgmentYAML
	wrapErr := func() error {
		var wrapped struct {
			Judgment judgmentYAML `yaml:"judgment"`
		}
		if err := yaml.Unmarshal([]byte(raw), &wrapped); err != nil {
			return err
		}
		if wrapped.Judgment.TotalScore == 0 && len(wrapped.Judgment.Dimensions) == 0 {
			return ErrEmptyJudgment
		}
		jy = wrapped.Judgment
		return nil
	}

	if err := yaml.Unmarshal([]byte(raw), &jy); err != nil || (jy.TotalScore == 0 && len(jy.Dimensions) == 0) {
		if err := wrapErr(); err != nil {
			return Result{}, false
		}
	}

	if jy.TotalScore == 0 {
		return Result{}, false
	}

	maxScore := jy.MaxScore
	if maxScore == 0 {
		maxScore = rubric.Total()
	}

	result := Result{
		TotalScore:  jy.TotalScore,
		MaxScore:    maxScore,
		TargetScore: rubric.Total(),
	}

	// Build dimension scores from YAML; fall back to rubric order when names match.
	yamlByName := make(map[string]int, len(jy.Dimensions))
	for _, d := range jy.Dimensions {
		yamlByName[strings.ToLower(d.Name)] = d.Score
	}

	for i, dim := range rubric.Dimensions {
		score := 0
		if s, ok := yamlByName[strings.ToLower(dim.Name)]; ok {
			score = s
		} else if i < len(jy.Dimensions) {
			score = jy.Dimensions[i].Score
		}
		result.Dimensions = append(result.Dimensions, DimensionScore{
			Name:     dim.Name,
			Score:    score,
			MaxScore: dim.Weight,
		})
	}

	result.Ready = result.TotalScore >= result.TargetScore
	return result, true
}

// ParseScore attempts to extract a total score from judge output.
// It first tries to parse a ```yaml block appended by the judge; if that
// fails it falls back to loose regex patterns for backward compatibility.
// TargetScore defaults to rubric.Total(); callers may override it afterward.
func ParseScore(judgeOutput string, rubric Rubric) Result {
	// Primary path: structured YAML block.
	if raw := extractYAMLBlock(judgeOutput); raw != "" {
		if result, ok := parseYAMLBlock(raw, rubric); ok {
			// Still extract improvements from the prose portion.
			result.Improvements = extractImprovements(judgeOutput)
			return result
		}
	}

	// Fallback: regex-based extraction for legacy / plain-text judge output.
	return parseScoreRegex(judgeOutput, rubric)
}

// parseScoreRegex is the original regex-based score extractor, kept as fallback.
func parseScoreRegex(judgeOutput string, rubric Rubric) Result {
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

	result.Improvements = extractImprovements(judgeOutput)
	result.Ready = result.TotalScore >= result.TargetScore
	return result
}

// extractImprovements parses numbered list items from the output.
// Matches lines like "1. Some improvement text".
func extractImprovements(text string) []string {
	lineRe := regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)
	lineMatches := lineRe.FindAllStringSubmatch(text, 5)
	var improvements []string
	for _, m := range lineMatches {
		if len(m) >= 2 {
			line := strings.TrimSpace(m[1])
			if line != "" {
				improvements = append(improvements, line)
			}
		}
	}
	return improvements
}
