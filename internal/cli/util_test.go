//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRoot_UsesFlag(t *testing.T) {
	root := t.TempDir()
	flagRoot = root
	t.Cleanup(func() { flagRoot = "" })

	got, err := resolveRoot()
	if err != nil {
		t.Fatalf("resolveRoot: %v", err)
	}
	if got != root {
		t.Errorf("resolveRoot() = %q, want %q", got, root)
	}
}

func TestResolveRoot_FallsBackToWd(t *testing.T) {
	flagRoot = ""
	got, err := resolveRoot()
	if err != nil {
		t.Fatalf("resolveRoot: %v", err)
	}
	wd, _ := os.Getwd()
	if got != wd {
		t.Errorf("resolveRoot() = %q, want wd %q", got, wd)
	}
}

func TestWritePromptToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "prompt.md")

	if err := writePromptToFile(path, "prompt content"); err != nil {
		t.Fatalf("writePromptToFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "prompt content" {
		t.Errorf("got %q, want %q", data, "prompt content")
	}

	// Verify restrictive permissions (0600).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("permissions: got %o, want %o", got, 0600)
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"code", "Code"},
		{"security", "Security"},
		{"", ""},
		{"A", "A"},
		{"already", "Already"},
	}
	for _, tt := range tests {
		got := capitalize(tt.input)
		if got != tt.want {
			t.Errorf("capitalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
