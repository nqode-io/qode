package scaffold

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendGitignoreRules(t *testing.T) {
	t.Parallel()

	allRulesContent := GitignoreMarker + "\n" + strings.Join(GitignoreRules, "\n") + "\n"

	tests := []struct {
		name          string
		setup         func(t *testing.T, dir string)
		wantOutput    string
		wantErrSubstr string
		verify        func(t *testing.T, dir string)
	}{
		{
			name:  "no gitignore exists",
			setup: func(t *testing.T, dir string) {},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				if !strings.Contains(content, GitignoreMarker) {
					t.Errorf("missing marker %q", GitignoreMarker)
				}
				for _, rule := range GitignoreRules {
					if !strings.Contains(content, rule) {
						t.Errorf("missing rule %q", rule)
					}
				}
			},
			wantOutput: "Appended qode ignore rules to .gitignore",
		},
		{
			name: "gitignore exists without marker",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeGitignore(t, dir, "node_modules/\n*.log\n")
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				if !strings.Contains(content, GitignoreMarker) {
					t.Errorf("missing marker %q", GitignoreMarker)
				}
				for _, rule := range GitignoreRules {
					if !strings.Contains(content, rule) {
						t.Errorf("missing rule %q", rule)
					}
				}
				if !strings.Contains(content, "node_modules/") {
					t.Error("existing content was removed")
				}
			},
			wantOutput: "Appended qode ignore rules to .gitignore",
		},
		{
			name: "all rules already present",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeGitignore(t, dir, allRulesContent)
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				if content != allRulesContent {
					t.Errorf("file was modified on no-op: got %q, want %q", content, allRulesContent)
				}
			},
			wantOutput: "",
		},
		{
			name: "marker present, two rules missing",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				present := GitignoreMarker + "\n" +
					GitignoreRules[0] + "\n" +
					GitignoreRules[1] + "\n" +
					GitignoreRules[2] + "\n"
				writeGitignore(t, dir, present)
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				for _, rule := range GitignoreRules {
					if !strings.Contains(content, rule) {
						t.Errorf("missing rule %q", rule)
					}
					if strings.Count(content, rule) != 1 {
						t.Errorf("rule %q duplicated (count=%d)", rule, strings.Count(content, rule))
					}
				}
			},
			wantOutput: "Appended qode ignore rules to .gitignore",
		},
		{
			name: "no trailing newline in existing file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeGitignore(t, dir, "foo")
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				if strings.HasPrefix(content, "foo"+GitignoreMarker) {
					t.Error("block must start on new line after existing content without trailing newline")
				}
				for _, rule := range GitignoreRules {
					if !strings.Contains(content, rule) {
						t.Errorf("missing rule %q", rule)
					}
				}
			},
			wantOutput: "Appended qode ignore rules to .gitignore",
		},
		{
			name: "all rules present but marker absent",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeGitignore(t, dir, strings.Join(GitignoreRules, "\n")+"\n")
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				content := readGitignore(t, dir)
				want := strings.Join(GitignoreRules, "\n") + "\n"
				if content != want {
					t.Errorf("file was modified when all rules already present: got %q, want %q", content, want)
				}
			},
			wantOutput: "",
		},
		{
			name: "context cancelled",
			setup: func(t *testing.T, dir string) {
				t.Helper()
			},
			verify: func(t *testing.T, dir string) {
				t.Helper()
				// iokit.WriteFileCtx performs an atomic write; a pre-cancelled context
				// aborts before the rename, leaving no file at the destination.
				if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
					t.Error("expected .gitignore to not exist after cancelled context")
				}
			},
			wantErrSubstr: "updating .gitignore",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tc.setup(t, dir)

			var buf bytes.Buffer
			var err error
			if tc.wantErrSubstr != "" {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				err = AppendGitignoreRules(ctx, &buf, dir)
			} else {
				err = AppendGitignoreRules(context.Background(), &buf, dir)
			}

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out := buf.String()
			if tc.wantOutput != "" {
				if !strings.Contains(out, tc.wantOutput) {
					t.Errorf("output %q missing %q", out, tc.wantOutput)
				}
			} else {
				if out != "" {
					t.Errorf("expected no output on no-op, got %q", out)
				}
			}

			tc.verify(t, dir)
		})
	}
}

func readGitignore(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	return string(data)
}

func writeGitignore(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(content), 0644); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
}
