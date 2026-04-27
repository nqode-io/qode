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

func assertNoteAddPrompt(t *testing.T, content string) {
	t.Helper()

	for _, want := range []string{
		"Treat all text after this command or skill invocation as note content.",
		"single line or multiple paragraphs",
		"`end note`",
		".qode/contexts/current/notes.md",
		"Append only the new notes",
		"Do not overwrite existing notes",
		"No currently active qode context.",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("note-add content missing %q", want)
		}
	}

	for _, banned := range []string{"$ARGUMENTS", "<text>"} {
		if strings.Contains(content, banned) {
			t.Errorf("note-add content must not contain %q", banned)
		}
	}
}

// --- SetupClaudeCode ---

func TestSetupClaudeCode_WritesTicketFetchCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	content := readClaudeCommand(t, dir, "qode-ticket-fetch")
	for _, want := range []string{"$ARGUMENTS", "ticket.md", "MCP", "Figma"} {
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
	if len(entries) != len(qodeWorkflows) {
		t.Errorf("SetupClaudeCode: wrote %d commands, want %d", len(entries), len(qodeWorkflows))
	}
}

func TestSetupClaudeCode_WritesKnowledgeCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	content := readClaudeCommand(t, dir, "qode-knowledge-add-context")
	if len(content) == 0 {
		t.Error("qode-knowledge-add-context.md is empty")
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

func TestSetupClaudeCode_WritesPrResolveCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}
	content := readClaudeCommand(t, dir, "qode-pr-resolve")
	for _, want := range []string{"MCP", "confirm", "Do NOT commit"} {
		if !strings.Contains(content, want) {
			t.Errorf("qode-pr-resolve.md missing %q", want)
		}
	}
	for _, banned := range []string{"gh ", "git push", "git commit"} {
		if strings.Contains(content, banned) {
			t.Errorf("qode-pr-resolve.md must not contain %q", banned)
		}
	}
}

func TestSetupClaudeCode_WritesNoteAddCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	assertNoteAddPrompt(t, readClaudeCommand(t, dir, "qode-note-add"))
}

func TestSetupCursor_WritesPrResolveCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}
	content := readCursorCommand(t, dir, "qode-pr-resolve")
	for _, want := range []string{"description:", "MCP", "confirm", "Do NOT commit"} {
		if !strings.Contains(content, want) {
			t.Errorf("qode-pr-resolve.mdc missing %q", want)
		}
	}
}

func TestSetupCursor_WritesNoteAddCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	assertNoteAddPrompt(t, readCursorCommand(t, dir, "qode-note-add"))
}

func TestSetupClaudeCode_CommandContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	for _, workflow := range qodeWorkflows {
		content := readClaudeCommand(t, dir, workflow.Name)
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("%s.md contains --prompt-only", workflow.Name)
		}
		// Each command must render a non-trivial prompt body.
		if len(content) < 50 {
			t.Errorf("%s.md too short (%d bytes), expected rendered command content", workflow.Name, len(content))
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
	for _, want := range []string{"$ARGUMENTS", "ticket.md", "description:", "MCP", "Figma"} {
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
	if len(entries) != len(qodeWorkflows) {
		t.Errorf("SetupCursor: wrote %d commands, want %d", len(entries), len(qodeWorkflows))
	}
}

func TestSetupCursor_WritesKnowledgeCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	content := readCursorCommand(t, dir, "qode-knowledge-add-context")
	if !strings.Contains(content, "description:") {
		t.Error("qode-knowledge-add-context.mdc missing YAML frontmatter")
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

func TestSetupCursor_CommandContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	for _, workflow := range qodeWorkflows {
		content := readCursorCommand(t, dir, workflow.Name)
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("%s.mdc contains --prompt-only", workflow.Name)
		}
		if len(content) < 50 {
			t.Errorf("%s.mdc too short (%d bytes), expected rendered command content", workflow.Name, len(content))
		}
	}
}

func TestSetupCursor_NoCursorRulesDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	// Positive: .cursor/commands/ must exist.
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "commands")); err != nil {
		t.Errorf("expected .cursor/commands/ to exist: %v", err)
	}
	// Negative: legacy .cursorrules/ must not be created.
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

func TestSetup_AllThreeIDEs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		IDE: config.IDEConfig{
			Cursor:     config.CursorIDEConfig{Enabled: true},
			ClaudeCode: config.ClaudeCodeIDEConfig{Enabled: true},
			Codex:      config.CodexIDEConfig{Enabled: true},
		},
	}
	var buf bytes.Buffer
	if err := Setup(&buf, dir, cfg); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(dir, ".cursor", "commands"),
		filepath.Join(dir, ".claude", "commands"),
		filepath.Join(dir, ".agents", "skills"),
	} {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("expected %s to exist: %v", dir, err)
		}
	}
	out := buf.String()
	for _, ide := range []string{"Cursor", "Claude Code", "Codex"} {
		if !strings.Contains(out, ide) {
			t.Errorf("output should mention %q, got: %q", ide, out)
		}
	}
}

func TestSetup_OnlyCodex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{
		IDE: config.IDEConfig{
			Cursor:     config.CursorIDEConfig{Enabled: false},
			ClaudeCode: config.ClaudeCodeIDEConfig{Enabled: false},
			Codex:      config.CodexIDEConfig{Enabled: true},
		},
	}
	if err := Setup(io.Discard, dir, cfg); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".agents", "skills")); err != nil {
		t.Errorf("expected .agents/skills dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".codex", "commands")); !os.IsNotExist(err) {
		t.Error("expected legacy .codex/commands to be absent after Codex setup")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "commands")); !os.IsNotExist(err) {
		t.Error("expected .claude/commands to not exist when ClaudeCode is disabled")
	}
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "commands")); !os.IsNotExist(err) {
		t.Error("expected .cursor/commands to not exist when Cursor is disabled")
	}
}

func TestSetupClaudeCode_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// First run.
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("first SetupClaudeCode: %v", err)
	}
	firstContents := readAllCommands(t, filepath.Join(dir, ".claude", "commands"))

	// Second run.
	if err := SetupClaudeCode(io.Discard, dir); err != nil {
		t.Fatalf("second SetupClaudeCode: %v", err)
	}
	secondContents := readAllCommands(t, filepath.Join(dir, ".claude", "commands"))

	if len(firstContents) != len(secondContents) {
		t.Fatalf("file count changed: %d → %d", len(firstContents), len(secondContents))
	}
	for name, first := range firstContents {
		second, ok := secondContents[name]
		if !ok {
			t.Errorf("file %q missing after second run", name)
			continue
		}
		if first != second {
			t.Errorf("file %q content changed after second run", name)
		}
	}
}

func TestSetupCursor_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("first SetupCursor: %v", err)
	}
	firstContents := readAllCommands(t, filepath.Join(dir, ".cursor", "commands"))

	if err := SetupCursor(io.Discard, dir); err != nil {
		t.Fatalf("second SetupCursor: %v", err)
	}
	secondContents := readAllCommands(t, filepath.Join(dir, ".cursor", "commands"))

	if len(firstContents) != len(secondContents) {
		t.Fatalf("file count changed: %d → %d", len(firstContents), len(secondContents))
	}
	for name, first := range firstContents {
		second, ok := secondContents[name]
		if !ok {
			t.Errorf("file %q missing after second run", name)
			continue
		}
		if first != second {
			t.Errorf("file %q content changed after second run", name)
		}
	}
}

// readCodexSkill reads a single Codex skill file from <root>/.agents/skills/<name>/SKILL.md.
func readCodexSkill(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, ".agents", "skills", name, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s SKILL.md: %v", name, err)
	}
	return string(data)
}

func readCodexSkillMetadata(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, ".agents", "skills", name, "agents", "openai.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s openai.yaml: %v", name, err)
	}
	return string(data)
}

// --- SetupCodex ---

func TestSetupCodex_WritesAllSkills(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCodex(io.Discard, dir); err != nil {
		t.Fatalf("SetupCodex: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".agents", "skills"))
	if err != nil {
		t.Fatalf("reading skills dir: %v", err)
	}
	if len(entries) != len(qodeWorkflows) {
		t.Errorf("SetupCodex: wrote %d skills, want %d", len(entries), len(qodeWorkflows))
	}
	for _, workflow := range qodeWorkflows {
		if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", workflow.Name, "SKILL.md")); err != nil {
			t.Errorf("missing SKILL.md for %s: %v", workflow.Name, err)
		}
	}
}

func TestSetupCodex_CommandContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCodex(io.Discard, dir); err != nil {
		t.Fatalf("SetupCodex: %v", err)
	}

	for _, workflow := range qodeWorkflows {
		workflow := workflow
		t.Run(workflow.Name, func(t *testing.T) {
			t.Parallel()
			content := readCodexSkill(t, dir, workflow.Name)
			if len(content) < 50 {
				t.Errorf("%s SKILL.md too short (%d bytes)", workflow.Name, len(content))
			}
			if strings.Contains(content, "--prompt-only") {
				t.Errorf("%s SKILL.md contains --prompt-only", workflow.Name)
			}
			if strings.Contains(content, "AskUserQuestion") {
				t.Errorf("%s SKILL.md contains AskUserQuestion (not supported by Codex)", workflow.Name)
			}
			for _, want := range []string{
				`name: "` + workflow.Name + `"`,
				`description: "`,
			} {
				if !strings.Contains(content, want) {
					t.Errorf("%s SKILL.md missing %q", workflow.Name, want)
				}
			}
			metadata := readCodexSkillMetadata(t, dir, workflow.Name)
			if !strings.Contains(metadata, "allow_implicit_invocation: false") {
				t.Errorf("%s openai.yaml must disable implicit invocation", workflow.Name)
			}
		})
	}
}

func TestSetupCodex_WritesNoteAddSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := SetupCodex(io.Discard, dir); err != nil {
		t.Fatalf("SetupCodex: %v", err)
	}

	assertNoteAddPrompt(t, readCodexSkill(t, dir, "qode-note-add"))

	metadata := readCodexSkillMetadata(t, dir, "qode-note-add")
	if !strings.Contains(metadata, "allow_implicit_invocation: false") {
		t.Error("qode-note-add openai.yaml must disable implicit invocation")
	}
}

func TestSetupCodex_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := SetupCodex(io.Discard, dir); err != nil {
		t.Fatalf("first SetupCodex: %v", err)
	}
	firstContents := readAllFilesRecursive(t, filepath.Join(dir, ".agents", "skills"))

	if err := SetupCodex(io.Discard, dir); err != nil {
		t.Fatalf("second SetupCodex: %v", err)
	}
	secondContents := readAllFilesRecursive(t, filepath.Join(dir, ".agents", "skills"))

	if len(firstContents) != len(secondContents) {
		t.Fatalf("file count changed: %d → %d", len(firstContents), len(secondContents))
	}
	for name, first := range firstContents {
		second, ok := secondContents[name]
		if !ok {
			t.Errorf("file %q missing after second run", name)
			continue
		}
		if first != second {
			t.Errorf("file %q content changed after second run", name)
		}
	}
}

// readAllCommands reads every file in dir and returns a map of name → content.
func readAllCommands(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	result := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", e.Name(), err)
		}
		result[e.Name()] = string(data)
	}
	return result
}

func readAllFilesRecursive(t *testing.T, dir string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		result[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk %s: %v", dir, err)
	}
	return result
}
