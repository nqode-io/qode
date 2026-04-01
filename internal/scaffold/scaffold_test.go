package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- claudeSlashCommands ---

func TestClaudeSlashCommands_ContainsTicketFetch(t *testing.T) {
	cmds := claudeSlashCommands("testproject")
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("claudeSlashCommands: missing key qode-ticket-fetch")
	}
	if content != "!qode ticket fetch $ARGUMENTS" {
		t.Errorf("qode-ticket-fetch content = %q, want %q", content, "!qode ticket fetch $ARGUMENTS")
	}
}

func TestClaudeSlashCommands_HasNineEntries(t *testing.T) {
	cmds := claudeSlashCommands("testproject")
	if len(cmds) != 9 {
		t.Errorf("claudeSlashCommands: len = %d, want 9", len(cmds))
	}
}

func TestClaudeSlashCommands_IncludesKnowledge(t *testing.T) {
	cmds := claudeSlashCommands("testproject")
	for _, key := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		content, ok := cmds[key]
		if !ok {
			t.Errorf("claudeSlashCommands: missing key %s", key)
			continue
		}
		if content == "" {
			t.Errorf("claudeSlashCommands: %s has empty content", key)
		}
	}
}

func TestClaudeSlashCommands_IncludesQodeCheck(t *testing.T) {
	cmds := claudeSlashCommands("testproject")
	content, ok := cmds["qode-check"]
	if !ok {
		t.Fatal("claudeSlashCommands: missing key qode-check")
	}
	if content == "" {
		t.Error("claudeSlashCommands: qode-check has empty content")
	}
	// The prompt must not instruct the AI to use config fields from qode.yaml.
	for _, prohibited := range []string{"test.unit", "test.lint"} {
		if strings.Contains(content, prohibited) {
			t.Errorf("claudeSlashCommands: qode-check must not reference qode.yaml field %q", prohibited)
		}
	}
	for _, heading := range []string{"Phase 1", "Phase 2"} {
		if !strings.Contains(content, heading) {
			t.Errorf("claudeSlashCommands: qode-check missing heading %q", heading)
		}
	}
}

func TestClaudeSlashCommands_NoPromptOnly(t *testing.T) {
	cmds := claudeSlashCommands("testproject")
	for name, content := range cmds {
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("claudeSlashCommands: %s contains --prompt-only", name)
		}
	}
}

// --- slashCommands (Cursor) ---

func TestCursorSlashCommands_ContainsTicketFetch(t *testing.T) {
	cmds := slashCommands("testproject")
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("slashCommands: missing key qode-ticket-fetch")
	}
	if !strings.Contains(content, "qode ticket fetch $ARGUMENTS") {
		t.Errorf("qode-ticket-fetch content missing command, got:\n%s", content)
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("qode-ticket-fetch content missing YAML frontmatter description, got:\n%s", content)
	}
	if !strings.Contains(content, "testproject") {
		t.Errorf("qode-ticket-fetch content missing project name %q, got:\n%s", "testproject", content)
	}
}

func TestCursorSlashCommands_HasNineEntries(t *testing.T) {
	cmds := slashCommands("testproject")
	if len(cmds) != 9 {
		t.Errorf("slashCommands: len = %d, want 9", len(cmds))
	}
}

func TestCursorSlashCommands_IncludesKnowledge(t *testing.T) {
	cmds := slashCommands("testproject")
	for _, key := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		content, ok := cmds[key]
		if !ok {
			t.Errorf("slashCommands: missing key %s", key)
			continue
		}
		if !strings.Contains(content, "description:") {
			t.Errorf("slashCommands: %s missing YAML frontmatter", key)
		}
	}
}

func TestCursorSlashCommands_IncludesQodeCheck(t *testing.T) {
	cmds := slashCommands("testproject")
	content, ok := cmds["qode-check"]
	if !ok {
		t.Fatal("slashCommands: missing key qode-check")
	}
	if !strings.Contains(content, "description:") {
		t.Error("slashCommands: qode-check missing YAML frontmatter")
	}
	// The prompt must not instruct the AI to use config fields from qode.yaml.
	for _, prohibited := range []string{"test.unit", "test.lint"} {
		if strings.Contains(content, prohibited) {
			t.Errorf("slashCommands: qode-check must not reference qode.yaml field %q", prohibited)
		}
	}
	for _, heading := range []string{"Phase 1", "Phase 2"} {
		if !strings.Contains(content, heading) {
			t.Errorf("slashCommands: qode-check missing heading %q", heading)
		}
	}
}

func TestCursorSlashCommands_NoPromptOnly(t *testing.T) {
	cmds := slashCommands("testproject")
	for name, content := range cmds {
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("slashCommands: %s contains --prompt-only", name)
		}
	}
}

// --- SetupClaudeCode integration ---

func TestSetupClaudeCode_WritesTicketFetchCommand(t *testing.T) {
	dir := t.TempDir()
	if err := SetupClaudeCode(dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	path := filepath.Join(dir, ".claude", "commands", "qode-ticket-fetch.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading qode-ticket-fetch.md: %v", err)
	}
	if string(data) != "!qode ticket fetch $ARGUMENTS" {
		t.Errorf("qode-ticket-fetch.md content = %q, want %q", string(data), "!qode ticket fetch $ARGUMENTS")
	}
}

func TestSetupClaudeCode_WritesKnowledgeCommands(t *testing.T) {
	dir := t.TempDir()
	if err := SetupClaudeCode(dir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	for _, name := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		path := filepath.Join(dir, ".claude", "commands", name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s.md: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s.md is empty", name)
		}
	}
}

func TestSetupClaudeCode_CommandsContainRootName(t *testing.T) {
	dir := t.TempDir()
	// Rename the temp dir would be complex; use a subdirectory with a known name.
	projectDir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := SetupClaudeCode(projectDir); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	path := filepath.Join(projectDir, ".claude", "commands", "qode-plan-refine.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading qode-plan-refine.md: %v", err)
	}
	if !strings.Contains(string(data), "myproject") {
		t.Errorf("qode-plan-refine.md missing root dir name %q, got:\n%s", "myproject", string(data))
	}
}

// --- SetupCursor integration ---

func TestSetupCursor_WritesTicketFetchCommand(t *testing.T) {
	dir := t.TempDir()
	if err := SetupCursor(dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	path := filepath.Join(dir, ".cursor", "commands", "qode-ticket-fetch.mdc")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading qode-ticket-fetch.mdc: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "qode ticket fetch $ARGUMENTS") {
		t.Errorf("qode-ticket-fetch.mdc missing command, got:\n%s", content)
	}
}

func TestSetupCursor_WritesKnowledgeCommands(t *testing.T) {
	dir := t.TempDir()
	if err := SetupCursor(dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	for _, name := range []string{"qode-knowledge-add-context", "qode-knowledge-add-branch"} {
		path := filepath.Join(dir, ".cursor", "commands", name+".mdc")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s.mdc: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s.mdc is empty", name)
		}
	}
}

func TestSetupCursor_NoCursorRulesDir(t *testing.T) {
	dir := t.TempDir()
	if err := SetupCursor(dir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".cursorrules")); !os.IsNotExist(err) {
		t.Error("SetupCursor must not create .cursorrules/ directory")
	}
}

func TestSetupCursor_CommandsContainRootName(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := SetupCursor(projectDir); err != nil {
		t.Fatalf("SetupCursor: %v", err)
	}

	path := filepath.Join(projectDir, ".cursor", "commands", "qode-plan-refine.mdc")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading qode-plan-refine.mdc: %v", err)
	}
	if !strings.Contains(string(data), "myproject") {
		t.Errorf("qode-plan-refine.mdc missing root dir name %q, got:\n%s", "myproject", string(data))
	}
}
