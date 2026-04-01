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
	cfg.IDE.Cursor.Enabled = true
	return cfg
}

func minimalMCPConfig() *config.Config {
	cfg := minimalConfig()
	cfg.TicketSystem.Mode = "mcp"
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

func TestClaudeSlashCommands_HasNineEntries(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	if len(cmds) != 9 {
		t.Errorf("claudeSlashCommands: len = %d, want 9", len(cmds))
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

func TestClaudeSlashCommands_IncludesQodeCheck(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
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

func TestCursorSlashCommands_HasNineEntries(t *testing.T) {
	cmds := slashCommands(minimalConfig())
	if len(cmds) != 9 {
		t.Errorf("slashCommands: len = %d, want 9", len(cmds))
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

func TestCursorSlashCommands_IncludesQodeCheck(t *testing.T) {
	cfg := minimalConfig()
	cmds := slashCommands(cfg)
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
	cmds := slashCommands(minimalConfig())
	for name, content := range cmds {
		if strings.Contains(content, "--prompt-only") {
			t.Errorf("slashCommands: %s contains --prompt-only", name)
		}
	}
}

// --- SetupClaudeCode integration ---

// --- MCP mode slash commands ---

func TestClaudeSlashCommands_MCPMode_TicketFetchIsPrompt(t *testing.T) {
	cmds := claudeSlashCommands(minimalMCPConfig())
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("claudeSlashCommands: missing key qode-ticket-fetch")
	}
	if strings.HasPrefix(content, "!") {
		t.Error("claudeSlashCommands: qode-ticket-fetch in mcp mode must not start with '!'")
	}
	if !strings.Contains(content, "$ARGUMENTS") {
		t.Error("claudeSlashCommands: qode-ticket-fetch in mcp mode must contain $ARGUMENTS")
	}
	if !strings.Contains(content, "context/ticket.md") {
		t.Error("claudeSlashCommands: qode-ticket-fetch in mcp mode must reference context/ticket.md")
	}
}

func TestClaudeSlashCommands_APIMode_TicketFetchIsShellCmd(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("claudeSlashCommands: missing key qode-ticket-fetch")
	}
	if content != "!qode ticket fetch $ARGUMENTS" {
		t.Errorf("claudeSlashCommands: api mode qode-ticket-fetch = %q, want %q", content, "!qode ticket fetch $ARGUMENTS")
	}
}

func TestCursorSlashCommands_MCPMode_TicketFetchIsPrompt(t *testing.T) {
	cmds := slashCommands(minimalMCPConfig())
	content, ok := cmds["qode-ticket-fetch"]
	if !ok {
		t.Fatal("slashCommands: missing key qode-ticket-fetch")
	}
	if !strings.Contains(content, "$ARGUMENTS") {
		t.Error("slashCommands: qode-ticket-fetch in mcp mode must contain $ARGUMENTS")
	}
	if !strings.Contains(content, "context/ticket.md") {
		t.Error("slashCommands: qode-ticket-fetch in mcp mode must reference context/ticket.md")
	}
	if !strings.Contains(content, "description:") {
		t.Error("slashCommands: qode-ticket-fetch in mcp mode must have YAML frontmatter description")
	}
	if strings.Contains(content, "qode ticket fetch") {
		t.Error("slashCommands: qode-ticket-fetch in mcp mode must not reference qode ticket fetch CLI")
	}
}

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
