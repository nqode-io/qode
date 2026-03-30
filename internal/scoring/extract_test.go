package scoring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractScore(t *testing.T) {
	tests := []struct {
		name string
		text string
		want float64
	}{
		{"plain", "**Total Score: 8.5/10**", 8.5},
		{"integer", "Total Score: 7/10", 7.0},
		{"bold markdown", "**Total Score: 9.0/10**", 9.0},
		{"extra whitespace", "Total  Score:  6.5 / 10", 6.5},
		{"case insensitive", "total score: 5/10", 5.0},
		{"dynamic denominator", "**Total Score: 18/20**", 18.0},
		{"dynamic denominator 25", "Total Score: 22/25", 22.0},
		{"no match", "no score here", 0},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractScore(tt.text)
			if got != tt.want {
				t.Errorf("extractScore(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestExtractScoreFromFile(t *testing.T) {
	t.Run("file not found returns 0", func(t *testing.T) {
		got := ExtractScoreFromFile("/nonexistent/path/review.md")
		if got != 0 {
			t.Errorf("expected 0, got %v", got)
		}
	})

	t.Run("reads score from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "review.md")
		content := "# Code Review\n\n**Total Score: 8.0/10**\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := ExtractScoreFromFile(path)
		if got != 8.0 {
			t.Errorf("expected 8.0, got %v", got)
		}
	})
}
