package ide

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
)

func minimalConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Project.Name = "testproject"
	cfg.Project.Topology = "single"
	cfg.IDE.ClaudeCode.Enabled = true
	cfg.IDE.ClaudeCode.SlashCommands = true
	cfg.IDE.Cursor.Enabled = true
	return cfg
}

// --- claudeSlashCommands ---

func TestClaudeSlashCommands_ContainsTicketFetch(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("claudeSlashCommands: missing key qode-ticket-fetch")
	}
	if content != "!qode ticket fetch $ARGUMENTS" {
		t.Errorf("qode-ticket-fetch content = %q, want %q", content, "!qode ticket fetch $ARGUMENTS")
	}
}

func TestClaudeSlashCommands_HasEightEntries(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	if len(cmds) != 8 {
		t.Errorf("claudeSlashCommands: len = %d, want 8", len(cmds))
	}
}

func TestClaudeSlashCommands_IncludesKnowledge(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
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

func TestClaudeSlashCommands_NoPromptOnly(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	for name, content := range cmds {
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("claudeSlashCommands: %s contains --prompt-only", name)
		}
	}
}

// --- slashCommands (Cursor) ---

func TestCursorSlashCommands_ContainsTicketFetch(t *testing.T) {
	cfg := minimalConfig()
	cmds := slashCommands(cfg)
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
	if !strings.Contains(content, cfg.Project.Name) {
		t.Errorf("qode-ticket-fetch content missing project name %q, got:\n%s", cfg.Project.Name, content)
	}
}

func TestCursorSlashCommands_HasEightEntries(t *testing.T) {
	cmds := slashCommands(minimalConfig())
	if len(cmds) != 8 {
		t.Errorf("slashCommands: len = %d, want 8", len(cmds))
	}
}

func TestCursorSlashCommands_IncludesKnowledge(t *testing.T) {
	cmds := slashCommands(minimalConfig())
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

func TestCursorSlashCommands_NoPromptOnly(t *testing.T) {
	cmds := slashCommands(minimalConfig())
	for name, content := range cmds {
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("slashCommands: %s contains --prompt-only", name)
		}
	}
}

// --- SetupClaudeCode integration ---

func TestSetupClaudeCode_WritesTicketFetchCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := minimalConfig()

	if err := SetupClaudeCode(dir, cfg); err != nil {
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
	cfg := minimalConfig()

	if err := SetupClaudeCode(dir, cfg); err != nil {
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

// --- SetupCursor integration ---

func TestSetupCursor_WritesTicketFetchCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := minimalConfig()

	if err := SetupCursor(dir, cfg); err != nil {
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
	cfg := minimalConfig()

	if err := SetupCursor(dir, cfg); err != nil {
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
