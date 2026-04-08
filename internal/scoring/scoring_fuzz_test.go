package scoring

import "testing"

func FuzzParseScore(f *testing.F) {
	// Seed with adversarial cases from TestParseScore_Adversarial.
	f.Add("```yaml\njudgment:\n  total_sc")
	f.Add("```yaml\njudgment:\n  total_score: 10\n  max_score: 25\n```\n")
	f.Add("```yaml\njudgment:\n  total_score: high\n  max_score: 25\n```\n")
	f.Add("```yaml\njudgment:\n  total_score: 999999\n  max_score: 25\n```\n")
	f.Add("```yaml\njudgment:\n  total_score: -5\n  max_score: 25\n```\n")
	f.Add("```yaml\n```\n")
	f.Add("```yaml\n   \n```\n")
	f.Add("```yaml\ntotal_score: 20\nmax_score: 25\n```\n")
	f.Add("Total Score: 20/25\n\n## Improvements\n- Fix error handling")
	f.Add("")
	f.Add("no yaml here, just text")
	f.Add("```yaml\njudgment:\n  total_score: 22\n  max_score: 25\n  dimensions:\n    - name: clarity\n      score: 5\n      max: 5\n```\n")

	f.Fuzz(func(t *testing.T, input string) {
		// ParseScore must never panic regardless of input.
		_ = ParseScore(input, DefaultRefineRubric)
	})
}
