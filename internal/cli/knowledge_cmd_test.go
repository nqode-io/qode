//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/nqode/qode/internal/knowledge"
)

// TestParseBranchArgs_SingleDot verifies that "." is silently dropped because
// it would resolve to the branches root directory rather than a named branch.
func TestParseBranchArgs_SingleDot(t *testing.T) {
	result := parseBranchArgs([]string{"."})
	if len(result) != 0 {
		t.Errorf("expected empty result for \".\", got %v", result)
	}
}

// TestParseBranchArgs_DotInList verifies that "." is filtered out while valid
// branch names in the same list are preserved.
func TestParseBranchArgs_DotInList(t *testing.T) {
	result := parseBranchArgs([]string{"feat/my-feature,.,main"})
	for _, b := range result {
		if b == "." {
			t.Errorf("\".\", should have been filtered, got %v", result)
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 branches, got %v", result)
	}
}

// TestParseBranchArgs_DotDot verifies that ".." is still blocked.
func TestParseBranchArgs_DotDot(t *testing.T) {
	result := parseBranchArgs([]string{".."})
	if len(result) != 0 {
		t.Errorf("expected empty result for \"..\", got %v", result)
	}
}

// TestParseBranchArgs_SlashedBranch verifies that slashed branch names are
// accepted (sanitization happens downstream in context.Load).
func TestParseBranchArgs_SlashedBranch(t *testing.T) {
	result := parseBranchArgs([]string{"feat/jira-123"})
	if len(result) != 1 || result[0] != "feat/jira-123" {
		t.Errorf("expected [\"feat/jira-123\"], got %v", result)
	}
}

func TestTruncateLines(t *testing.T) {
	tests := []struct {
		name string
		input string
		max  int
		want string
	}{
		{"under limit", "a\nb\nc", 5, "a\nb\nc"},
		{"at limit", "a\nb\nc", 3, "a\nb\nc"},
		{"over limit", "a\nb\nc\nd\ne", 3, "a\nb\nc\n\n(truncated)"},
		{"empty", "", 5, ""},
		{"single line", "hello", 1, "hello"},
		{"single line over", "a\nb", 1, "a\n\n(truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLines(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncateLines(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestFormatLessonsList(t *testing.T) {
	tests := []struct {
		name    string
		lessons []knowledge.LessonSummary
		want    string
	}{
		{"empty", nil, ""},
		{
			"single",
			[]knowledge.LessonSummary{{Title: "Lesson 1", Summary: "First lesson"}},
			"- **Lesson 1**: First lesson\n",
		},
		{
			"multiple",
			[]knowledge.LessonSummary{
				{Title: "A", Summary: "first"},
				{Title: "B", Summary: "second"},
			},
			"- **A**: first\n- **B**: second\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLessonsList(tt.lessons)
			if got != tt.want {
				t.Errorf("formatLessonsList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateLines_PreservesContent(t *testing.T) {
	input := strings.Repeat("line\n", 600)
	got := truncateLines(input, 500)
	if !strings.HasSuffix(got, "(truncated)") {
		t.Error("expected (truncated) suffix for input over limit")
	}
	lines := strings.Split(got, "\n")
	const maxLines = 500
	const wantLines = maxLines + 2 // content lines + empty line + "(truncated)"
	if len(lines) != wantLines {
		t.Errorf("expected %d lines, got %d", wantLines, len(lines))
	}
}
