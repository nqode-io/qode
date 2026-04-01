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

	if raw["qode_version"] != "dev" {
		t.Errorf("expected qode_version \"dev\", got %v", raw["qode_version"])
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

func TestRunInitExisting_CreatesScoringYaml(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	if _, err := os.Stat(scoringPath); os.IsNotExist(err) {
		t.Error(".qode/scoring.yaml not created on first run")
	}
}

func TestRunInitExisting_RerunPreservesScoringYaml(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(dir); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}

	// Record scoring.yaml content after first run.
	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	firstScoring, err := os.ReadFile(scoringPath)
	if err != nil {
		t.Fatalf("reading scoring.yaml after first run: %v", err)
	}

	// Second run must succeed and must not overwrite .qode/scoring.yaml.
	if err := runInitExisting(dir); err != nil {
		t.Fatalf("second runInitExisting: %v", err)
	}

	secondScoring, err := os.ReadFile(scoringPath)
	if err != nil {
		t.Fatalf("reading scoring.yaml after second run: %v", err)
	}

	if string(firstScoring) != string(secondScoring) {
		t.Error(".qode/scoring.yaml was overwritten on re-run")
	}
}

func TestRootCmd_NoIDESubcommand(t *testing.T) {
	// Confirm 'ide' is not a registered subcommand.
	ideCmd, _, findErr := rootCmd.Find([]string{"ide"})
	if findErr == nil && ideCmd != rootCmd {
		t.Error("'ide' subcommand must not be registered on rootCmd")
	}
}
