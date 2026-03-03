package scoring

import (
	"os"
	"regexp"
	"strconv"
)

var totalScoreRe = regexp.MustCompile(`(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*10`)

// ExtractScoreFromFile reads a review markdown file and returns the total
// score found on a "Total Score: X/10" line. Returns 0 if the file does not
// exist or no score line is present.
func ExtractScoreFromFile(path string) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return extractScore(string(data))
}

func extractScore(text string) float64 {
	m := totalScoreRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0
	}
	score, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return score
}
