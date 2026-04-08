package scaffold

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
)

// helpers

func setupProject(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readClaudeCommand(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, ".claude", "commands", name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s.md: %v", name, err)
	}
	return string(data)
}

func readCursorCommand(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, ".cursor", "commands", name+".mdc")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s.mdc: %v", name, err)
	}
	return string(data)
}

// --- SetupClaudeCode ---

func TestSetupClaudeCode_WritesTicketFetchCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	content := readClaudeCommand(t, dir, "qode-ticket-fetch")
	for _, want := range []string{"$ARGUMENTS", "context/ticket.md", "MCP", "Figma"} {
		if !strings.Contains(content, want) {
			t.Errorf("qode-ticket-fetch.md missing %q", want)
		}
	}
	if strings.Contains(content, "qode ticket fetch") {
		t.Error("qode-ticket-fetch.md must not reference CLI command")
	}
}

func TestSetupClaudeCode_WritesAllCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".claude", "commands"))
	if err != nil {
		t.Fatalf("reading commands dir: %v", err)
	}
	if len(entries) != len(claudeCommands) {
		t.Errorf("SetupClaudeCode: wrote %d commands, want %d", len(entries), len(claudeCommands))
	}
}

func TestSetupClaudeCode_WritesKnowledgeCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	for _, name := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		content := readClaudeCommand(t, dir, name)
		if len(content) == 0 {
			t.Errorf("%s.md is empty", name)
		}
	}
}

func TestSetupClaudeCode_IncludesQodeCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	content := readClaudeCommand(t, dir, "qode-check")
	if content == "" {
		t.Fatal("qode-check.md is empty")
	}
	for _, prohibited := range []string{"test.unit", "test.lint"} {
		if strings.Contains(content, prohibited) {
			t.Errorf("qode-check.md must not reference qode.yaml field %q", prohibited)
		}
	}
	for _, heading := range []string{"Phase 1", "Phase 2"} {
		if !strings.Contains(content, heading) {
			t.Errorf("qode-check.md missing heading %q", heading)
		}
	}
}

func TestSetupClaudeCode_NoPromptOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	for _, name := range claudeCommands {
		content := readClaudeCommand(t, dir, name)
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("%s.md contains --prompt-only", name)
		}
	}
}

func TestSetupClaudeCode_CommandsContainRootName(t *testing.T) {
	t.Parallel()
	projectDir := setupProject(t, "myproject")
	if err := SetupClaudeCode(io.Discard, projectDir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	content := readClaudeCommand(t, projectDir, "qode-plan-refine")
	if !strings.Contains(content, "myproject") {
		t.Errorf("qode-plan-refine.md missing root dir name %q, got:\n%s", "myproject", content)
	}
}

// --- SetupCursor ---

func TestSetupCursor_WritesTicketFetchCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	content := readCursorCommand(t, dir, "qode-ticket-fetch")
	for _, want := range []string{"$ARGUMENTS", "context/ticket.md", "description:", "MCP", "Figma"} {
		if !strings.Contains(content, want) {
			t.Errorf("qode-ticket-fetch.mdc missing %q", want)
		}
	}
	if strings.Contains(content, "qode ticket fetch") {
		t.Error("qode-ticket-fetch.mdc must not reference CLI command")
	}
}

func TestSetupCursor_WritesAllCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".cursor", "commands"))
	if err != nil {
		t.Fatalf("reading cursor commands dir: %v", err)
	}
	if len(entries) != len(cursorCommands) {
		t.Errorf("SetupCursor: wrote %d commands, want %d", len(entries), len(cursorCommands))
	}
}

func TestSetupCursor_WritesKnowledgeCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	for _, name := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		content := readCursorCommand(t, dir, name)
		if !strings.Contains(content, "description:") {
			t.Errorf("%s.mdc missing YAML frontmatter", name)
		}
	}
}

func TestSetupCursor_IncludesQodeCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	content := readCursorCommand(t, dir, "qode-check")
	if !strings.Contains(content, "description:") {
		t.Error("qode-check.mdc missing YAML frontmatter")
	}
	for _, prohibited := range []string{"test.unit", "test.lint"} {
		if strings.Contains(content, prohibited) {
			t.Errorf("qode-check.mdc must not reference qode.yaml field %q", prohibited)
		}
	}
	for _, heading := range []string{"Phase 1", "Phase 2"} {
		if !strings.Contains(content, heading) {
			t.Errorf("qode-check.mdc missing heading %q", heading)
		}
	}
}

func TestSetupCursor_NoPromptOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	for _, name := range cursorCommands {
		content := readCursorCommand(t, dir, name)
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("%s.mdc contains --prompt-only", name)
		}
	}
}

func TestSetupCursor_NoCursorRulesDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".cursorrules")); !os.IsNotExist(err) {
		t.Error("SetupCursor must not create .cursorrules/ directory")
	}
}

func TestSetupCursor_CommandsContainRootName(t *testing.T) {
	t.Parallel()
	projectDir := setupProject(t, "myproject")
	if err := SetupCursor(io.Discard, projectDir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	content := readCursorCommand(t, projectDir, "qode-plan-refine")
	if !strings.Contains(content, "myproject") {
		t.Errorf("qode-plan-refine.mdc missing root dir name %q, got:\n%s", "myproject", content)
	}
}

// --- Setup orchestration ---

func TestSetup_BothIDEs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		IDE: config.IDEConfig{
			Cursor:     config.CursorIDEConfig{Enabled: true},
			ClaudeCode: config.ClaudeCodeIDEConfig{Enabled: true},
		},
	}
	var buf bytes.Buffer
	if err := Setup(&buf, dir, cfg); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Both command dirs should exist.
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "commands")); err != nil {
		t.Errorf("expected .cursor/commands dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "commands")); err != nil {
		t.Errorf("expected .claude/commands dir: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Cursor") || !strings.Contains(out, "Claude Code") {
		t.Errorf("output should mention both IDEs, got: %q", out)
	}
}

func TestSetup_OnlyOneIDE(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		IDE: config.IDEConfig{
			Cursor:     config.CursorIDEConfig{Enabled: true},
			ClaudeCode: config.ClaudeCodeIDEConfig{Enabled: false},
		},
	}
	var buf bytes.Buffer
	if err := Setup(&buf, dir, cfg); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".cursor", "commands")); err != nil {
		t.Errorf("expected .cursor/commands dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "commands")); !os.IsNotExist(err) {
		t.Error("expected .claude/commands to not exist when ClaudeCode is disabled")
	}
}

func TestSetup_NoIDEs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		IDE: config.IDEConfig{
			Cursor:     config.CursorIDEConfig{Enabled: false},
			ClaudeCode: config.ClaudeCodeIDEConfig{Enabled: false},
		},
	}
	var buf bytes.Buffer
	if err := Setup(&buf, dir, cfg); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if !strings.Contains(buf.String(), "No IDEs enabled") {
		t.Errorf("expected 'No IDEs enabled' message, got: %q", buf.String())
	}
}
