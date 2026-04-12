//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/nqode/qode/internal/knowledge"
)

func TestTruncateLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
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
