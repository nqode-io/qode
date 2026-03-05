package ide

import (
	"encoding/json"
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
	cfg.IDE.ClaudeCode.ClaudeMD = false
	cfg.IDE.Cursor.Enabled = true
	cfg.IDE.VSCode.Enabled = true
	cfg.IDE.VSCode.Tasks = true
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

// --- buildTasksJSON ---

func TestBuildTasksJSON_ContainsTicketFetchTask(t *testing.T) {
	result := buildTasksJSON(minimalConfig())

	tasks, ok := result["tasks"].([]map[string]interface{})
	if !ok {
		t.Fatal("buildTasksJSON: tasks not []map[string]interface{}")
	}

	var found bool
	for _, task := range tasks {
		if task["label"] == "qode: fetch ticket" {
			found = true
			if task["command"] != "qode ticket fetch ${input:ticketUrl}" {
				t.Errorf("fetch ticket command = %q, want %q", task["command"], "qode ticket fetch ${input:ticketUrl}")
			}
			if task["type"] != "shell" {
				t.Errorf("fetch ticket type = %q, want %q", task["type"], "shell")
			}
		}
	}
	if !found {
		t.Error("buildTasksJSON: task 'qode: fetch ticket' not found")
	}
}

func TestBuildTasksJSON_ContainsInputsEntry(t *testing.T) {
	result := buildTasksJSON(minimalConfig())

	inputs, ok := result["inputs"].([]map[string]interface{})
	if !ok {
		t.Fatal("buildTasksJSON: inputs not []map[string]interface{}")
	}
	if len(inputs) == 0 {
		t.Fatal("buildTasksJSON: inputs is empty")
	}

	found := false
	for _, inp := range inputs {
		if inp["id"] == "ticketUrl" {
			found = true
			if inp["type"] != "promptString" {
				t.Errorf("ticketUrl input type = %q, want %q", inp["type"], "promptString")
			}
		}
	}
	if !found {
		t.Error("buildTasksJSON: input with id 'ticketUrl' not found")
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

func TestSetupClaudeCode_DoesNotOverwriteExistingClaudeMD(t *testing.T) {
	dir := t.TempDir()
	cfg := minimalConfig()
	cfg.IDE.ClaudeCode.ClaudeMD = true

	claudeMDPath := filepath.Join(dir, "CLAUDE.md")
	existingContent := "# Custom CLAUDE.md\n\nUser-maintained content.\n"
	if err := os.WriteFile(claudeMDPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("writing existing CLAUDE.md: %v", err)
	}

	if err := SetupClaudeCode(dir, cfg); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, err := os.ReadFile(claudeMDPath)
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if string(data) != existingContent {
		t.Errorf("CLAUDE.md was overwritten.\ngot:\n%s\nwant:\n%s", string(data), existingContent)
	}
}

func TestSetupClaudeCode_CreatesClaudeMDWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg := minimalConfig()
	cfg.IDE.ClaudeCode.ClaudeMD = true

	if err := SetupClaudeCode(dir, cfg); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	claudeMDPath := filepath.Join(dir, "CLAUDE.md")
	data, err := os.ReadFile(claudeMDPath)
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(data), cfg.Project.Name) {
		t.Errorf("CLAUDE.md missing project name %q, got:\n%s", cfg.Project.Name, string(data))
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

// --- SetupVSCode integration ---

func TestSetupVSCode_WritesTicketFetchTask(t *testing.T) {
	dir := t.TempDir()
	cfg := minimalConfig()

	if err := SetupVSCode(dir, cfg); err != nil {
		t.Fatalf("SetupVSCode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".vscode", "tasks.json"))
	if err != nil {
		t.Fatalf("reading tasks.json: %v", err)
	}

	var tasks map[string]interface{}
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("parsing tasks.json: %v", err)
	}

	taskList, ok := tasks["tasks"].([]interface{})
	if !ok {
		t.Fatal("tasks.json: tasks field not an array")
	}

	foundTask := false
	for _, item := range taskList {
		task, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if task["label"] == "qode: fetch ticket" {
			foundTask = true
			if task["command"] != "qode ticket fetch ${input:ticketUrl}" {
				t.Errorf("task command = %q, want %q", task["command"], "qode ticket fetch ${input:ticketUrl}")
			}
		}
	}
	if !foundTask {
		t.Error("tasks.json: 'qode: fetch ticket' task not found")
	}

	inputs, ok := tasks["inputs"].([]interface{})
	if !ok {
		t.Fatal("tasks.json: inputs field not an array")
	}

	foundInput := false
	for _, item := range inputs {
		inp, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if inp["id"] == "ticketUrl" {
			foundInput = true
		}
	}
	if !foundInput {
		t.Error("tasks.json: input 'ticketUrl' not found")
	}
}
