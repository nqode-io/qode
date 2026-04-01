package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRunInitExisting_WritesQodeVersion(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "qode.yaml"))
	if err != nil {
		t.Fatalf("reading qode.yaml: %v", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshaling qode.yaml: %v", err)
	}

	if raw["qode_version"] != "0.1" {
		t.Errorf("expected qode_version \"0.1\", got %v", raw["qode_version"])
	}
	if _, ok := raw["project"]; ok {
		t.Error("qode.yaml must not contain a 'project' key")
	}
}

func TestRunInitExisting_CreatesDirs(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	for _, sub := range []string{"branches", "knowledge", "prompts"} {
		path := filepath.Join(dir, ".qode", sub)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf(".qode/%s/ not created", sub)
		}
	}
}

func TestRunInitExisting_CopiesTemplates(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	promptsDir := filepath.Join(dir, ".qode", "prompts")
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		t.Fatalf("reading .qode/prompts/: %v", err)
	}

	var tmplCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md.tmpl") || e.IsDir() {
			tmplCount++
		}
	}
	// Walk subdirectories too.
	var total int
	if err := filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md.tmpl") {
			total++
		}
		return nil
	}); err != nil {
		t.Fatalf("walking .qode/prompts/: %v", err)
	}
	_ = tmplCount
	if total == 0 {
		t.Error("expected at least one .md.tmpl file under .qode/prompts/")
	}
}

func TestRunInitExisting_CreatesIDEConfigs(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	claudePath := filepath.Join(dir, ".claude", "commands", "qode-plan-refine.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Error(".claude/commands/qode-plan-refine.md not created")
	}

	cursorPath := filepath.Join(dir, ".cursor", "commands", "qode-plan-refine.mdc")
	if _, err := os.Stat(cursorPath); os.IsNotExist(err) {
		t.Error(".cursor/commands/qode-plan-refine.mdc not created")
	}
}

func TestRunInitExisting_NoCursorRules(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".cursorrules")); !os.IsNotExist(err) {
		t.Error("runInitExisting must not create .cursorrules/ directory")
	}
}

func TestRunInitExisting_NoDetectionOutput(t *testing.T) {
	dir := t.TempDir()

	output := captureStdout(t, func() {
		if err := runInitExisting(dir); err != nil {
			t.Errorf("runInitExisting: %v", err)
		}
	})

	for _, forbidden := range []string{"Detected", "Scanning", "qode ide setup"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("output must not contain %q, got: %s", forbidden, output)
		}
	}
}

func TestRootCmd_NoIDESubcommand(t *testing.T) {
	// Confirm 'ide' is not a registered subcommand.
	ideCmd, _, findErr := rootCmd.Find([]string{"ide"})
	if findErr == nil && ideCmd != rootCmd {
		t.Error("'ide' subcommand must not be registered on rootCmd")
	}
}
