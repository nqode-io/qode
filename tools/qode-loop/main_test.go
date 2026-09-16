package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want any
	}{
		{name: "null", raw: "null", want: nil},
		{name: "integer", raw: "42", want: 42},
		{name: "zero", raw: "0", want: 0},
		{name: "string", raw: "refine", want: "refine"},
		{name: "negative stays string", raw: "-1", want: "-1"},
		{name: "plus sign stays string", raw: "+1", want: "+1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseValue(tc.raw); got != tc.want {
				t.Fatalf("parseValue(%q) = %#v, want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestRun_UsageErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "unknown command", args: []string{"bogus"}},
		{name: "state without subcommand", args: []string{"state"}},
		{name: "unknown state subcommand", args: []string{"state", "bogus"}},
		{name: "set without equals", args: []string{"state", "set", "stage"}},
		{name: "set with empty key", args: []string{"state", "set", "=refine"}},
		{name: "refine-score too few args", args: []string{"refine-score", "1", "25"}},
		{name: "refine-score non-integer", args: []string{"refine-score", "one", "25", "25"}},
		{name: "refine-score over max", args: []string{"refine-score", "1", "26", "25"}},
		{name: "refine-score zero iteration", args: []string{"refine-score", "0", "25", "25"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, stateFile), `{"stage":"setup"}`)
			err := run(tc.args, &bytes.Buffer{}, dir)
			if !errors.Is(err, errUsage) {
				t.Fatalf("run(%v) error = %v, want errUsage", tc.args, err)
			}
		})
	}
}

func TestRunState_InitSetGet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var out bytes.Buffer
	if err := run([]string{"state", "init", "ticket=72"}, &out, dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out.String(), `"ticket": 72`) || !strings.Contains(out.String(), `"stage": "setup"`) {
		t.Fatalf("init output = %s", out.String())
	}

	out.Reset()
	if err := run([]string{"state", "set", "stage=refine", "refineIteration=2", "prNumber=null"}, &out, dir); err != nil {
		t.Fatalf("set: %v", err)
	}

	out.Reset()
	if err := run([]string{"state", "get", "stage"}, &out, dir); err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `"refine"` {
		t.Fatalf("get stage = %q, want %q", got, `"refine"`)
	}

	out.Reset()
	if err := run([]string{"state", "get"}, &out, dir); err != nil {
		t.Fatalf("get all: %v", err)
	}
	for _, want := range []string{`"refineIteration": 2`, `"prNumber": null`, `"ticket": 72`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("get all output missing %s:\n%s", want, out.String())
		}
	}
}

func TestRunState_GetMissingFile(t *testing.T) {
	t.Parallel()
	err := run([]string{"state", "get"}, &bytes.Buffer{}, t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("get on missing state error = %v, want os.ErrNotExist", err)
	}
}

func TestRunRefineScore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		content   string
		wantFirst string
		wantLines int
	}{
		{
			name:      "replaces existing header",
			content:   "<!-- qode:iteration=2 -->\n# Analysis\nbody\n",
			wantFirst: "<!-- qode:iteration=2 score=23/25 -->",
			wantLines: 4,
		},
		{
			name:      "prepends when header missing",
			content:   "# Analysis\nbody\n",
			wantFirst: "<!-- qode:iteration=2 score=23/25 -->",
			wantLines: 4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, analysisFile)
			writeFile(t, path, tc.content)

			var out bytes.Buffer
			if err := run([]string{"refine-score", "2", "23", "25"}, &out, dir); err != nil {
				t.Fatalf("refine-score: %v", err)
			}

			stamped := readFile(t, path)
			lines := strings.Split(stamped, "\n")
			if lines[0] != tc.wantFirst {
				t.Errorf("first line = %q, want %q", lines[0], tc.wantFirst)
			}
			if len(lines) != tc.wantLines {
				t.Errorf("line count = %d, want %d:\n%s", len(lines), tc.wantLines, stamped)
			}
			archive := filepath.Join(dir, "refined-analysis-2-score-23.md")
			if readFile(t, archive) != stamped {
				t.Errorf("archive %s differs from stamped analysis", archive)
			}
			if !strings.Contains(out.String(), "score=23/25") {
				t.Errorf("output = %q, want score summary", out.String())
			}
		})
	}
}

func TestRunRefineScore_MissingAnalysis(t *testing.T) {
	t.Parallel()
	err := run([]string{"refine-score", "1", "25", "25"}, &bytes.Buffer{}, t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refine-score on missing analysis error = %v, want os.ErrNotExist", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), filePerm); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
