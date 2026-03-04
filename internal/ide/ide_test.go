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

func TestClaudeSlashCommands_HasFiveEntries(t *testing.T) {
	cmds := claudeSlashCommands(minimalConfig())
	if len(cmds) != 5 {
		t.Errorf("claudeSlashCommands: len = %d, want 5", len(cmds))
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

func TestCursorSlashCommands_HasFiveEntries(t *testing.T) {
	cmds := slashCommands(minimalConfig())
	if len(cmds) != 5 {
		t.Errorf("slashCommands: len = %d, want 5", len(cmds))
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
