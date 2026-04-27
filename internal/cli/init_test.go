//go:build !integration

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scaffold"
	"gopkg.in/yaml.v3"
)

func TestRunInitExisting_WritesQodeVersion(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
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

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	for _, sub := range []string{"contexts", "knowledge", "prompts"} {
		path := filepath.Join(dir, ".qode", sub)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf(".qode/%s/ not created", sub)
		}
	}
}

func TestRunInitExisting_CopiesTemplates(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
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
	embedded, _ := prompt.EmbeddedTemplates()
	// Only non-scaffold templates are copied to .qode/prompts/.
	wantTemplates := 0
	for name := range embedded {
		if !strings.HasPrefix(name, "scaffold/") {
			wantTemplates++
		}
	}
	if total != wantTemplates {
		t.Errorf("expected %d .md.tmpl files under .qode/prompts/, got %d", wantTemplates, total)
	}
}

func TestRunInitExisting_CreatesIDEConfigs(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
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

	codexPath := filepath.Join(dir, ".agents", "skills", "qode-plan-refine", "SKILL.md")
	if _, err := os.Stat(codexPath); os.IsNotExist(err) {
		t.Error(".agents/skills/qode-plan-refine/SKILL.md not created")
	}
	if _, err := os.Stat(filepath.Join(dir, ".codex", "commands")); !os.IsNotExist(err) {
		t.Error("legacy .codex/commands directory should not be created")
	}
}

func TestRunInitExisting_NoCursorRules(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	// Positive: .qode/prompts/ must exist.
	if _, err := os.Stat(filepath.Join(dir, ".qode", "prompts")); err != nil {
		t.Errorf("expected .qode/prompts/ to exist: %v", err)
	}
	// Negative: legacy .cursorrules/ must not be created.
	if _, err := os.Stat(filepath.Join(dir, ".cursorrules")); !os.IsNotExist(err) {
		t.Error("runInitExisting must not create .cursorrules/ directory")
	}
}

func TestRunInitExisting_NoDetectionOutput(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	if err := runInitExisting(&buf, dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	// The writer must not contain legacy detection phrases.
	out := buf.String()
	for _, forbidden := range []string{"Detected", "Scanning", "qode ide setup"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("output must not contain %q, got: %s", forbidden, out)
		}
	}

	// Sanity-check that the writer received expected content.
	if !strings.Contains(out, "Generated:") {
		t.Errorf("expected 'Generated:' in output, got: %s", out)
	}
}

func TestRunInitExisting_CreatesScoringYaml(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	if _, err := os.Stat(scoringPath); os.IsNotExist(err) {
		t.Error(".qode/scoring.yaml not created on first run")
	}
}

func TestRunInitExisting_RerunPreservesScoringYaml(t *testing.T) {
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}

	// Record scoring.yaml content after first run.
	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	firstScoring, err := os.ReadFile(scoringPath)
	if err != nil {
		t.Fatalf("reading scoring.yaml after first run: %v", err)
	}

	// Second run must succeed and must not overwrite .qode/scoring.yaml.
	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
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

func TestRunInitExisting_AppendsGitignoreRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)

	for _, rule := range scaffold.GitignoreRules {
		if !strings.Contains(content, rule) {
			t.Errorf(".gitignore missing rule %q", rule)
		}
	}
}

func TestRunInitExisting_GitignoreIsIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}
	if err := runInitExisting(&bytes.Buffer{}, dir); err != nil {
		t.Fatalf("second runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)

	for _, rule := range scaffold.GitignoreRules {
		if count := strings.Count(content, rule); count != 1 {
			t.Errorf("rule %q appears %d times, want 1", rule, count)
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
